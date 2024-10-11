// Package replication defines interfaces and types for the SyzgyDB replication system.
package replication

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
}
