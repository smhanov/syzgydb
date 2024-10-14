package replication

import (
	"testing"
	"time"
)

func TestTimestampComparison(t *testing.T) {
	ts1 := Timestamp{UnixTime: 1000, LamportClock: 1}
	ts2 := Timestamp{UnixTime: 1000, LamportClock: 2}
	if ts1.Compare(ts2) != -1 {
		t.Error("Expected ts1 < ts2")
	}
}

func TestBasicOperations(t *testing.T) {
	storage := NewMockStorage(0)

	// Test creating a database
	createDB := Update{
		Timestamp:    Timestamp{UnixTime: time.Now().UnixMilli(), LamportClock: 1},
		Type:         CreateDatabase,
		DatabaseName: "test_db",
	}
	err := storage.CommitUpdates([]Update{createDB})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Test upserting a record
	upsertRecord := Update{
		Timestamp:    Timestamp{UnixTime: time.Now().UnixMilli(), LamportClock: 2},
		Type:         UpsertRecord,
		RecordID:     "record1",
		DataStreams:  []DataStream{{StreamID: 1, Data: []byte("test data")}},
		DatabaseName: "test_db",
	}
	err = storage.CommitUpdates([]Update{upsertRecord})
	if err != nil {
		t.Fatalf("Failed to upsert record: %v", err)
	}

	// Test deleting a record
	deleteRecord := Update{
		Timestamp:    Timestamp{UnixTime: time.Now().UnixMilli(), LamportClock: 3},
		Type:         DeleteRecord,
		RecordID:     "record1",
		DatabaseName: "test_db",
	}
	err = storage.CommitUpdates([]Update{deleteRecord})
	if err != nil {
		t.Fatalf("Failed to delete record: %v", err)
	}

	// Test dropping a database
	dropDB := Update{
		Timestamp:    Timestamp{UnixTime: time.Now().UnixMilli(), LamportClock: 4},
		Type:         DropDatabase,
		DatabaseName: "test_db",
	}
	err = storage.CommitUpdates([]Update{dropDB})
	if err != nil {
		t.Fatalf("Failed to drop database: %v", err)
	}
}

func TestUpdateOrdering(t *testing.T) {
	storage := NewMockStorage(0)

	updates := []Update{
		{
			Timestamp:    Timestamp{UnixTime: 1000, LamportClock: 2},
			Type:         UpsertRecord,
			RecordID:     "record1",
			DataStreams:  []DataStream{{StreamID: 1, Data: []byte("data2")}},
			DatabaseName: "test_db",
		},
		{
			Timestamp:    Timestamp{UnixTime: 1000, LamportClock: 1},
			Type:         UpsertRecord,
			RecordID:     "record1",
			DataStreams:  []DataStream{{StreamID: 1, Data: []byte("data1")}},
			DatabaseName: "test_db",
		},
	}

	err := storage.CommitUpdates(updates)
	if err != nil {
		t.Fatalf("Failed to commit updates: %v", err)
	}

	record, err := storage.GetRecord("test_db", "record1")
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}

	if string(record[0].Data) != "data2" {
		t.Errorf("Expected final data to be 'data2', got '%s'", string(record[0].Data))
	}
}

func TestUpdateBuffering(t *testing.T) {
	re := &ReplicationEngine{
		storage:         NewMockStorage(0),
		bufferedUpdates: make(map[string][]Update),
	}

	// Try to apply an update for a non-existent database
	update := Update{
		Timestamp:    Timestamp{UnixTime: time.Now().UnixMilli(), LamportClock: 1},
		Type:         UpsertRecord,
		RecordID:     "record1",
		DataStreams:  []DataStream{{StreamID: 1, Data: []byte("test data")}},
		DatabaseName: "non_existent_db",
	}

	re.handleReceivedUpdate(update)

	// Check if the update was buffered
	if len(re.bufferedUpdates["non_existent_db"]) != 1 {
		t.Error("Expected update to be buffered")
	}

	// Create the database
	createDB := Update{
		Timestamp:    Timestamp{UnixTime: time.Now().UnixMilli(), LamportClock: 2},
		Type:         CreateDatabase,
		DatabaseName: "non_existent_db",
	}

	re.handleReceivedUpdate(createDB)

	// Use a timeout to ensure the test doesn't run indefinitely
	timeout := time.After(5 * time.Second)
	done := make(chan bool)

	go func() {
		// Check if the buffered update was applied
		for len(re.bufferedUpdates["non_existent_db"]) != 0 {
			time.Sleep(100 * time.Millisecond)
		}

		record, err := re.storage.GetRecord("non_existent_db", "record1")
		if err != nil {
			t.Errorf("Failed to get record: %v", err)
		} else if len(record) == 0 || string(record[0].Data) != "test data" {
			t.Error("Expected buffered update to be applied correctly")
		}

		done <- true
	}()

	select {
	case <-timeout:
		t.Fatal("Test timed out")
	case <-done:
		// Test completed successfully
	}
}

func TestEngineConflictResolution(t *testing.T) {
	storage := NewMockStorage(0)

	update1 := Update{
		Timestamp:    Timestamp{UnixTime: 1000, LamportClock: 1},
		Type:         UpsertRecord,
		RecordID:     "record1",
		DataStreams:  []DataStream{{StreamID: 1, Data: []byte("data1")}},
		DatabaseName: "test_db",
	}

	update2 := Update{
		Timestamp:    Timestamp{UnixTime: 1000, LamportClock: 2},
		Type:         UpsertRecord,
		RecordID:     "record1",
		DataStreams:  []DataStream{{StreamID: 1, Data: []byte("data2")}},
		DatabaseName: "test_db",
	}

	resolvedUpdate, err := storage.ResolveConflict(update1, update2)
	if err != nil {
		t.Fatalf("Failed to resolve conflict: %v", err)
	}

	if string(resolvedUpdate.DataStreams[0].Data) != "data2" {
		t.Errorf("Expected resolved update to have data 'data2', got '%s'", string(resolvedUpdate.DataStreams[0].Data))
	}
}

// Additional tests for replication logic, including out-of-order updates and dependency buffering.
