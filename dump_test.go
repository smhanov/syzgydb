package syzgydb

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/smhanov/syzgydb/replication"
)

func TestExportImportJSON(t *testing.T) {
	// Create a temporary file for the collection
	tempFile, err := os.CreateTemp("", "test_collection_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// Create a test collection
	options := CollectionOptions{
		Name:           tempFile.Name(),
		DistanceMethod: Euclidean,
		DimensionCount: 3,
		Quantization:   64,
		FileMode:       CreateAndOverwrite,
		Timestamp:      replication.Timestamp{UnixTime: 0, LamportClock: 1},
	}

	collection, err := NewCollection(options)
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Add some test documents
	testDocs := []struct {
		id       uint64
		vector   []float64
		metadata []byte
	}{
		{1, []float64{1.0, 2.0, 3.0}, []byte(`{"name": "doc1"}`)},
		{2, []float64{4.0, 5.0, 6.0}, []byte(`{"name": "doc2"}`)},
		{3, []float64{7.0, 8.0, 9.0}, []byte(`{"name": "doc3"}`)},
	}

	for _, doc := range testDocs {
		collection.AddDocument(doc.id, doc.vector, doc.metadata)
	}

	// Export the collection to JSON
	var buf bytes.Buffer
	err = ExportJSON(collection, &buf)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	// Close the original collection
	collection.Close()

	// Create a new temporary file for the imported collection
	importTempFile, err := os.CreateTemp("", "test_import_collection_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file for import: %v", err)
	}
	defer os.Remove(importTempFile.Name())
	importTempFile.Close()

	// Import the JSON into a new collection
	err = ImportJSON(importTempFile.Name(), &buf)
	if err != nil {
		t.Fatalf("ImportJSON failed: %v", err)
	}

	// Open the imported collection
	importedCollection, err := NewCollection(CollectionOptions{Name: importTempFile.Name(), FileMode: ReadOnly})
	if err != nil {
		t.Fatalf("Failed to open imported collection: %v", err)
	}

	// Verify the imported collection
	importedOptions := importedCollection.GetOptions()
	// Ignore Name and FileMode in the comparison
	options.Name = ""
	options.FileMode = 0
	importedOptions.Name = ""
	importedOptions.FileMode = 0
	if !reflect.DeepEqual(options, importedOptions) {
		t.Errorf("Imported collection options do not match original. Got %+v, want %+v", importedOptions, options)
	}

	for _, doc := range testDocs {
		importedDoc, err := importedCollection.GetDocument(doc.id)
		if err != nil {
			t.Errorf("Failed to get imported document with id %d: %v", doc.id, err)
			continue
		}

		if !reflect.DeepEqual(importedDoc.Vector, doc.vector) {
			t.Errorf("Imported vector does not match for document %d. Got %v, want %v", doc.id, importedDoc.Vector, doc.vector)
		}

		var originalMetadata, importedMetadata map[string]interface{}
		json.Unmarshal(doc.metadata, &originalMetadata)
		json.Unmarshal(importedDoc.Metadata, &importedMetadata)

		if !reflect.DeepEqual(importedMetadata, originalMetadata) {
			t.Errorf("Imported metadata does not match for document %d. Got %v, want %v", doc.id, importedMetadata, originalMetadata)
		}
	}

	// Close the imported collection
	importedCollection.Close()
}

/* too sensitive to whitespace
func TestExportJSONFormat(t *testing.T) {
	// Create a temporary file for the collection
	tempFile, err := os.CreateTemp("", "test_collection_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// Create a test collection
	options := CollectionOptions{
		Name:           tempFile.Name(),
		DistanceMethod: Euclidean,
		DimensionCount: 2,
		Quantization:   32,
		FileMode:       CreateAndOverwrite,
	}

	collection, err := NewCollection(options)
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Add a test document
	collection.AddDocument(1, []float64{1.5, 2.5}, []byte(`{"name": "test"}`))

	// Export the collection to JSON
	var buf bytes.Buffer
	err = ExportJSON(collection, &buf)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	// Close the collection
	collection.Close()

	// Read the exported JSON
	exported, err := io.ReadAll(&buf)
	if err != nil {
		t.Fatalf("Failed to read exported JSON: %v", err)
	}

	// Define the expected JSON structure
	expected := `{
  "collection": {
    "name": "` + tempFile.Name() + `",
    "distance_method": 0,
    "dimension_count": 2,
    "quantization": 32
  },
  "records": [{
    "id": 1,
    "vector": [1.500000, 2.500000],
    "metadata": {
      "name": "test"
    }
  }]
}`

	// Compare the exported JSON with the expected structure
	if string(exported) != expected {
		t.Errorf("Exported JSON does not match expected format.\nGot:\n%s\nWant:\n%s", string(exported), expected)
	}
}

*/

func TestExportToFileAndImport(t *testing.T) {
	// Create a temporary file for the original collection
	originalFile, err := os.CreateTemp("", "original_collection_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file for original collection: %v", err)
	}
	defer os.Remove(originalFile.Name())
	originalFile.Close()

	// Create the original collection
	options := CollectionOptions{
		Name:           originalFile.Name(),
		DistanceMethod: Cosine,
		DimensionCount: 4,
		Quantization:   32,
		FileMode:       CreateAndOverwrite,
	}

	originalCollection, err := NewCollection(options)
	if err != nil {
		t.Fatalf("Failed to create original collection: %v", err)
	}

	// Add some test documents
	testDocs := []struct {
		id       uint64
		vector   []float64
		metadata []byte
	}{
		{1, []float64{1.0, 2.0, 3.0, 4.0}, []byte(`{"name": "doc1", "category": "A"}`)},
		{2, []float64{5.0, 6.0, 7.0, 8.0}, []byte(`{"name": "doc2", "category": "B"}`)},
		{3, []float64{9.0, 10.0, 11.0, 12.0}, []byte(`{"name": "doc3", "category": "C"}`)},
	}

	for _, doc := range testDocs {
		originalCollection.AddDocument(doc.id, doc.vector, doc.metadata)
	}

	// Create a temporary file for the JSON export
	jsonFile, err := os.CreateTemp("", "export_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file for JSON export: %v", err)
	}
	defer os.Remove(jsonFile.Name())

	// Export the collection to the JSON file
	err = ExportJSON(originalCollection, jsonFile)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}
	jsonFile.Close()

	// Close the original collection
	originalCollection.Close()

	// Create a new temporary file for the imported collection
	importedFile, err := os.CreateTemp("", "imported_collection_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file for imported collection: %v", err)
	}
	defer os.Remove(importedFile.Name())
	importedFile.Close()

	// Import the JSON file into a new collection
	jsonFile, err = os.Open(jsonFile.Name())
	if err != nil {
		t.Fatalf("Failed to open JSON file for import: %v", err)
	}
	err = ImportJSON(importedFile.Name(), jsonFile)
	if err != nil {
		t.Fatalf("ImportJSON failed: %v", err)
	}
	jsonFile.Close()

	// Open the imported collection
	importedCollection, err := NewCollection(CollectionOptions{Name: importedFile.Name(), FileMode: ReadOnly})
	if err != nil {
		t.Fatalf("Failed to open imported collection: %v", err)
	}

	// Verify the imported collection options
	importedOptions := importedCollection.GetOptions()
	if importedOptions.DistanceMethod != options.DistanceMethod {
		t.Errorf("Imported DistanceMethod does not match. Got %v, want %v", importedOptions.DistanceMethod, options.DistanceMethod)
	}
	if importedOptions.DimensionCount != options.DimensionCount {
		t.Errorf("Imported DimensionCount does not match. Got %v, want %v", importedOptions.DimensionCount, options.DimensionCount)
	}
	if importedOptions.Quantization != options.Quantization {
		t.Errorf("Imported Quantization does not match. Got %v, want %v", importedOptions.Quantization, options.Quantization)
	}

	// Verify all expected records exist in the imported collection
	for _, doc := range testDocs {
		importedDoc, err := importedCollection.GetDocument(doc.id)
		if err != nil {
			t.Errorf("Failed to get imported document with id %d: %v", doc.id, err)
			continue
		}

		if !reflect.DeepEqual(importedDoc.Vector, doc.vector) {
			t.Errorf("Imported vector does not match for document %d. Got %v, want %v", doc.id, importedDoc.Vector, doc.vector)
		}

		var originalMetadata, importedMetadata map[string]interface{}
		json.Unmarshal(doc.metadata, &originalMetadata)
		json.Unmarshal(importedDoc.Metadata, &importedMetadata)

		if !reflect.DeepEqual(importedMetadata, originalMetadata) {
			t.Errorf("Imported metadata does not match for document %d. Got %v, want %v", doc.id, importedMetadata, originalMetadata)
		}
	}

	// Close the imported collection
	importedCollection.Close()
}
