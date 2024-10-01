package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
)

// DumpIndex reads the specified file and displays its contents in a human-readable format.
func DumpIndex(filename string) {
	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Read and display the header
	header := make([]byte, headerSize)
	if _, err := file.ReadAt(header, 0); err != nil {
		log.Fatalf("Failed to read header: %v", err)
	}

	version := header[0]
	distanceMethod := header[1]
	dimensionCount := binary.BigEndian.Uint64(header[2:])

	fmt.Printf("Header:\n")
	fmt.Printf("  Version: %d\n", version)
	fmt.Printf("  Distance Method: %d\n", distanceMethod)
	fmt.Printf("  Number of Dimensions: %d\n", dimensionCount)

	// Iterate over all records
	fmt.Println("Records:")
	offset := int64(headerSize)
	for {
		// Read the total length of the record
		recordLengthBuf := make([]byte, 8)
		if _, err := file.ReadAt(recordLengthBuf, offset); err != nil {
			break // End of file
		}
		recordLength := binary.BigEndian.Uint64(recordLengthBuf)
		fmt.Printf("    Total Length: %d\n", recordLength)

		// Read the ID
		recordIDBuf := make([]byte, 8)
		if _, err := file.ReadAt(recordIDBuf, offset+8); err != nil {
			break
		}
		recordID := binary.BigEndian.Uint64(recordIDBuf)

		if recordID == 0xffffffffffffffff {
			fmt.Printf("  Record at offset %d is deleted\n", offset)
			offset += int64(recordLength)
			continue
		}

		fmt.Printf("  Record ID: %d\n", recordID)

		// Read the vector
		vector := make([]float64, dimensionCount)
		vectorOffset := offset + 16
		for i := range vector {
			vectorBuf := make([]byte, 8)
			if _, err := file.ReadAt(vectorBuf, vectorOffset+int64(i*8)); err != nil {
				break
			}
			vector[i] = math.Float64frombits(binary.BigEndian.Uint64(vectorBuf))
		}

		fmt.Printf("    Vector: %v\n", vector)

		// Read the metadata length
		metadataLengthBuf := make([]byte, 4)
		if _, err := file.ReadAt(metadataLengthBuf, vectorOffset+int64(dimensionCount*8)); err != nil {
			break
		}
		metadataLength := binary.BigEndian.Uint32(metadataLengthBuf)
		fmt.Printf("    Metadata length: %v\n", metadataLength)

		// Read the metadata
		metadataOffset := vectorOffset + int64(dimensionCount*8) + 4
		metadata := make([]byte, metadataLength)
		if _, err := file.ReadAt(metadata, metadataOffset); err != nil {
			break
		}
		fmt.Printf("    Metadata: %s\n", string(metadata))

		// Move to the next record
		offset += int64(recordLength)
	}
}
