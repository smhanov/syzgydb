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
			re.mu.Lock()
			for _, peer := range re.peers {
				if peer.IsConnected() {
					go re.sendGossipMessage(peer)
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
func (re *ReplicationEngine) sendGossipMessage(peer *Peer) {
	msg := &pb.GossipMessage{
		NodeId:        re.config.OwnURL,
		KnownPeers:    re.getPeerURLs(),
		LastTimestamp: re.lastTimestamp.toProto(),
	}
	err := peer.SendGossipMessage(msg)
	if err != nil {
		log.Println("Failed to send gossip message:", err)
	}
}

// HandleGossipMessage processes a received gossip message, updating the peer list
// and requesting updates if necessary.
func (re *ReplicationEngine) HandleGossipMessage(msg *pb.GossipMessage) {
	re.updatePeerList(msg.KnownPeers)
	peerTimestamp := fromProtoTimestamp(msg.LastTimestamp)
	
	re.mu.Lock()
	peer, exists := re.peers[msg.NodeId]
	if exists {
		peer.mu.Lock()
		if peerTimestamp.After(peer.lastKnownTimestamp) {
			peer.lastKnownTimestamp = peerTimestamp
		}
		peer.mu.Unlock()
	}
	re.mu.Unlock()

	if re.lastTimestamp.Compare(peerTimestamp) < 0 {
		go re.requestUpdatesFromPeer(msg.NodeId)
	}
}

// updatePeerList adds new peers to the ReplicationEngine's peer list.
func (re *ReplicationEngine) updatePeerList(newPeers []string) {
	re.mu.Lock()
	defer re.mu.Unlock()

	for _, url := range newPeers {
		if url != re.config.OwnURL && re.peers[url] == nil {
			re.peers[url] = NewPeer(url, re)
			go re.peers[url].Connect(re.config.JWTSecret)
		}
	}
}

// getPeerURLs returns a list of all known peer URLs.
func (re *ReplicationEngine) getPeerURLs() []string {
	urls := make([]string, 0, len(re.peers))
	for url := range re.peers {
		urls = append(urls, url)
	}
	return urls
}

// requestUpdatesFromPeer initiates an update request to a specific peer.
func (re *ReplicationEngine) requestUpdatesFromPeer(peerURL string) {
	re.mu.Lock()
	defer re.mu.Unlock()

	peer, exists := re.peers[peerURL]
	if !exists {
		log.Printf("Peer %s not found", peerURL)
		return
	}

	peer.mu.Lock()
	since := peer.lastKnownTimestamp
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

		err := peer.RequestUpdates(req.since, MaxUpdateResults)
		if err != nil {
			log.Printf("Error requesting updates from peer %s: %v", peerURL, err)
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

	log.Printf("Received %d updates from peer %s (exists=%v)", len(batchUpdate.Updates), peerURL, exists)

	if !exists || !peerExists {
		log.Printf("[!] Peer %s is not in the updateRequests map or peers map", peerURL)
		return
	}

	var latestTimestamp Timestamp
	for _, protoUpdate := range batchUpdate.Updates {
		update := fromProtoUpdate(protoUpdate)
		re.handleReceivedUpdate(update)
		if update.Timestamp.After(latestTimestamp) {
			latestTimestamp = update.Timestamp
		}
	}

	peer.mu.Lock()
	if latestTimestamp.After(peer.lastKnownTimestamp) {
		peer.lastKnownTimestamp = latestTimestamp
	}
	peer.mu.Unlock()

	req.since = latestTimestamp

	// Signal the fetchUpdatesFromPeer goroutine
	req.responseChan <- batchUpdate.HasMore
}
