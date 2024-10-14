package replication

import (
	"log"

	pb "github.com/smhanov/syzgydb/replication/proto"
)

// GossipLoop runs the gossip protocol, periodically sending gossip messages to peers.
func (re *ReplicationEngine) GossipLoop() {
	for {
		select {
		case <-re.gossipTicker.C:
			ts := re.NextTimestamp(false)
			re.mu.Lock()
			for _, peer := range re.peers {
				if peer.IsConnected() {
					go re.sendGossipMessage(peer, ts)
				}
			}
			re.mu.Unlock()
		case <-re.gossipDone:
			re.gossipTicker.Stop()
			return
		}
	}
}

// sendGossipMessage sends a gossip message to a specific peer.
func (re *ReplicationEngine) sendGossipMessage(peer *Peer, vc *VectorClock) {
	re.mu.Lock()
	defer re.mu.Unlock()
	msg := &pb.GossipMessage{
		NodeId:           re.name,
		KnownPeers:       re.getPeerURLs(),
		LastVectorClock:  vc.toProto(),
	}
	err := peer.SendGossipMessage(msg, vc)
	if err != nil {
		log.Println("Failed to send gossip message:", err)
	}
}

// HandleGossipMessage processes a received gossip message, updating the peer list
// and requesting updates if necessary.
func (re *ReplicationEngine) HandleGossipMessage(peer *Peer, msg *pb.GossipMessage) {
	re.updatePeerList(msg.KnownPeers)
	peerVectorClock := fromProtoVectorClock(msg.LastVectorClock)

	peer.mu.Lock()
	if peerVectorClock.After(peer.lastKnownVectorClock) {
		peer.lastKnownVectorClock = peerVectorClock
	}
	peer.name = msg.NodeId
	peer.mu.Unlock()

	if re.lastKnownVectorClock.Before(peerVectorClock) {
		go re.requestUpdatesFromPeer(peer.url)
	}
}

// updatePeerList adds new peers to the ReplicationEngine's peer list.
func (re *ReplicationEngine) updatePeerList(newPeers []string) {
	re.mu.Lock()
	defer re.mu.Unlock()

	for _, url := range newPeers {
		if url != re.config.OwnURL && re.peers[url] == nil {
			re.peers[url] = NewPeer("o:?", url, re)
			go re.peers[url].Connect(re.config.JWTSecret)
		}
	}
}

// getPeerURLs returns a list of all known peer URLs.
func (re *ReplicationEngine) getPeerURLs() []string {
	urls := make([]string, 0, len(re.peers))
	for _, peer := range re.peers {
		urls = append(urls, peer.url)
	}
	return urls
}

// requestUpdatesFromPeer initiates an update request to a specific peer.
func (re *ReplicationEngine) requestUpdatesFromPeer(peerURL string) {
	re.mu.Lock()
	defer re.mu.Unlock()

	log.Printf("[%s] requesting updates from peer %s", re.name, peerURL)

	peer, exists := re.peers[peerURL]
	if !exists {
		log.Printf("Peer url:%s not found", peerURL)
		return
	}

	peer.mu.Lock()
	since := peer.lastKnownVectorClock
	peer.mu.Unlock()

	if re.updateRequests == nil {
		re.updateRequests = make(map[string]*updateRequest)
	}

	if _, exists := re.updateRequests[peerURL]; !exists {
		re.updateRequests[peerURL] = &updateRequest{
			peerURL:      peerURL,
			since:        since,
			responseChan: make(chan bool),
		}
	}

	if !re.updateRequests[peerURL].inProgress {
		re.updateRequests[peerURL].inProgress = true
		go re.fetchUpdatesFromPeer(peerURL)
	}
}

func (re *ReplicationEngine) fetchUpdatesFromPeer(peerURL string) {
	for {
		re.mu.Lock()
		req, exists := re.updateRequests[peerURL]
		if !exists {
			re.mu.Unlock()
			break
		}
		re.mu.Unlock()

		peer := re.peers[peerURL]
		if peer == nil || !peer.IsConnected() {
			break
		}

		responseChan := make(chan bool)
		req.responseChan = responseChan

		err := peer.RequestUpdates(req.since, MaxUpdateResults, re.NextTimestamp(false))
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

	re.mu.Lock()
	delete(re.updateRequests, peerURL)
	re.mu.Unlock()
}

func (re *ReplicationEngine) handleReceivedBatchUpdate(peerURL string, batchUpdate *pb.BatchUpdate) {
	re.mu.Lock()
	req, exists := re.updateRequests[peerURL]
	peer, peerExists := re.peers[peerURL]
	re.mu.Unlock()

	log.Printf("Received %d updates from peer url:%s (exists=%v)", len(batchUpdate.Updates), peerURL, exists)

	if !exists || !peerExists {
		log.Printf("[!] Peer url %s is not in the updateRequests map or peers map", peerURL)
		return
	}

	latestVectorClock := NewVectorClock()
	for _, protoUpdate := range batchUpdate.Updates {
		update := fromProtoUpdate(protoUpdate)
		re.handleReceivedUpdate(update)
		latestVectorClock.Merge(update.VectorClock)
	}

	peer.mu.Lock()
	if latestVectorClock.After(peer.lastKnownVectorClock) {
		peer.lastKnownVectorClock = latestVectorClock.Clone()
	}
	peer.mu.Unlock()

	req.since = latestVectorClock.Clone()

	// Signal the fetchUpdatesFromPeer goroutine
	req.responseChan <- batchUpdate.HasMore
}
