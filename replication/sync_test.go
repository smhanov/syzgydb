package replication

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	pb "github.com/smhanov/syzgydb/replication/proto"
	"google.golang.org/protobuf/proto"
)

func TestGossipMessageReceived(t *testing.T) {
	// Set up a single node
	storage := NewMockStorage(0)
	config := ReplicationConfig{
		OwnURL:    "ws://localhost:8080",
		PeerURLs:  []string{},
		JWTSecret: []byte("test_secret"),
		NodeID:    0,
	}
	re, err := Init(storage, config, NewVectorClock().Update(0, Now()))
	if err != nil {
		t.Fatalf("Failed to initialize ReplicationEngine: %v", err)
	}

	// Start listening
	err = re.Listen(":8080")
	if err != nil {
		t.Fatalf("Failed to start listening: %v", err)
	}
	defer re.Close()

	// Connect to the WebSocket
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/"}
	header := http.Header{}
	token, err := GenerateToken("test_client", "ws://localhost:8081", config.JWTSecret)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	header.Add("Authorization", "Bearer "+token)

	c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer c.Close()

	// Wait for and read the gossip message
	done := make(chan struct{})
	var gossipMsg *pb.GossipMessage

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
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

			if msg.Type == pb.Message_GOSSIP {
				gossipMsg = msg.GetGossipMessage()
				return
			}
		}
	}()

	select {
	case <-done:
		if gossipMsg == nil {
			t.Fatalf("Did not receive gossip message")
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("Timed out waiting for gossip message")
	}

	// Verify the contents of the gossip message
	if gossipMsg.NodeId != "0" {
		t.Errorf("Expected NodeId to be '0', got '%s'", gossipMsg.NodeId)
	}

	if len(gossipMsg.KnownPeers) != 1 || gossipMsg.KnownPeers[0] != "ws://localhost:8081" {
		t.Errorf("Expected KnownPeers to be ['ws://localhost:8081'], got %v", gossipMsg.KnownPeers)
	}

	vcJSON, _ := json.MarshalIndent(gossipMsg.LastVectorClock, "", "  ")
	fmt.Printf("Received VectorClock: %s\n", vcJSON)

	// You may want to add more specific checks for the LastVectorClock
	if gossipMsg.LastVectorClock == nil {
		t.Errorf("Expected non-nil LastVectorClock")
	}
}
