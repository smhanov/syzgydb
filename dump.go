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
			fmt.Printf("[%08x] Reached end of file")
			break
		}
		at += 4

		log.Printf("[%08x] Magic: %08x (%s)", at, magic, magicNumberToString(magic))

		length, err := readUint32(buffer, at)
		if err != nil {
			fmt.Printf("[%08x] Could not read length.", at)
			break
		}

		//TODO: fill in the rest of the code.

	}
}
