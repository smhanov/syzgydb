package syzgydb

import (
	"fmt"
	"log"
	"math/rand"
	"os"
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

func TestCosineDistancePrecisionComparison(t *testing.T) {
	// Define collection options with Cosine distance
	options := CollectionOptions{
		Name:           "test_cosine_precision_comparison.dat",
		DistanceMethod: Cosine,
		DimensionCount: 3,
		Create:         true,
	}

	// Remove any existing file
	os.Remove(options.Name)

	// Create a new collection
	collection := NewCollection(options)
	defer collection.Close()

	// Add 200 random vectors to the collection
	numDocuments := 200
	vectors := make([][]float64, numDocuments)
	for i := 0; i < numDocuments; i++ {
		vector := make([]float64, options.DimensionCount)
		for d := 0; d < options.DimensionCount; d++ {
			vector[d] = rand.Float64()
		}
		vectors[i] = vector
		collection.AddDocument(uint64(i), vector, []byte(fmt.Sprintf("metadata_%d", i)))
	}

	// Retrieve the 10 closest points to the first vector with precision=exact
	searchArgsExact := SearchArgs{
		Vector:    vectors[0],
		K:         10,
		Precision: "exact",
	}
	resultsExact := collection.Search(searchArgsExact)

	// Retrieve the 10 closest points to the first vector with precision=medium
	searchArgsMedium := SearchArgs{
		Vector:    vectors[0],
		K:         10,
		Precision: "medium",
	}
	resultsMedium := collection.Search(searchArgsMedium)

	// Compare the results
	if len(resultsExact.Results) != len(resultsMedium.Results) {
		t.Fatalf("Expected the same number of results, got %d (exact) and %d (medium)", len(resultsExact.Results), len(resultsMedium.Results))
	}

	// Check if the IDs of the results are the same
	matched := true
	for i := range resultsExact.Results {
		log.Printf(" Exact: %v %v", resultsExact.Results[i].ID, resultsExact.Results[i].Distance)
		log.Printf("Medium: %v %v", resultsMedium.Results[i].ID, resultsMedium.Results[i].Distance)
		if resultsExact.Results[i].ID != resultsMedium.Results[i].ID {
			matched = false
		}
	}
	if !matched {
		t.Error("Results do not match")
	}
}

func TestComputeAverageDistance(t *testing.T) {
	// Define collection options
	options := CollectionOptions{
		Name:           "test_collection_avg_distance.dat",
		DistanceMethod: Euclidean,
		DimensionCount: 3,
		Create:         true,
	}

	// Remove any existing file
	os.Remove(options.Name)

	// Create a new collection
	collection := NewCollection(options)
	defer collection.Close()

	// Add documents to the collection
	numDocuments := 100
	for i := 0; i < numDocuments; i++ {
		vector := []float64{rand.Float64() * 100, rand.Float64() * 100, rand.Float64() * 100}
		collection.AddDocument(uint64(i), vector, []byte("metadata"))
	}

	// Compute the average distance
	samples := 50
	averageDistance := collection.computeAverageDistance(samples)

	// Check that the average distance is greater than zero
	if averageDistance <= 0 {
		t.Errorf("Expected average distance to be greater than zero, got %f", averageDistance)
	}

	// Optionally, log the average distance for manual verification
	t.Logf("Average distance: %f", averageDistance)
}

func TestRemoveDocumentRealWorld(t *testing.T) {
	// Create a collection with some documents
	collectionName := "test_collection.dat"
	options := CollectionOptions{
		Name:           collectionName,
		DistanceMethod: Euclidean,
		DimensionCount: 3,
	}
	// Remove any existing file
	os.Remove(collectionName)

	// Create a new collection
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

func TestUpdateDocument(t *testing.T) {
	// Create a collection with some documents
	options := CollectionOptions{
		Name:           "test_collection",
		DistanceMethod: Euclidean,
		DimensionCount: 3,
		Create:         true,
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
	// Create a new collection with appropriate options
	options := CollectionOptions{
		Name:           "test_collection",
		DistanceMethod: Euclidean,
		DimensionCount: 10,
		Quantization:   64,
		Create:         true,
	}
	collection := NewCollection(options)
	defer collection.Close()

	// Add 200 documents
	for i := 0; i < 200; i++ {
		vector := make([]float64, options.DimensionCount)
		for j := range vector {
			vector[j] = float64(i + j)
		}
		collection.AddDocument(uint64(i), vector, []byte(fmt.Sprintf("metadata_%d", i)))
	}

	// Choose a document to remove
	docToRemove := uint64(100)

	// Remove the chosen document
	err := collection.removeDocument(docToRemove)
	if err != nil {
		t.Fatalf("Failed to remove document: %v", err)
	}

	// Verify the document is removed
	_, err = collection.GetDocument(docToRemove)
	if err == nil {
		t.Errorf("Document %d was not removed", docToRemove)
	}

	// Verify other documents are still present
	for i := 0; i < 200; i++ {
		if uint64(i) == docToRemove {
			continue
		}
		_, err := collection.GetDocument(uint64(i))
		if err != nil {
			t.Errorf("Document %d is missing", i)
		}
	}
}

func TestCollectionSearch(t *testing.T) {
	// Create a collection with some documents
	options := CollectionOptions{
		Name:           "test_collection.dat",
		DistanceMethod: Euclidean,
		DimensionCount: 2,
	}
	os.Remove(options.Name)

	// Search with Empty Collection
	t.Run("Empty Collection", func(t *testing.T) {
		emptyCollection := NewCollection(options)
		defer emptyCollection.Close()
		searchVector := []float64{50, 50}
		args := SearchArgs{
			Vector: searchVector,
			K:      5,
		}
		results := emptyCollection.Search(args)
		if len(results.Results) != 0 {
			t.Errorf("Expected no results, got %d", len(results.Results))
		}

	})

	os.Remove(options.Name)

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
			Vector: searchVector,
			K:      5,
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
			Vector: searchVector,
			K:      3,
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

	// Search with Filter Function
	t.Run("Filter Function", func(t *testing.T) {
		searchVector := []float64{50, 50}
		args := SearchArgs{
			Vector: searchVector,
			K:      5,
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
	collectionName := "persistent_test_collection.dat"
	options := CollectionOptions{
		Name:           collectionName,
		DistanceMethod: Cosine,
		DimensionCount: 3,
	}

	os.Remove(collectionName)

	// Create a new collection
	collection := NewCollection(options)

	// Add some records to the collection
	numRecords := 1000 // Ensure enough records to trigger pivot creation
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
		Vector: searchVector,
		K:      5,
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

func TestCollectionAddDeleteAndRetrieve(t *testing.T) {
	// Define collection options
	collectionName := "test_collection_add_delete_retrieve.dat"
	options := CollectionOptions{
		Name:           collectionName,
		DistanceMethod: Euclidean,
		DimensionCount: 3,
	}

	os.Remove(collectionName)

	// Create a new collection
	collection := NewCollection(options)

	// Add some records to the collection
	numRecords := 10
	for i := 0; i < numRecords; i++ {
		vector := []float64{float64(i), float64(i + 1), float64(i + 2)}
		metadata := []byte("metadata")
		collection.AddDocument(uint64(i), vector, metadata)
	}

	// Delete all records
	for i := 0; i < numRecords; i++ {
		err := collection.removeDocument(uint64(i))
		if err != nil {
			t.Errorf("Failed to remove document with ID %d: %v", i, err)
		}
	}

	// Close the collection
	collection.Close()

	// Reopen the collection
	collection = NewCollection(options)

	// Add a single record with slightly larger metadata
	vector := []float64{1.0, 2.0, 3.0}
	metadata := []byte("larger metadata")
	collection.AddDocument(1, vector, metadata)

	// Close the collection
	collection.Close()

	// Reopen the collection
	collection = NewCollection(options)

	// Retrieve the record
	doc, err := collection.GetDocument(1)
	if err != nil {
		t.Errorf("Failed to retrieve document: %v", err)
	}

	// Verify the record's metadata
	if string(doc.Metadata) != "larger metadata" {
		t.Errorf("Expected metadata 'larger metadata', got '%s'", doc.Metadata)
	}

	// Verify the record's vector
	expectedVector := []float64{1.0, 2.0, 3.0}
	if !equalVectors(doc.Vector, expectedVector) {
		t.Errorf("Expected vector %v, got %v", expectedVector, doc.Vector)
	}
}

func TestExhaustiveSearch(t *testing.T) {
	// Define collection options
	options := CollectionOptions{
		Name:           "test_exhaustive_search.dat",
		DistanceMethod: Euclidean,
		DimensionCount: 3,
		Create:         true,
	}

	// Remove any existing file
	os.Remove(options.Name)

	// Create a new collection
	collection := NewCollection(options)
	defer collection.Close()

	// Add documents to the collection
	documents := []struct {
		id       uint64
		vector   []float64
		metadata []byte
	}{
		{1, []float64{1.0, 2.0, 3.0}, []byte("doc1")},
		{2, []float64{4.0, 5.0, 6.0}, []byte("doc2")},
		{3, []float64{7.0, 8.0, 9.0}, []byte("doc3")},
	}

	for _, doc := range documents {
		collection.AddDocument(doc.id, doc.vector, doc.metadata)
	}

	// Define search arguments for exhaustive search
	searchArgs := SearchArgs{
		Vector:    []float64{1.0, 2.0, 3.0},
		Precision: "exact",
		K:         3, // Request all documents
	}

	// Perform the exhaustive search
	results := collection.Search(searchArgs)

	// Verify the number of results
	if len(results.Results) != len(documents) {
		t.Errorf("Expected %d results, got %d", len(documents), len(results.Results))
	}

	// Verify that all documents are returned
	expectedIDs := map[uint64]bool{1: true, 2: true, 3: true}
	for _, result := range results.Results {
		if !expectedIDs[result.ID] {
			t.Errorf("Unexpected document ID %d in results", result.ID)
		}
		delete(expectedIDs, result.ID)
	}

	// Verify that PercentSearched is 100
	if results.PercentSearched != 100.0 {
		t.Errorf("Expected PercentSearched to be 100, got %f", results.PercentSearched)
	}
}

func TestVectorSearchWith4BitQuantization(t *testing.T) {
	// Define collection options with 4-bit quantization
	collectionName := "test_collection_4bit.dat"
	options := CollectionOptions{
		Name:           collectionName,
		DistanceMethod: Euclidean,
		DimensionCount: 3, // Example dimension count
		Quantization:   4, // 4-bit quantization
	}

	os.Remove(collectionName)
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
		Vector: searchVector,
		K:      5, // Limit to top 5 results
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
