package replication

import (
	"encoding/json"
	"sync"
	"time"

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
	scheduledEvents map[string]bool
	lastSavedState  []byte
	stateTimer      *time.Ticker
}

type replicationState struct {
	NodeSequences *NodeSequences `json:"node_sequences"`
	Timestamp     Timestamp      `json:"timestamp"`
	PeerURLs      []string      `json:"peer_urls"`
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
			PeerURLs:     make([]string, 0),
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
		lastSavedState:  state,
		stateTimer:      time.NewTicker(5 * time.Second),
	}

	// Initialize peers from saved state
	for _, url := range rstate.PeerURLs {
		if url != config.OwnURL { // Don't add ourselves as a peer
			sm.peers[url] = NewPeer("", url, sm)
		}
	}

	go sm.eventLoop()
	go sm.stateCheckLoop() // Start the state check loop
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
	sm.stateTimer.Stop()
	close(sm.done)
}

func (sm *StateMachine) updateTimestamp(t Timestamp) {
	if t.After(sm.timestamp) {
		sm.timestamp = t
	}
}

func (sm *StateMachine) saveState() ([]byte, error) {
	// Get peer URLs and sort them
	peerURLs := make([]string, 0, len(sm.peers))
	for url := range sm.peers {
		peerURLs = append(peerURLs, url)
	}
	sort.Strings(peerURLs)

	state := replicationState{
		NodeSequences: sm.nodeSequences,
		Timestamp:     sm.timestamp,
		PeerURLs:      peerURLs,
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

func (sm *StateMachine) scheduleEvent(eventType string, event Event, delay time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.scheduledEvents == nil {
		sm.scheduledEvents = make(map[string]bool)
	}

	if !sm.scheduledEvents[eventType] {
		sm.scheduledEvents[eventType] = true
		time.AfterFunc(delay, func() {
			sm.eventChan <- event
			sm.mu.Lock()
			delete(sm.scheduledEvents, eventType)
			sm.mu.Unlock()
		})
	}
}
func (sm *StateMachine) stateCheckLoop() {
	for {
		select {
		case <-sm.stateTimer.C:
			currentState, err := sm.saveState()
			if err != nil {
				log.Printf("Error generating state: %v", err)
				continue
			}

			// Only save if state has changed
			if string(currentState) != string(sm.lastSavedState) {
				err = sm.storage.SaveState(currentState)
				if err != nil {
					log.Printf("Error saving state: %v", err)
					continue
				}
				sm.lastSavedState = currentState
			}
		case <-sm.done:
			sm.stateTimer.Stop()
			return
		}
	}
}
