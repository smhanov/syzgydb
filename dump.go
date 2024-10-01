package main

import (
	"encoding/binary"
	"fmt"
	"math"
)

func dump_memfile(mf *memfile) {
	// Read and display the header
	header := make([]byte, headerSize)
	mf.ReadAt(header, 0)

	version := header[0]
	distanceMethod := header[1]
	dimensionCount := binary.LittleEndian.Uint64(header[2:])

	fmt.Printf("Header:\n")
	fmt.Printf("  Version: %d\n", version)
	fmt.Printf("  Distance Method: %d\n", distanceMethod)
	fmt.Printf("  Number of Dimensions: %d\n", dimensionCount)

	// Iterate over all records
	fmt.Println("Records:")
	for id, offset := range mf.idOffsets {
		// Read the total length of the record
		recordLength := mf.readUint64(offset)

		// Read the ID
		recordID := mf.readUint64(offset + 8)

		// Check if the record is deleted
		if recordID == 0xffffffffffffffff {
			fmt.Printf("  Record at offset %d is deleted\n", offset)
			continue
		}

		// Read the vector
		vector := make([]float64, dimensionCount)
		vectorOffset := offset + 16
		for i := range vector {
			vector[i] = math.Float64frombits(mf.readUint64(vectorOffset + int64(i*8)))
		}

		// Read the metadata length
		metadataLength := mf.readUint32(vectorOffset + int64(dimensionCount*8))

		// Read the metadata
		metadataOffset := vectorOffset + int64(dimensionCount*8) + 4
		metadata := make([]byte, metadataLength)
		mf.ReadAt(metadata, metadataOffset)

		// Display the record
		fmt.Printf("  Record ID: %d\n", id)
		fmt.Printf("    Total Length: %d\n", recordLength)
		fmt.Printf("    Vector: %v\n", vector)
		fmt.Printf("    Metadata: %s\n", string(metadata))
	}
}
