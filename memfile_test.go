package main

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

	// Create a memfile instance
	mf, err := createMemFile(fileName, 0)
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
}
