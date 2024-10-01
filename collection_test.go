package syzygy

import (
	"math/rand"
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
	t.Logf("Adding 1000 documents")
	// Add 1000 documents to the collection
	for i := 0; i < 1000; i++ {
		vector := []float64{float64(i), float64(i + 1), float64(i + 2)}
		metadata := []byte("metadata")
		collection.AddDocument(uint64(i), vector, metadata)
	}

	// Remove every 10th document
	t.Logf(("Removing every 10th document"))
	for i := 0; i < 1000; i += 10 {
		err := collection.removeDocument(uint64(i))
		if err != nil {
			t.Errorf("Failed to remove document with ID %d: %v", i, err)
		}
	}

	t.Logf("Verifying that removed documents are not accessible")
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
	collection.AddDocument(1, []float64{1.0, 2.0, 3.0}, []byte("original"))

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
	doc := collection.decodeDocument(data, 1)

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
	collection.AddDocument(1, []float64{1.0, 2.0, 3.0}, []byte("to be removed"))

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

func TestCollectionSearch(t *testing.T) {
	// Create a collection with some documents
	options := CollectionOptions{
		Name:           "test_collection",
		DistanceMethod: Euclidean,
		DimensionCount: 2,
	}
	collection := NewCollection(options)

	// Add documents to the collection
	for i := 0; i < 10; i++ {
		vector := []float64{rand.Float64() * 100, rand.Float64() * 100}
		collection.AddDocument(uint64(i), vector, []byte("metadata"))
	}

	// Basic Search Test
	t.Run("Basic Search", func(t *testing.T) {
		searchVector := []float64{50, 50}
		args := SearchArgs{
			Vector:   searchVector,
			MaxCount: 5,
		}
		results := collection.Search(args)
		if len(results.Results) == 0 {
			t.Errorf("Expected results, got none")
		}
	})

	// Search with Maximum Count
	t.Run("Max Count", func(t *testing.T) {
		searchVector := []float64{50, 50}
		args := SearchArgs{
			Vector:   searchVector,
			MaxCount: 3,
		}
		results := collection.Search(args)
		if len(results.Results) > 3 {
			t.Errorf("Expected at most 3 results, got %d", len(results.Results))
		}
	})

	// Search with Radius
	t.Run("Radius Search", func(t *testing.T) {
		searchVector := []float64{50, 50}
		args := SearchArgs{
			Vector: searchVector,
			Radius: 10,
		}
		results := collection.Search(args)
		for _, result := range results.Results {
			if result.Distance > 10 {
				t.Errorf("Expected distance <= 10, got %f", result.Distance)
			}
		}
	})

	// Search with Empty Collection
	t.Run("Empty Collection", func(t *testing.T) {
		emptyCollection := NewCollection(options)
		searchVector := []float64{50, 50}
		args := SearchArgs{
			Vector:   searchVector,
			MaxCount: 5,
		}
		results := emptyCollection.Search(args)
		if len(results.Results) != 0 {
			t.Errorf("Expected no results, got %d", len(results.Results))
		}
	})

	// Search with Filter Function
	t.Run("Filter Function", func(t *testing.T) {
		searchVector := []float64{50, 50}
		args := SearchArgs{
			Vector:   searchVector,
			MaxCount: 5,
			Filter: func(id uint64, metadata []byte) bool {
				return id%2 == 0 // Exclude odd IDs
			},
		}
		results := collection.Search(args)
		for _, result := range results.Results {
			if result.ID%2 != 0 {
				t.Errorf("Expected only even IDs, got %d", result.ID)
			}
		}
	})
}

func TestCollectionPersistence(t *testing.T) {
	// Define collection options
	options := CollectionOptions{
		Name:           "persistent_test_collection",
		DistanceMethod: Euclidean,
		DimensionCount: 3,
	}

	// Create a new collection
	collection := NewCollection(options)

	// Add some records to the collection
	numRecords := 100 // Ensure enough records to trigger pivot creation
	for i := 0; i < numRecords; i++ {
		vector := []float64{float64(i), float64(i + 1), float64(i + 2)}
		metadata := []byte("metadata")
		collection.AddDocument(uint64(i), vector, metadata)
	}

	// Close the collection (assuming there's a method to close it)
	collection.Close()

	// Reopen the collection (assuming there's a method to open it)
	collection = NewCollection(options)

	// Verify that the records are still available
	for i := 0; i < numRecords; i++ {
		doc, err := collection.GetDocument(uint64(i))
		if err != nil {
			t.Errorf("Failed to retrieve document with ID %d: %v", i, err)
		}
		expectedVector := []float64{float64(i), float64(i + 1), float64(i + 2)}
		if !equalVectors(doc.Vector, expectedVector) {
			t.Errorf("Expected vector %v, got %v", expectedVector, doc.Vector)
		}
		if string(doc.Metadata) != "metadata" {
			t.Errorf("Expected metadata 'metadata', got '%s'", doc.Metadata)
		}
	}

	// Perform a search to test pivot usage
	searchVector := []float64{50, 51, 52}
	args := SearchArgs{
		Vector:   searchVector,
		MaxCount: 5,
	}
	results := collection.Search(args)

	// Check that the search results are not empty
	if len(results.Results) == 0 {
		t.Errorf("Expected search results, but got none")
	}

	t.Logf("Percent searched: %v", results.PercentSearched)

	// Ensure that PercentSearched is less than 100
	if results.PercentSearched >= 100 {
		t.Errorf("Expected PercentSearched to be less than 100, got %f", results.PercentSearched)
	}
}

func TestVectorSearchWith4BitQuantization(t *testing.T) {
	// Define collection options with 4-bit quantization
	options := CollectionOptions{
		Name:           "test_collection_4bit",
		DistanceMethod: Euclidean,
		DimensionCount: 3, // Example dimension count
		Quantization:   4, // 4-bit quantization
	}

	collection := NewCollection(options)

	// Add documents to the collection
	numDocuments := 10
	for i := 0; i < numDocuments; i++ {
		vector := make([]float64, options.DimensionCount)
		for d := 0; d < options.DimensionCount; d++ {
			vector[d] = rand.Float64() // Random float values
		}
		collection.AddDocument(uint64(i), vector, []byte("metadata"))
	}

	// Define a search vector
	searchVector := make([]float64, options.DimensionCount)
	for d := 0; d < options.DimensionCount; d++ {
		searchVector[d] = rand.Float64()
	}

	// Define search arguments
	args := SearchArgs{
		Vector:   searchVector,
		MaxCount: 5, // Limit to top 5 results
	}

	// Perform the search
	results := collection.Search(args)

	// Check the results
	if len(results.Results) == 0 {
		t.Errorf("Expected search results, but got none")
	}

	// Optionally, print results for manual verification
	for _, result := range results.Results {
		t.Logf("ID: %d, Distance: %.4f, Metadata: %s", result.ID, result.Distance, string(result.Metadata))
	}

	//DumpIndex("test_collection_4bit")
}
