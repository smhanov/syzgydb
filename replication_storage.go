package syzgydb

import (
    "fmt"
    "sync"
    "encoding/json"
    "strconv"
    "github.com/smhanov/syzgydb/replication"
)

type ReplicationStorage struct {
    node *Node
    mutex sync.RWMutex
    vectorClock *replication.VectorClock
}

func NewReplicationStorage(node *Node) *ReplicationStorage {
    return &ReplicationStorage{
        node: node,
        vectorClock: replication.NewVectorClock(),
    }
}

func (rs *ReplicationStorage) CommitUpdates(updates []replication.Update) error {
    rs.mutex.Lock()
    defer rs.mutex.Unlock()

    for _, update := range updates {
        err := rs.applyUpdate(update)
        if err != nil {
            return err
        }
        rs.vectorClock.Update(update.NodeID, update.Timestamp)
    }
    return nil
}

func (rs *ReplicationStorage) applyUpdate(update replication.Update) error {
    switch update.Type {
    case replication.CreateDatabase:
        return rs.createCollection(update)
    case replication.DropDatabase:
        return rs.dropCollection(update)
    case replication.UpsertRecord:
        return rs.upsertRecord(update)
    case replication.DeleteRecord:
        return rs.deleteRecord(update)
    default:
        return fmt.Errorf("unknown update type: %v", update.Type)
    }
}

func (rs *ReplicationStorage) createCollection(update replication.Update) error {
    var opts CollectionOptions
    err := json.Unmarshal(update.DataStreams[0].Data, &opts)
    if err != nil {
        return fmt.Errorf("failed to unmarshal collection options: %v", err)
    }
    opts.Name = update.DatabaseName
    _, err = rs.node.CreateCollection(opts)
    return err
}

func (rs *ReplicationStorage) dropCollection(update replication.Update) error {
    return rs.node.DropCollection(update.DatabaseName)
}

func (rs *ReplicationStorage) upsertRecord(update replication.Update) error {
    collection, exists := rs.node.GetCollection(update.DatabaseName)
    if !exists {
        return fmt.Errorf("collection %s does not exist", update.DatabaseName)
    }

    var metadata []byte
    var vector []float64

    for _, stream := range update.DataStreams {
        switch stream.StreamID {
        case 0:
            metadata = stream.Data
        case 1:
            vector = DecodeVector(stream.Data, collection.DimensionCount, collection.Quantization)
        }
    }

    id, err := strconv.ParseUint(update.RecordID, 10, 64)
    if err != nil {
        return fmt.Errorf("invalid record ID: %v", err)
    }

    return collection.AddDocument(id, vector, metadata)
}

func (rs *ReplicationStorage) deleteRecord(update replication.Update) error {
    collection, exists := rs.node.GetCollection(update.DatabaseName)
    if !exists {
        return fmt.Errorf("collection %s does not exist", update.DatabaseName)
    }

    id, err := strconv.ParseUint(update.RecordID, 10, 64)
    if err != nil {
        return fmt.Errorf("invalid record ID: %v", err)
    }

    return collection.removeDocument(id)
}

func (rs *ReplicationStorage) GetUpdatesSince(vectorClock *replication.VectorClock, maxResults int) (map[string][]replication.Update, bool, error) {
    // This method would require implementing a way to store and retrieve updates.
    // For now, we'll return an empty result.
    return make(map[string][]replication.Update), false, nil
}

func (rs *ReplicationStorage) ResolveConflict(update1, update2 replication.Update) (replication.Update, error) {
    // Implement conflict resolution logic here.
    // For now, we'll use a simple last-write-wins strategy based on the timestamp.
    if update1.Timestamp.After(update2.Timestamp) {
        return update1, nil
    }
    return update2, nil
}

func (rs *ReplicationStorage) Exists(dependency string) bool {
    return rs.node.CollectionExists(dependency)
}

func (rs *ReplicationStorage) GetRecord(databaseName, recordID string) ([]replication.DataStream, error) {
    collection, exists := rs.node.GetCollection(databaseName)
    if !exists {
        return nil, fmt.Errorf("collection %s does not exist", databaseName)
    }

    id, err := strconv.ParseUint(recordID, 10, 64)
    if err != nil {
        return nil, fmt.Errorf("invalid record ID: %v", err)
    }

    doc, err := collection.GetDocument(id)
    if err != nil {
        return nil, err
    }

    encodedVector := EncodeVector(doc.Vector, collection.Quantization)

    return []replication.DataStream{
        {StreamID: 0, Data: doc.Metadata},
        {StreamID: 1, Data: encodedVector},
    }, nil
}

// Helper function to create an Update from a local change
func (rs *ReplicationStorage) CreateUpdate(updateType replication.UpdateType, databaseName, recordID string, dataStreams []replication.DataStream) replication.Update {
    rs.mutex.Lock()
    defer rs.mutex.Unlock()

    timestamp := rs.node.spanfile.NextTimestamp()
    rs.vectorClock.Update(rs.node.nodeID, timestamp)

    return replication.Update{
        NodeID:       rs.node.nodeID,
        Timestamp:    timestamp,
        Type:         updateType,
        RecordID:     recordID,
        DataStreams:  dataStreams,
        DatabaseName: databaseName,
    }
}
