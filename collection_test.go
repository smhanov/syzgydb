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

func TestRemoveDocumentRealWorld(t *testing.T) {
	// Create a collection with some documents
	options := CollectionOptions{
		Name:           "test_collection",
		DistanceMethod: Euclidean,
		DimensionCount: 3,
	}
	collection := NewCollection(options)

	// Add 1000 documents to the collection
	for i := 0; i < 1000; i++ {
		vector := []float64{float64(i), float64(i + 1), float64(i + 2)}
		metadata := []byte("metadata")
		collection.addDocument(uint64(i), vector, metadata)
	}

	// Remove every 10th document
	for i := 0; i < 1000; i += 10 {
		err := collection.removeDocument(uint64(i))
		if err != nil {
			t.Errorf("Failed to remove document with ID %d: %v", i, err)
		}
	}

	// Verify that removed documents are not accessible
	for i := 0; i < 1000; i++ {
		_, err := collection.memfile.readRecord(uint64(i))
		if i%10 == 0 {
			// Expect an error for removed documents
			if err == nil {
				t.Errorf("Expected error when reading removed document with ID %d, but got none", i)
			}
		} else {
			// Expect no error for existing documents
			if err != nil {
				t.Errorf("Unexpected error when reading document with ID %d: %v", i, err)
			}
		}
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
		Vector:   []float64{1.0, 2.0, 3.0},
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
