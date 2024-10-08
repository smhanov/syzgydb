package syzgydb

import (
	"os"
)

func ensureTestdataDir() {
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		os.Mkdir("testdata", os.ModePerm)
	}
}

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func setupTestServer() *Server {
	if GlobalConfig == nil {
		GlobalConfig = &Config{}
	}
	ensureTestdataDir()
	GlobalConfig.DataFolder = "testdata" // Set the data folder to the testdata directory
	server := &Server{
		collections: make(map[string]*Collection),
	}

	os.Remove("test_collection.dat")

	return server
}

func TestGetCollectionIDs(t *testing.T) {
	server := setupTestServer()

	// Create the collection explicitly for this test
	server.collections["test_collection"] = NewCollection(CollectionOptions{
		Name:           "testdata/test_collection.dat",
		DistanceMethod: Cosine,
		DimensionCount: 5,
		Quantization:   64,
	})
	server.collections["test_collection"].AddDocument(1234567890, []float64{0.1, 0.2, 0.3, 0.4, 0.5}, []byte(`{"key1":"value1"}`))
	server.collections["test_collection"].AddDocument(1234567891, []float64{0.5, 0.4, 0.3, 0.2, 0.1}, []byte(`{"key2":"value2"}`))

	req, err := http.NewRequest(http.MethodGet, "/api/v1/collections/test_collection/ids", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleGetCollectionIDs)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var ids []uint64
	if err := json.NewDecoder(rr.Body).Decode(&ids); err != nil {
		t.Fatal(err)
	}

	expectedIDs := []uint64{1234567890, 1234567891}
	if !equalUint64Slices(ids, expectedIDs) {
		t.Errorf("handler returned unexpected IDs: got %v want %v", ids, expectedIDs)
	}
}

func equalUint64Slices(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDeleteCollection(t *testing.T) {
	server := setupTestServer()

	// Create the collection explicitly for this test
	collectionName := "test_collection"
	fileName := server.collectionNameToFileName(collectionName)
	server.collections[collectionName] = NewCollection(CollectionOptions{
		Name:           fileName,
		DistanceMethod: Cosine,
		DimensionCount: 128,
		Quantization:   64,
	})

	req, err := http.NewRequest(http.MethodDelete, "/api/v1/collections/test_collection", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleCollection)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := `{"message":"Collection deleted successfully."}`
	actual := strings.TrimSpace(rr.Body.String())
	if actual != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", actual, expected)
	}
}

func TestSearchRecords(t *testing.T) {
	server := setupTestServer()

	// Create the collection explicitly for this test
	server.collections["test_collection"] = NewCollection(CollectionOptions{
		Name:           "test_collection.dat",
		DistanceMethod: Cosine,
		DimensionCount: 5,
		Quantization:   64,
	})
	server.collections["test_collection"].AddDocument(1234567890, []float64{0.1, 0.2, 0.3, 0.4, 0.5}, []byte(`{"key1":"value1"}`))

	reqBody := `{"vector": [0.1, 0.2, 0.3, 0.4, 0.5], "k": 1}`
	req, err := http.NewRequest(http.MethodPost, "/api/v1/collections/test_collection/search", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleSearchRecords)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	type JSSearchResult struct {
		SearchResult
		Metadata interface{} `json:"metadata"`
	}

	type JSSearchResults struct {
		Results []JSSearchResult `json:"results"`
	}

	var response JSSearchResults
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if len(response.Results) == 0 {
		t.Errorf("expected at least one search result, got %v", len(response.Results))
	}
}

func TestCreateCollection(t *testing.T) {
	server := setupTestServer()

	// Define the request body for creating a collection
	reqBody := `{
		"name": "test_collection",
		"vector_size": 128,
		"quantization": 64,
		"distance_function": "cosine"
	}`
	req, err := http.NewRequest(http.MethodPost, "/api/v1/collections", bytes.NewBufferString(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleCollections)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Decode the actual response
	var actualResponse map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&actualResponse); err != nil {
		t.Fatal(err)
	}

	// Define the expected response
	expectedResponse := map[string]string{
		"message":         "Collection created successfully.",
		"collection_name": "test_collection",
	}

	// Compare the actual and expected responses
	if actualResponse["message"] != expectedResponse["message"] || actualResponse["collection_name"] != expectedResponse["collection_name"] {
		t.Errorf("handler returned unexpected body: got %v want %v", actualResponse, expectedResponse)
	}
}

func TestGetCollectionInfo(t *testing.T) {
	server := setupTestServer()

	// Create the collection explicitly for this test
	server.collections["test_collection"] = NewCollection(CollectionOptions{
		Name:           "test_collection.dat",
		DistanceMethod: Cosine,
		DimensionCount: 128,
		Quantization:   64,
	})

	req, err := http.NewRequest(http.MethodGet, "/api/v1/collections/test_collection", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleCollection)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if response["name"] != "test_collection" {
		t.Errorf("handler returned unexpected name: got %v want %v", response["name"], "test_collection")
	}
}

func mockEmbedText(texts []string) ([][]float64, error) {
	// Return a fixed vector for each input text
	mockVector := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	vectors := make([][]float64, len(texts))
	for i := range texts {
		vectors[i] = mockVector
	}
	return vectors, nil
}

func TestInsertRecords(t *testing.T) {
	// Set up the mock embedding function
	embedText = mockEmbedText
	server := setupTestServer()

	// Create the collection explicitly for this test
	server.collections["test_collection"] = NewCollection(CollectionOptions{
		Name:           "test_collection.dat",
		DistanceMethod: Cosine,
		DimensionCount: 5,
		Quantization:   64,
	})

	reqBody := `[
		{
			"id": 1234567890,
			"vector": [0.1, 0.2, 0.3, 0.4, 0.5],
			"metadata": {"key1": "value1"}
		},
		{
			"id": 1234567891,
			"text": "example text",
			"metadata": {"key2": "value2"}
		}
	]`
	req, err := http.NewRequest(http.MethodPost, "/api/v1/collections/test_collection/records", bytes.NewBufferString(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleInsertRecord)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		// print out the body of the response.
		t.Logf("Response body: %v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Decode the actual response
	var actualResponse map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&actualResponse); err != nil {
		t.Fatal(err)
	}

	// Define the expected response
	expectedResponse := map[string]interface{}{
		"message": "Records inserted successfully.",
	}

	// Compare the actual and expected responses
	if actualResponse["message"] != expectedResponse["message"] {
		t.Errorf("handler returned unexpected body: got %v want %v", actualResponse, expectedResponse)
	}
}

func TestUpdateRecordMetadata(t *testing.T) {
	server := setupTestServer()
	server.collections["test_collection"] = NewCollection(CollectionOptions{
		Name:           "test_collection.dat",
		DistanceMethod: Cosine,
		DimensionCount: 5,
		Quantization:   64,
	})
	server.collections["test_collection"].AddDocument(1234567890, []float64{0.1, 0.2, 0.3, 0.4, 0.5}, []byte(`{"key1":"value1"}`))

	reqBody := `{
		"metadata": {"key1": "new_value1"}
	}`
	req, err := http.NewRequest(http.MethodPut, "/api/v1/collections/test_collection/records/1234567890/metadata", bytes.NewBufferString(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleUpdateMetadata)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Decode the actual response
	var actualResponse map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&actualResponse); err != nil {
		t.Fatal(err)
	}

	// Define the expected response
	expectedResponse := map[string]interface{}{
		"message": "Metadata updated successfully.",
		"id":      float64(1234567890), // JSON numbers are decoded as float64
	}

	// Compare the actual and expected responses
	if actualResponse["message"] != expectedResponse["message"] || actualResponse["id"] != expectedResponse["id"] {
		t.Errorf("handler returned unexpected body: got %v want %v", actualResponse, expectedResponse)
	}
}

func TestDeleteRecord(t *testing.T) {
	server := setupTestServer()

	server.collections["test_collection"] = NewCollection(CollectionOptions{
		Name:           "test_collection.dat",
		DistanceMethod: Cosine,
		DimensionCount: 5,
		Quantization:   64,
	})
	metadata, err := json.Marshal(map[string]string{"key1": "value1"})
	if err != nil {
		t.Fatal(err)
	}
	server.collections["test_collection"].AddDocument(1234567890, []float64{0.1, 0.2, 0.3, 0.4, 0.5}, []byte(metadata))

	req, err := http.NewRequest(http.MethodDelete, "/api/v1/collections/test_collection/records/1234567890", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleDeleteRecord)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Decode the actual response
	var actualResponse map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&actualResponse); err != nil {
		t.Fatal(err)
	}

	// Define the expected response
	expectedResponse := map[string]interface{}{
		"message": "Record deleted successfully.",
		"id":      float64(1234567890), // JSON numbers are decoded as float64
	}

	// Compare the actual and expected responses
	if actualResponse["message"] != expectedResponse["message"] || actualResponse["id"] != expectedResponse["id"] {
		t.Errorf("handler returned unexpected body: got %v want %v", actualResponse, expectedResponse)
	}
}
func TestGetAllCollections(t *testing.T) {
	server := setupTestServer()

	// Create some collections for testing
	server.collections["collection1"] = NewCollection(CollectionOptions{
		Name:           "collection1.dat",
		DistanceMethod: Cosine,
		DimensionCount: 128,
		Quantization:   64,
		FileMode:       CreateAndOverwrite,
	})
	server.collections["collection2"] = NewCollection(CollectionOptions{
		Name:           "collection2.dat",
		DistanceMethod: Euclidean,
		DimensionCount: 64,
		Quantization:   32,
		FileMode:       CreateAndOverwrite,
	})

	req, err := http.NewRequest(http.MethodGet, "/api/v1/collections", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleCollections)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if len(response) != 2 {
		t.Errorf("expected 2 collections, got %v", len(response))
	}

	expectedNames := map[string]bool{"collection1": true, "collection2": true}
	for _, collectionInfo := range response {
		name := collectionInfo["name"].(string)
		if !expectedNames[name] {
			t.Errorf("unexpected collection name: %v", name)
		}
	}
}
