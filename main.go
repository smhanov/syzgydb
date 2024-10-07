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
	dataFolder := GlobalConfig.DataFolder
	files, err := filepath.Glob(filepath.Join(dataFolder, "*.dat"))
	if err != nil {
		log.Fatalf("Failed to list .dat files: %v", err)
	}

	for _, file := range files {
		collectionName := server.fileNameToCollectionName(file)
		log.Printf("Loading collection from file: %s", file)

		// Create a collection with empty CollectionOptions
		opts := CollectionOptions{Name: file}
		server.collections[collectionName] = NewCollection(opts)
		log.Printf("Collection %s loaded successfully", collectionName)
	}

	http.Handle("/api/v1/collections", gzipMiddleware(http.HandlerFunc(server.handleCollections)))
	http.Handle("/api/v1/collections/", gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
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
	})))

	host := GlobalConfig.SygyHost
	log.Printf("Starting server on %s", host)
	http.ListenAndServe(host, nil)
}
