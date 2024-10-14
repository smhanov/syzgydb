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

func (sm *StateMachine) handlePeerHeartbeat(peers []*Peer) {
	for _, peer := range peers {
		err := sm.sendHeartbeatToPeer(peer)
		if err != nil {
			log.Printf("Error sending heartbeat to peer %s: %v", peer.url, err)
			sm.eventChan <- PeerDisconnectedEvent{Peer: peer}
		}
	}
}

func (sm *StateMachine) sendHeartbeatToPeer(peer *Peer) error {
	heartbeat := &pb.Heartbeat{
		VectorClock: sm.lastKnownVectorClock.toProto(),
	}

	msg := &pb.Message{
		Type:        pb.Message_HEARTBEAT,
		VectorClock: sm.lastKnownVectorClock.toProto(),
		Content: &pb.Message_Heartbeat{
			Heartbeat: heartbeat,
		},
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error marshaling heartbeat: %w", err)
	}

	err = peer.connection.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		return fmt.Errorf("error sending heartbeat to peer %s: %w", peer.url, err)
	}

	return nil
}
