package replication

import (
    "errors"
    "log"
    "net/http"
    "sync"
    "time"
)

type ReplicationEngine struct {
    storage         StorageInterface
    ownURL          string
    peers           map[string]*Peer
    updatesChan     <-chan Update
    jwtSecret       []byte
    lastTimestamp   Timestamp
    mu              sync.Mutex
    bufferedUpdates map[string][]Update
    bufferMu        sync.Mutex
}

func Init(storage StorageInterface, ownURL string, peerURLs []string, jwtSecret []byte) (*ReplicationEngine, error) {
    if storage == nil {
        return nil, errors.New("storage cannot be nil")
    }
    if ownURL == "" {
        return nil, errors.New("ownURL cannot be empty")
    }
    if jwtSecret == nil || len(jwtSecret) == 0 {
        return nil, errors.New("jwtSecret cannot be empty")
    }

    updatesChan, err := storage.SubscribeUpdates()
    if err != nil {
        return nil, err
    }

    re := &ReplicationEngine{
        storage:         storage,
        ownURL:          ownURL,
        peers:           make(map[string]*Peer),
        updatesChan:     updatesChan,
        jwtSecret:       jwtSecret,
        lastTimestamp:   Timestamp{UnixTime: time.Now().UnixMilli(), LamportClock: 0},
        bufferedUpdates: make(map[string][]Update),
    }

    for _, url := range peerURLs {
        re.peers[url] = NewPeer(url)
    }

    go re.processLocalUpdates()
    go re.GossipLoop()
    go re.ConnectToPeers()
    go re.startBufferedUpdatesProcessor()

    return re, nil
}

func (re *ReplicationEngine) GetHandler() http.Handler {
    return http.HandlerFunc(re.HandleWebSocket)
}

func (re *ReplicationEngine) processLocalUpdates() {
    for update := range re.updatesChan {
        re.broadcastUpdate(update)
    }
}

func (re *ReplicationEngine) broadcastUpdate(update Update) {
    re.mu.Lock()
    defer re.mu.Unlock()

    for _, peer := range re.peers {
        if peer.IsConnected() {
            go peer.SendUpdate(update)
        }
    }
}

func (re *ReplicationEngine) handleReceivedUpdate(update Update) {
    if re.dependenciesSatisfied(update.Dependencies) {
        err := re.applyUpdate(update)
        if err != nil {
            log.Println("Failed to apply update:", err)
        }
    } else {
        re.bufferUpdate(update)
    }
}

func (re *ReplicationEngine) dependenciesSatisfied(dependencies []string) bool {
    for _, dep := range dependencies {
        if !re.storage.Exists(dep) {
            return false
        }
    }
    return true
}

func (re *ReplicationEngine) bufferUpdate(update Update) {
    re.bufferMu.Lock()
    defer re.bufferMu.Unlock()
    for _, dep := range update.Dependencies {
        re.bufferedUpdates[dep] = append(re.bufferedUpdates[dep], update)
    }
}

func (re *ReplicationEngine) applyUpdate(update Update) error {
    err := re.storage.CommitUpdates([]Update{update})
    if err != nil {
        return err
    }

    re.mu.Lock()
    if update.Timestamp.Compare(re.lastTimestamp) > 0 {
        re.lastTimestamp = update.Timestamp
    }
    re.mu.Unlock()

    re.processBufferedUpdates(update)
    return nil
}

func (re *ReplicationEngine) processBufferedUpdates(update Update) {
    re.bufferMu.Lock()
    defer re.bufferMu.Unlock()
    depKey := update.DatabaseName
    buffered, exists := re.bufferedUpdates[depKey]
    if exists {
        for _, bufferedUpdate := range buffered {
            if re.dependenciesSatisfied(bufferedUpdate.Dependencies) {
                err := re.applyUpdate(bufferedUpdate)
                if err != nil {
                    log.Println("Failed to apply buffered update:", err)
                }
            } else {
                continue
            }
        }
        delete(re.bufferedUpdates, depKey)
    }
}

func (re *ReplicationEngine) startBufferedUpdatesProcessor() {
    ticker := time.NewTicker(10 * time.Second)
    go func() {
        for range ticker.C {
            re.bufferMu.Lock()
            for dep, updates := range re.bufferedUpdates {
                var remainingUpdates []Update
                for _, update := range updates {
                    if re.dependenciesSatisfied(update.Dependencies) {
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
