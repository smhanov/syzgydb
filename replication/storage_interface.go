// Package replication defines interfaces and types for the SyzgyDB replication system.
package replication

import (
	"bytes"
	"fmt"
)

// UpdateType represents the type of update operation.
type UpdateType int32

const (
	DeleteRecord   UpdateType = 0
	UpsertRecord   UpdateType = 1
	CreateDatabase UpdateType = 2
	DropDatabase   UpdateType = 3
	Superceded     UpdateType = 4
)

type DataStream struct {
	StreamID uint8
	Data     []byte
}

// Update represents a single update operation in the replication system.
type Update struct {
	NodeID       uint64       `json:"node_id"`
	SequenceNo   uint64       `json:"sequence_no"`
	Timestamp    Timestamp    `json:"timestamp"`
	Type         UpdateType   `json:"type"`
	RecordID     string       `json:"record_id"`
	DataStreams  []DataStream `json:"data_streams"`
	DatabaseName string       `json:"database_name"`
}

// Compare compares two Updates based on their vector clocks and record IDs.
func (u Update) Compare(other Update) int {
	vcComp := u.Timestamp.Compare(other.Timestamp)
	if vcComp != 0 {
		return vcComp
	}
	return bytes.Compare([]byte(u.RecordID), []byte(other.RecordID))
}

// String returns a string representation of the Update.
func (u Update) String() string {
	typeStr := ""
	switch u.Type {
	case DeleteRecord:
		typeStr = "DeleteRecord"
	case UpsertRecord:
		typeStr = "UpsertRecord"
	case CreateDatabase:
		typeStr = "CreateDatabase"
	case DropDatabase:
		typeStr = "DropDatabase"
	default:
		typeStr = fmt.Sprintf("Unknown(%d)", u.Type)
	}
	return fmt.Sprintf("Update{%s %s/%s %d@%d}",
		typeStr, u.DatabaseName, u.RecordID, u.NodeID, u.SequenceNo)
}

// ReplicationConfig holds the configuration settings for the replication engine.
type ReplicationConfig struct {
	OwnURL    string   `json:"own_url"`
	PeerURLs  []string `json:"peer_urls"`
	JWTSecret []byte   `json:"jwt_secret"`
	NodeID    uint64   `json:"node_id"`
}

// StorageInterface defines the methods that must be implemented by any storage backend
// to be compatible with the replication system.
type StorageInterface interface {
	// CommitUpdates applies a list of updates to the storage.
	CommitUpdates(updates []Update) error

	// GetUpdatesSince retrieves updates that occurred after the given vector clock, up to maxResults.
	// It returns the updates, a boolean indicating if there are more results, and an error if any.
	GetUpdatesSince(nodeSequences *NodeSequences, maxResults int) ([]Update, bool, error)

	// ResolveConflict determines which of two conflicting updates should be applied.
	ResolveConflict(update1, update2 Update) (Update, error)

	// Exists checks if a given dependency (usually a database) exists in the storage.
	Exists(dependency string) bool

	// GetRecord retrieves a record by its ID and database name.
	GetRecord(databaseName, recordID string) ([]DataStream, error)

	// Save replication state to storage.
	SaveState(state []byte) error
}

const MaxUpdateResults = 100
