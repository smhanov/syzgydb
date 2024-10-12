// Package replication defines interfaces and types for the SyzgyDB replication system.
package replication

import "bytes"

// UpdateType represents the type of update operation.
type UpdateType int32

const (
    DeleteRecord   UpdateType = 0
    UpsertRecord   UpdateType = 1
    CreateDatabase UpdateType = 2
    DropDatabase   UpdateType = 3
)

type DataStream struct {
    StreamID uint8
    Data     []byte
}

// Update represents a single update operation in the replication system.
type Update struct {
    Timestamp    Timestamp   `json:"timestamp"`
    Type         UpdateType  `json:"type"`
    RecordID     string      `json:"record_id"`
    DataStreams  []DataStream `json:"data_streams"`
    DatabaseName string      `json:"database_name"`
}

// Compare compares two Updates based on their timestamps and record IDs.
func (u Update) Compare(other Update) int {
    tsComp := u.Timestamp.Compare(other.Timestamp)
    if tsComp != 0 {
        return tsComp
    }
    return bytes.Compare([]byte(u.RecordID), []byte(other.RecordID))
}

// ReplicationConfig holds the configuration settings for the replication engine.
type ReplicationConfig struct {
    OwnURL    string   `json:"own_url"`
    PeerURLs  []string `json:"peer_urls"`
    JWTSecret []byte   `json:"jwt_secret"`
}

// StorageInterface defines the methods that must be implemented by any storage backend
// to be compatible with the replication system.
type StorageInterface interface {
    // CommitUpdates applies a list of updates to the storage.
    CommitUpdates(updates []Update) error

    // GetUpdatesSince retrieves all updates that occurred after the given timestamp.
    GetUpdatesSince(timestamp Timestamp) (map[string][]Update, error)

    // ResolveConflict determines which of two conflicting updates should be applied.
    ResolveConflict(update1, update2 Update) (Update, error)

    // SubscribeUpdates returns a channel that receives new updates as they occur.
    SubscribeUpdates() (<-chan Update, error)

    // Exists checks if a given dependency (usually a database) exists in the storage.
    Exists(dependency string) bool

    // GetRecord retrieves a record by its ID and database name.
    GetRecord(databaseName, recordID string) ([]DataStream, error)
}
