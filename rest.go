package syzgydb

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type Server struct {
	collections map[string]*Collection
	mutex       sync.Mutex
}

func (s *Server) collectionNameToFileName(name string) string {
	return filepath.Join(GlobalConfig.DataFolder, name+".dat")
}

func (s *Server) fileNameToCollectionName(fileName string) string {
	return strings.TrimSuffix(fileName, ".dat")
}
func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request for %s", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodPost:
		// Existing code for creating a collection
		var temp struct {
			Name           string `json:"name"`
			DistanceMethod string `json:"distance_function"`
			DimensionCount int    `json:"vector_size"`
			Quantization   int    `json:"quantization"`
		}

		if err := json.NewDecoder(r.Body).Decode(&temp); err != nil {
			log.Printf("Error decoding request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Printf("Creating collection with options: %+v", temp)

		s.mutex.Lock()
		defer s.mutex.Unlock()

		opts := CollectionOptions{
			Name:           temp.Name,
			DimensionCount: temp.DimensionCount,
			Quantization:   temp.Quantization,
		}

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

	case http.MethodGet:
		// New code to handle GET request
		var collectionsInfo []map[string]interface{}

		s.mutex.Lock()
		defer s.mutex.Unlock()

		for _, collection := range s.collections {
			stats := collection.ComputeStats()
			info := map[string]interface{}{
				"name":              s.fileNameToCollectionName(collection.Name),
				"vector_size":       stats.DimensionCount,
				"quantization":      stats.Quantization,
				"distance_function": stats.DistanceMethod,
				"storage_space":     stats.StorageSize,
				"num_vectors":       stats.DocumentCount,
				"average_distance":  stats.AverageDistance,
			}
			collectionsInfo = append(collectionsInfo, info)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(collectionsInfo)
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
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "Collection did not exist."})
			return
		}
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		log.Printf("Fetching info for collection %s", collectionName)

		// Use ComputeStats to get the collection statistics
		stats := collection.ComputeStats()

		// Populate the response with the statistics
		info := map[string]interface{}{
			"name":              s.fileNameToCollectionName(collection.Name),
			"vector_size":       stats.DimensionCount,
			"quantization":      stats.Quantization,
			"distance_function": stats.DistanceMethod,
			"storage_space":     stats.StorageSize,
			"num_vectors":       stats.DocumentCount,
			"average_distance":  stats.AverageDistance,
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

	var records []struct {
		ID       uint64            `json:"id"`
		Vector   []float64         `json:"vector,omitempty"`
		Text     string            `json:"text,omitempty"`
		Metadata map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&records); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Collect all texts that need to be embedded
	var textsToEmbed []string
	textIndices := make(map[int]int) // Map from text index to record index
	for i, record := range records {
		if record.Text != "" && record.Vector == nil {
			textIndices[len(textsToEmbed)] = i
			textsToEmbed = append(textsToEmbed, record.Text)
		}
	}

	// Call embedText once for all texts
	if len(textsToEmbed) > 0 {
		vectors, err := embedText(textsToEmbed)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to convert text to vector: %v", err), http.StatusInternalServerError)
			return
		}

		// Assign the resulting vectors back to the corresponding records
		for textIndex, recordIndex := range textIndices {
			records[recordIndex].Vector = vectors[textIndex]
		}
	}

	for _, record := range records {
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
	}
	log.Printf("Done")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Records inserted successfully."})
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

	var searchArgs SearchArgs

	var searchRequest struct {
		Vector []float64 `json:"vector,omitempty"`
		Text   string    `json:"text,omitempty"`
		Offset int       `json:"offset,omitempty"`
		Limit  int       `json:"limit,omitempty"`
		Radius float64   `json:"radius,omitempty"`
		K      int       `json:"k,omitempty"`
	}

	if r.Method == http.MethodGet {
		query := r.URL.Query()
		searchArgs.Offset, _ = strconv.Atoi(query.Get("offset"))
		searchArgs.Limit, _ = strconv.Atoi(query.Get("limit"))
		searchArgs.Radius, _ = strconv.ParseFloat(query.Get("radius"), 64)
		searchArgs.Radius, _ = strconv.ParseFloat(query.Get("radius"), 64)
		searchArgs.K, _ = strconv.Atoi(query.Get("k"))
		searchRequest.Text = query.Get("text")
	} else if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&searchRequest); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		searchArgs = SearchArgs{
			Vector: searchRequest.Vector,
			Offset: searchRequest.Offset,
			Limit:  searchRequest.Limit,
			Radius: searchRequest.Radius,
			K:      searchRequest.K,
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if searchRequest.Text != "" {
		vector, err := embedText([]string{searchRequest.Text})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to convert text to vector: %v", err), http.StatusInternalServerError)
			return
		}
		searchArgs.Vector = vector[0]
	}

	//log.Printf("Using searchArgs: %+v", searchArgs)
	results := collection.Search(searchArgs)

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

	json.NewEncoder(w).Encode(struct {
		Results         []jsonSearchResult `json:"results"`
		PercentSearched float64            `json:"percent_searched"`
	}{
		Results:         jsonResults,
		PercentSearched: results.PercentSearched,
	})
}
