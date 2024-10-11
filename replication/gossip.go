package replication

import (
    "time"
    pb "your_project/replication/proto"
)

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

func (re *ReplicationEngine) HandleGossipMessage(msg *pb.GossipMessage) {
    re.updatePeerList(msg.KnownPeers)
    if re.lastTimestamp.Compare(fromProtoTimestamp(msg.LastTimestamp)) < 0 {
        go re.requestUpdatesFromPeer(msg.NodeId, fromProtoTimestamp(msg.LastTimestamp))
    }
}

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

func (re *ReplicationEngine) getPeerURLs() []string {
    urls := make([]string, 0, len(re.peers))
    for url := range re.peers {
        urls = append(urls, url)
    }
    return urls
}

func (re *ReplicationEngine) requestUpdatesFromPeer(peerURL string, since Timestamp) {
    peer := re.peers[peerURL]
    if peer != nil && peer.IsConnected() {
        peer.RequestUpdates(since)
    }
}
