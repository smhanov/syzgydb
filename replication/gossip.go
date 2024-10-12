package replication

import (
	"log"
	"time"

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
	if re.lastTimestamp.Compare(fromProtoTimestamp(msg.LastTimestamp)) < 0 {
		go re.requestUpdatesFromPeer(msg.NodeId, fromProtoTimestamp(msg.LastTimestamp))
	}
}

// updatePeerList adds new peers to the ReplicationEngine's peer list.
func (re *ReplicationEngine) updatePeerList(newPeers []string) {
	re.mu.Lock()
	defer re.mu.Unlock()

	for _, url := range newPeers {
		if url != re.config.OwnURL && re.peers[url] == nil {
			re.peers[url] = NewPeer(url)
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
func (re *ReplicationEngine) requestUpdatesFromPeer(peerURL string, since Timestamp) {
    re.mu.Lock()
    defer re.mu.Unlock()

    if re.updateRequests == nil {
        re.updateRequests = make(map[string]*updateRequest)
    }

    if _, exists := re.updateRequests[peerURL]; !exists {
        re.updateRequests[peerURL] = &updateRequest{
            peerURL: peerURL,
            since:   since,
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
        req := re.updateRequests[peerURL]
        re.mu.Unlock()

        peer := re.peers[peerURL]
        if peer == nil || !peer.IsConnected() {
            break
        }

        err := peer.RequestUpdates(req.since, MaxUpdateResults)
        if err != nil {
            log.Printf("Error requesting updates from peer %s: %v", peerURL, err)
            break
        }

        // Wait for the response in handleReceivedBatchUpdate
        // If no more updates, the loop will be broken there
    }

    re.mu.Lock()
    delete(re.updateRequests, peerURL)
    re.mu.Unlock()
}

func (re *ReplicationEngine) handleReceivedBatchUpdate(peerURL string, batchUpdate *pb.BatchUpdate) {
    re.mu.Lock()
    req, exists := re.updateRequests[peerURL]
    re.mu.Unlock()

    if !exists {
        return
    }

    for _, protoUpdate := range batchUpdate.Updates {
        update := fromProtoUpdate(protoUpdate)
        re.handleReceivedUpdate(update)
        if update.Timestamp.After(req.since) {
            req.since = update.Timestamp
        }
    }

    if !batchUpdate.HasMore {
        re.mu.Lock()
        delete(re.updateRequests, peerURL)
        re.mu.Unlock()
    }
}
