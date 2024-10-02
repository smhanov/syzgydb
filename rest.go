package syzgydb

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
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
	log.Printf("Received %s request for %s", r.Method, r.URL.Path)

	if r.Method == http.MethodPost {
		// Define a temporary struct to match the JSON structure
		var temp struct {
			Name           string `json:"name"`
			DistanceMethod string `json:"distance_function"`
			DimensionCount int    `json:"vector_size"`
			Quantization   int    `json:"quantization"`
		}

		// Decode the JSON request into the temporary struct
		if err := json.NewDecoder(r.Body).Decode(&temp); err != nil {
			log.Printf("Error decoding request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Printf("Creating collection with options: %+v", temp)

		s.mutex.Lock()
		defer s.mutex.Unlock()

		// Map the fields from the temporary struct to the CollectionOptions struct
		opts := CollectionOptions{
			Name:           temp.Name,
			DimensionCount: temp.DimensionCount,
			Quantization:   temp.Quantization,
		}

		// Convert DistanceMethod from string to int
		switch temp.DistanceMethod {
		case "euclidean":
			opts.DistanceMethod = Euclidean
		case "cosine":
			opts.DistanceMethod = Cosine
		default:
			log.Printf("Invalid distance method: %s", temp.DistanceMethod)
			http.Error(w, "Invalid distance method", http.StatusBadRequest)
			return
		}

		name := opts.Name
		opts.Name = s.collectionNameToFileName(name)

		if _, exists := s.collections[name]; exists {
			log.Printf("Collection %s already exists", name)
			http.Error(w, "Collection already exists", http.StatusBadRequest)
			return
		}

		s.collections[name] = NewCollection(opts)
		log.Printf("Collection %s created successfully", name)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Collection created successfully.", "collection_name": name})
	}
}

func (s *Server) handleCollection(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request for %s", r.Method, r.URL.Path)

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		log.Println("Invalid path")
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	collectionName := parts[4]

	s.mutex.Lock()
	defer s.mutex.Unlock()

	collection, exists := s.collections[collectionName]

	if !exists {
		log.Printf("Collection %s not found", collectionName)
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		log.Printf("Fetching info for collection %s", collectionName)
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
		log.Printf("Deleting collection %s", collectionName)
		delete(s.collections, collectionName)
		collection.Close()
		os.Remove(s.collectionNameToFileName(collectionName))
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
	collectionName := parts[4]

	s.mutex.Lock()
	collection, exists := s.collections[collectionName]
	s.mutex.Unlock()

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	var record struct {
		ID       uint64            `json:"id"`
		Vector   []float64         `json:"vector,omitempty"`
		Text     string            `json:"text,omitempty"`
		Metadata map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert text to vector if text is provided
	if record.Text != "" {
		vector, err := embedText(record.Text)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to convert text to vector: %v", err), http.StatusInternalServerError)
			return
		}
		record.Vector = vector
	}

	// Ensure a vector is present
	if record.Vector == nil {
		http.Error(w, "Either vector or text must be provided", http.StatusBadRequest)
		return
	}
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
	collectionName := parts[4]
	id, err := strconv.ParseUint(parts[len(parts)-2], 10, 64)
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
	if len(parts) < 7 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	collectionName := parts[4]
	id, err := strconv.ParseUint(parts[6], 10, 64)
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
	collectionName := parts[4]

	s.mutex.Lock()
	collection, exists := s.collections[collectionName]
	s.mutex.Unlock()

	if !exists {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	offsetStr := query.Get("offset")
	limitStr := query.Get("limit")
	includeVectorsStr := query.Get("include_vectors")
	radiusStr := query.Get("radius")
	kStr := query.Get("k")

	// Set defaults
	offset := 0
	limit := 10
	includeVectors := false

	// Convert query parameters to appropriate types
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if includeVectorsStr != "" {
		if iv, err := strconv.ParseBool(includeVectorsStr); err == nil {
			includeVectors = iv
		}
	}

	// Initialize SearchArgs
	searchArgs := SearchArgs{
		Offset: offset,
		Limit:  limit,
	}

	// Parse optional body for vector or text
	var searchRequest struct {
		Vector []float64 `json:"vector,omitempty"`
		Text   string    `json:"text,omitempty"`
	}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&searchRequest); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	}

	// Convert text to vector if text is provided
	if searchRequest.Text != "" {
		vector, err := ollama_embed_text(searchRequest.Text)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to convert text to vector: %v", err), http.StatusInternalServerError)
			return
		}
		searchRequest.Vector = vector
	}

	searchArgs.Vector = searchRequest.Vector
	if radiusStr != "" {
		if radius, err := strconv.ParseFloat(radiusStr, 64); err == nil {
			searchArgs.Radius = radius
		}
	}
	if kStr != "" {
		if k, err := strconv.Atoi(kStr); err == nil {
			searchArgs.MaxCount = k
		}
	}

	if includeVectors {
		// Collect all document IDs
		ids := make([]uint64, 0, len(collection.memfile.idOffsets))
		for id := range collection.memfile.idOffsets {
			ids = append(ids, id)
		}

		// Sort IDs
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

		// Apply offset and limit
		start := offset
		if start > len(ids) {
			start = len(ids)
		}
		end := start + limit
		if end > len(ids) {
			end = len(ids)
		}

		// Collect results
		results := make([]SearchResult, 0, end-start)
		for _, id := range ids[start:end] {
			doc, err := collection.GetDocument(id)
			if err != nil {
				continue
			}
			results = append(results, SearchResult{
				ID:       doc.ID,
				Metadata: doc.Metadata,
				Distance: 0, // Distance is not applicable here
			})
		}

		json.NewEncoder(w).Encode(SearchResults{
			Results:         results,
			PercentSearched: 100.0, // All records are considered
		})
		return
	}

	results := collection.Search(searchArgs)

	// Decode metadata and prepare results with lowercase JSON field names
	type jsonSearchResult struct {
		ID       uint64                 `json:"id"`
		Metadata map[string]interface{} `json:"metadata"`
		Distance float64                `json:"distance"`
	}

	jsonResults := make([]jsonSearchResult, len(results.Results))
	for i, result := range results.Results {
		var metadata map[string]interface{}
		if err := json.Unmarshal(result.Metadata, &metadata); err != nil {
			log.Printf("Error decoding metadata for ID %d: %v", result.ID, err)
			continue
		}
		jsonResults[i] = jsonSearchResult{
			ID:       result.ID,
			Metadata: metadata,
			Distance: result.Distance,
		}
	}

	// Encode the results with lowercase field names
	json.NewEncoder(w).Encode(struct {
		Results         []jsonSearchResult `json:"results"`
		PercentSearched float64            `json:"percent_searched"`
	}{
		Results:         jsonResults,
		PercentSearched: results.PercentSearched,
	})
}
