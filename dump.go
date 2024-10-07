package syzgydb

import (
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

	// Get the file size
	stat, err := file.Stat()
	if err != nil {
		log.Fatalf("Failed to get file stats: %v", err)
	}

	// Read the file into a buffer
	size := stat.Size()
	buffer := make([]byte, size)
	_, err = io.ReadFull(file, buffer)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	at := 0
	for {
		start := at
		magic, err := readUint32(buffer, at)
		if err != nil {
			fmt.Printf("[%08x] Reached end of file\n", start)
			break
		}
		at += 4

		log.Printf("[%08x] Magic: %08x (%s)", start, magic, magicNumberToString(magic))

		length, err := readUint32(buffer, at)
		if err != nil {
			fmt.Printf("[%08x] Could not read length.\n", at)
			break
		}
		at += 4

		if int(at)+int(length) > len(buffer) {
			fmt.Printf("[%08x] Span length exceeds buffer size.\n", at)
			break
		}

		if magic == activeMagic {
			spanData := buffer[start : start+int(length)]
			span, err := parseSpan(spanData)
			if err != nil {
				fmt.Printf("[%08x] Error parsing span: %v\n", start, err)
				at += int(length)
				continue
			}

			fmt.Printf("[%08x] Length: %d bytes\n", start, span.Length)
			fmt.Printf("[%08x] Sequence Number: %d\n", start, span.SequenceNumber)
			fmt.Printf("[%08x] Record ID: %s\n", start, span.RecordID)
			fmt.Printf("[%08x] Data Streams:\n", start)
			for _, ds := range span.DataStreams {
				fmt.Printf("  Stream ID: %d, Length: %d bytes\n", ds.StreamID, len(ds.Data))
			}
			fmt.Printf("[%08x] Checksum: %x\n", start, span.Checksum)
		} else if magic == freeMagic {
			fmt.Printf("[%08x] Free span of length: %d bytes\n", start, length)
		}

		at = start + int(length)
	}
}
