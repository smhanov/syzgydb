package syzgydb

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// Helper function to create a test file path
func testFilePath(fileName string) string {
	return filepath.Join("./testdata", fileName)
}

// Helper function to ensure testdata exists
func ensureTestFolder(t *testing.T) {
	err := os.MkdirAll("./testdata", 0755)
	if err != nil {
		t.Fatalf("Failed to create testdata: %v", err)
	}
}

// Helper function to ensure testdata directory exists
func ensureTestdataDir() {
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		os.Mkdir("testdata", os.ModePerm)
	}
}

// Setup function for SpanFile tests
func setupTestDB(t *testing.T) (*SpanFile, func()) {
	ensureTestFolder(t)
	ensureTestdataDir()

	tempFile, err := ioutil.TempFile("testdata", "spanfile_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	db, err := OpenFile(tempFile.Name(), CreateIfNotExists)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(tempFile.Name())
	}

	return db, cleanup
}

// Setup function for Server tests
func setupTestServer() *Server {
	ensureTestFolder(nil) // We're not in a test context here, so pass nil

	globalConfig.DataFolder = "./testdata" // Set the data folder to the testfolder
	node := NewNode(globalConfig.DataFolder, 0)

	return &Server{node: node}
}
