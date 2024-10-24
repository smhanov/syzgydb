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

	"github.com/smhanov/syzgydb/replication"
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
	db.WriteRecord("record1", dataStreams, 0, 0, db.NextTimestamp())

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
	db.WriteRecord("record1", dataStreams, 0, 0, db.NextTimestamp())

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

func TestWriteRecord(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	dataStreams := []DataStream{
		{StreamID: 1, Data: []byte("Hello")},
	}
	err := db.WriteRecord("record1", dataStreams, 0, 0, db.NextTimestamp())
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
	db.WriteRecord("record1", dataStreams, 0, 0, db.NextTimestamp())

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
	db.WriteRecord("record1", dataStreams, 0, 0, db.NextTimestamp())

	updatedStreams := []DataStream{
		{StreamID: 1, Data: []byte("Updated")},
	}
	err := db.WriteRecord("record1", updatedStreams, 0, 0, db.NextTimestamp())
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
	db.WriteRecord("record1", dataStreams, 0, 0, db.NextTimestamp())
	db.WriteRecord("record2", dataStreams, 0, 0, db.NextTimestamp())

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
	db.WriteRecord("record1", dataStreams, 0, 0, db.NextTimestamp())

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
	err := db.WriteRecord("record1", []DataStream{{StreamID: 1, Data: data100}}, 0, 0, db.NextTimestamp())
	if err != nil {
		t.Fatalf("Failed to write record of length 100: %v", err)
	}

	// Update the record to be of length 200
	data200 := make([]byte, 200)
	for i := range data200 {
		data200[i] = byte('B' + i%26)
	}
	err = db.WriteRecord("record1", []DataStream{{StreamID: 1, Data: data200}}, 0, 0, db.NextTimestamp())
	if err != nil {
		t.Fatalf("Failed to update record to length 200: %v", err)
	}

	// Write another record of length 50
	data50 := make([]byte, 50)
	for i := range data50 {
		data50[i] = byte('C' + i%26)
	}
	err = db.WriteRecord("record2", []DataStream{{StreamID: 1, Data: data50}}, 0, 0, db.NextTimestamp())
	if err != nil {
		t.Fatalf("Failed to write record of length 50: %v", err)
	}

	// Write another record of length 25
	data25 := make([]byte, 25)
	for i := range data25 {
		data25[i] = byte('D' + i%26)
	}
	err = db.WriteRecord("record3", []DataStream{{StreamID: 1, Data: data25}}, 0, 0, db.NextTimestamp())
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
					err := db.WriteRecord(recordID, []DataStream{{StreamID: 1, Data: data}}, 0, 0, db.NextTimestamp())
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
					err := db.WriteRecord(recordID, []DataStream{{StreamID: 1, Data: newData}}, 0, 0, db.NextTimestamp())
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
					err := db.RemoveRecord(recordID, 0, db.NextTimestamp())
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
func TestDeleteRecord(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	dataStreams := []DataStream{
		{StreamID: 1, Data: []byte("Hello")},
	}
	err := db.WriteRecord("record1", dataStreams, 0, 0, db.NextTimestamp())
	if err != nil {
		t.Fatalf("Failed to write record: %v", err)
	}

	oldOffset := db.index["record1"]

	// Delete the record
	err = db.RemoveRecord("record1", 0, db.NextTimestamp())
	if err != nil {
		t.Fatalf("Failed to delete record: %v", err)
	}

	// Verify the old span is marked as free
	oldSpan, err := parseSpanAtOffset(db.mmapData, oldOffset)
	if err != nil {
		t.Fatalf("Failed to parse old span: %v", err)
	}
	if oldSpan.MagicNumber != freeMagic {
		t.Errorf("Expected old span to be marked as free, got: %v", magicNumberToString(oldSpan.MagicNumber))
	}

	// Verify the record is in the deletedIndex
	newOffset, exists := db.deletedIndex["record1"]
	if !exists {
		t.Fatal("Expected record to be in deletedIndex")
	}
	if newOffset == oldOffset {
		t.Fatal("Expected new offset to be different from old offset")
	}

	// Verify the record is not in the main index
	if _, exists := db.index["record1"]; exists {
		t.Fatal("Expected record to be removed from main index")
	}

	deletedSpan, err := parseSpanAtOffset(db.mmapData, newOffset)
	if err != nil {
		t.Fatalf("Failed to parse deleted span: %v", err)
	}

	if deletedSpan.MagicNumber != deletedMagic {
		t.Errorf("Expected deleted magic number, got: %v", magicNumberToString(deletedSpan.MagicNumber))
	}

	if len(deletedSpan.DataStreams) != 0 {
		t.Errorf("Expected zero data streams in deleted span, got: %d", len(deletedSpan.DataStreams))
	}

	if deletedSpan.RecordID != "record1" {
		t.Errorf("Expected RecordID to be preserved, got: %s", deletedSpan.RecordID)
	}

	// Verify IsRecordDeleted method
	if !db.IsRecordDeleted("record1") {
		t.Error("Expected IsRecordDeleted to return true")
	}

	// Try to read the deleted record
	_, err = db.ReadRecord("record1")
	if err == nil {
		t.Fatal("Expected error when reading deleted record, got nil")
	}
}
func TestWriteDeletedRecord(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Write initial record
	initialData := []DataStream{{StreamID: 1, Data: []byte("Initial")}}
	err := db.WriteRecord("record1", initialData, 0, 0, db.NextTimestamp())
	if err != nil {
		t.Fatalf("Failed to write initial record: %v", err)
	}

	// Delete the record
	err = db.RemoveRecord("record1", 0, db.NextTimestamp())
	if err != nil {
		t.Fatalf("Failed to delete record: %v", err)
	}

	// Verify the record is deleted
	if !db.IsRecordDeleted("record1") {
		t.Fatal("Expected record to be marked as deleted")
	}

	// Write the record again
	newData := []DataStream{{StreamID: 1, Data: []byte("New data")}}
	err = db.WriteRecord("record1", newData, 0, 0, db.NextTimestamp())
	if err != nil {
		t.Fatalf("Failed to write new record: %v", err)
	}

	// Verify the record is no longer marked as deleted
	if db.IsRecordDeleted("record1") {
		t.Fatal("Expected record to no longer be marked as deleted")
	}

	// Read the record and verify its contents
	span, err := db.ReadRecord("record1")
	if err != nil {
		t.Fatalf("Failed to read record: %v", err)
	}
	if string(span.DataStreams[0].Data) != "New data" {
		t.Errorf("Expected 'New data', got '%s'", span.DataStreams[0].Data)
	}

	// Verify that the deleted span has been freed
	_, exists := db.deletedIndex["record1"]
	if exists {
		t.Error("Expected record to be removed from deletedIndex")
	}
}

func TestParseFreedSpan(t *testing.T) {
	// Create a freed span
	freedSpan := make([]byte, 8)
	binary.BigEndian.PutUint32(freedSpan[0:4], freeMagic)
	binary.BigEndian.PutUint32(freedSpan[4:8], 8) // Length of the span

	// Parse the freed span
	span, err := parseSpan(freedSpan)
	if err != nil {
		t.Fatalf("Failed to parse freed span: %v", err)
	}

	// Check the parsed span
	if span.MagicNumber != freeMagic {
		t.Errorf("Expected magic number %d, got %d", freeMagic, span.MagicNumber)
	}
	if span.Length != 8 {
		t.Errorf("Expected length 8, got %d", span.Length)
	}
	if span.RecordID != "" {
		t.Errorf("Expected empty RecordID, got %s", span.RecordID)
	}
	if len(span.DataStreams) != 0 {
		t.Errorf("Expected no data streams, got %d", len(span.DataStreams))
	}
}

func TestGetUpdatesSince(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create some test data with different site IDs and sequence numbers
	testData := []struct {
		recordID string
		siteID   uint64
		seqNum   uint64
		data     string
		isDelete bool
	}{
		{"record1", 1, 1, "data1", false},
		{"record2", 1, 2, "data2", false},
		{"record3", 2, 1, "data3", false},
		{"record4", 2, 2, "data4", true}, // This one will be deleted
		{"record5", 3, 1, "data5", false},
	}

	// Write the records
	for _, td := range testData {
		ts := replication.Timestamp{
			UnixTime:     1000,
			LamportClock: int64(td.seqNum),
		}

		if !td.isDelete {
			err := db.WriteRecord(td.recordID, []DataStream{{StreamID: 1, Data: []byte(td.data)}}, td.siteID, td.seqNum, ts)
			if err != nil {
				t.Fatalf("Failed to write record %s: %v", td.recordID, err)
			}
		} else {
			err := db.RemoveRecord(td.recordID, td.siteID, ts)
			if err != nil {
				t.Fatalf("Failed to delete record %s: %v", td.recordID, err)
			}
		}
	}

	// Test cases with different since values and expected results
	tests := []struct {
		name          string
		since         map[uint64]uint64 // map of siteID to sequence number
		maxResults    int
		expectedCount int
		expectedFirst string // recordID of first expected result
		expectedLast  string // recordID of last expected result
	}{
		{
			name:          "Get all updates",
			since:         map[uint64]uint64{},
			maxResults:    10,
			expectedCount: 5,
			expectedFirst: "record1",
			expectedLast:  "record5",
		},
		{
			name:          "Filter by sequence number for site 1",
			since:         map[uint64]uint64{1: 1},
			maxResults:    10,
			expectedCount: 4,
			expectedFirst: "record2",
			expectedLast:  "record5",
		},
		{
			name:          "Limited results",
			since:         map[uint64]uint64{},
			maxResults:    3,
			expectedCount: 3,
			expectedFirst: "record1",
			expectedLast:  "record3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create NodeSequences from the test case
			since := replication.NewNodeSequences()
			for siteID, seqNum := range tt.since {
				since.Update(siteID, seqNum)
			}

			// Get updates
			updates, err := db.GetUpdatesSince(since, tt.maxResults)
			if err != nil {
				t.Fatalf("GetUpdatesSince failed: %v", err)
			}

			// Verify count
			if len(updates) != tt.expectedCount {
				t.Errorf("Expected %d updates, got %d", tt.expectedCount, len(updates))
			}

			// Verify ordering and contents
			if len(updates) > 0 {
				// Check first result
				if updates[0].RecordID != tt.expectedFirst {
					t.Errorf("Expected first record to be %s, got %s", tt.expectedFirst, updates[0].RecordID)
				}

				// Check last result
				if updates[len(updates)-1].RecordID != tt.expectedLast {
					t.Errorf("Expected last record to be %s, got %s", tt.expectedLast, updates[len(updates)-1].RecordID)
				}

				// Verify ordering
				for i := 1; i < len(updates); i++ {
					prev := updates[i-1]
					curr := updates[i]
					if prev.NodeID > curr.NodeID ||
						(prev.NodeID == curr.NodeID && prev.SequenceNo > curr.SequenceNo) {
						t.Errorf("Updates not properly ordered at index %d: %v -> %v",
							i, prev, curr)
					}
				}

				// Verify update types
				for _, update := range updates {
					expectedType := replication.UpsertRecord
					if update.RecordID == "record4" {
						expectedType = replication.DeleteRecord
					}
					if update.Type != expectedType {
						t.Errorf("Wrong update type for %s: expected %v, got %v",
							update.RecordID, expectedType, update.Type)
					}

					// Verify data streams for non-deleted records
					if update.Type == replication.UpsertRecord {
						if len(update.DataStreams) != 1 {
							t.Errorf("Expected 1 data stream for %s, got %d",
								update.RecordID, len(update.DataStreams))
						} else {
							expectedData := fmt.Sprintf("data%c", update.RecordID[len(update.RecordID)-1])
							if string(update.DataStreams[0].Data) != expectedData {
								t.Errorf("Wrong data for %s: expected %s, got %s",
									update.RecordID, expectedData, string(update.DataStreams[0].Data))
							}
						}
					}
				}
			}
		})
	}
}
