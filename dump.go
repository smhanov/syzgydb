package syzgydb

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
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
	version, _ := readUint(file, 4)
	headerLength, _ := readUint(file, 4)
	distanceMethod, _ := readUint(file, 1)
	dimensionCount, _ := readUint(file, 4)
	quantization, _ := readUint(file, 1) // Add this line

	fmt.Printf("Header:\n")
	fmt.Printf("  Version: %d\n", version)
	fmt.Printf("  Header Length: %d\n", headerLength)
	fmt.Printf("  Distance Method: %d\n", distanceMethod)
	fmt.Printf("  Number of Dimensions: %d\n", dimensionCount)
	fmt.Printf("  Quantization: %d-bit\n", quantization) // Add this line

	// Iterate over all records
	fmt.Println("Records:")
	for {
		recordLength, err := readUint(file, 8)
		if err != nil {
			break
		}

		fmt.Printf("    Total Length: %d\n", recordLength)
		if recordLength == 0 {
			fmt.Println("     (Indicates end of usable records)")
			break
		}

		// Read the ID
		recordID, _ := readUint(file, 8)

		if recordID == deletedRecordMarker {
			fmt.Printf("  Record is deleted\n")
			file.Seek(int64(recordLength)-16, io.SeekCurrent)
			continue
		}

		fmt.Printf("  Record ID: %d\n", recordID)

		// Calculate the vector size based on quantization
		vectorSize := getVectorSize(int(quantization), int(dimensionCount))
		vectorData := make([]byte, vectorSize)
		if _, err := file.Read(vectorData); err != nil {
			break
		}

		vector := make([]float64, dimensionCount)
		vectorOffset := 0

		for i := range vector {
			var quantizedValue uint64
			switch quantization {
			case 4:
				if i%2 == 0 {
					quantizedValue = uint64(vectorData[vectorOffset+i/2] >> 4)
				} else {
					quantizedValue = uint64(vectorData[vectorOffset+i/2] & 0x0F)
				}
			case 8:
				quantizedValue = uint64(vectorData[vectorOffset+i])
			case 16:
				quantizedValue = uint64(binary.BigEndian.Uint16(vectorData[vectorOffset+i*2:]))
			case 32:
				quantizedValue = uint64(binary.BigEndian.Uint32(vectorData[vectorOffset+i*4:]))
			case 64:
				quantizedValue = binary.BigEndian.Uint64(vectorData[vectorOffset+i*8:])
			}

			vector[i] = dequantize(quantizedValue, int(quantization))
		}

		fmt.Printf("    Vector: %v\n", vector)

		// Read the metadata length
		metadataLength, _ := readUint(file, 4)
		fmt.Printf("    Metadata length: %v\n", metadataLength)

		// Read the metadata
		metadata := make([]byte, metadataLength)
		if _, err := file.Read(metadata); err != nil {
			break
		}
		fmt.Printf("    Metadata: %s\n", string(metadata))
	}
}

func readUint(f io.Reader, size int) (uint64, error) {
	buf := make([]byte, size)
	if _, err := f.Read(buf); err != nil {
		return 0, err
	}

	switch size {
	case 1:
		return uint64(buf[0]), nil
	case 2:
		return uint64(binary.BigEndian.Uint16(buf)), nil
	case 4:
		return uint64(binary.BigEndian.Uint32(buf)), nil
	case 8:
		return uint64(binary.BigEndian.Uint64(buf)), nil
	default:
		log.Fatalf("Invalid number of bytes: %d", size)
	}
	return 0, nil
}
