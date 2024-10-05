package syzgydb

import (
	"bytes"
	"os"
	"testing"
)

func TestMemfile(t *testing.T) {
	// Create a temporary file for testing
	fileName := "testfile"
	file, err := os.Create(fileName)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()
	defer os.Remove(fileName)

	// Create a memfile instance with an 8-byte header
	header := make([]byte, 8)
	mf, err := createMemFile(fileName, header, CreateAndOverwrite)
	if err != nil {
		t.Fatalf("Failed to create memfile: %v", err)
	}

	// Test addRecord and readRecord
	data := []byte("testdata")
	mf.addRecord(1, data)

	readData, err := mf.readRecord(1)
	if err != nil {
		t.Fatalf("Failed to read record: %v", err)
	}
	if !bytes.Equal(data, readData) {
		t.Errorf("Expected %v, got %v", data, readData)
	}

	// Test readUint64 and writeUint64
	offset := int64(0)
	value := uint64(123456789)
	mf.writeUint64(offset, value)

	readValue := mf.readUint64(offset)
	if value != readValue {
		t.Errorf("Expected %d, got %d", value, readValue)
	}
	// Test deleteRecord
	err = mf.deleteRecord(1)
	if err != nil {
		t.Fatalf("Failed to delete record: %v", err)
	}

	_, err = mf.readRecord(1)
	if err == nil {
		t.Errorf("Expected error when reading deleted record, got nil")
	}
}

func TestMemfileExpansion(t *testing.T) {
	// Create a temporary file for testing
	fileName := "testfile_expansion"

	os.Remove(fileName)

	// Create a memfile instance with an 8-byte header
	header := make([]byte, 8)
	mf, err := createMemFile(fileName, header, CreateAndOverwrite)
	if err != nil {
		t.Fatalf("Failed to create memfile: %v", err)
	}

	// Add a record that ends 8 bytes before 4096
	largeData := make([]byte, minGrowthBytes-8-16-8) // 16 bytes for length and ID
	largeData[0] = 'a'
	mf.addRecord(2, largeData)

	// Add a second record to trigger file expansion
	secondData := []byte("second")
	mf.addRecord(3, secondData)

	// Close and re-open the file
	mf.File.Close()
	mf, err = createMemFile(fileName, header)
	if err != nil {
		t.Fatalf("Failed to re-open memfile: %v", err)
	}

	// Verify both records can be read
	readLargeData, err := mf.readRecord(2)
	if err != nil {
		t.Fatalf("Failed to read large record: %v", err)
	}
	if !bytes.Equal(largeData, readLargeData[:len(largeData)]) {
		t.Errorf("Expected large data, got %v", readLargeData)
	}

	readSecondData, err := mf.readRecord(3)
	if err != nil {
		t.Fatalf("Failed to read second record: %v", err)
	}
	if !bytes.Equal(secondData, readSecondData) {
		t.Errorf("Expected second data, got %v", readSecondData)
	}
}
