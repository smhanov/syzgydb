package replication

import (
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	pb "github.com/smhanov/syzgydb/replication/proto"
	"google.golang.org/protobuf/proto"
)

var upgrader = websocket.Upgrader{}

func (re *ReplicationEngine) ConnectToPeers() {
	for {
		re.mu.Lock()
		for _, peer := range re.peers {
			if !peer.IsConnected() {
				go peer.Connect(re.jwtSecret)
			}
		}
		re.mu.Unlock()
		time.Sleep(10 * time.Second)
	}
}

func (re *ReplicationEngine) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	tokenString := r.Header.Get("Authorization")
	_, err := ValidateToken(tokenString, re.jwtSecret)
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

type Peer struct {
	url        string
	connection *websocket.Conn
	lastActive time.Time
	mu         sync.Mutex
}

func NewPeer(url string) *Peer {
	return &Peer{
		url:        url,
		lastActive: time.Now(),
	}
}

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

func (p *Peer) IsConnected() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connection != nil
}

func (p *Peer) SetConnection(conn *websocket.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connection = conn
}

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

func (p *Peer) processMessage(data []byte, re *ReplicationEngine) error {
	var msg pb.Message
	if err := proto.Unmarshal(data, &msg); err != nil {
		return err
	}

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
			updates, err := re.storage.GetUpdatesSince(since)
			if err != nil {
				return err
			}
			for dbName, dbUpdates := range updates {
				batchUpdate := &pb.BatchUpdate{
					DatabaseName: dbName,
					Updates:      toProtoUpdates(dbUpdates),
				}
				re.sendBatchUpdate(p, batchUpdate)
			}
		}
	case pb.Message_BATCH_UPDATE:
		if re != nil {
			batchUpdate := msg.GetBatchUpdate()
			for _, protoUpdate := range batchUpdate.Updates {
				update := fromProtoUpdate(protoUpdate)
				re.handleReceivedUpdate(update)
			}
		}
	default:
		return errors.New("unknown message type")
	}
	return nil
}

func (p *Peer) SendGossipMessage(msg *pb.GossipMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connection == nil {
		return errors.New("not connected")
	}
	message := &pb.Message{
		Type:          pb.Message_GOSSIP,
		GossipMessage: msg,
	}
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	return p.connection.WriteMessage(websocket.BinaryMessage, data)
}

func (p *Peer) SendUpdate(update Update) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connection == nil {
		return errors.New("not connected")
	}
	message := &pb.Message{
		Type:   pb.Message_UPDATE,
		Update: update.toProto(),
	}
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	return p.connection.WriteMessage(websocket.BinaryMessage, data)
}

func (p *Peer) RequestUpdates(since Timestamp) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connection == nil {
		return errors.New("not connected")
	}
	request := &pb.UpdateRequest{
		Since: since.toProto(),
	}
	message := &pb.Message{
		Type:          pb.Message_UPDATE_REQUEST,
		UpdateRequest: request,
	}
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	return p.connection.WriteMessage(websocket.BinaryMessage, data)
}

func (re *ReplicationEngine) sendBatchUpdate(peer *Peer, batchUpdate *pb.BatchUpdate) {
	peer.mu.Lock()
	defer peer.mu.Unlock()
	if peer.connection == nil {
		return
	}
	message := &pb.Message{
		Type:        pb.Message_BATCH_UPDATE,
		BatchUpdate: batchUpdate,
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

func toProtoUpdates(updates []Update) []*pb.Update {
	var protoUpdates []*pb.Update
	for _, update := range updates {
		protoUpdates = append(protoUpdates, update.toProto())
	}
	return protoUpdates
}
