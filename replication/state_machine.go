package replication

import (
	"encoding/json"
	"sync"

	pb "github.com/smhanov/syzgydb/replication/proto"
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

func (sm *StateMachine) updateTimestamp(t Timestamp) {
	if t.After(sm.timestamp) {
		sm.timestamp = t
	}
}

func (sm *StateMachine) saveState() ([]byte, error) {
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

func (sm *StateMachine) nextTimestamp() Timestamp {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	return sm.timestamp.Next(true)
}

func (sm *StateMachine) incrementAndGetTimestamp() *pb.Timestamp {
	ts := sm.nextTimestamp()
	return &pb.Timestamp{
		UnixTime:     ts.UnixTime,
		LamportClock: ts.LamportClock,
	}
}
