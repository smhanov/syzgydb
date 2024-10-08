package syzgydb

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestOpenFileWithInvalidMagicNumber(t *testing.T) {
	// Create testdata if it doesn't exist
	err := os.MkdirAll("./testdata", 0755)
	if err != nil {
		t.Fatalf("Failed to create testdata: %v", err)
	}

	tempFile, err := ioutil.TempFile("./testdata", "invalid_magic_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write an invalid magic number to the file
	invalidMagic := []byte{0x00, 0x00, 0x00, 0x00}
	_, err = tempFile.Write(invalidMagic)
	if err != nil {
		t.Fatalf("Failed to write invalid magic number: %v", err)
	}
	tempFile.Close()

	// Attempt to open the file
	_, err = OpenFile(tempFile.Name(), ReadWrite)
	if err == nil || !strings.Contains(err.Error(), "invalid magic number") {
		t.Fatalf("Expected error for invalid magic number, got: %v", err)
	}
}

func TestGetSpanReader(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	dataStreams := []DataStream{
		{StreamID: 1, Data: []byte("Hello")},
	}
	db.WriteRecord("record1", dataStreams)

	spanReader, err := db.getSpanReader("record1")
	if err != nil {
		t.Fatalf("Failed to get SpanReader: %v", err)
	}

	streamData, err := spanReader.getStream(1)
	if err != nil {
		t.Fatalf("Failed to get stream data: %v", err)
	}

	if string(streamData) != "Hello" {
		t.Errorf("Expected stream data 'Hello', got '%s'", streamData)
	}
}

func TestChecksumVerification(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	dataStreams := []DataStream{
		{StreamID: 1, Data: []byte("Hello")},
	}
	db.WriteRecord("record1", dataStreams)

	// Log the length of the file
	fileLength := len(db.mmapData)
	t.Logf("File length: %d bytes after writing record", fileLength)

	// Manually corrupt the checksum
	offset := db.index["record1"]

	// Log the span length
	t.Logf("Span was written at offset %v", offset)

	// Corrupt the record
	db.mmapData[offset+9] ^= 0xFF

	_, err := db.ReadRecord("record1")
	if err == nil {
		t.Fatal("Expected checksum verification to fail")
	}

	// check if error contained the string "checksum"
	if !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("Expected error to contain 'checksum', got: %v", err)
	}
}

func TestInvalidSpanHandling(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Manually write an invalid span
	invalidSpan := make([]byte, 20)
	binary.BigEndian.PutUint32(invalidSpan, 0xDEADBEEF) // Invalid magic number
	db.appendToFile(invalidSpan)

	db.scanFile()

	// Ensure no invalid spans are in the index
	// other than the "header" span
	if len(db.index) != 1 {
		t.Errorf("Expected no valid records, got %d", len(db.index))
	}
}

func TestSequenceNumberWraparound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.sequenceNumber = ^uint32(0) // Set sequence number near max value

	dataStreams := []DataStream{
		{StreamID: 1, Data: []byte("Hello")},
	}
	err := db.WriteRecord("record1", dataStreams)
	if err != nil {
		t.Fatalf("Failed to write record: %v", err)
	}

	if db.sequenceNumber != 0 {
		t.Errorf("Expected sequence number to wrap around to 0, got %d", db.sequenceNumber)
	}
}

func TestWriteRecord(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	dataStreams := []DataStream{
		{StreamID: 1, Data: []byte("Hello")},
	}
	err := db.WriteRecord("record1", dataStreams)
	if err != nil {
		t.Fatalf("Failed to write record: %v", err)
	}
}

func TestReadRecord(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	dataStreams := []DataStream{
		{StreamID: 1, Data: []byte("Hello")},
	}
	db.WriteRecord("record1", dataStreams)

	span, err := db.ReadRecord("record1")
	if err != nil {
		t.Fatalf("Failed to read record: %v", err)
	}

	if span.RecordID != "record1" {
		t.Errorf("Expected RecordID 'record1', got '%s'", span.RecordID)
	}
}

func TestUpdateRecord(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	dataStreams := []DataStream{
		{StreamID: 1, Data: []byte("Hello")},
	}
	db.WriteRecord("record1", dataStreams)

	updatedStreams := []DataStream{
		{StreamID: 1, Data: []byte("Updated")},
	}
	err := db.WriteRecord("record1", updatedStreams)
	if err != nil {
		t.Fatalf("Failed to update record: %v", err)
	}

	span, err := db.ReadRecord("record1")
	if err != nil {
		t.Fatalf("Failed to read updated record: %v", err)
	}

	if string(span.DataStreams[0].Data) != "Updated" {
		t.Errorf("Expected data 'Updated', got '%s'", span.DataStreams[0].Data)
	}
}

func TestIterateRecords(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	dataStreams := []DataStream{
		{StreamID: 1, Data: []byte("Hello")},
	}
	db.WriteRecord("record1", dataStreams)
	db.WriteRecord("record2", dataStreams)

	count := 0
	err := db.IterateRecords(func(recordID string, sr *SpanReader) error {
		// Use the SpanReader to get the stream data
		streamData, err := sr.getStream(1)
		if err != nil {
			return err
		}

		// Verify the stream data
		if string(streamData) != "Hello" {
			t.Errorf("Expected stream data 'Hello', got '%s'", streamData)
		}

		count++
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to iterate records: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 records, got %d", count)
	}
}

func TestGetStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	dataStreams := []DataStream{
		{StreamID: 1, Data: []byte("Hello")},
	}
	db.WriteRecord("record1", dataStreams)

	size, numRecords := db.GetStats()
	if numRecords != 1 {
		t.Errorf("Expected 1 record, got %d", numRecords)
	}

	if size == 0 {
		t.Error("Expected non-zero database size")
	}
}

func TestRecordUpdateAndPersistence(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Write a record of length 100
	data100 := make([]byte, 100)
	for i := range data100 {
		data100[i] = byte('A' + i%26)
	}
	err := db.WriteRecord("record1", []DataStream{{StreamID: 1, Data: data100}})
	if err != nil {
		t.Fatalf("Failed to write record of length 100: %v", err)
	}

	// Update the record to be of length 200
	data200 := make([]byte, 200)
	for i := range data200 {
		data200[i] = byte('B' + i%26)
	}
	err = db.WriteRecord("record1", []DataStream{{StreamID: 1, Data: data200}})
	if err != nil {
		t.Fatalf("Failed to update record to length 200: %v", err)
	}

	// Write another record of length 50
	data50 := make([]byte, 50)
	for i := range data50 {
		data50[i] = byte('C' + i%26)
	}
	err = db.WriteRecord("record2", []DataStream{{StreamID: 1, Data: data50}})
	if err != nil {
		t.Fatalf("Failed to write record of length 50: %v", err)
	}

	// Write another record of length 25
	data25 := make([]byte, 25)
	for i := range data25 {
		data25[i] = byte('D' + i%26)
	}
	err = db.WriteRecord("record3", []DataStream{{StreamID: 1, Data: data25}})
	if err != nil {
		t.Fatalf("Failed to write record of length 25: %v", err)
	}

	name := db.file.Name()
	// Close and reopen the file
	db.Close() // Use the new Close method
	db, err = OpenFile(name, ReadWrite)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}

	// Verify all records are available
	span, err := db.ReadRecord("record1")
	if err != nil {
		t.Fatalf("Failed to read record1: %v", err)
	}
	if string(span.DataStreams[0].Data) != string(data200) {
		t.Errorf("Data mismatch for record1: expected %s, got %s", data200, span.DataStreams[0].Data)
	}

	span, err = db.ReadRecord("record2")
	if err != nil {
		t.Fatalf("Failed to read record2: %v", err)
	}
	if string(span.DataStreams[0].Data) != string(data50) {
		t.Errorf("Data mismatch for record2: expected %s, got %s", data50, span.DataStreams[0].Data)
	}

	span, err = db.ReadRecord("record3")
	if err != nil {
		t.Fatalf("Failed to read record3: %v", err)
	}
	if string(span.DataStreams[0].Data) != string(data25) {
		t.Errorf("Data mismatch for record3: expected %s, got %s", data25, span.DataStreams[0].Data)
	}
}

func TestBatchOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Seed the random number generator
	rsource := rand.NewSource(0)
	r := rand.New(rsource)

	// Map to keep track of expected records and their contents
	expectedRecords := make(map[string][]byte)

	// Function to generate random data of a given size
	generateRandomData := func(size int) []byte {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte('A' + i%26) // Simple pattern for test data
		}
		return data
	}

	chooseRandomRecord := func() string {
		if len(expectedRecords) == 0 {
			return ""
		}
		keys := make([]string, 0, len(expectedRecords))
		for k := range expectedRecords {
			keys = append(keys, k)
		}

		sort.Strings(keys)
		return keys[r.Intn(len(keys))]
	}

	const count = 10000
	const batchSize = 100
	nextRecordID := 0
	var recordID string
	// Perform operations in batches of 100
	for batch := 0; batch < count/batchSize; batch++ {
		for i := 0; i < batchSize; i++ {
			operation := r.Intn(3) // Randomly choose an operation: 0=create, 1=update, 2=delete

			switch operation {
			case 0: // Create a new record
				recordID := fmt.Sprintf("record%d", nextRecordID)
				nextRecordID++
				if _, exists := expectedRecords[recordID]; !exists {
					dataSize := 100 + r.Intn(101) // Random size between 100 and 200 bytes
					data := generateRandomData(dataSize)
					err := db.WriteRecord(recordID, []DataStream{{StreamID: 1, Data: data}})
					if err != nil {
						t.Fatalf("Failed to write record: %v", err)
					}
					expectedRecords[recordID] = data
				}
			case 1: // Update an existing record
				recordID = chooseRandomRecord()
				if _, exists := expectedRecords[recordID]; exists {
					dataSize := 100 + r.Intn(101) // Random size between 100 and 200 bytes
					newData := generateRandomData(dataSize)
					err := db.WriteRecord(recordID, []DataStream{{StreamID: 1, Data: newData}})
					if err != nil {
						t.Fatalf("Failed to update record: %v", err)
					}
					expectedRecords[recordID] = newData
				}
			case 2: // Delete an existing record
				recordID = chooseRandomRecord()
				if _, exists := expectedRecords[recordID]; exists {
					delete(expectedRecords, recordID)
					// Simulate deletion by writing an empty data stream
					err := db.RemoveRecord(recordID)
					if err != nil {
						t.Fatalf("Failed to delete record: %v", err)
					}
				}
			}
		}

		name := db.file.Name()

		// Close and reopen the spanfile
		db.Close() // Use the new Close method
		var err error
		db, err = OpenFile(name, 0)
		if err != nil {
			t.Fatalf("Failed to reopen database: %v", err)
		}

		// Verify all expected records are present
		for recordID, expectedData := range expectedRecords {
			t.Logf("Verifying record %s", recordID)
			span, err := db.ReadRecord(recordID)
			if err != nil {
				DumpIndex(name)
				t.Fatalf("Failed to read record %s: %v", recordID, err)
			}
			if string(span.DataStreams[0].Data) != string(expectedData) {
				t.Errorf("Data mismatch for record %s: expected %s, got %s", recordID, expectedData, span.DataStreams[0].Data)
			}
		}
	}
}
