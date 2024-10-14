package replication

import pb "github.com/smhanov/syzgydb/replication/proto"

type ReceivedUpdateEvent struct {
	Update Update
}

func (e ReceivedUpdateEvent) process(sm *StateMachine) {
	sm.handleReceivedUpdate(e.Update)
}

type GossipMessageEvent struct {
	Peer    *Peer
	Message *pb.GossipMessage
}

func (e GossipMessageEvent) process(sm *StateMachine) {
	sm.handleGossipMessage(e.Peer, e.Message)
}

type UpdateRequestEvent struct {
	Peer       *Peer
	Since      *VectorClock
	MaxResults int
}

func (e UpdateRequestEvent) process(sm *StateMachine) {
	updates, hasMore, err := sm.storage.GetUpdatesSince(e.Since, e.MaxResults)
	if err != nil {
		log.Println("Failed to get updates:", err)
		return
	}
	batchUpdate := &pb.BatchUpdate{
		Updates: toProtoUpdates(flattenUpdates(updates)),
		HasMore: hasMore,
	}
	e.Peer.SendBatchUpdate(batchUpdate, sm.NextTimestamp(false))
}

type BatchUpdateEvent struct {
	Peer        *Peer
	BatchUpdate *pb.BatchUpdate
}

func (e BatchUpdateEvent) process(sm *StateMachine) {
	sm.handleReceivedBatchUpdate(e.Peer.url, e.BatchUpdate)
}

type AddPeerEvent struct {
	URL string
}

func (e AddPeerEvent) process(sm *StateMachine) {
	if _, exists := sm.peers[e.URL]; !exists {
		peer := NewPeer("a:?", e.URL, sm)
		sm.peers[e.URL] = peer
		go peer.Connect(sm.config.JWTSecret)
	}
}

type SendGossipEvent struct {
	Peer *Peer
}

func (e SendGossipEvent) process(sm *StateMachine) {
	msg := &pb.GossipMessage{
		NodeId:          sm.config.OwnURL,
		KnownPeers:      sm.getPeerURLs(),
		LastVectorClock: sm.lastKnownVectorClock.toProto(),
	}
	err := e.Peer.SendGossipMessage(msg, sm.NextTimestamp(false))
	if err != nil {
		log.Println("Failed to send gossip message:", err)
	}
}

func toProtoUpdates(updates []Update) []*pb.Update {
	var protoUpdates []*pb.Update
	for _, update := range updates {
		protoUpdates = append(protoUpdates, update.toProto())
	}
	return protoUpdates
}

func flattenUpdates(updates map[string][]Update) []Update {
	var flattened []Update
	for _, dbUpdates := range updates {
		flattened = append(flattened, dbUpdates...)
	}
	return flattened
}
