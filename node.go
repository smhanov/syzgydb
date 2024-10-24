package syzgydb

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/smhanov/syzgydb/replication"
)

/*
The node class is responsible for maintaining the list of local collections and
also the replication system. When a write is made to the collection, it is converted
into an update and forwarded to the replication system. The node class itself
then processes updates that come in from the replication system.
*/

type Node struct {
	collections map[string]*Collection
	mutex       sync.RWMutex
	dataFolder  string
	nodeID      uint64
	initialized bool
	re          *replication.ReplicationEngine
	config      Config
	spanfile    *SpanFile
}

func NewNode(config Config) *Node {
	node := &Node{
		collections: make(map[string]*Collection),
		dataFolder:  config.DataFolder,
		nodeID:      config.NodeID,
		config:      config,
	}

	return node
}

// GetCollectionNames returns a list of all collection names
func (n *Node) GetCollectionNames() []string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	names := make([]string, 0, len(n.collections))
	for name := range n.collections {
		names = append(names, name)
	}
	return names
}

// Initialize loads all collections from disk.
func (n *Node) Initialize(openStored bool) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	log.Printf("Initailizing")

	if openStored {
		log.Printf("Openstored")
		files, err := filepath.Glob(filepath.Join(n.dataFolder, "*.dat"))
		if err != nil {
			return fmt.Errorf("failed to list .dat files: %v", err)
		}

		for _, file := range files {
			collectionName := n.fileNameToCollectionName(file)
			log.Printf("Loading collection from file: %s", file)

			opts := CollectionOptions{Name: file}
			collection, err := NewCollection(opts)
			if err != nil {
				return fmt.Errorf("failed to create collection %s: %v", collectionName, err)
			}
			n.collections[collectionName] = collection
			log.Printf("Collection %s loaded successfully", collectionName)
		}
	}

	// create / open the clusterdata file
	var err error
	state, err := n.createClusterDataFile()

	if err != nil {
		return fmt.Errorf("failed to create clusterdata file: %v", err)
	}

	n.initialized = true

	reConfig := replication.ReplicationConfig{
		OwnURL:    n.config.ReplicationOwnURL,
		PeerURLs:  n.config.ReplicationPeerURLs,
		JWTSecret: []byte(n.config.ReplicationJWTKey),
		NodeID:    n.config.NodeID,
	}

	n.re, err = replication.Init(n, reConfig, state)
	if err != nil {
		return err
	}

	log.Printf("n.re is %p", n.re)

	return nil
}

func (n *Node) createClusterDataFile() ([]byte, error) {
	spanfile, err := OpenFile(filepath.Join(n.dataFolder, "clusterdata.span"), CreateIfNotExists)
	if err != nil {
		return nil, fmt.Errorf("failed to create span file: %v", err)
	}
	n.spanfile = spanfile
	span, err := spanfile.ReadRecord("replication_state")
	if err != nil {
		return nil, fmt.Errorf("failed to read span file: %v", err)
	}

	return span.DataStreams[0].Data, nil
}

func (n *Node) SaveState(state []byte) error {
	// save the replication state
	return n.spanfile.WriteRecord("replication_state", []DataStream{
		{StreamID: 0, Data: state},
	}, 0, 0, n.re.NextTimestamp())
}

func (n *Node) CreateCollection(opts CollectionOptions) (ICollection, error) {
	// Prepare an update and send it to the replication engine.
	// The data of the update should be the JSON encoding of the CollectionOptions.
	data, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection options: %v", err)
	}

	update := replication.Update{
		NodeID: n.nodeID,
		Type:   replication.CreateDatabase,
		DataStreams: []replication.DataStream{
			{StreamID: 0, Data: data},
		},
		DatabaseName: opts.Name,
	}

	err = n.re.SubmitUpdates([]replication.Update{update})
	if err != nil {
		return nil, err
	}

	n.mutex.RLock()
	defer n.mutex.RUnlock()
	coll, exists := n.collections[opts.Name]
	if !exists {
		return nil, fmt.Errorf("collection %s failed to create", opts.Name)
	}
	return newCollectionProxy(n, coll), nil
}

func (n *Node) GetCollection(name string) (ICollection, bool) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	collection, exists := n.collections[name]
	if !exists {
		return nil, false
	}
	return newCollectionProxy(n, collection), true
}

func (n *Node) createCollectionImpl(update replication.Update, opts CollectionOptions) (ICollection, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// record the created collection.
	err := n.spanfile.WriteRecord("create:"+opts.Name, nil, update.NodeID, update.SequenceNo, update.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to record created collection: %v", err)
	}

	if current, exists := n.collections[opts.Name]; exists {
		if opts.FileMode == CreateAndOverwrite {
			err := current.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to overwrite existing collection: %v", err)
			}
		} else if opts.FileMode == CreateIfNotExists {
			return current, nil
		} else {
			return nil, fmt.Errorf("collection %s already exists", opts.Name)
		}
	}

	key := opts.Name
	opts.Name = n.collectionNameToFileName(opts.Name)
	collection, err := NewCollection(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %v", err)
	}

	n.collections[key] = collection
	return newCollectionProxy(n, collection), nil
}

func (n *Node) DropCollection(name string) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Prepare an update and send it to the replication engine.
	update := replication.Update{
		NodeID:       n.nodeID,
		Type:         replication.DropDatabase,
		DatabaseName: name,
	}

	return n.re.SubmitUpdates([]replication.Update{update})
}

func (n *Node) dropCollectionImpl(update replication.Update) error {
	name := update.DatabaseName

	// record the deleted collection.
	err := n.spanfile.WriteRecord("delete:"+name, nil, update.NodeID, update.SequenceNo, update.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to record deleted collection: %v", err)
	}

	collection, exists := n.collections[name]
	if !exists {
		return fmt.Errorf("collection %s does not exist", name)
	}

	err = collection.Close()
	if err != nil {
		return fmt.Errorf("failed to close collection: %v", err)
	}

	delete(n.collections, name)

	return nil
}

func (n *Node) CollectionExists(name string) bool {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	_, exists := n.collections[name]
	return exists
}

func (n *Node) collectionNameToFileName(name string) string {
	return filepath.Join(n.dataFolder, name+".dat")
}

func (n *Node) fileNameToCollectionName(fileName string) string {
	return strings.TrimSuffix(filepath.Base(fileName), ".dat")
}

// CommitUpdates applies a list of updates to the storage.
func (n *Node) CommitUpdates(updates []replication.Update) error {
	// Go through the updates and call DropCollection, CreateCollection, UpdateDocument, RemoveDocument
	// as necessary of the underlying collections.
	for _, update := range updates {
		switch update.Type {
		case replication.CreateDatabase:
			//TODO: Check if the create is before the database was dropped and ignore. Need somewhere to store this
			var opts CollectionOptions
			err := json.Unmarshal(update.DataStreams[0].Data, &opts)
			if err != nil {
				return fmt.Errorf("failed to unmarshal collection options: %v", err)
			}
			opts.Name = update.DatabaseName
			opts.Timestamp = update.Timestamp
			opts.NodeID = update.NodeID
			opts.SequenceNumber = update.SequenceNo
			_, err = n.createCollectionImpl(update, opts)
			if err != nil {
				return err
			}

		case replication.DropDatabase:
			// TODO: Check if the drop is after the database was created.
			err := n.dropCollectionImpl(update)
			if err != nil {
				return err
			}

		case replication.UpsertRecord:
			collection, exists := n.collections[update.DatabaseName]
			if !exists {
				return fmt.Errorf("collection %s does not exist", update.DatabaseName)
			}
			id, err := strconv.ParseUint(update.RecordID, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid record ID: %v", err)
			}
			if len(update.DataStreams) == 2 {
				// This is an AddDocument operation
				dataStreams := make([]DataStream, len(update.DataStreams))
				for i, ds := range update.DataStreams {
					dataStreams[i] = DataStream{
						StreamID: uint8(ds.StreamID),
						Data:     ds.Data,
					}
				}
				err = collection.AddRecordDirect(id, dataStreams, update.NodeID, update.SequenceNo, update.Timestamp)
			} else if len(update.DataStreams) == 1 {
				// This is an UpdateDocument operation
				err = collection.UpdateDocumentDirect(id, update.DataStreams[0].Data, update.NodeID, update.SequenceNo, update.Timestamp)
			} else {
				return fmt.Errorf("invalid number of data streams for UpsertRecord")
			}
			if err != nil {
				return err
			}

		case replication.DeleteRecord:
			collection, exists := n.collections[update.DatabaseName]
			if !exists {
				return fmt.Errorf("collection %s does not exist", update.DatabaseName)
			}
			id, err := strconv.ParseUint(update.RecordID, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid record ID: %v", err)
			}
			err = collection.removeDocumentDirect(id, update.NodeID, update.Timestamp)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetUpdatesSince retrieves updates that occurred after the given vector clock, up to maxResults.
func (n *Node) GetUpdatesSince(vectorClock *replication.NodeSequences, maxResults int) ([]replication.Update, bool, error) {
    n.mutex.RLock()
    defer n.mutex.RUnlock()

    var wg sync.WaitGroup
    var mu sync.Mutex // For protecting the combined results
    var allUpdates []replication.Update
    
    // Channel to collect errors from goroutines
    errChan := make(chan error, len(n.collections))

    // Process each collection in parallel
    for name, coll := range n.collections {
        wg.Add(1)
        go func(dbName string, c *Collection) {
            defer wg.Done()
            
            updates, err := c.GetUpdatesSince(vectorClock, maxResults)
            if err != nil {
                errChan <- fmt.Errorf("error getting updates from collection %s: %v", dbName, err)
                return
            }

            // Set the database name for each update
            for i := range updates {
                updates[i].DatabaseName = dbName
            }

            mu.Lock()
            allUpdates = append(allUpdates, updates...)
            mu.Unlock()
        }(name, coll)
    }

    // Wait for all collection processing to complete
    wg.Wait()
    close(errChan)

    // Check for any errors
    for err := range errChan {
        if err != nil {
            return nil, false, err
        }
    }

    // Get updates from node.spanfile for collection creation/deletion
    spanUpdates, err := n.spanfile.GetUpdatesSince(vectorClock, maxResults)
    if err != nil {
        return nil, false, fmt.Errorf("error getting updates from spanfile: %v", err)
    }

    // Process spanfile updates to convert them to appropriate Update types
    for _, update := range spanUpdates {
        if strings.HasPrefix(update.RecordID, "create:") {
            dbName := strings.TrimPrefix(update.RecordID, "create:")
            createUpdate := replication.Update{
                NodeID:       update.NodeID,
                SequenceNo:   update.SequenceNo,
                Timestamp:    update.Timestamp,
                Type:        replication.CreateDatabase,
                DatabaseName: dbName,
            }
            allUpdates = append(allUpdates, createUpdate)
        } else if strings.HasPrefix(update.RecordID, "delete:") {
            dbName := strings.TrimPrefix(update.RecordID, "delete:")
            deleteUpdate := replication.Update{
                NodeID:       update.NodeID,
                SequenceNo:   update.SequenceNo,
                Timestamp:    update.Timestamp,
                Type:        replication.DropDatabase,
                DatabaseName: dbName,
            }
            allUpdates = append(allUpdates, deleteUpdate)
        }
    }

    // Sort updates by NodeID and SequenceNo
    sort.Slice(allUpdates, func(i, j int) bool {
        if allUpdates[i].NodeID != allUpdates[j].NodeID {
            return allUpdates[i].NodeID < allUpdates[j].NodeID
        }
        return allUpdates[i].SequenceNo < allUpdates[j].SequenceNo
    })

    // Truncate to maxResults if needed
    hasMore := false
    if len(allUpdates) > maxResults {
        hasMore = true
        allUpdates = allUpdates[:maxResults]
    }

    return allUpdates, hasMore, nil
}

// Exists checks if a given dependency (usually a database) exists in the storage.
func (n *Node) Exists(dependency string) bool {
	return n.CollectionExists(dependency)
}

// GetRecord retrieves a record by its ID and database name.
func (n *Node) GetRecord(databaseName, recordID string) ([]replication.DataStream, error) {
	return nil, fmt.Errorf("not implemented")
}

// This proxy allows local access to the collection but forwards writes to the replication engine.
type CollectionProxy struct {
	node       *Node
	collection *Collection
}

func newCollectionProxy(node *Node, collection *Collection) *CollectionProxy {
	return &CollectionProxy{
		node:       node,
		collection: collection,
	}
}

func (cf *CollectionProxy) AddDocument(id uint64, vector []float64, metadata []byte) error {
	log.Printf("AddDocument(%d) called", id)
	// Prepare an update and send it to the replication engine
	options := cf.collection.GetOptions()

	// Encode the vector using the collection's quantization
	encodedVector := EncodeVector(vector, options.Quantization)

	update := replication.Update{
		NodeID:       cf.node.nodeID,
		Type:         replication.UpsertRecord,
		RecordID:     fmt.Sprintf("%d", id),
		DatabaseName: cf.collection.GetName(),
		DataStreams: []replication.DataStream{
			{StreamID: 0, Data: metadata},
			{StreamID: 1, Data: encodedVector},
		},
	}

	return cf.node.re.SubmitUpdates([]replication.Update{update})
}

func (cf *CollectionProxy) Close() error {
	return cf.collection.Close()
}

func (cf *CollectionProxy) ComputeStats() CollectionStats {
	return cf.collection.ComputeStats()
}

func (cf *CollectionProxy) GetAllIDs() []uint64 {
	return cf.collection.GetAllIDs()
}

func (cf *CollectionProxy) GetDocument(id uint64) (*Document, error) {
	return cf.collection.GetDocument(id)
}

func (cf *CollectionProxy) GetDocumentCount() int {
	return cf.collection.GetDocumentCount()
}

func (cf *CollectionProxy) GetOptions() CollectionOptions {
	return cf.collection.GetOptions()
}

func (cf *CollectionProxy) RemoveDocument(id uint64) error {
	update := replication.Update{
		NodeID:       cf.node.nodeID,
		Type:         replication.DeleteRecord,
		RecordID:     fmt.Sprintf("%d", id),
		DatabaseName: cf.collection.GetName(),
	}

	return cf.node.re.SubmitUpdates([]replication.Update{update})
}

func (cf *CollectionProxy) Search(args SearchArgs) SearchResults {
	return cf.collection.Search(args)
}

func (cf *CollectionProxy) UpdateDocument(id uint64, newMetadata []byte) error {
	update := replication.Update{
		NodeID:   cf.node.nodeID,
		Type:     replication.UpsertRecord,
		RecordID: fmt.Sprintf("%d", id),
		DataStreams: []replication.DataStream{
			{StreamID: 0, Data: newMetadata},
		},
		DatabaseName: cf.collection.GetName(),
	}

	return cf.node.re.SubmitUpdates([]replication.Update{update})
}
