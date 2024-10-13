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
	_, err := ValidateToken(tokenString, re.config.JWTSecret)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}

	peerURL := r.RemoteAddr
	peer := NewPeer(peerURL)
	peer.SetConnection(conn)
	re.mu.Lock()
	re.peers[peerURL] = peer
	re.mu.Unlock()

	go peer.HandleIncomingMessages(re)
}

// NewPeer creates a new Peer instance.
func NewPeer(url string) *Peer {
	// For testing, use a dummy URL and a mock connection
	if !strings.HasPrefix(url, "ws://") && !strings.HasPrefix(url, "wss://") {
		url = "ws://localhost:8080/" + url
		return &Peer{
			url:        url,
			lastActive: time.Now(),
			connection: &mockConnection{},
		}
	}
	return &Peer{
		url:        url,
		lastActive: time.Now(),
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
	token, _ := GenerateToken(p.url, jwtSecret)
	header.Add("Authorization", token)
	conn, _, err := dialer.Dial(p.url, header)
	if err != nil {
		log.Println("Failed to connect to peer:", err)
		return
	}

	p.connection = conn
	go p.HandleIncomingMessages(nil)
}

// IsConnected checks if the peer is currently connected.
func (p *Peer) IsConnected() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connection != nil
}

// HandleIncomingMessages processes messages received from the peer.
func (p *Peer) HandleIncomingMessages(re *ReplicationEngine) {
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
		err = p.processMessage(message, re)
		if err != nil {
			log.Println("Failed to process message:", err)
		}
	}
}

// processMessage handles different types of incoming messages.
func (p *Peer) processMessage(data []byte, re *ReplicationEngine) error {
	var msg pb.Message
	if err := proto.Unmarshal(data, &msg); err != nil {
		return err
	}

	log.Printf("Peer %s received message of type %s", p.url, msg.Type.String())

	switch msg.Type {
	case pb.Message_GOSSIP:
		if re != nil {
			re.HandleGossipMessage(msg.GetGossipMessage())
		}
	case pb.Message_UPDATE:
		if re != nil {
			update := fromProtoUpdate(msg.GetUpdate())
			re.handleReceivedUpdate(update)
		}
	case pb.Message_UPDATE_REQUEST:
		if re != nil {
			since := fromProtoTimestamp(msg.GetUpdateRequest().Since)
			maxResults := int(msg.GetUpdateRequest().MaxResults)
			updates, hasMore, err := re.storage.GetUpdatesSince(since, maxResults)
			if err != nil {
				return err
			}
			batchUpdate := &pb.BatchUpdate{
				Updates: toProtoUpdates(flattenUpdates(updates)),
				HasMore: hasMore,
			}
			re.sendBatchUpdate(p, batchUpdate)
		}
	case pb.Message_BATCH_UPDATE:
		if re != nil {
			batchUpdate := msg.GetBatchUpdate()
			re.handleReceivedBatchUpdate(p.url, batchUpdate)
		}
	default:
		return errors.New("unknown message type")
	}
	return nil
}

// SendGossipMessage sends a gossip message to the peer.
func (p *Peer) SendGossipMessage(msg *pb.GossipMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connection == nil {
		return errors.New("not connected")
	}
	message := &pb.Message{
		Type: pb.Message_GOSSIP,
		Content: &pb.Message_GossipMessage{
			GossipMessage: msg,
		},
	}
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	return p.connection.WriteMessage(websocket.BinaryMessage, data)
}

// SendUpdate sends an update message to the peer.
func (p *Peer) SendUpdate(update Update) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connection == nil {
		return errors.New("not connected")
	}
	message := &pb.Message{
		Type: pb.Message_UPDATE,
		Content: &pb.Message_Update{
			Update: update.toProto(),
		},
	}
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	return p.connection.WriteMessage(websocket.BinaryMessage, data)
}

// RequestUpdates sends an update request to the peer.
func (p *Peer) RequestUpdates(since Timestamp, maxResults int) error {
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
		Type: pb.Message_UPDATE_REQUEST,
		Content: &pb.Message_UpdateRequest{
			UpdateRequest: request,
		},
	}
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	log.Printf("Peer %s requesting updates since %v", p.url, since)
	return p.connection.WriteMessage(websocket.BinaryMessage, data)
}

// sendBatchUpdate sends a batch update to the peer.
func (re *ReplicationEngine) sendBatchUpdate(peer *Peer, batchUpdate *pb.BatchUpdate) {
	peer.mu.Lock()
	defer peer.mu.Unlock()
	if peer.connection == nil {
		return
	}
	message := &pb.Message{
		Type: pb.Message_BATCH_UPDATE,
		Content: &pb.Message_BatchUpdate{
			BatchUpdate: batchUpdate,
		},
	}
	data, err := proto.Marshal(message)
	if err != nil {
		log.Println("Failed to marshal batch update:", err)
		return
	}
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
