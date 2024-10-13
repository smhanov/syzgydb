package replication

import (
	"log"
	"sort"
	"sync"
	"time"
)

// MockStorage is an in-memory implementation of the StorageInterface for testing purposes.
type MockStorage struct {
	updates      map[string][]Update
	records      map[string]map[string][]DataStream
	updatesMutex sync.Mutex
	subscribers  []chan Update
	subMutex     sync.Mutex
	databases    map[string]bool
}

// NewMockStorage creates a new instance of MockStorage.
func NewMockStorage() *MockStorage {
	return &MockStorage{
		updates:     make(map[string][]Update),
		records:     make(map[string]map[string][]DataStream),
		databases:   make(map[string]bool),
		subscribers: make([]chan Update, 0),
	}
}

// CommitUpdates applies a list of updates to the mock storage.
func (ms *MockStorage) CommitUpdates(updates []Update) error {
	ms.updatesMutex.Lock()
	defer ms.updatesMutex.Unlock()

	sort.Slice(updates, func(i, j int) bool {
		return updates[i].Timestamp.Compare(updates[j].Timestamp) < 0
	})

	for _, update := range updates {
		log.Printf("Committing update: %+v", update)
		if update.Type == CreateDatabase {
			ms.databases[update.DatabaseName] = true
			if _, ok := ms.records[update.DatabaseName]; !ok {
				ms.records[update.DatabaseName] = make(map[string][]DataStream)
			}
		} else if update.Type == DropDatabase {
			delete(ms.databases, update.DatabaseName)
			delete(ms.records, update.DatabaseName)
		} else if update.Type == UpsertRecord {
			if _, ok := ms.records[update.DatabaseName]; !ok {
				ms.records[update.DatabaseName] = make(map[string][]DataStream)
			}
			ms.records[update.DatabaseName][update.RecordID] = update.DataStreams
		} else if update.Type == DeleteRecord {
			if dbRecords, ok := ms.records[update.DatabaseName]; ok {
				delete(dbRecords, update.RecordID)
			}
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

// GetUpdatesSince retrieves updates that occurred after the given timestamp, up to maxResults.
func (ms *MockStorage) GetUpdatesSince(timestamp Timestamp, maxResults int) (map[string][]Update, bool, error) {
	ms.updatesMutex.Lock()
	defer ms.updatesMutex.Unlock()

	result := make(map[string][]Update)
	totalUpdates := 0
	hasMore := false

	// Helper function to add an update to the result
	addUpdate := func(dbName string, update Update) bool {
		if totalUpdates >= maxResults {
			hasMore = true
			return false
		}
		if _, ok := result[dbName]; !ok {
			result[dbName] = []Update{}
		}
		result[dbName] = append(result[dbName], update)
		totalUpdates++
		return true
	}

	for dbName, updates := range ms.updates {
		for _, update := range updates {
			if update.Timestamp.Compare(timestamp) > 0 {
				if !addUpdate(dbName, update) {
					break
				}
			}
		}
		if hasMore {
			break
		}
	}

	// Include CreateDatabase and DropDatabase updates
	for dbName, exists := range ms.databases {
		createUpdate, ok := ms.findDatabaseCreateUpdate(dbName)
		if ok && createUpdate.Timestamp.Compare(timestamp) > 0 {
			if !addUpdate(dbName, createUpdate) {
				break
			}
		}
		if !exists {
			dropUpdate, ok := ms.findDatabaseDropUpdate(dbName)
			if ok && dropUpdate.Timestamp.Compare(timestamp) > 0 {
				if !addUpdate(dbName, dropUpdate) {
					break
				}
			}
		}
	}

	log.Printf("Updates since %v: %v", timestamp, result)
	return result, hasMore, nil
}

// findDatabaseCreateUpdate finds the CreateDatabase update for a given database
func (ms *MockStorage) findDatabaseCreateUpdate(dbName string) (Update, bool) {
	updates := ms.updates[dbName]
	for _, update := range updates {
		if update.Type == CreateDatabase {
			return update, true
		}
	}
	return Update{}, false
}

// findDatabaseDropUpdate finds the DropDatabase update for a given database
func (ms *MockStorage) findDatabaseDropUpdate(dbName string) (Update, bool) {
	updates := ms.updates[dbName]
	for i := len(updates) - 1; i >= 0; i-- {
		if updates[i].Type == DropDatabase {
			return updates[i], true
		}
	}
	return Update{}, false
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

// GetRecord retrieves a record by its ID and database name.
func (ms *MockStorage) GetRecord(databaseName, recordID string) ([]DataStream, error) {
	ms.updatesMutex.Lock()
	defer ms.updatesMutex.Unlock()

	if dbRecords, ok := ms.records[databaseName]; ok {
		if record, ok := dbRecords[recordID]; ok {
			return record, nil
		}
	}
	return nil, nil
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
		Type:     UpsertRecord,
		RecordID: "test_record_" + time.Now().String(),
		DataStreams: []DataStream{
			{StreamID: 1, Data: []byte("Sample data")},
		},
		DatabaseName: dbName,
	}
	ms.updates[dbName] = append(ms.updates[dbName], update)

	ms.subMutex.Lock()
	for _, ch := range ms.subscribers {
		ch <- update
	}
	ms.subMutex.Unlock()
}
