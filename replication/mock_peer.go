package replication

import (
    "sync"
)

type MockPeer struct {
    *Peer
    connectCalled bool
    mu            sync.Mutex
}

func NewMockPeer(url string) *MockPeer {
    return &MockPeer{
        Peer: NewPeer(url),
    }
}

func (mp *MockPeer) Connect(jwtSecret []byte) {
    mp.mu.Lock()
    defer mp.mu.Unlock()
    mp.connectCalled = true
    // Simulate a successful connection without actually connecting
    mp.SetConnection(&mockConnection{})
}

func (mp *MockPeer) WasConnectCalled() bool {
    mp.mu.Lock()
    defer mp.mu.Unlock()
    return mp.connectCalled
}

type mockConnection struct{}

func (mc *mockConnection) Close() error                   { return nil }
func (mc *mockConnection) WriteMessage(int, []byte) error { return nil }
func (mc *mockConnection) ReadMessage() (int, []byte, error) { return 0, nil, nil }
