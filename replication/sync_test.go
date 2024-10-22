package replication

import (
	"bytes"
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

func TestRejectMissingSequence(t *testing.T) {
	// 1. Create a node
	node := setupTestNode(t, 0, 8080)
	defer node.RE.Close()

	// 2. Connect to the node and expect a gossip message
	conn := connectToNode(t, node, "1", "ws://localhost:8081")
	defer conn.Close()

	initialGossip := receiveMessage(t, conn, 5*time.Second)
	if initialGossip.Type != pb.Message_GOSSIP {
		t.Fatalf("Expected initial GOSSIP message, got %v", initialGossip.Type)
	}

	// 3. Send a gossip message with nodeSequences advertising nodeid 1 and sequence 2
	gossipMsg := &pb.GossipMessage{
		NodeId: "1",
		NodeSequences: &pb.NodeSequences{
			Clock: map[uint64]uint64{1: 2},
		},
	}
	sendMessage(t, conn, createProtoMessage(pb.Message_GOSSIP, gossipMsg))

	// 4. Expect an update_request message with since equal to nodeid 1 and sequence 0
	updateRequestMsg := receiveMessage(t, conn, 5*time.Second)
	if updateRequestMsg.Type != pb.Message_UPDATE_REQUEST {
		t.Fatalf("Expected UPDATE_REQUEST message, got %v", updateRequestMsg.Type)
	}
	updateRequest := updateRequestMsg.GetUpdateRequest()
	if updateRequest == nil {
		t.Fatalf("Received nil UpdateRequest")
	}
	if updateRequest.Since.Clock[1] != 0 {
		t.Fatalf("Expected UpdateRequest with since sequence 0 for node 1, got %d", updateRequest.Since.Clock[1])
	}

	// 5. Send batch_update containing an update beginning with sequence 1
	update := createProtoUpdate(
		1,
		Timestamp{UnixTime: time.Now().UnixMilli(), LamportClock: 1},
		UpsertRecord,
		"testRecord",
		[]DataStream{{StreamID: 1, Data: []byte("test data")}},
	)
	update.SequenceNumber = 1
	batchUpdate := createProtoBatchUpdate([]Update{fromProtoUpdate(update)})
	sendMessage(t, conn, createProtoMessage(pb.Message_BATCH_UPDATE, batchUpdate))

	// 6. Wait 5s
	time.Sleep(5 * time.Second)

	// 7. Expect a gossip message with nodeid 1 and sequence 0
	finalGossip := receiveMessage(t, conn, 5*time.Second)
	if finalGossip.Type != pb.Message_GOSSIP {
		t.Fatalf("Expected final GOSSIP message, got %v", finalGossip.Type)
	}
	finalGossipMsg := finalGossip.GetGossipMessage()
	if finalGossipMsg == nil {
		t.Fatalf("Received nil GossipMessage")
	}
	if finalGossipMsg.NodeSequences.Clock[1] != 0 {
		t.Fatalf("Expected final gossip with sequence 0 for node 1, got %d", finalGossipMsg.NodeSequences.Clock[1])
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

func TestMultipleUpdatesAndSync(t *testing.T) {
	// 1. Start a node
	node := setupTestNode(t, 1, 8080)
	defer node.RE.Close()

	// 2. Update a record with the same id twice using replication engine.SubmitUpdates
	updates := []Update{
		{
			Type:     UpsertRecord,
			RecordID: "testRecord",
			DataStreams: []DataStream{
				{StreamID: 1, Data: []byte("first update")},
			},
		},
		{
			Type:     UpsertRecord,
			RecordID: "testRecord",
			DataStreams: []DataStream{
				{StreamID: 1, Data: []byte("second update")},
			},
		},
	}

	err := node.RE.SubmitUpdates(updates)
	if err != nil {
		t.Fatalf("Failed to submit updates: %v", err)
	}

	// 3. Connect to it as a peer and expect the gossip message
	conn := connectToNode(t, node, "2", "ws://localhost:8081")
	defer conn.Close()

	gossipMsg := receiveMessage(t, conn, 5*time.Second)
	if gossipMsg.Type != pb.Message_GOSSIP {
		t.Fatalf("Expected GOSSIP message, got %v", gossipMsg.Type)
	}

	// 4. Send the update_request starting at the correct node id and sequence number 0
	updateRequest := createProtoUpdateRequest(NewNodeSequences(), MaxUpdateResults)
	updateRequestMsg := createProtoMessage(pb.Message_UPDATE_REQUEST, updateRequest)
	sendMessage(t, conn, updateRequestMsg)

	// 5. Expect both updates to come through
	batchUpdateMsg := receiveMessage(t, conn, 5*time.Second)
	if batchUpdateMsg.Type != pb.Message_BATCH_UPDATE {
		t.Fatalf("Expected BATCH_UPDATE message, got %v", batchUpdateMsg.Type)
	}

	batchUpdate := batchUpdateMsg.GetBatchUpdate()
	if batchUpdate == nil {
		t.Fatalf("Received nil BatchUpdate")
	}

	if len(batchUpdate.Updates) != 2 {
		t.Fatalf("Expected 2 updates, got %d", len(batchUpdate.Updates))
	}

	// Verify the contents of the updates
	for i, update := range batchUpdate.Updates {
		if update.RecordId != "testRecord" {
			t.Errorf("Update %d: Expected RecordId 'testRecord', got '%s'", i, update.RecordId)
		}
		if update.NodeId != 1 {
			t.Errorf("Update %d: Expected NodeId 1, got %d", i, update.NodeId)
		}
		if update.SequenceNumber != uint64(i+1) {
			t.Errorf("Update %d: Expected SequenceNumber %d, got %d", i, i+1, update.SequenceNumber)
		}
		if len(update.DataStreams) != 1 {
			t.Errorf("Update %d: Expected 1 DataStream, got %d", i, len(update.DataStreams))
		} else {
			expectedData := []byte(fmt.Sprintf("%s update", []string{"first", "second"}[i]))
			if !bytes.Equal(update.DataStreams[0].Data, expectedData) {
				t.Errorf("Update %d: Expected data '%s', got '%s'", i, expectedData, update.DataStreams[0].Data)
			}
		}
	}
}
