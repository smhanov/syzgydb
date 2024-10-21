package replication

import (
	"fmt"
	"log"
	"sync"
)

// MockStorage is an in-memory implementation of the StorageInterface for testing purposes.
type MockStorage struct {
	updates      map[string]Update
	updatesMutex sync.Mutex
	nodeID       int
}

// NewMockStorage creates a new instance of MockStorage.
func NewMockStorage(peerID int) *MockStorage {
	return &MockStorage{
		updates: make(map[string]Update),
		nodeID:  peerID,
	}
}

func (ms *MockStorage) writeUpdate(key string, update Update) {
	// If the update is already in the storage, and its existing timestamp is newer, then ignore the new update.
	if existingUpdate, ok := ms.updates[key]; ok {
		if existingUpdate.Timestamp.After(update.Timestamp) {
			log.Printf("Ignoring update %s %+v because existing update %+v is newer", key, update, existingUpdate)
			return
		}
	}

	ms.updates[key] = update
}

// CommitUpdates applies a list of updates to the mock storage.
func (ms *MockStorage) CommitUpdates(updates []Update) error {
	ms.updatesMutex.Lock()
	defer ms.updatesMutex.Unlock()

	for _, update := range updates {
		log.Printf("[%d] Committing update: %+v", ms.nodeID, update)
		key := update.DatabaseName + ":" + update.RecordID
		if update.Type == CreateDatabase || update.Type == DropDatabase {
			key = fmt.Sprintf("_db:%s", update.DatabaseName)
		}
		ms.writeUpdate(key, update)
	}

	return nil
}

// GetUpdatesSince retrieves updates that occurred after the given vector clock, up to maxResults.
func (ms *MockStorage) GetUpdatesSince(sequences *NodeSequences, maxResults int) (map[string][]Update, bool, error) {
	ms.updatesMutex.Lock()
	defer ms.updatesMutex.Unlock()

	result := make(map[string][]Update)
	totalUpdates := 0
	hasMore := false

	for _, update := range ms.updates {
		if sequences.BeforeNode(update.NodeID, update.SequenceNo) {
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

	log.Printf("[node%d] Updates since %v: %v", ms.nodeID, sequences, result)
	return result, hasMore, nil
}

// ResolveConflict determines which of two conflicting updates should be applied.
func (ms *MockStorage) ResolveConflict(update1, update2 Update) (Update, error) {
	comp := update1.Timestamp.Compare(update2.Timestamp)
	if comp > 0 {
		return update1, nil
	} else if comp < 0 {
		return update2, nil
	} else if update1.NodeID < update2.NodeID {
		return update1, nil
	}
	return update2, nil
}

// Exists checks if a given dependency (usually a database) exists in the storage.
func (ms *MockStorage) Exists(dependency string) bool {
	ms.updatesMutex.Lock()
	defer ms.updatesMutex.Unlock()
	key := fmt.Sprintf("_db:%s", dependency)
	update, ok := ms.updates[key]
	return ok && update.Type != DropDatabase
}

// GetRecord retrieves a record by its ID and database name.
func (ms *MockStorage) GetRecord(databaseName, recordID string) ([]DataStream, error) {
	ms.updatesMutex.Lock()
	defer ms.updatesMutex.Unlock()

	if record, ok := ms.updates[databaseName+":"+recordID]; ok {
		if record.Type == UpsertRecord {
			return record.DataStreams, nil
		}
	}
	return nil, fmt.Errorf("record not found")
}

func (ms *MockStorage) SaveState(state []byte) error {
	return nil
}
