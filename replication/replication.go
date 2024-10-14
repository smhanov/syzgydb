// Package replication implements a distributed replication system for SyzgyDB.
package replication

import (
	"errors"
	"log"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Connection interface {
	Close() error
	WriteMessage(int, []byte) error
	ReadMessage() (int, []byte, error)
}

type Peer struct {
	url                string
	name               string
	connection         Connection
	lastActive         time.Time
	lastKnownTimestamp Timestamp
	mu                 sync.Mutex
	re                 *ReplicationEngine
}

// SetConnection sets the WebSocket connection for the peer.
func (p *Peer) SetConnection(conn Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connection = conn
}

// ReplicationEngine is the core component of the replication system.
// It manages peer connections, handles local and remote updates,
// and coordinates the gossip protocol.
type ReplicationEngine struct {
	storage         StorageInterface
	config          ReplicationConfig
	peers           map[string]*Peer
	lastTimestamp   Timestamp
	mu              sync.Mutex
	bufferedUpdates map[string][]Update
	bufferMu        sync.Mutex
	updateRequests  map[string]*updateRequest
	gossipTicker    *time.Ticker
	gossipDone      chan bool
	server          *http.Server
	listener        net.Listener
	name            string
}

// GetUpdatesSince retrieves updates that occurred after the given timestamp, up to maxResults,
// and combines them with any buffered updates that have unmet dependencies.
func (re *ReplicationEngine) GetUpdatesSince(timestamp Timestamp, maxResults int) (map[string][]Update, bool, error) {
	// Get updates from storage
	updates, hasMore, err := re.storage.GetUpdatesSince(timestamp, maxResults)
	if err != nil {
		return nil, false, err
	}

	// Combine with buffered updates
	re.bufferMu.Lock()
	defer re.bufferMu.Unlock()

	for dbName, bufferedUpdates := range re.bufferedUpdates {
		for _, update := range bufferedUpdates {
			if update.Timestamp.After(timestamp) {
				if updates[dbName] == nil {
					updates[dbName] = []Update{}
				}
				updates[dbName] = append(updates[dbName], update)
			}
		}
	}

	// Sort and limit the combined updates
	for dbName := range updates {
		sortAndLimitUpdates(updates[dbName], maxResults)
	}

	return updates, hasMore, nil
}

// sortAndLimitUpdates sorts the updates by timestamp and limits them to maxResults
func sortAndLimitUpdates(updates []Update, maxResults int) {
	sort.Slice(updates, func(i, j int) bool {
		return updates[i].Timestamp.Before(updates[j].Timestamp)
	})
	if len(updates) > maxResults {
		updates = updates[:maxResults]
	}
}

type updateRequest struct {
	peerURL      string
	since        *VectorClock
	inProgress   bool
	responseChan chan bool
}

// Init initializes a new ReplicationEngine with the given parameters.
// It sets up the necessary structures, starts background processes,
// and prepares the engine for operation.
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

	re := &ReplicationEngine{
		storage:              storage,
		config:               config,
		peers:                make(map[string]*Peer),
		lastKnownVectorClock: localVectorClock,
		bufferedUpdates:      make(map[string][]Update),
		gossipTicker:         time.NewTicker(5 * time.Second),
		gossipDone:           make(chan bool),
		name:                 config.OwnURL,
	}

	for _, url := range config.PeerURLs {
		re.peers[url] = NewPeer("c:?", url, re)
	}

	// Start background processes
	go re.GossipLoop()
	go re.ConnectToPeers()
	go re.startBufferedUpdatesProcessor()

	return re, nil
}

// GetHandler returns an http.Handler for handling WebSocket connections.
// This is used to set up the WebSocket endpoint for peer communication.
func (re *ReplicationEngine) GetHandler() http.Handler {
	return http.HandlerFunc(re.HandleWebSocket)
}

// SubmitUpdates commits a batch of updates to storage and broadcasts them to peers.
func (re *ReplicationEngine) SubmitUpdates(updates []Update) error {
	// Commit updates to storage
	err := re.storage.CommitUpdates(updates)
	if err != nil {
		return err
	}

	// Update lastTimestamp
	re.mu.Lock()
	if len(updates) > 0 && updates[len(updates)-1].Timestamp.After(re.lastTimestamp) {
		re.lastTimestamp = updates[len(updates)-1].Timestamp
	}
	re.mu.Unlock()

	// Broadcast updates to peers
	for _, update := range updates {
		re.broadcastUpdate(update)
	}

	return nil
}

// broadcastUpdate sends an update to all connected peers.
func (re *ReplicationEngine) broadcastUpdate(update Update) {
	ts := re.NextTimestamp(false)
	re.mu.Lock()
	defer re.mu.Unlock()

	for _, peer := range re.peers {
		if peer.IsConnected() {
			go peer.SendUpdate(update, ts)
		}
	}
}

// dependenciesSatisfied checks if the database of an update exists.
func (re *ReplicationEngine) dependenciesSatisfied(update Update) bool {
	return update.Type == CreateDatabase || re.storage.Exists(update.DatabaseName)
}

// bufferUpdate stores an update that can't be applied immediately due to unmet dependencies.
func (re *ReplicationEngine) bufferUpdate(update Update) {
	re.bufferMu.Lock()
	defer re.bufferMu.Unlock()
	re.bufferedUpdates[update.DatabaseName] = append(re.bufferedUpdates[update.DatabaseName], update)
}

// applyUpdate commits an update to storage.
func (re *ReplicationEngine) applyUpdate(update Update) error {
	log.Printf("Apply update %+v", update)
	err := re.storage.CommitUpdates([]Update{update})
	if err != nil {
		return err
	}

	re.mu.Lock()
	if update.Timestamp.Compare(re.lastTimestamp) > 0 {
		re.lastTimestamp = update.Timestamp
	}
	re.mu.Unlock()

	return nil
}

// applyUpdateAndProcessBuffer applies an update and processes buffered updates.
func (re *ReplicationEngine) applyUpdateAndProcessBuffer(update Update) error {
	err := re.applyUpdate(update)
	if err != nil {
		return err
	}

	re.processBufferedUpdates(update)
	return nil
}

// handleReceivedUpdate processes an update received from a peer.
// It checks if the database exists and either applies the update or buffers it.
func (re *ReplicationEngine) handleReceivedUpdate(update Update) {
	log.Printf("Received update: %+v", update)
	if update.Type == CreateDatabase {
		log.Printf("Applying CreateDatabase update for %s", update.DatabaseName)
		err := re.applyUpdateAndProcessBuffer(update)
		if err != nil {
			log.Println("Failed to apply CreateDatabase update:", err)
		} else {
			log.Printf("Successfully applied CreateDatabase update for %s", update.DatabaseName)
		}
	} else if re.dependenciesSatisfied(update) {
		log.Printf("Dependencies satisfied for update: %+v", update)
		err := re.applyUpdateAndProcessBuffer(update)
		if err != nil {
			log.Println("Failed to apply update:", err)
		} else {
			log.Printf("Successfully applied update: %+v", update)
		}
	} else {
		log.Printf("Buffering update due to unsatisfied dependencies: %+v", update)
		re.bufferUpdate(update)
	}
}

// processBufferedUpdates attempts to apply buffered updates whose dependencies are now satisfied.
func (re *ReplicationEngine) processBufferedUpdates(update Update) {
	re.bufferMu.Lock()
	defer re.bufferMu.Unlock()
	depKey := update.DatabaseName
	buffered, exists := re.bufferedUpdates[depKey]
	if exists {
		var remainingUpdates []Update
		for _, bufferedUpdate := range buffered {
			if re.dependenciesSatisfied(bufferedUpdate) {
				err := re.applyUpdate(bufferedUpdate)
				if err != nil {
					log.Println("Failed to apply buffered update:", err)
					remainingUpdates = append(remainingUpdates, bufferedUpdate)
				}
			} else {
				remainingUpdates = append(remainingUpdates, bufferedUpdate)
			}
		}
		if len(remainingUpdates) > 0 {
			re.bufferedUpdates[depKey] = remainingUpdates
		} else {
			delete(re.bufferedUpdates, depKey)
		}
	}
}

// startBufferedUpdatesProcessor periodically attempts to apply buffered updates.
func (re *ReplicationEngine) startBufferedUpdatesProcessor() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			re.bufferMu.Lock()
			for dep, updates := range re.bufferedUpdates {
				var remainingUpdates []Update
				for _, update := range updates {
					if re.dependenciesSatisfied(update) {
						err := re.applyUpdate(update)
						if err != nil {
							log.Println("Failed to apply buffered update:", err)
							remainingUpdates = append(remainingUpdates, update)
						}
					} else {
						remainingUpdates = append(remainingUpdates, update)
					}
				}
				if len(remainingUpdates) > 0 {
					re.bufferedUpdates[dep] = remainingUpdates
				} else {
					delete(re.bufferedUpdates, dep)
				}
			}
			re.bufferMu.Unlock()
		}
	}()
}

// NextTimestamp generates and returns the next logical timestamp.
func (re *ReplicationEngine) NextTimestamp(local bool) Timestamp {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.lastTimestamp = re.lastTimestamp.Next(local)
	return re.lastTimestamp
}

// NextTimestamp generates and returns the next logical timestamp.
func (re *ReplicationEngine) handleReceivedTimestamp(ts Timestamp) {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.lastTimestamp.LamportClock = max(re.lastTimestamp.LamportClock, ts.LamportClock)
}

// Listen starts the ReplicationEngine's HTTP server on the specified address.
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

// Close shuts down the ReplicationEngine's HTTP server and listener.
func (re *ReplicationEngine) Close() error {
	if re.server != nil {
		re.server.Close()
	}
	if re.listener != nil {
		re.listener.Close()
	}
	return nil
}

// AddPeer adds a new peer to the ReplicationEngine and immediately triggers a connection request.
func (re *ReplicationEngine) AddPeer(url string) {
	re.mu.Lock()
	defer re.mu.Unlock()

	if _, exists := re.peers[url]; !exists {
		peer := NewPeer("a:?", url, re)
		re.peers[url] = peer
		go peer.Connect(re.config.JWTSecret)
	}
}
