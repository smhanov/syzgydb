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
