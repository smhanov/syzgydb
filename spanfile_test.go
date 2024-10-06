package main

import (
    "os"
    "testing"
    "io/ioutil"
)

func setupTestDB(t *testing.T) (*DB, func()) {
    tempFile, err := ioutil.TempFile("", "spanfile_test")
    if err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }

    db, err := OpenFile(tempFile.Name(), OpenOptions{CreateIfNotExists: true})
    if err != nil {
        t.Fatalf("Failed to open database: %v", err)
    }

    cleanup := func() {
        db.file.Close()
        os.Remove(tempFile.Name())
    }

    return db, cleanup
}

func TestOpenFile(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    if db == nil {
        t.Fatal("Expected non-nil DB")
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

func TestFreeSpaceReuse(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    dataStreams := []DataStream{
        {StreamID: 1, Data: []byte("Hello")},
    }
    db.WriteRecord("record1", dataStreams)
    db.WriteRecord("record2", dataStreams)

    db.WriteRecord("record1", []DataStream{{StreamID: 1, Data: []byte("Updated")}})

    // Check if the free space from the first record1 is reused
    // This requires inspecting the free list or file structure
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
    err := db.IterateRecords(func(recordID string, dataStreams []DataStream) error {
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

func TestDumpFile(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    dataStreams := []DataStream{
        {StreamID: 1, Data: []byte("Hello")},
    }
    db.WriteRecord("record1", dataStreams)

    err := db.DumpFile(os.Stdout)
    if err != nil {
        t.Fatalf("Failed to dump file: %v", err)
    }
}
