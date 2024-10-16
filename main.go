package syzgydb

import (
	"log"
	"net/http"
	"strings"
)

func RunServer() {
	if globalConfig.NodeID == 0 {
		var err error
		globalConfig.NodeID, err = GetServerHash()
		if err != nil {
			log.Printf("Warning: Can't get server hash: %v", err)
			globalConfig.NodeID = uint64(myRandom.Intn(100) + 1)
		}
	}
	node := NewNode(globalConfig)
	err := node.Initialize(true)
	if err != nil {
		log.Fatalf("Failed to initialize node: %v", err)
	}

	server := &Server{node: node}

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

	host := globalConfig.SyzgyHost
	log.Printf("Starting server on %s", host)
	http.ListenAndServe(host, nil)
}
