package syzgydb

import (
	"net/http"
	"strings"
)

func RunServer() {
	server := &Server{
		collections: make(map[string]*Collection),
	}

	http.HandleFunc("/api/v1/collections", server.handleCollections)
	http.HandleFunc("/api/v1/collections/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/records") && r.Method == http.MethodPost {
			server.handleInsertRecord(w, r)
		} else if strings.Contains(r.URL.Path, "/records/") && r.Method == http.MethodPut {
			server.handleUpdateMetadata(w, r)
		} else if strings.Contains(r.URL.Path, "/records/") && r.Method == http.MethodDelete {
			server.handleDeleteRecord(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/search") && r.Method == http.MethodGet {
			server.handleSearchRecords(w, r)
		} else {
			server.handleCollection(w, r)
		}
	})

	http.ListenAndServe(":8080", nil)
}
