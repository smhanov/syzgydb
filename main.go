package syzgydb

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

func RunServer() {
	server := &Server{
		collections: make(map[string]*Collection),
	}

	// Scan for existing .dat files and create collections
	files, err := filepath.Glob("*.dat")
	if err != nil {
		log.Fatalf("Failed to list .dat files: %v", err)
	}

	for _, file := range files {
		collectionName := strings.TrimSuffix(file, ".dat")
		log.Printf("Loading collection from file: %s", file)

		// Create a collection with empty CollectionOptions
		opts := CollectionOptions{Name: file}
		server.collections[collectionName] = NewCollection(opts)
		log.Printf("Collection %s loaded successfully", collectionName)
	}

	http.HandleFunc("/api/v1/collections", server.handleCollections)
	http.HandleFunc("/api/v1/collections/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/records") && r.Method == http.MethodPost {
			server.handleInsertRecord(w, r)
		} else if strings.Contains(r.URL.Path, "/records/") && r.Method == http.MethodPut {
			server.handleUpdateMetadata(w, r)
		} else if strings.Contains(r.URL.Path, "/records/") && r.Method == http.MethodDelete {
			server.handleDeleteRecord(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/search") && (r.Method == http.MethodGet || r.Method == http.MethodPost) {
			server.handleSearchRecords(w, r)
		} else {
			server.handleCollection(w, r)
		}
	})

	http.ListenAndServe(":8080", nil)
}
