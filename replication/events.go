package replication

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	pb "github.com/smhanov/syzgydb/replication/proto"
	"google.golang.org/protobuf/proto"
)

func (sm *StateMachine) incrementAndGetTimestamp() *pb.Timestamp {
	sm.timestamp = sm.timestamp.Next(true)
	return &pb.Timestamp{
		UnixTime:     sm.timestamp.UnixTime,
		LamportClock: sm.timestamp.LamportClock,
	}
}

type UpdateRequestEvent struct {
	Peer       *Peer
	Since      *VectorClock
	MaxResults int
}

func (e UpdateRequestEvent) process(sm *StateMachine) {
	log.Printf("[%d]->[%s] Processing UpdateRequestEvent", sm.config.NodeID, e.Peer.name)
	updates, hasMore, err := sm.storage.GetUpdatesSince(e.Since, e.MaxResults)
	if err != nil {
		log.Println("Failed to get updates:", err)
		return
	}
	batchUpdate := &pb.BatchUpdate{
		Updates: toProtoUpdates(flattenUpdates(updates)),
		HasMore: hasMore,
	}

	msg := &pb.Message{
		Type:      pb.Message_BATCH_UPDATE,
		TimeStamp: sm.incrementAndGetTimestamp(),
		Content: &pb.Message_BatchUpdate{
			BatchUpdate: batchUpdate,
		},
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling batch update: %v", err)
		return
	}

	err = e.Peer.connection.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		log.Printf("Error sending batch update to peer %s: %v", e.Peer.url, err)
	}
}

type BatchUpdateEvent struct {
	Peer        *Peer
	BatchUpdate *pb.BatchUpdate
}

func (e BatchUpdateEvent) process(sm *StateMachine) {
	log.Printf("[%d]<-[%s] Processing BatchUpdateEvent", sm.config.NodeID, e.Peer.name)
	updates := make([]Update, 0, len(e.BatchUpdate.Updates))
	for _, protoUpdate := range e.BatchUpdate.Updates {
		update := fromProtoUpdate(protoUpdate)
		updates = append(updates, update)

		e.Peer.lastKnownVectorClock.Update(update.NodeID, update.Timestamp)
	}

	sm.storage.CommitUpdates(updates)

	// TODO: If there were more updates, request them
}

type AddPeerEvent struct {
	URL string
}

func (e AddPeerEvent) process(sm *StateMachine) {
	log.Printf("[%d] Processing AddPeerEvent %s", sm.config.NodeID, e.URL)
	if _, exists := sm.peers[e.URL]; !exists {
		sm.peers[e.URL] = &Peer{
			url:                  e.URL,
			lastKnownVectorClock: NewVectorClock(),
			stateMachine:         sm,
		}
		// Schedule a ConnectPeerEvent instead of directly connecting
		sm.eventChan <- ConnectPeerEvent(e)
	}
}

type ConnectPeerEvent struct {
	URL string
}

func (e ConnectPeerEvent) process(sm *StateMachine) {
	log.Printf("[%d] Processing ConnectPeerEvent", sm.config.NodeID)
	peer, exists := sm.peers[e.URL]
	if !exists {
		log.Printf("Peer %s not found for connection", e.URL)
		return
	}

	if peer.connection != nil {
		log.Printf("[%d] Peer %s already connected", sm.config.NodeID, e.URL)
		return
	}

	name := fmt.Sprintf("%d", sm.config.NodeID)
	dialWebSocket(name, sm.config.OwnURL, e.URL, sm.config.JWTSecret, sm.eventChan)
}

type WebSocketDialSucceededEvent struct {
	URL        string
	Connection *websocket.Conn
}

func (e WebSocketDialSucceededEvent) process(sm *StateMachine) {
	log.Printf("[%d] Processing WebSocketDialSucceededEvent for %s", sm.config.NodeID, e.URL)
	peer, exists := sm.peers[e.URL]
	if !exists {
		log.Printf("Peer %s not found for successful connection", e.URL)
		e.Connection.Close()
		return
	}

	if peer.connection != nil {
		log.Printf("[%d] Peer %s already connected", sm.config.NodeID, e.URL)
		e.Connection.Close()
		return
	}

	peer.connection = e.Connection

	// Schedule a SendGossipEvent for the new peer
	sm.eventChan <- SendGossipEvent{Peer: peer}

	// Start reading messages from this peer
	go peer.ReadLoop(sm.eventChan)
}

type WebSocketDialFailedEvent struct {
	URL   string
	Error error
}

func (e WebSocketDialFailedEvent) process(sm *StateMachine) {
	//log.Printf("[%d] Processing WebSocketDialFailedEvent for %s: %v", sm.config.NodeID, e.URL, e.Error)
	// Optionally, schedule a retry after some time
	//time.AfterFunc(5*time.Second, func() { sm.eventChan <- ConnectPeerEvent{URL: e.URL} })
}

type WebSocketConnectionEvent struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	ReplyChan      chan<- struct{}
}

func (e WebSocketConnectionEvent) process(sm *StateMachine) {
	log.Printf("[%d] Processing WebSocketConnectionEvent", sm.config.NodeID)
	defer close(e.ReplyChan) // Signal that the event has been processed

	tokenString := e.Request.Header.Get("Authorization")
	if tokenString == "" || len(tokenString) <= 7 || tokenString[:7] != "Bearer " {
		http.Error(e.ResponseWriter, "Missing or invalid Authorization header", http.StatusUnauthorized)
		return
	}
	tokenString = tokenString[7:]

	peerName, peerURL, err := ValidateToken(tokenString, sm.config.JWTSecret)
	if err != nil {
		http.Error(e.ResponseWriter, "Invalid token", http.StatusUnauthorized)
		return
	}

	if peer, ok := sm.peers[peerURL]; ok {
		log.Printf("[%d] Ignore new connection from connected peer [%s]", sm.config.NodeID, peer.name)
		http.Error(e.ResponseWriter, "Peer already connected", http.StatusConflict)
		return
	}

	if peerName == "" || peerURL == "" {
		log.Panicf("Peer has blank name: %s, URL: %s", peerName, peerURL)
	}

	conn, err := upgradeToWebSocket(e.ResponseWriter, e.Request)
	if err != nil {
		log.Printf("Failed to upgrade connection to WebSocket: %v", err)
		http.Error(e.ResponseWriter, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}

	peer := NewPeer(peerName, peerURL, sm)
	peer.connection = conn
	sm.peers[peerURL] = peer
	log.Printf("[%d]<-[%s] Connected to peer (incoming)", sm.config.NodeID, peerName)

	// Schedule a SendGossipEvent for the new peer
	sm.eventChan <- SendGossipEvent{Peer: peer}

	go peer.ReadLoop(sm.eventChan)
}

type SendGossipEvent struct {
	Peer *Peer
}

func (e SendGossipEvent) process(sm *StateMachine) {
	log.Printf("[%d]->[%s] Processing SendGossipEvent", sm.config.NodeID, e.Peer.name)
	msg := &pb.GossipMessage{
		NodeId:        fmt.Sprintf("%d", sm.config.NodeID),
		KnownPeers:    sm.getPeerURLs(),
		NodeSequences: sm.nodeSequences.toProto(),
	}

	protoMsg := &pb.Message{
		Type:      pb.Message_GOSSIP,
		TimeStamp: sm.incrementAndGetTimestamp(),
		Content: &pb.Message_GossipMessage{
			GossipMessage: msg,
		},
	}

	data, err := proto.Marshal(protoMsg)
	if err != nil {
		log.Printf("Error marshaling gossip message: %v", err)
		return
	}

	err = e.Peer.connection.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		log.Printf("Error sending gossip message to peer %s: %v", e.Peer.url, err)
	}
}

type GossipMessageEvent struct {
	Peer    *Peer
	Message *pb.GossipMessage
}

func (e GossipMessageEvent) process(sm *StateMachine) {
	log.Printf("[%d]<-[%s] Processing GossipMessageEvent", sm.config.NodeID, e.Message.NodeId)
	peerVectorClock := fromProtoVectorClock(e.Message.LastVectorClock)

	e.Peer.lastKnownVectorClock = peerVectorClock
	e.Peer.name = e.Message.NodeId

	for _, peer := range sm.peers {
		log.Printf("[%d]     Known peer: %s", sm.config.NodeID, peer.url)
	}

	for _, peerURL := range e.Message.KnownPeers {
		if _, exists := sm.peers[peerURL]; !exists && peerURL != sm.config.OwnURL {
			log.Printf("[%d]    New peer: %s", sm.config.NodeID, peerURL)
			sm.eventChan <- AddPeerEvent{URL: peerURL}
		}
	}

	if sm.lastKnownVectorClock.Before(peerVectorClock) {
		log.Printf("[%d]->[%s] Request updates", sm.config.NodeID, e.Peer.name)
		updateRequest := &pb.UpdateRequest{
			Since:      sm.lastKnownVectorClock.toProto(),
			MaxResults: MaxUpdateResults,
		}

		msg := &pb.Message{
			Type:        pb.Message_UPDATE_REQUEST,
			TimeStamp:   sm.incrementAndGetTimestamp(),
			VectorClock: sm.lastKnownVectorClock.toProto(),
			Content: &pb.Message_UpdateRequest{
				UpdateRequest: updateRequest,
			},
		}

		data, err := proto.Marshal(msg)
		if err != nil {
			log.Printf("Error marshaling update request: %v", err)
			return
		}

		err = e.Peer.connection.WriteMessage(websocket.BinaryMessage, data)
		if err != nil {
			log.Printf("Error sending update request to peer %s: %v", e.Peer.url, err)
		}
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

type PeerHeartbeatEvent struct {
}

func (e PeerHeartbeatEvent) process(sm *StateMachine) {
	log.Printf("[%d] Processing PeerHeartbeatEvent", sm.config.NodeID)
	for _, peer := range sm.peers {
		heartbeat := &pb.Heartbeat{
			VectorClock: sm.lastKnownVectorClock.toProto(),
		}

		msg := &pb.Message{
			Type:        pb.Message_HEARTBEAT,
			TimeStamp:   sm.incrementAndGetTimestamp(),
			VectorClock: sm.lastKnownVectorClock.toProto(),
			Content: &pb.Message_Heartbeat{
				Heartbeat: heartbeat,
			},
		}

		data, err := proto.Marshal(msg)
		if err != nil {
			log.Printf("Error marshaling heartbeat for peer %s: %v", peer.url, err)
			continue
		}

		err = peer.connection.WriteMessage(websocket.BinaryMessage, data)
		if err != nil {
			log.Printf("Error sending heartbeat to peer %s: %v", peer.url, err)
			sm.eventChan <- PeerDisconnectEvent{Peer: peer}
		}
	}
}

type LocalUpdatesEvent struct {
	Updates   []Update
	ReplyChan chan<- error
}

func (e LocalUpdatesEvent) process(sm *StateMachine) {
	log.Printf("[%d] Processing LocalUpdatesEvent", sm.config.NodeID)

	// Commit updates locally
	err := sm.storage.CommitUpdates(e.Updates)
	if err != nil {
		e.ReplyChan <- err
		return
	}

	// Create a BatchUpdate message
	batchUpdate := &pb.BatchUpdate{
		Updates: toProtoUpdates(e.Updates),
		HasMore: false,
	}

	// Create a Message containing the BatchUpdate
	msg := &pb.Message{
		Type:        pb.Message_BATCH_UPDATE,
		TimeStamp:   sm.incrementAndGetTimestamp(),
		VectorClock: sm.lastKnownVectorClock.toProto(),
		Content: &pb.Message_BatchUpdate{
			BatchUpdate: batchUpdate,
		},
	}

	// Marshal the message
	data, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("[%d] Error marshaling BatchUpdate message: %v", sm.config.NodeID, err)
		e.ReplyChan <- err
		return
	}

	// Send the BatchUpdate to all connected peers
	for _, peer := range sm.peers {
		if peer.connection != nil {
			log.Printf("[%d] Sending BatchUpdate to peer [%s]", sm.config.NodeID, peer.name)
			err := peer.connection.WriteMessage(websocket.BinaryMessage, data)
			if err != nil {
				log.Printf("[%d] Error sending BatchUpdate to peer %s: %v", sm.config.NodeID, peer.url, err)
				// Note: We're not treating this as a fatal error, just logging it
			}
		}
	}

	// Signal that the updates have been processed
	e.ReplyChan <- nil
}

type PeerDisconnectEvent struct {
	Peer *Peer
}

func (e PeerDisconnectEvent) process(sm *StateMachine) {
	log.Printf("[%d] Processing PeerDisconnectEvent from [%s]", sm.config.NodeID, e.Peer.name)

	// Close the connection
	if e.Peer.connection != nil {
		e.Peer.connection.Close()
	}

	// Remove the peer from the peers map
	delete(sm.peers, e.Peer.url)

	// Trigger a reconnection attempt after a delay
	go func() {
		time.Sleep(5 * time.Second)
		sm.eventChan <- ConnectPeerEvent{URL: e.Peer.url}
	}()
}
