package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Server struct {
	collections map[string]*Collection
	mutex       sync.Mutex
}

func main() {
	server := &Server{
		collections: make(map[string]*Collection),
	}

	http.HandleFunc("/api/v1/collections", server.handleCollections)
	http.HandleFunc("/api/v1/collections/", server.handleCollection)

	http.ListenAndServe(":8080", nil)
}

func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var opts CollectionOptions
		if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.mutex.Lock()
		defer s.mutex.Unlock()

		if _, exists := s.collections[opts.Name]; exists {
			http.Error(w, "Collection already exists", http.StatusBadRequest)
			return
		}

		s.collections[opts.Name] = NewCollection(opts)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Collection created successfully.", "collection_name": opts.Name})
	}
}

func (s *Server) handleCollection(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	collectionName := parts[3]

	s.mutex.Lock()
	defer s.mutex.Unlock()

	collection, exists := s.collections[collectionName]
	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		info := map[string]interface{}{
			"name":             collection.Name,
			"vector_size":     collection.DimensionCount,
			"quantization":    collection.Quantization,
			"distance_function": collection.DistanceMethod,
			"storage_space":   0, // Placeholder
			"num_vectors":     len(collection.memfile.idOffsets),
			"average_distance": 0.0, // Placeholder
		}
		json.NewEncoder(w).Encode(info)

	case http.MethodDelete:
		delete(s.collections, collectionName)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Collection deleted successfully."})
	}
}
