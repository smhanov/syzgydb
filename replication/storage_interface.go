package replication

type StorageInterface interface {
    CommitUpdates(updates []Update) error
    GetUpdatesSince(timestamp Timestamp) (map[string][]Update, error)
    ResolveConflict(update1, update2 Update) (Update, error)
    SubscribeUpdates() (<-chan Update, error)
    Exists(dependency string) bool
}
