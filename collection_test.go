package main

import (
	"testing"
)

func TestEuclideanDistance(t *testing.T) {
	vec1 := []float64{1.0, 2.0, 3.0}
	vec2 := []float64{4.0, 5.0, 6.0}
	expected := 5.196152422706632 // Pre-calculated Euclidean distance

	result := euclideanDistance(vec1, vec2)
	if result != expected {
		t.Errorf("Expected %f, got %f", expected, result)
	}
}

func TestCosineDistance(t *testing.T) {
	vec1 := []float64{1.0, 0.0, 0.0}
	vec2 := []float64{0.0, 1.0, 0.0}
	expected := 1.0 // Orthogonal vectors have cosine distance of 1

	result := cosineDistance(vec1, vec2)
	if result != expected {
		t.Errorf("Expected %f, got %f", expected, result)
	}
}

func TestUpdateDocument(t *testing.T) {
	// Create a collection with some documents
	options := CollectionOptions{
		Name:           "test_collection",
		DistanceMethod: Euclidean,
		DimensionCount: 3,
	}
	collection := NewCollection(options)

	// Add a document to the collection
	collection.addDocument(1, []float64{1.0, 2.0, 3.0}, []byte("original"))

	// Update the document's metadata
	err := collection.UpdateDocument(1, []byte("updated"))
	if err != nil {
		t.Errorf("Failed to update document: %v", err)
	}

	// Read the updated document
	data, err := collection.memfile.readRecord(1)
	if err != nil {
		t.Errorf("Failed to read updated document: %v", err)
	}

	// Decode the document
	doc := decodeDocument(data)

	// Check if the metadata was updated
	if string(doc.Metadata) != "updated" {
		t.Errorf("Expected metadata 'updated', got '%s'", doc.Metadata)
	}
}

func TestRemoveDocument(t *testing.T) {
	// Create a collection with some documents
	options := CollectionOptions{
		Name:           "test_collection",
		DistanceMethod: Euclidean,
		DimensionCount: 3,
	}
	collection := NewCollection(options)

	// Add a document to the collection
	collection.addDocument(1, []float64{1.0, 2.0, 3.0}, []byte("to be removed"))

	// Remove the document
	err := collection.removeDocument(1)
	if err != nil {
		t.Errorf("Failed to remove document: %v", err)
	}

	// Attempt to read the removed document
	_, err = collection.memfile.readRecord(1)
	if err == nil {
		t.Errorf("Expected error when reading removed document, but got none")
	}
}
	vec1 := []float64{1.0, 0.0, 0.0}
	vec2 := []float64{0.0, 1.0, 0.0}
	expected := 1.0 // Orthogonal vectors have cosine distance of 1

	result := cosineDistance(vec1, vec2)
	if result != expected {
		t.Errorf("Expected %f, got %f", expected, result)
	}
}

func TestSearch(t *testing.T) {
	// Create a collection with some documents
	options := CollectionOptions{
		Name:           "test_collection",
		DistanceMethod: Euclidean,
		DimensionCount: 3,
	}
	collection := NewCollection(options)

	// Add documents to the collection
	collection.addDocument(1, []float64{1.0, 2.0, 3.0}, []byte("doc1"))
	collection.addDocument(2, []float64{4.0, 5.0, 6.0}, []byte("doc2"))

	// Define search arguments
	args := SearchArgs{
		Vector:  []float64{1.0, 2.0, 3.0},
		MaxCount: 1,
	}

	// Perform search
	results := collection.Search(args)

	// Check the results
	if len(results.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results.Results))
	}

	if results.Results[0].ID != 1 {
		t.Errorf("Expected document ID 1, got %d", results.Results[0].ID)
	}
}
