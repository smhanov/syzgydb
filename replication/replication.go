// Package replication implements a distributed replication system for SyzgyDB.
package replication

import (
	"errors"
	"net"
	"net/http"
)

type Connection interface {
	Close() error
	WriteMessage(int, []byte) error
	ReadMessage() (int, []byte, error)
}

type ReplicationEngine struct {
	stateMachine *StateMachine
	server       *http.Server
	listener     net.Listener
	eventChan    chan Event
}

func Init(storage StorageInterface, config ReplicationConfig, localVectorClock *VectorClock) (*ReplicationEngine, error) {
	if storage == nil {
		return nil, errors.New("storage cannot be nil")
	}
	if config.OwnURL == "" {
		return nil, errors.New("config.OwnURL cannot be empty")
	}
	if len(config.JWTSecret) == 0 {
		return nil, errors.New("config.JWTSecret cannot be empty")
	}

	eventChan := make(chan Event, 1000)
	sm := NewStateMachine(storage, config, localVectorClock, eventChan)

	re := &ReplicationEngine{
		stateMachine: sm,
		eventChan:    eventChan,
	}

	go sm.eventLoop()

	return re, nil
}

func (re *ReplicationEngine) GetHandler() http.Handler {
	return http.HandlerFunc(re.HandleWebSocket)
}

func (re *ReplicationEngine) SubmitUpdates(updates []Update) error {
	for _, update := range updates {
		re.stateMachine.eventChan <- ReceivedUpdateEvent{Update: update}
	}
	return nil
}

func (re *ReplicationEngine) Listen(address string) error {
	var err error
	re.listener, err = net.Listen("tcp", address)
	if err != nil {
		return err
	}

	re.server = &http.Server{
		Handler: re.GetHandler(),
	}

	go re.server.Serve(re.listener)
	return nil
}

func (re *ReplicationEngine) Close() error {
	re.stateMachine.Stop()
	if re.server != nil {
		re.server.Close()
	}
	if re.listener != nil {
		re.listener.Close()
	}
	return nil
}

func (re *ReplicationEngine) AddPeer(url string) {
	re.eventChan <- AddPeerEvent{URL: url}
}

func (re *ReplicationEngine) NextTimestamp(local bool) *VectorClock {
	return re.stateMachine.NextTimestamp(local)
}

func (re *ReplicationEngine) NextLocalTimestamp() Timestamp {
	return re.stateMachine.NextLocalTimestamp()
}

func (re *ReplicationEngine) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	re.eventChan <- WebSocketConnectionEvent{ResponseWriter: w, Request: r}
}
