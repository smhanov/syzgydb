package syzgydb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	verbose := false
	for _, arg := range os.Args {
		if arg == "-test.v" {
			verbose = true
			break
		}
	}

	if !verbose {
		log.SetOutput(ioutil.Discard)
	}

	os.Exit(m.Run())
}

func TestGetCollectionIDs(t *testing.T) {
	ensureTestFolder(t)
	server := setupTestServer()

	// Create the collection explicitly for this test
	collection, err := NewCollection(CollectionOptions{
		Name:           testFilePath("test_collection.dat"),
		DistanceMethod: Cosine,
		DimensionCount: 5,
		Quantization:   64,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create test collection: %v", err))
	}
	server.node.collections["test_collection"] = collection
	server.node.collections["test_collection"].AddDocument(1234567890, []float64{0.1, 0.2, 0.3, 0.4, 0.5}, []byte(`{"key1":"value1"}`))
	server.node.collections["test_collection"].AddDocument(1234567891, []float64{0.5, 0.4, 0.3, 0.2, 0.1}, []byte(`{"key2":"value2"}`))

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
	ensureTestFolder(t)
	server := setupTestServer()

	// Create the collection explicitly for this test
	collectionName := "test_collection"
	fileName := server.node.collectionNameToFileName(collectionName)
	collection, err := NewCollection(CollectionOptions{
		Name:           fileName,
		DistanceMethod: Cosine,
		DimensionCount: 128,
		Quantization:   64,
	})
	if err != nil {
		t.Fatalf("Failed to create test collection: %v", err)
	}
	server.node.collections[collectionName] = collection

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
	ensureTestFolder(t)
	server := setupTestServer()

	// Create the collection explicitly for this test
	collection, err := NewCollection(CollectionOptions{
		Name:           testFilePath("test_collection.dat"),
		DistanceMethod: Cosine,
		DimensionCount: 5,
		Quantization:   64,
	})
	if err != nil {
		t.Fatalf("Failed to create test collection: %v", err)
	}
	server.node.collections["test_collection"] = collection
	server.node.collections["test_collection"].AddDocument(1234567890, []float64{0.1, 0.2, 0.3, 0.4, 0.5}, []byte(`{"key1":"value1"}`))

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
	ensureTestFolder(t)
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
	ensureTestFolder(t)
	server := setupTestServer()

	// Create the collection explicitly for this test
	collection, err := NewCollection(CollectionOptions{
		Name:           testFilePath("test_collection.dat"),
		DistanceMethod: Cosine,
		DimensionCount: 128,
		Quantization:   64,
	})
	if err != nil {
		t.Fatalf("Failed to create test collection: %v", err)
	}
	server.node.collections["test_collection"] = collection

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

func mockEmbedText(texts []string, useCache bool) ([][]float64, error) {
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
	ensureTestFolder(t)
	server := setupTestServer()

	// Create the collection explicitly for this test
	collection, err := NewCollection(CollectionOptions{
		Name:           testFilePath("test_collection.dat"),
		DistanceMethod: Cosine,
		DimensionCount: 5,
		Quantization:   64,
	})
	if err != nil {
		t.Fatalf("Failed to create test collection: %v", err)
	}
	server.node.collections["test_collection"] = collection

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
	ensureTestFolder(t)
	server := setupTestServer()
	collection, err := NewCollection(CollectionOptions{
		Name:           testFilePath("test_collection.dat"),
		DistanceMethod: Cosine,
		DimensionCount: 5,
		Quantization:   64,
	})
	if err != nil {
		t.Fatalf("Failed to create test collection: %v", err)
	}
	server.node.collections["test_collection"] = collection
	server.node.collections["test_collection"].AddDocument(1234567890, []float64{0.1, 0.2, 0.3, 0.4, 0.5}, []byte(`{"key1":"value1"}`))

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

func TestRestDeleteRecord(t *testing.T) {
	ensureTestFolder(t)
	server := setupTestServer()

	collection, err := NewCollection(CollectionOptions{
		Name:           testFilePath("test_collection.dat"),
		DistanceMethod: Cosine,
		DimensionCount: 5,
		Quantization:   64,
	})
	if err != nil {
		t.Fatalf("Failed to create test collection: %v", err)
	}
	server.node.collections["test_collection"] = collection
	metadata, err := json.Marshal(map[string]string{"key1": "value1"})
	if err != nil {
		t.Fatal(err)
	}
	server.node.collections["test_collection"].AddDocument(1234567890, []float64{0.1, 0.2, 0.3, 0.4, 0.5}, []byte(metadata))

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

func TestSearchRecordsWithFilter(t *testing.T) {
	ensureTestFolder(t)
	server := setupTestServer()

	// Create the collection explicitly for this test
	collection, err := NewCollection(CollectionOptions{
		Name:           testFilePath("test_collection.dat"),
		DistanceMethod: Cosine,
		DimensionCount: 5,
		Quantization:   64,
	})
	if err != nil {
		t.Fatalf("Failed to create test collection: %v", err)
	}
	server.node.collections["test_collection"] = collection

	// Add documents with different metadata
	server.node.collections["test_collection"].AddDocument(1, []float64{0.1, 0.2, 0.3, 0.4, 0.5}, []byte(`{"category":"A", "score":80}`))
	server.node.collections["test_collection"].AddDocument(2, []float64{0.2, 0.3, 0.4, 0.5, 0.6}, []byte(`{"category":"B", "score":90}`))
	server.node.collections["test_collection"].AddDocument(3, []float64{0.3, 0.4, 0.5, 0.6, 0.7}, []byte(`{"category":"A", "score":70}`))

	// Prepare the search request with a filter
	reqBody := `{
		"vector": [0.1, 0.2, 0.3, 0.4, 0.5],
		"k": 3,
		"filter": "category == \"A\" AND score > 75"
	}`
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

	var response struct {
		Results []struct {
			ID       uint64                 `json:"id"`
			Metadata map[string]interface{} `json:"metadata"`
			Distance float64                `json:"distance"`
		} `json:"results"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	// Check that we got only one result (ID 1) that matches the filter
	if len(response.Results) != 1 {
		t.Errorf("expected 1 search result, got %v", len(response.Results))
	}

	if len(response.Results) > 0 {
		if response.Results[0].ID != 1 {
			t.Errorf("expected result with ID 1, got %v", response.Results[0].ID)
		}
		if response.Results[0].Metadata["category"] != "A" {
			t.Errorf("expected result with category A, got %v", response.Results[0].Metadata["category"])
		}
		if score, ok := response.Results[0].Metadata["score"].(float64); !ok || score <= 75 {
			t.Errorf("expected result with score > 75, got %v", response.Results[0].Metadata["score"])
		}
	}
}
func TestGetAllCollections(t *testing.T) {
	ensureTestFolder(t)
	server := setupTestServer()

	// Create some collections for testing
	collection1, err := NewCollection(CollectionOptions{
		Name:           testFilePath("collection1.dat"),
		DistanceMethod: Cosine,
		DimensionCount: 128,
		Quantization:   64,
		FileMode:       CreateAndOverwrite,
	})
	if err != nil {
		t.Fatalf("Failed to create collection1: %v", err)
	}
	server.collections["collection1"] = collection1

	collection2, err := NewCollection(CollectionOptions{
		Name:           testFilePath("collection2.dat"),
		DistanceMethod: Euclidean,
		DimensionCount: 64,
		Quantization:   32,
		FileMode:       CreateAndOverwrite,
	})
	if err != nil {
		t.Fatalf("Failed to create collection2: %v", err)
	}
	server.collections["collection2"] = collection2

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
