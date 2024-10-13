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
		Peer:       NewPeer(url),
		connection: newMockConnection(),
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
	readChan        chan []byte
	mu              sync.Mutex
}

func newMockConnection() *mockConnection {
	return &mockConnection{
		readChan: make(chan []byte, 100),
	}
}

func (mc *mockConnection) Close() error { return nil }

func (mc *mockConnection) WriteMessage(_ int, data []byte) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.writtenMessages = append(mc.writtenMessages, data)
	mc.readChan <- data
	log.Printf("Mock connection: Message written: %v", data)
	return nil
}

func (mc *mockConnection) ReadMessage() (int, []byte, error) {
	msg := <-mc.readChan
	return 1, msg, nil
}
