package replication

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	pb "github.com/smhanov/syzgydb/replication/proto"
	"google.golang.org/protobuf/proto"
)

// TestNode represents a node in the test environment
type TestNode struct {
	RE     *ReplicationEngine
	Config ReplicationConfig
}

// setupTestNode creates and starts a test node
func setupTestNode(t *testing.T, nodeID uint64, port int) *TestNode {
	storage := NewMockStorage(int(nodeID))
	config := ReplicationConfig{
		OwnURL:    fmt.Sprintf("ws://localhost:%d", port),
		PeerURLs:  []string{},
		JWTSecret: []byte("test_secret"),
		NodeID:    nodeID,
	}
	re, err := Init(storage, config, nil)
	if err != nil {
		t.Fatalf("Failed to initialize ReplicationEngine: %v", err)
	}

	err = re.Listen(fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("Failed to start listening: %v", err)
	}

	return &TestNode{RE: re, Config: config}
}

// connectToNode establishes a WebSocket connection to a node
func connectToNode(t *testing.T, node *TestNode, clientID, clientURL string) *websocket.Conn {
	u := url.URL{Scheme: "ws", Host: node.Config.OwnURL[5:], Path: "/"}
	header := http.Header{}
	token, err := GenerateToken(clientID, clientURL, node.Config.JWTSecret)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	header.Add("Authorization", "Bearer "+token)

	c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	return c
}

// Helper function to create a protocol buffer Message
func createProtoMessage(messageType pb.Message_MessageType, content proto.Message) *pb.Message {
	msg := &pb.Message{
		Type:      messageType,
		TimeStamp: Timestamp{UnixTime: time.Now().UnixMilli(), LamportClock: 1}.toProto(),
	}

	switch v := content.(type) {
	case *pb.GossipMessage:
		msg.Content = &pb.Message_GossipMessage{GossipMessage: v}
	case *pb.BatchUpdate:
		msg.Content = &pb.Message_BatchUpdate{BatchUpdate: v}
	case *pb.UpdateRequest:
		msg.Content = &pb.Message_UpdateRequest{UpdateRequest: v}
	case *pb.Heartbeat:
		msg.Content = &pb.Message_Heartbeat{Heartbeat: v}
	}

	return msg
}

// Helper function to create a protocol buffer Update
func createProtoUpdate(nodeID uint64, timestamp Timestamp, updateType UpdateType, recordID string, dataStreams []DataStream) *pb.Update {
	return (&Update{
		NodeID:      nodeID,
		Timestamp:   timestamp,
		Type:        updateType,
		RecordID:    recordID,
		DataStreams: dataStreams,
	}).toProto()
}

// Helper function to create a protocol buffer BatchUpdate
func createProtoBatchUpdate(updates []Update) *pb.BatchUpdate {
	return &pb.BatchUpdate{
		Updates: toProtoUpdates(updates),
		HasMore: false,
	}
}

// Helper function to create a protocol buffer UpdateRequest
func createProtoUpdateRequest(since *NodeSequences, maxResults int32) *pb.UpdateRequest {
	return &pb.UpdateRequest{
		Since:      since.toProto(),
		MaxResults: maxResults,
	}
}

// sendMessage sends a protobuf message over the WebSocket connection
func sendMessage(t *testing.T, conn *websocket.Conn, msg proto.Message) {
	data, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}
	err = conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}
}

// receiveMessage waits for and receives a specific type of message
func receiveMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) *pb.Message {
	done := make(chan struct{})
	var receivedMsg *pb.Message

	go func() {
		defer close(done)
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("Failed to read message: %v", err)
			return
		}

		var msg pb.Message
		err = proto.Unmarshal(message, &msg)
		if err != nil {
			t.Errorf("Failed to unmarshal message: %v", err)
			return
		}

		receivedMsg = &msg
	}()

	select {
	case <-done:
		if receivedMsg == nil {
			t.Fatalf("Did not receive expected message")
		}
		return receivedMsg
	case <-time.After(timeout):
		t.Fatalf("Timed out waiting for message")
		return nil
	}
}

func TestGossipMessageReceived(t *testing.T) {
	node := setupTestNode(t, 0, 8080)
	defer node.RE.Close()

	conn := connectToNode(t, node, "1", "ws://localhost:8081")
	defer conn.Close()

	msg := receiveMessage(t, conn, 5*time.Second)

	if msg.Type != pb.Message_GOSSIP {
		t.Fatalf("Expected GOSSIP message, got %v", msg.Type)
	}

	gossipMsg := msg.GetGossipMessage()

	// Verify the contents of the gossip message
	if gossipMsg.NodeId != "0" {
		t.Errorf("Expected NodeId to be '0', got '%s'", gossipMsg.NodeId)
	}

	if len(gossipMsg.KnownPeers) != 1 || gossipMsg.KnownPeers[0] != "ws://localhost:8081" {
		t.Errorf("Expected KnownPeers to be ['ws://localhost:8081'], got %v", gossipMsg.KnownPeers)
	}

	fmt.Printf("Received VectorClock: %s\n", gossipMsg.NodeSequences)

	if gossipMsg.NodeSequences == nil {
		t.Errorf("Expected non-nil NodeSequences")
	}
}

func TestSendAndReceiveUpdate(t *testing.T) {
	node := setupTestNode(t, 0, 8080)
	defer node.RE.Close()

	conn := connectToNode(t, node, "1", "ws://localhost:8081")
	defer conn.Close()

	// Receive initial gossip message
	receiveMessage(t, conn, 5*time.Second)

	// Create and send an update message
	update := createProtoUpdate(
		1,
		Timestamp{UnixTime: 123456789, LamportClock: 1},
		UpsertRecord,
		"testRecord",
		[]DataStream{{StreamID: 1, Data: []byte("test data")}},
	)
	batchUpdate := createProtoBatchUpdate([]Update{fromProtoUpdate(update)})
	updateMsg := createProtoMessage(pb.Message_BATCH_UPDATE, batchUpdate)
	sendMessage(t, conn, updateMsg)

	// Wait for and verify the response
	response := receiveMessage(t, conn, 5*time.Second)
	// Add assertions to check the response
	if response.Type != pb.Message_BATCH_UPDATE {
		t.Fatalf("Expected BATCH_UPDATE response, got %v", response.Type)
	}
	// Add more specific checks for the response content
}
