package replication

type Event interface {
	process(sm *StateMachine)
}

type StateMachine struct {
	storage              StorageInterface
	config               ReplicationConfig
	peers                map[string]*Peer
	lastKnownVectorClock *VectorClock
	bufferedUpdates      map[string][]Update
	eventChan            chan Event
	done                 chan struct{}
}

func NewStateMachine(storage StorageInterface, config ReplicationConfig, localVectorClock *VectorClock) *StateMachine {
	sm := &StateMachine{
		storage:              storage,
		config:               config,
		peers:                make(map[string]*Peer),
		lastKnownVectorClock: localVectorClock.Clone(),
		bufferedUpdates:      make(map[string][]Update),
		eventChan:            make(chan Event, 1000),
		done:                 make(chan struct{}),
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

func (sm *StateMachine) getPeerURLs() []string {
	keys := make([]string, 0, len(sm.peers))
	for key := range sm.peers {
		keys = append(keys, key)
	}
	return keys
}

func (sm *StateMachine) NextTimestamp(local bool) *VectorClock {
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
	cur, _ := sm.lastKnownVectorClock.Get(uint64(sm.config.NodeID))
	cur = cur.Next(true)
	sm.lastKnownVectorClock.Update(uint64(sm.config.NodeID), cur)
	return cur
}
