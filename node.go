package syzgydb

import (
    "fmt"
    "log"
    "path/filepath"
    "sync"
    "strings"
)

type Node struct {
    collections map[string]*Collection
    mutex       sync.RWMutex
    dataFolder  string
}

func NewNode(dataFolder string) *Node {
    return &Node{
        collections: make(map[string]*Collection),
        dataFolder:  dataFolder,
    }
}

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

    return nil
}

func (n *Node) GetCollection(name string) (*Collection, bool) {
    n.mutex.RLock()
    defer n.mutex.RUnlock()
    collection, exists := n.collections[name]
    return collection, exists
}

func (n *Node) CreateCollection(opts CollectionOptions) error {
    n.mutex.Lock()
    defer n.mutex.Unlock()

    if _, exists := n.collections[opts.Name]; exists {
        return fmt.Errorf("collection %s already exists", opts.Name)
    }

    opts.Name = n.collectionNameToFileName(opts.Name)
    collection, err := NewCollection(opts)
    if err != nil {
        return fmt.Errorf("failed to create collection: %v", err)
    }

    n.collections[opts.Name] = collection
    return nil
}

func (n *Node) DropCollection(name string) error {
    n.mutex.Lock()
    defer n.mutex.Unlock()

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
