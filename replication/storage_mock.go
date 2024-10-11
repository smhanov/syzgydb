package replication

import (
    "sync"
    "time"
)

// MockStorage is an in-memory implementation of the StorageInterface for testing purposes.
type MockStorage struct {
    updates      map[string][]Update
    updatesMutex sync.Mutex
    subscribers  []chan Update
    subMutex     sync.Mutex
    databases    map[string]bool
}

// NewMockStorage creates a new instance of MockStorage.
func NewMockStorage() *MockStorage {
    return &MockStorage{
        updates:     make(map[string][]Update),
        databases:   make(map[string]bool),
        subscribers: make([]chan Update, 0),
    }
}

// CommitUpdates applies a list of updates to the mock storage.
func (ms *MockStorage) CommitUpdates(updates []Update) error {
    ms.updatesMutex.Lock()
    defer ms.updatesMutex.Unlock()

    for _, update := range updates {
        if update.Type == CreateDatabase {
            ms.databases[update.DatabaseName] = true
        } else if update.Type == DropDatabase {
            delete(ms.databases, update.DatabaseName)
        }
        ms.updates[update.DatabaseName] = append(ms.updates[update.DatabaseName], update)
    }

    ms.subMutex.Lock()
    for _, update := range updates {
        for _, ch := range ms.subscribers {
            ch <- update
        }
    }
    ms.subMutex.Unlock()

    return nil
}

// GetUpdatesSince retrieves all updates that occurred after the given timestamp.
func (ms *MockStorage) GetUpdatesSince(timestamp Timestamp) (map[string][]Update, error) {
    ms.updatesMutex.Lock()
    defer ms.updatesMutex.Unlock()

    result := make(map[string][]Update)
    for dbName, updates := range ms.updates {
        for _, update := range updates {
            if update.Timestamp.Compare(timestamp) > 0 {
                result[dbName] = append(result[dbName], update)
            }
        }
    }
    return result, nil
}

// ResolveConflict determines which of two conflicting updates should be applied.
func (ms *MockStorage) ResolveConflict(update1, update2 Update) (Update, error) {
    comp := update1.Compare(update2)
    if comp >= 0 {
        return update1, nil
    }
    return update2, nil
}

// SubscribeUpdates returns a channel that receives new updates as they occur.
func (ms *MockStorage) SubscribeUpdates() (<-chan Update, error) {
    ms.subMutex.Lock()
    defer ms.subMutex.Unlock()

    ch := make(chan Update, 100)
    ms.subscribers = append(ms.subscribers, ch)
    return ch, nil
}

// Exists checks if a given dependency (usually a database) exists in the storage.
func (ms *MockStorage) Exists(dependency string) bool {
    ms.updatesMutex.Lock()
    defer ms.updatesMutex.Unlock()
    return ms.databases[dependency]
}

// GenerateUpdate creates a new update for testing purposes.
func (ms *MockStorage) GenerateUpdate(dbName string) {
    ms.updatesMutex.Lock()
    defer ms.updatesMutex.Unlock()

    update := Update{
        Timestamp: Timestamp{
            UnixTime:     time.Now().UnixMilli(),
            LamportClock: int64(len(ms.updates[dbName]) + 1),
        },
        Type:         UpsertRecord,
        Data:         []byte("Sample data"),
        DatabaseName: dbName,
        Dependencies: []string{dbName},
    }
    ms.updates[dbName] = append(ms.updates[dbName], update)

    ms.subMutex.Lock()
    for _, ch := range ms.subscribers {
        ch <- update
    }
    ms.subMutex.Unlock()
}