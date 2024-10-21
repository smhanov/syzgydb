package replication

import (
	"encoding/json"
	"sync"
)

type Event interface {
	process(sm *StateMachine)
}

type StateMachine struct {
	mu              sync.Mutex
	storage         StorageInterface
	config          ReplicationConfig
	peers           map[string]*Peer
	nodeSequences   *NodeSequences
	bufferedUpdates map[string][]Update
	eventChan       chan Event
	done            chan struct{}
	timestamp       Timestamp
}

type replicationState struct {
	NodeSequences *NodeSequences `json:"node_sequences"`
	Timestamp     Timestamp      `json:"timestamp"`
}

func NewStateMachine(storage StorageInterface, config ReplicationConfig, state []byte) *StateMachine {
	var rstate replicationState
	if len(state) > 0 {
		if err := json.Unmarshal(state, &rstate); err != nil {
			panic("failed to unmarshal replication state: " + err.Error())
		}
	} else {
		rstate = replicationState{
			NodeSequences: NewNodeSequences(),
			Timestamp:     Now(),
		}
	}

	sm := &StateMachine{
		storage:         storage,
		config:          config,
		peers:           make(map[string]*Peer),
		nodeSequences:   rstate.NodeSequences,
		bufferedUpdates: make(map[string][]Update),
		eventChan:       make(chan Event, 1000),
		done:            make(chan struct{}),
		timestamp:       rstate.Timestamp,
	}

	go sm.eventLoop()
	return sm
}

func (sm *StateMachine) eventLoop() {
	for {
		select {
		case event := <-sm.eventChan:
			event.process(sm)
		case <-sm.done:
			return
		}
	}
}

func (sm *StateMachine) Stop() {
	close(sm.done)
}

func (sm *StateMachine) UpdateTimestamp(t Timestamp) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if t.After(sm.timestamp) {
		sm.timestamp = t
	}
}

func (sm *StateMachine) SaveState() ([]byte, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	state := replicationState{
		NodeSequences: sm.nodeSequences,
		Timestamp:     sm.timestamp,
	}
	return json.Marshal(state)
}

func (sm *StateMachine) getPeerURLs() []string {
	keys := make([]string, 0, len(sm.peers))
	for key := range sm.peers {
		keys = append(keys, key)
	}
	return keys
}

func (sm *StateMachine) NextTimestamp(local bool) *VectorClock {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// Increment the vector clock for this node
	nodeID := uint64(sm.config.NodeID)
	currentTimestamp, exists := sm.lastKnownVectorClock.Get(nodeID)
	if !exists {
		currentTimestamp = Timestamp{}
	}
	newTimestamp := currentTimestamp.Next(local)
	sm.lastKnownVectorClock.Update(nodeID, newTimestamp)

	// Return a copy of the updated vector clock
	return sm.lastKnownVectorClock.Clone()
}

func (sm *StateMachine) NextLocalTimestamp() Timestamp {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	cur, _ := sm.lastKnownVectorClock.Get(uint64(sm.config.NodeID))
	cur = cur.Next(true)
	sm.lastKnownVectorClock.Update(uint64(sm.config.NodeID), cur)
	return cur
}
