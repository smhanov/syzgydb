package replication

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	pb "github.com/smhanov/syzgydb/replication/proto"
	"google.golang.org/protobuf/proto"
)

var upgrader = websocket.Upgrader{}

// ConnectToPeers attempts to establish WebSocket connections with all known peers.
func (re *ReplicationEngine) ConnectToPeers() {
	for {
		re.mu.Lock()
		for _, peer := range re.peers {
			if !peer.IsConnected() {
				go peer.Connect(re.config.JWTSecret)
			}
		}
		re.mu.Unlock()
		time.Sleep(10 * time.Second)
	}
}

// HandleWebSocket upgrades an HTTP connection to a WebSocket and handles the peer connection.
func (re *ReplicationEngine) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	tokenString := r.Header.Get("Authorization")
	nodeID, peerURL, err := ValidateToken(tokenString, re.config.JWTSecret)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}

	peer := NewPeer(nodeID, peerURL, re)
	peer.SetConnection(conn)
	re.mu.Lock()
	re.peers[peerURL] = peer
	re.mu.Unlock()

	go peer.HandleIncomingMessages()

	// Trigger a gossip message to the new peer
	go re.sendGossipMessage(peer, re.NextTimestamp(false))
}

// NewPeer creates a new Peer instance.
func NewPeer(name string, url string, re *ReplicationEngine) *Peer {
	if !strings.HasPrefix(url, "ws://") && !strings.HasPrefix(url, "wss://") {
		log.Panicf("Bad url: %s", url)
	}
	return &Peer{
		url:                url,
		lastActive:         time.Now(),
		lastKnownTimestamp: Timestamp{},
		re:                 re,
		name:               name,
	}
}

// Connect establishes a WebSocket connection with the peer.
func (p *Peer) Connect(jwtSecret []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.connection != nil {
		return
	}

	dialer := websocket.Dialer{}
	header := http.Header{}
	token, _ := GenerateToken(p.name, p.url, jwtSecret)
	header.Add("Authorization", token)
	conn, _, err := dialer.Dial(p.url, header)
	if err != nil {
		log.Printf("Failed to connect to peer url:%v: %v", p.url, err)
		return
	}

	p.connection = conn
	go p.HandleIncomingMessages()

	// Trigger a gossip message to the newly connected peer
	go p.re.sendGossipMessage(p, p.re.NextTimestamp(false))
}

// IsConnected checks if the peer is currently connected.
func (p *Peer) IsConnected() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connection != nil
}

// HandleIncomingMessages processes messages received from the peer.
func (p *Peer) HandleIncomingMessages() {
	if p.re == nil {
		log.Panicf("HandleIncomingMessages called with nil ReplicationEngine")
	}
	for {
		_, message, err := p.connection.ReadMessage()
		if err != nil {
			log.Println("Error reading from peer:", err)
			p.mu.Lock()
			p.connection.Close()
			p.connection = nil
			p.mu.Unlock()
			return
		}
		p.lastActive = time.Now()
		err = p.processMessage(message)
		if err != nil {
			log.Println("Failed to process message:", err)
		}
	}
}

// processMessage handles different types of incoming messages.
func (p *Peer) processMessage(data []byte) error {
	var msg pb.Message
	if err := proto.Unmarshal(data, &msg); err != nil {
		return err
	}

	vc := fromProtoVectorClock(msg.VectorClock)
	log.Printf("[%s] received  %s from %s vc=%s", p.re.name, msg.Type.String(), p.name, vc)
	p.re.handleReceivedVectorClock(vc)

	switch msg.Type {
	case pb.Message_GOSSIP:
		p.re.HandleGossipMessage(p, msg.GetGossipMessage())
	case pb.Message_UPDATE:
		update := fromProtoUpdate(msg.GetUpdate())
		p.re.handleReceivedUpdate(update)
	case pb.Message_UPDATE_REQUEST:
		since := fromProtoTimestamp(msg.GetUpdateRequest().Since)
		maxResults := int(msg.GetUpdateRequest().MaxResults)
		updates, hasMore, err := p.re.storage.GetUpdatesSince(since, maxResults)
		if err != nil {
			return err
		}
		batchUpdate := &pb.BatchUpdate{
			Updates: toProtoUpdates(flattenUpdates(updates)),
			HasMore: hasMore,
		}
		p.re.sendBatchUpdate(p, batchUpdate, p.re.NextTimestamp(false))
	case pb.Message_BATCH_UPDATE:
		log.Printf("[%s] Processing batch update from peer %s", p.re.name, p.name)
		batchUpdate := msg.GetBatchUpdate()
		p.re.handleReceivedBatchUpdate(p.url, batchUpdate)
	default:
		return errors.New("unknown message type")
	}
	return nil
}

// SendGossipMessage sends a gossip message to the peer.
func (p *Peer) SendGossipMessage(msg *pb.GossipMessage, vc *VectorClock) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connection == nil {
		return errors.New("not connected")
	}
	message := &pb.Message{
		Type:      pb.Message_GOSSIP,
		VectorClock: vc.toProto(),
		Content: &pb.Message_GossipMessage{
			GossipMessage: msg,
		},
	}
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	log.Printf("[%s] sending gossip message to %s", p.re.name, p.name)
	return p.connection.WriteMessage(websocket.BinaryMessage, data)
}

// SendUpdate sends an update message to the peer.
func (p *Peer) SendUpdate(update Update, vc *VectorClock) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connection == nil {
		return errors.New("not connected")
	}
	message := &pb.Message{
		Type:        pb.Message_UPDATE,
		VectorClock: vc.toProto(),
		Content: &pb.Message_Update{
			Update: update.toProto(),
		},
	}
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	log.Printf("[%s] sending update to %s", p.re.name, p.name)
	return p.connection.WriteMessage(websocket.BinaryMessage, data)
}

// RequestUpdates sends an update request to the peer.
func (p *Peer) RequestUpdates(since *VectorClock, maxResults int, vc *VectorClock) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connection == nil {
		return errors.New("not connected")
	}
	request := &pb.UpdateRequest{
		Since:      since.toProto(),
		MaxResults: int32(maxResults),
	}
	message := &pb.Message{
		Type:        pb.Message_UPDATE_REQUEST,
		VectorClock: vc.toProto(),
		Content: &pb.Message_UpdateRequest{
			UpdateRequest: request,
		},
	}
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	log.Printf("[%s] requesting updates since %v from %s", p.re.name, since, p.name)
	return p.connection.WriteMessage(websocket.BinaryMessage, data)
}

// sendBatchUpdate sends a batch update to the peer.
func (re *ReplicationEngine) sendBatchUpdate(peer *Peer, batchUpdate *pb.BatchUpdate, vc *VectorClock) {
	peer.mu.Lock()
	defer peer.mu.Unlock()
	if peer.connection == nil {
		return
	}
	message := &pb.Message{
		Type:      pb.Message_BATCH_UPDATE,
		VectorClock: vc.toProto(),
		Content: &pb.Message_BatchUpdate{
			BatchUpdate: batchUpdate,
		},
	}
	data, err := proto.Marshal(message)
	if err != nil {
		log.Println("Failed to marshal batch update:", err)
		return
	}
	log.Printf("[%s] Reply to %s with batch update", re.name, peer.name)
	err = peer.connection.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		log.Println("Failed to send batch update:", err)
	}
}

// toProtoUpdates converts a slice of Updates to a slice of protobuf Updates.
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
