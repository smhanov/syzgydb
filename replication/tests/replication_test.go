package replication_test

import (
    "testing"
    "time"

    "your_project/replication"
)

func TestTimestampComparison(t *testing.T) {
    ts1 := replication.Timestamp{UnixTime: 1000, LamportClock: 1}
    ts2 := replication.Timestamp{UnixTime: 1000, LamportClock: 2}
    if ts1.Compare(ts2) != -1 {
        t.Error("Expected ts1 < ts2")
    }
}

func TestMockStorage(t *testing.T) {
    storage := replication.NewMockStorage()

    // Create a database
    storage.CommitUpdates([]replication.Update{
        {
            Timestamp:    replication.Timestamp{UnixTime: time.Now().UnixMilli(), LamportClock: 1},
            Type:         replication.CreateDatabase,
            Data:         []byte("test_db"),
            DatabaseName: "test_db",
        },
    })

    // Subscribe to updates
    ch, err := storage.SubscribeUpdates()
    if err != nil {
        t.Fatal("Failed to subscribe to updates:", err)
    }

    // Generate an update
    storage.GenerateUpdate("test_db")

    select {
    case update := <-ch:
        if update.Type != replication.UpsertRecord {
            t.Error("Expected UpsertRecord update type")
        }
    case <-time.After(1 * time.Second):
        t.Error("Timeout waiting for update")
    }
}

// Additional tests for replication logic, including out-of-order updates and dependency buffering.
