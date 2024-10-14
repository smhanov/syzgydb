package replication

import (
    "log"
    "time"

    pb "github.com/smhanov/syzgydb/replication/proto"
)

type Event interface {
    process(sm *StateMachine)
}

type StateMachine struct {
    storage              StorageInterface
    config               ReplicationConfig
    peers                map[string]*Peer
    lastKnownVectorClock *VectorClock
    bufferedUpdates      map[string][]Update
    updateRequests       map[string]*updateRequest
    eventChan            chan Event
    done                 chan struct{}
}

func (sm *StateMachine) getPeerURLs() []string {
    urls := make([]string, 0, len(sm.peers))
    for url := range sm.peers {
        urls = append(urls, url)
    }
    return urls
}

func NewStateMachine(storage StorageInterface, config ReplicationConfig, localVectorClock *VectorClock) *StateMachine {
    sm := &StateMachine{
        storage:              storage,
        config:               config,
        peers:                make(map[string]*Peer),
        lastKnownVectorClock: localVectorClock.Clone(),
        bufferedUpdates:      make(map[string][]Update),
        updateRequests:       make(map[string]*updateRequest),
        eventChan:            make(chan Event, 1000),
        done:                 make(chan struct{}),
    }

    go sm.eventLoop()
    return sm
}

func (sm *StateMachine) eventLoop() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case event := <-sm.eventChan:
            event.process(sm)
        case <-ticker.C:
            sm.processBufferedUpdates()
        case <-sm.done:
            return
        }
    }
}

func (sm *StateMachine) Stop() {
    close(sm.done)
}

func (sm *StateMachine) handleReceivedUpdate(update Update) {
    log.Printf("[%s] Received update: %+v", sm.config.OwnURL, update)
    if update.Type == CreateDatabase {
        err := sm.applyUpdateAndProcessBuffer(update)
        if err != nil {
            log.Println("Failed to apply CreateDatabase update:", err)
        } else {
            log.Printf("Successfully applied CreateDatabase update for %s", update.DatabaseName)
        }
    } else if sm.dependenciesSatisfied(update) {
        err := sm.applyUpdateAndProcessBuffer(update)
        if err != nil {
            log.Println("Failed to apply update:", err)
        } else {
            log.Printf("Successfully applied update: %+v", update)
        }
    } else {
        sm.bufferUpdate(update)
    }
}

func (sm *StateMachine) dependenciesSatisfied(update Update) bool {
    return update.Type == CreateDatabase || sm.storage.Exists(update.DatabaseName)
}

func (sm *StateMachine) bufferUpdate(update Update) {
    sm.bufferedUpdates[update.DatabaseName] = append(sm.bufferedUpdates[update.DatabaseName], update)
}

func (sm *StateMachine) applyUpdate(update Update) error {
    err := sm.storage.CommitUpdates([]Update{update})
    if err != nil {
        return err
    }

    sm.lastKnownVectorClock.Update(update.NodeID, update.Timestamp)
    return nil
}

func (sm *StateMachine) applyUpdateAndProcessBuffer(update Update) error {
    err := sm.applyUpdate(update)
    if err != nil {
        return err
    }

    sm.processBufferedUpdates()
    return nil
}

func (sm *StateMachine) processBufferedUpdates() {
    for depKey, buffered := range sm.bufferedUpdates {
        var remainingUpdates []Update
        for _, bufferedUpdate := range buffered {
            if sm.dependenciesSatisfied(bufferedUpdate) {
                err := sm.applyUpdate(bufferedUpdate)
                if err != nil {
                    log.Println("Failed to apply buffered update:", err)
                    remainingUpdates = append(remainingUpdates, bufferedUpdate)
                }
            } else {
                remainingUpdates = append(remainingUpdates, bufferedUpdate)
            }
        }
        if len(remainingUpdates) > 0 {
            sm.bufferedUpdates[depKey] = remainingUpdates
        } else {
            delete(sm.bufferedUpdates, depKey)
        }
    }
}

func (sm *StateMachine) handleGossipMessage(peer *Peer, msg *pb.GossipMessage) {
    sm.updatePeerList(msg.KnownPeers)
    peerVectorClock := fromProtoVectorClock(msg.LastVectorClock)

    peer.lastKnownVectorClock = peerVectorClock
    peer.name = msg.NodeId

    if sm.lastKnownVectorClock.Before(peerVectorClock) {
        sm.requestUpdatesFromPeer(peer.url)
    }
}

func (sm *StateMachine) updatePeerList(newPeers []string) {
    for _, url := range newPeers {
        if url != sm.config.OwnURL && sm.peers[url] == nil {
            sm.peers[url] = NewPeer("c:?", url, sm)
            go sm.peers[url].Connect(sm.config.JWTSecret)
        }
    }
}

func (sm *StateMachine) requestUpdatesFromPeer(peerURL string) {
    log.Printf("[%s] requesting updates from peer %s", sm.config.OwnURL, peerURL)

    peer, exists := sm.peers[peerURL]
    if !exists {
        log.Printf("Peer url:%s not found", peerURL)
        return
    }

    since := peer.lastKnownVectorClock

    if sm.updateRequests == nil {
        sm.updateRequests = make(map[string]*updateRequest)
    }

    if _, exists := sm.updateRequests[peerURL]; !exists {
        sm.updateRequests[peerURL] = &updateRequest{
            peerURL:      peerURL,
            since:        since,
            responseChan: make(chan bool),
        }
    }

    if !sm.updateRequests[peerURL].inProgress {
        sm.updateRequests[peerURL].inProgress = true
        go sm.fetchUpdatesFromPeer(peerURL)
    }
}

func (sm *StateMachine) fetchUpdatesFromPeer(peerURL string) {
    for {
        req, exists := sm.updateRequests[peerURL]
        if !exists {
            break
        }

        peer := sm.peers[peerURL]
        if peer == nil || !peer.IsConnected() {
            break
        }

        responseChan := make(chan bool)
        req.responseChan = responseChan

        err := peer.RequestUpdates(req.since, MaxUpdateResults, sm.NextTimestamp(false))
        if err != nil {
            log.Printf("Error requesting updates from peer %s: %v", peer.name, err)
            break
        }

        // Wait for the response
        hasMore := <-responseChan
        if !hasMore {
            break
        }
    }

    delete(sm.updateRequests, peerURL)
}

func (sm *StateMachine) handleReceivedBatchUpdate(peerURL string, batchUpdate *pb.BatchUpdate) {
    req, exists := sm.updateRequests[peerURL]
    peer, peerExists := sm.peers[peerURL]

    log.Printf("Received %d updates from peer url:%s (exists=%v)", len(batchUpdate.Updates), peerURL, exists)

    if !exists || !peerExists {
        log.Printf("[!] Peer url %s is not in the updateRequests map or peers map", peerURL)
        return
    }

    latestVectorClock := NewVectorClock()
    for _, protoUpdate := range batchUpdate.Updates {
        update := fromProtoUpdate(protoUpdate)
        sm.handleReceivedUpdate(update)
        latestVectorClock.Update(update.NodeID, update.Timestamp)
    }

    if latestVectorClock.After(peer.lastKnownVectorClock) {
        peer.lastKnownVectorClock = latestVectorClock.Clone()
    }

    req.since = latestVectorClock.Clone()

    // Signal the fetchUpdatesFromPeer goroutine
    req.responseChan <- batchUpdate.HasMore
}

func (sm *StateMachine) NextTimestamp(local bool) *VectorClock {
    // Increment the vector clock for this node
    nodeID := uint64(sm.config.NodeID)
    currentTimestamp, exists := sm.lastKnownVectorClock.Get(nodeID)
    if !exists {
        currentTimestamp = Timestamp{}
    }
    newTimestamp := currentTimestamp.Next(local)
    sm.lastKnownVectorClock.Update(nodeID, newTimestamp)

    // Return a copy of the updated vector clock
    return sm.lastKnownVectorClock.Clone()
}

func (sm *StateMachine) NextLocalTimestamp() Timestamp {
    cur, _ := sm.lastKnownVectorClock.Get(uint64(sm.config.NodeID))
    cur = cur.Next(true)
    sm.lastKnownVectorClock.Update(uint64(sm.config.NodeID), cur)
    return cur
}