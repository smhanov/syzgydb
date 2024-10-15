package syzgydb

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/smhanov/syzgydb/replication"
)

type Node struct {
	collections map[string]*Collection
	mutex       sync.RWMutex
	dataFolder  string
	nodeID      uint64
	initialized bool
}

func NewNode(dataFolder string, nodeID uint64) *Node {
	return &Node{
		collections: make(map[string]*Collection),
		dataFolder:  dataFolder,
		nodeID:      nodeID,
	}
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
func (n *Node) Initialize() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

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

	n.initialized = true

	return nil
}

func (n *Node) isInitialized() bool {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.initialized
}

// Returns the vector clock from the disk files for the node.
func (n *Node) getStoredVectorClock() *replication.VectorClock {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	clock := replication.NewVectorClock()
	for _, collection := range n.collections {
		clock.Merge(collection.getLatestVectorClock())
	}
	return clock
}

func (n *Node) GetCollection(name string) (*Collection, bool) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	collection, exists := n.collections[name]
	return collection, exists
}

func (n *Node) CreateCollection(opts CollectionOptions) (*Collection, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if current, exists := n.collections[opts.Name]; exists {
		if opts.FileMode == CreateAndOverwrite {
			err := current.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to existing collection: %v", err)
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
	return collection, nil
}

func (n *Node) DropCollection(name string) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	return n.dropCollection(name)
}

func (n *Node) dropCollection(name string) error {
	collection, exists := n.collections[name]
	if !exists {
		return fmt.Errorf("collection %s does not exist", name)
	}

	err := collection.Close()
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
