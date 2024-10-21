// Package replication implements a distributed replication system for SyzgyDB.
package replication

import (
	"errors"
	"net"
	"net/http"
	"time"
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
	storage      StorageInterface
}

func Init(storage StorageInterface, config ReplicationConfig, state []byte) (*ReplicationEngine, error) {
	if storage == nil {
		return nil, errors.New("storage cannot be nil")
	}
	if config.OwnURL == "" {
		return nil, errors.New("config.OwnURL cannot be empty")
	}
	if len(config.JWTSecret) == 0 {
		return nil, errors.New("config.JWTSecret cannot be empty")
	}

	sm := NewStateMachine(storage, config, state)

	re := &ReplicationEngine{
		stateMachine: sm,
		storage:      storage,
	}

	re.startHeartbeatTimer()

	return re, nil
}

func (re *ReplicationEngine) GetHandler() http.Handler {
	return http.HandlerFunc(re.HandleWebSocket)
}

func (re *ReplicationEngine) SubmitUpdates(updates []Update) error {
	replyChan := make(chan error)
	re.stateMachine.eventChan <- LocalUpdatesEvent{Updates: updates, ReplyChan: replyChan}
	return <-replyChan
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

func (re *ReplicationEngine) NextTimestamp() Timestamp {
	return re.stateMachine.nextTimestamp()
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
	re.stateMachine.eventChan <- AddPeerEvent{URL: url}
}

func (re *ReplicationEngine) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	replyChan := make(chan struct{})
	re.stateMachine.eventChan <- WebSocketConnectionEvent{ResponseWriter: w, Request: r, ReplyChan: replyChan}
	<-replyChan // Wait for the event to be processed
}

func (re *ReplicationEngine) startHeartbeatTimer() {
	ticker := time.NewTicker(30 * time.Second) // Adjust the interval as needed
	go func() {
		for {
			select {
			case <-ticker.C:
				re.stateMachine.eventChan <- PeerHeartbeatEvent{}
			case <-re.stateMachine.done:
				ticker.Stop()
				return
			}
		}
	}()
}
