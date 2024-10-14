package replication

import (
	"fmt"
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
	databases    map[string]bool
	nodeID       int
}

// NewMockStorage creates a new instance of MockStorage.
func NewMockStorage(peerID int) *MockStorage {
	return &MockStorage{
		updates:   make([]Update, 0),
		records:   make(map[string]map[string][]DataStream),
		databases: make(map[string]bool),
		nodeID:    peerID,
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
		log.Printf("[node%d] Committing update: %+v", ms.nodeID, update)
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

	log.Printf("[node%d] Updates since %v: %v", ms.nodeID, timestamp, result)
	return result, hasMore, nil
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
	return nil, fmt.Errorf("record not found")
}

// GenerateUpdate creates a new update for testing purposes.
func (ms *MockStorage) GenerateUpdate(dbName string, ts Timestamp) Update {
	ms.updatesMutex.Lock()
	defer ms.updatesMutex.Unlock()

	update := Update{
		Timestamp: ts,
		Type:      UpsertRecord,
		RecordID:  "test_record_" + time.Now().String(),
		DataStreams: []DataStream{
			{StreamID: 1, Data: []byte("Sample data")},
		},
		DatabaseName: dbName,
	}
	ms.updates = append(ms.updates, update)
	return update
}
