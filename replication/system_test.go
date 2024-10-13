package replication

import (
	"fmt"
	"testing"
	"time"
)

type mockNetwork struct {
    nodes map[string]*ReplicationEngine
    peers map[string]*MockPeer
}

func TestTimestampOrdering(t *testing.T) {
	_, nodes := setupTestEnvironment(t, 2)
	defer tearDownTestEnvironment(nodes)

	// Generate updates with out-of-order timestamps
	update1 := Update{
		Timestamp:    Timestamp{UnixTime: 1000, LamportClock: 1},
		Type:         UpsertRecord,
		RecordID:     "record1",
		DataStreams:  []DataStream{{StreamID: 1, Data: []byte("data1")}},
		DatabaseName: "testdb",
	}
	update2 := Update{
		Timestamp:    Timestamp{UnixTime: 1000, LamportClock: 2},
		Type:         UpsertRecord,
		RecordID:     "record1",
		DataStreams:  []DataStream{{StreamID: 1, Data: []byte("data2")}},
		DatabaseName: "testdb",
	}

	// Submit updates in reverse order
	nodes[0].SubmitUpdates([]Update{update2, update1})

	time.Sleep(1 * time.Second)

	// Check if the final state reflects the correct ordering
	record, _ := nodes[0].storage.GetRecord("testdb", "record1")
	if string(record[0].Data) != "data2" {
		t.Errorf("Expected final data to be 'data2', got '%s'", string(record[0].Data))
	}
}

func TestBufferedUpdates(t *testing.T) {
	network, nodes := setupTestEnvironment(t, 2)
	defer tearDownTestEnvironment(nodes)

	// Disconnect the nodes
	network.disconnect("node0", "node1")

	// Submit updates to both nodes
	update1 := Update{
		Timestamp:    nodes[0].NextTimestamp(),
		Type:         CreateDatabase,
		DatabaseName: "newdb",
	}
	update2 := Update{
		Timestamp:    nodes[1].NextTimestamp(),
		Type:         UpsertRecord,
		RecordID:     "record1",
		DataStreams:  []DataStream{{StreamID: 1, Data: []byte("test data")}},
		DatabaseName: "newdb",
	}

	err := nodes[0].SubmitUpdates([]Update{update1})
	if err != nil {
		t.Fatalf("Failed to submit update1: %v", err)
	}
	err = nodes[1].SubmitUpdates([]Update{update2})
	if err != nil {
		t.Fatalf("Failed to submit update2: %v", err)
	}

	// Reconnect the nodes
	network.connect("node0", "node1")

	// Manually trigger update exchange
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			if i != j {
				err = nodes[i].peers[fmt.Sprintf("node%d", j)].RequestUpdates(Timestamp{}, MaxUpdateResults)
				if err != nil {
					t.Fatalf("Failed to request updates from node%d to node%d: %v", j, i, err)
				}
			}
		}
	}

	// Wait for replication and buffered updates to be processed
	time.Sleep(5 * time.Second)

	// Check if both nodes have the database and the record
	for i, node := range nodes {
		if !node.storage.Exists("newdb") {
			t.Errorf("Node %d: Expected 'newdb' to exist", i)
		}
		record, err := node.storage.GetRecord("newdb", "record1")
		if err != nil {
			t.Errorf("Node %d: Failed to get record: %v", i, err)
		} else if len(record) == 0 {
			t.Errorf("Node %d: Expected record to exist, but it doesn't", i)
		} else if string(record[0].Data) != "test data" {
			t.Errorf("Node %d: Expected record data 'test data', got '%s'", i, string(record[0].Data))
		}
	}
}

func TestScalability(t *testing.T) {
	nodeCount := 10
	updateCount := 100

	_, nodes := setupTestEnvironment(t, nodeCount)
	defer tearDownTestEnvironment(nodes)

	// Submit multiple updates to random nodes
	for i := 0; i < updateCount; i++ {
		nodeIndex := i % nodeCount
		update := Update{
			Timestamp:    nodes[nodeIndex].NextTimestamp(),
			Type:         UpsertRecord,
			RecordID:     fmt.Sprintf("record%d", i),
			DataStreams:  []DataStream{{StreamID: 1, Data: []byte(fmt.Sprintf("data%d", i))}},
			DatabaseName: "testdb",
		}
		err := nodes[nodeIndex].SubmitUpdates([]Update{update})
		if err != nil {
			t.Fatalf("Failed to submit update: %v", err)
		}
	}

	// Wait for replication
	time.Sleep(5 * time.Second)

	// Check if all nodes have all records
	for i, node := range nodes {
		for j := 0; j < updateCount; j++ {
			record, err := node.storage.GetRecord("testdb", fmt.Sprintf("record%d", j))
			if err != nil {
				t.Errorf("Node %d: Failed to get record%d: %v", i, j, err)
			}
			if len(record) == 0 || string(record[0].Data) != fmt.Sprintf("data%d", j) {
				t.Errorf("Node %d: Expected record%d data 'data%d', got '%s'", i, j, j, string(record[0].Data))
			}
		}
	}
}

func newMockNetwork() *mockNetwork {
    return &mockNetwork{
        nodes: make(map[string]*ReplicationEngine),
        peers: make(map[string]*MockPeer),
    }
}

func (mn *mockNetwork) addNode(nodeID string, re *ReplicationEngine) {
    mn.nodes[nodeID] = re
    for _, peer := range re.peers {
        mockPeer := NewMockPeer(peer.url)
        mn.peers[peer.url] = mockPeer
        re.peers[peer.url] = mockPeer.Peer
    }
}

func (mn *mockNetwork) connect(nodeID1, nodeID2 string) {
    node1 := mn.nodes[nodeID1]
    node2 := mn.nodes[nodeID2]

    mockPeer1 := NewMockPeer(nodeID2)
    mockPeer2 := NewMockPeer(nodeID1)

    node1.peers[nodeID2] = mockPeer1.Peer
    node2.peers[nodeID1] = mockPeer2.Peer

    mn.peers[nodeID2] = mockPeer1
    mn.peers[nodeID1] = mockPeer2
}

func (mn *mockNetwork) disconnect(nodeID1, nodeID2 string) {
    delete(mn.nodes[nodeID1].peers, nodeID2)
    delete(mn.nodes[nodeID2].peers, nodeID1)
    delete(mn.peers, nodeID2)
    delete(mn.peers, nodeID1)
}

func setupTestEnvironment(t *testing.T, nodeCount int) (*mockNetwork, []*ReplicationEngine) {
    network := newMockNetwork()
    nodes := make([]*ReplicationEngine, nodeCount)

    for i := 0; i < nodeCount; i++ {
        nodeID := fmt.Sprintf("node%d", i)
        storage := NewMockStorage()
        config := ReplicationConfig{
            OwnURL:    nodeID,
            PeerURLs:  []string{},
            JWTSecret: []byte("test_secret"),
        }
        re, err := Init(storage, config, Now())
        if err != nil {
            t.Fatalf("Failed to initialize ReplicationEngine: %v", err)
        }
        nodes[i] = re
        network.addNode(nodeID, re)
    }

    // Connect all nodes in a fully connected topology
    for i := 0; i < nodeCount; i++ {
        for j := i + 1; j < nodeCount; j++ {
            network.connect(fmt.Sprintf("node%d", i), fmt.Sprintf("node%d", j))
        }
    }

    // Simulate connections for all peers
    for _, mockPeer := range network.peers {
        mockPeer.Connect([]byte("test_secret"))
    }

    return network, nodes
}

func tearDownTestEnvironment(nodes []*ReplicationEngine) {
	// Implement any necessary cleanup
}

func TestBasicReplication(t *testing.T) {
	_, nodes := setupTestEnvironment(t, 3)
	defer tearDownTestEnvironment(nodes)

	// Submit an update to node0
	update := Update{
		Timestamp:    nodes[0].NextTimestamp(),
		Type:         UpsertRecord,
		RecordID:     "record1",
		DataStreams:  []DataStream{{StreamID: 1, Data: []byte("test data")}},
		DatabaseName: "testdb",
	}
	err := nodes[0].SubmitUpdates([]Update{update})
	if err != nil {
		t.Fatalf("Failed to submit update: %v", err)
	}

	// Wait for replication
	time.Sleep(1 * time.Second)

	// Check if the update is replicated to all nodes
	for i, node := range nodes {
		record, err := node.storage.GetRecord("testdb", "record1")
		if err != nil {
			t.Errorf("Node %d: Failed to get record: %v", i, err)
		}
		if len(record) == 0 || string(record[0].Data) != "test data" {
			t.Errorf("Node %d: Expected record data 'test data', got '%s'", i, string(record[0].Data))
		}
	}
}

func TestConflictResolution(t *testing.T) {
	_, nodes := setupTestEnvironment(t, 2)
	defer tearDownTestEnvironment(nodes)

	// Submit conflicting updates to both nodes
	update1 := Update{
		Timestamp:    nodes[0].NextTimestamp(),
		Type:         UpsertRecord,
		RecordID:     "record1",
		DataStreams:  []DataStream{{StreamID: 1, Data: []byte("data from node0")}},
		DatabaseName: "testdb",
	}
	update2 := Update{
		Timestamp:    nodes[1].NextTimestamp(),
		Type:         UpsertRecord,
		RecordID:     "record1",
		DataStreams:  []DataStream{{StreamID: 1, Data: []byte("data from node1")}},
		DatabaseName: "testdb",
	}

	nodes[0].SubmitUpdates([]Update{update1})
	nodes[1].SubmitUpdates([]Update{update2})

	// Wait for replication and conflict resolution
	time.Sleep(2 * time.Second)

	// Check if both nodes have the same final state
	record0, _ := nodes[0].storage.GetRecord("testdb", "record1")
	record1, _ := nodes[1].storage.GetRecord("testdb", "record1")

	if string(record0[0].Data) != string(record1[0].Data) {
		t.Errorf("Conflict resolution failed: Node0 has '%s', Node1 has '%s'", string(record0[0].Data), string(record1[0].Data))
	}
}

func TestNetworkPartition(t *testing.T) {
	network, nodes := setupTestEnvironment(t, 3)
	defer tearDownTestEnvironment(nodes)

	// Disconnect node2 from the network
	network.disconnect("node0", "node2")
	network.disconnect("node1", "node2")

	// Submit updates to node0 and node2
	update1 := Update{
		Timestamp:    nodes[0].NextTimestamp(),
		Type:         UpsertRecord,
		RecordID:     "record1",
		DataStreams:  []DataStream{{StreamID: 1, Data: []byte("data from node0")}},
		DatabaseName: "testdb",
	}
	update2 := Update{
		Timestamp:    nodes[2].NextTimestamp(),
		Type:         UpsertRecord,
		RecordID:     "record2",
		DataStreams:  []DataStream{{StreamID: 1, Data: []byte("data from node2")}},
		DatabaseName: "testdb",
	}

	nodes[0].SubmitUpdates([]Update{update1})
	nodes[2].SubmitUpdates([]Update{update2})

	// Wait for replication within partitions
	time.Sleep(1 * time.Second)

	// Reconnect node2
	network.connect("node0", "node2")
	network.connect("node1", "node2")

	// Wait for replication after partition healing
	time.Sleep(2 * time.Second)

	// Check if all nodes have both records
	for i, node := range nodes {
		record1, _ := node.storage.GetRecord("testdb", "record1")
		record2, _ := node.storage.GetRecord("testdb", "record2")

		if len(record1) == 0 || string(record1[0].Data) != "data from node0" {
			t.Errorf("Node %d: Expected record1 data 'data from node0', got '%s'", i, string(record1[0].Data))
		}
		if len(record2) == 0 || string(record2[0].Data) != "data from node2" {
			t.Errorf("Node %d: Expected record2 data 'data from node2', got '%s'", i, string(record2[0].Data))
		}
	}
}
