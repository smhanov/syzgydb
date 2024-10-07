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
			fmt.Printf("[%08d] Reached end of file\n", start)
			break
		}
		at += 4

		fmt.Printf("[%08d] Magic: %08d (%s)\n", start, magic, magicNumberToString(magic))

		length, err := readUint32(buffer, at)
		if err != nil {
			fmt.Printf("[%08d] Could not read length.\n", at)
			break
		}
		at += 4

		if int(start)+int(length) > len(buffer) {
			fmt.Printf("[%08d] Span length exceeds buffer size.\n", at)
			break
		}
		fmt.Printf("[%08d] Length: %d bytes\n", at, length)

		if magic == activeMagic {
			var seq uint64
			seq, at, err = read7Code(buffer, at)
			if err != nil {
				fmt.Printf("[%08d] Could not read sequence number.\n", at)
				at = start + int(length)
				continue
			}
			fmt.Printf("[%08d] Sequence Number: %d\n", start, seq)

			var idlen uint64
			was := at
			idlen, at, err = read7Code(buffer, at)
			if err != nil {
				fmt.Printf("[%08d] Could not read rec id len.\n", at)
				at = start + int(length)
				continue
			}
			fmt.Printf("[%08d] Record ID length: %d\n", was, idlen)
			fmt.Printf("[%08d] Record ID: %s\n", at, string(buffer[at:at+int(idlen)]))
			at += int(idlen)

			numStreams := buffer[at]
			fmt.Printf("[%08d] Data Streams: %d\n", at, buffer[at])
			at++
			for i := 0; i < int(numStreams); i++ {
				streamId := buffer[at]
				fmt.Printf("[%08d] Stream ID: %d\n", at, streamId)
				at++
				var streamLen uint64
				streamLen, at, err = read7Code(buffer, at)
				if err != nil {
					fmt.Printf("[%08d] Could not read stream length.\n", at)
					at = start + int(length)
					continue
				}
				fmt.Printf("[%08d] Stream Length: %d\n", at, streamLen)
				at += int(streamLen)
			}
			checksum, err := readUint32(buffer, at)
			if err != nil {
				fmt.Printf("[%08d] Could not read checksum.\n", at)
			}
			fmt.Printf("[%08d] Checksum: %x\n", start, checksum)
		} else if magic == freeMagic {
			fmt.Printf("[%08d] Free span of length: %d bytes\n", start, length)
		}

		at = start + int(length)
	}
}
