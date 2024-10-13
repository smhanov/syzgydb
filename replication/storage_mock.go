package replication

import (
	"log"
	"sort"
	"sync"
	"time"
)

// MockStorage is an in-memory implementation of the StorageInterface for testing purposes.
type MockStorage struct {
	updates      []Update
	records      map[string]map[string][]DataStream
	updatesMutex sync.Mutex
	subMutex     sync.Mutex
	databases    map[string]bool
}

// NewMockStorage creates a new instance of MockStorage.
func NewMockStorage() *MockStorage {
	return &MockStorage{
		updates:   make([]Update, 0),
		records:   make(map[string]map[string][]DataStream),
		databases: make(map[string]bool),
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
		ms.updates = append(ms.updates, update)
	}

	return nil
}

// GetUpdatesSince retrieves updates that occurred after the given timestamp, up to maxResults.
func (ms *MockStorage) GetUpdatesSince(timestamp Timestamp, maxResults int) (map[string][]Update, bool, error) {
	ms.updatesMutex.Lock()
	defer ms.updatesMutex.Unlock()

	result := make(map[string][]Update)
	totalUpdates := 0
	hasMore := false

	for _, update := range ms.updates {
		if update.Timestamp.Compare(timestamp) > 0 {
			if totalUpdates >= maxResults {
				hasMore = true
				break
			}
			if _, ok := result[update.DatabaseName]; !ok {
				result[update.DatabaseName] = []Update{}
			}
			result[update.DatabaseName] = append(result[update.DatabaseName], update)
			totalUpdates++
		}
	}

	log.Printf("Updates since %v: %v", timestamp, result)
	return result, hasMore, nil
}

// findDatabaseCreateUpdate finds the CreateDatabase update for a given database
func (ms *MockStorage) findDatabaseCreateUpdate(dbName string) (Update, bool) {
	for _, update := range ms.updates {
		if update.Type == CreateDatabase && update.DatabaseName == dbName {
			return update, true
		}
	}
	return Update{}, false
}

// findDatabaseDropUpdate finds the DropDatabase update for a given database
func (ms *MockStorage) findDatabaseDropUpdate(dbName string) (Update, bool) {
	for i := len(ms.updates) - 1; i >= 0; i-- {
		if ms.updates[i].Type == DropDatabase && ms.updates[i].DatabaseName == dbName {
			return ms.updates[i], true
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
}
