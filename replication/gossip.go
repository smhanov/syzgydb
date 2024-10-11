package replication

import (
	"log"
	"time"

	pb "github.com/smhanov/syzgydb/replication/proto"
)

// GossipLoop runs the gossip protocol, periodically sending gossip messages to peers.
func (re *ReplicationEngine) GossipLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		re.mu.Lock()
		for _, peer := range re.peers {
			if peer.IsConnected() {
				go re.sendGossipMessage(peer)
			}
		}
		re.mu.Unlock()
	}
}

// sendGossipMessage sends a gossip message to a specific peer.
func (re *ReplicationEngine) sendGossipMessage(peer *Peer) {
	msg := &pb.GossipMessage{
		NodeId:        re.ownURL,
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
		if url != re.ownURL && re.peers[url] == nil {
			re.peers[url] = NewPeer(url)
			go re.peers[url].Connect(re.jwtSecret)
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

// requestUpdatesFromPeer sends an update request to a specific peer.
func (re *ReplicationEngine) requestUpdatesFromPeer(peerURL string, since Timestamp) {
	peer := re.peers[peerURL]
	if peer != nil && peer.IsConnected() {
		peer.RequestUpdates(since)
	}
}
