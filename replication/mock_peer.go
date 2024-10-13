package replication

import (
	"log"
	"sync"
	"time"
)

type MockPeer struct {
	*Peer
	connectCalled bool
	connection    *mockConnection
	mu            sync.Mutex
}

func NewMockPeer(url string) *MockPeer {
	return &MockPeer{
		Peer: NewPeer(url),
		connection: &mockConnection{
			writtenMessages: make([][]byte, 0),
		},
	}
}

func (mp *MockPeer) Connect(jwtSecret []byte) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.connectCalled = true
	// Simulate a successful connection without actually connecting
	mp.SetConnection(&mockConnection{})
	// Set lastActive to simulate a recent connection
	mp.Peer.lastActive = time.Now()
	// Set the peer as connected
	mp.Peer.connection = &mockConnection{}
}

func (mp *MockPeer) WasConnectCalled() bool {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.connectCalled
}

type mockConnection struct {
	writtenMessages [][]byte
}

func (mc *mockConnection) Close() error { return nil }
func (mc *mockConnection) WriteMessage(_ int, data []byte) error {
	mc.writtenMessages = append(mc.writtenMessages, data)
	log.Printf("Mock connection: Message written: %v", data)
	return nil
}
func (mc *mockConnection) ReadMessage() (int, []byte, error) {
	if len(mc.writtenMessages) > 0 {
		msg := mc.writtenMessages[0]
		mc.writtenMessages = mc.writtenMessages[1:]
		return 1, msg, nil
	}
	return 0, nil, nil
}
