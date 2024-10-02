package syzgydb

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

func (s *Server) collectionNameToFileName(name string) string {
    return name + ".dat"
}

func (s *Server) fileNameToCollectionName(fileName string) string {
    return strings.TrimSuffix(fileName, ".dat")
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

		// Transform collection name to filename
		fileName := s.collectionNameToFileName(opts.Name)

		// Pass the transformed name to NewCollection
		opts.Name = fileName
		s.collections[opts.Name] = NewCollection(opts)
		w.WriteHeader(http.StatusCreated)
		collectionName := s.fileNameToCollectionName(opts.Name)
		json.NewEncoder(w).Encode(map[string]string{"message": "Collection created successfully.", "collection_name": collectionName})
	}
}

func (s *Server) handleCollection(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	collectionName := s.collectionNameToFileName(parts[3])

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
			"name":              s.fileNameToCollectionName(collection.Name),
			"vector_size":       collection.DimensionCount,
			"quantization":      collection.Quantization,
			"distance_function": collection.DistanceMethod,
			"storage_space":     0, // Placeholder
			"num_vectors":       len(collection.memfile.idOffsets),
			"average_distance":  0.0, // Placeholder
		}
		json.NewEncoder(w).Encode(info)

	case http.MethodDelete:
		delete(s.collections, collectionName)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Collection deleted successfully."})
	}
}
func (s *Server) handleInsertRecord(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	collectionName := parts[3]

	s.mutex.Lock()
	collection, exists := s.collections[collectionName]
	s.mutex.Unlock()

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	var record struct {
		ID       uint64            `json:"id"`
		Vector   []float64         `json:"vector"`
		Metadata map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Encode metadata to JSON
	metadataBytes, err := json.Marshal(record.Metadata)
	if err != nil {
		http.Error(w, "Failed to encode metadata", http.StatusInternalServerError)
		return
	}

	collection.AddDocument(record.ID, record.Vector, metadataBytes)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Record inserted successfully.", "id": record.ID})
}

func (s *Server) handleUpdateMetadata(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	collectionName := parts[3]
	id, err := strconv.ParseUint(parts[len(parts)-1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	s.mutex.Lock()
	collection, exists := s.collections[collectionName]
	s.mutex.Unlock()

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	var metadata struct {
		Metadata map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Encode new metadata to JSON
	metadataBytes, err := json.Marshal(metadata.Metadata)
	if err != nil {
		http.Error(w, "Failed to encode metadata", http.StatusInternalServerError)
		return
	}

	if err := collection.UpdateDocument(id, metadataBytes); err != nil {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Metadata updated successfully.", "id": id})
}

func (s *Server) handleDeleteRecord(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	collectionName := parts[3]
	id, err := strconv.ParseUint(parts[5], 10, 64)
	if err != nil {
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	s.mutex.Lock()
	collection, exists := s.collections[collectionName]
	s.mutex.Unlock()

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	if err := collection.removeDocument(id); err != nil {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Record deleted successfully.", "id": id})
}

func (s *Server) handleSearchRecords(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	collectionName := parts[3]

	s.mutex.Lock()
	collection, exists := s.collections[collectionName]
	s.mutex.Unlock()

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	var searchArgs SearchArgs
	if err := json.NewDecoder(r.Body).Decode(&searchArgs); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	results := collection.Search(searchArgs)
	json.NewEncoder(w).Encode(results)
}
