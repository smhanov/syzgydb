package syzgydb

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
	server := &Server{
		collections: make(map[string]*Collection),
	}

	os.Remove("test_collection.dat")

	return server
}

func TestDeleteCollection(t *testing.T) {
	server := setupTestServer()

	// Create the collection explicitly for this test
	server.collections["test_collection"] = NewCollection(CollectionOptions{
		Name:           "test_collection",
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
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestSearchRecords(t *testing.T) {
	server := setupTestServer()

	// Create the collection explicitly for this test
	server.collections["test_collection"] = NewCollection(CollectionOptions{
		Name:           "test_collection",
		DistanceMethod: Cosine,
		DimensionCount: 128,
		Quantization:   64,
	})
	server.collections["test_collection"].AddDocument(1234567890, []float64{0.1, 0.2, 0.3, 0.4, 0.5}, []byte(`{"key1":"value1"}`))

	reqBody := `{"vector": [0.1, 0.2, 0.3, 0.4, 0.5]}`
	req, err := http.NewRequest(http.MethodGet, "/api/v1/collections/test_collection/search", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleSearchRecords)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response SearchResults
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

	expected := `{"collection_name":"test_collection","message":"Collection created successfully."}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestGetCollectionInfo(t *testing.T) {
	server := setupTestServer()

	// Create the collection explicitly for this test
	server.collections["test_collection"] = NewCollection(CollectionOptions{
		Name:           "test_collection",
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

func TestInsertRecord(t *testing.T) {
	server := setupTestServer()

	// Create the collection explicitly for this test
	server.collections["test_collection"] = NewCollection(CollectionOptions{
		Name:           "test_collection",
		DistanceMethod: Cosine,
		DimensionCount: 128,
		Quantization:   64,
	})

	reqBody := `{
		"id": 1234567890,
		"vector": [0.1, 0.2, 0.3, 0.4, 0.5],
		"metadata": {"key1": "value1"}
	}`
	req, err := http.NewRequest(http.MethodPost, "/api/v1/collections/test_collection/records", bytes.NewBufferString(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleInsertRecord)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	expected := `{"message":"Record inserted successfully.","id":1234567890}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestUpdateRecordMetadata(t *testing.T) {
	server := setupTestServer()
	server.collections["test_collection"] = NewCollection(CollectionOptions{
		Name:           "test_collection",
		DistanceMethod: Cosine,
		DimensionCount: 128,
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

	expected := `{"message":"Metadata updated successfully.","id":1234567890}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestDeleteRecord(t *testing.T) {
	server := setupTestServer()
	server.collections["test_collection"] = NewCollection(CollectionOptions{
		Name:           "test_collection",
		DistanceMethod: Cosine,
		DimensionCount: 128,
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

	expected := `{"message":"Record deleted successfully.","id":1234567890}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}
