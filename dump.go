package syzgydb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// indentWriter is a custom writer that adds an indent to the start of each line
type indentWriter struct {
	w          io.Writer
	prefix     string
	needIndent bool
}

func (iw *indentWriter) Write(p []byte) (n int, err error) {
	var written int
	lines := bytes.Split(p, []byte("\n"))
	for i, line := range lines {
		if len(line) > 0 {
			if iw.needIndent {
				_, err = iw.w.Write([]byte(iw.prefix))
				if err != nil {
					return written, err
				}
			}
			n, err = iw.w.Write(line)
			written += n
			if err != nil {
				return written, err
			}
		}
		if i < len(lines)-1 {
			n, err = iw.w.Write([]byte("\n"))
			written += n
			if err != nil {
				return written, err
			}
			iw.needIndent = true
		}
	}
	return written, nil
}

func ExportJSON(c *Collection, w io.Writer) error {
	// Write the opening brace
	fmt.Fprintln(w, "{")

	// Write the collection options
	fmt.Fprint(w, "  \"collection\": ")
	options := c.GetOptions()
	optionsJSON, err := json.MarshalIndent(options, "  ", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode collection options: %v", err)
	}
	optionsJSON = bytes.TrimRightFunc(optionsJSON, func(r rune) bool {
		return r == '\n' || r == ' ' || r == '\t'
	})
	w.Write(optionsJSON)

	// Write the records array
	fmt.Fprint(w, ",\n  \"records\": [")

	ids := c.GetAllIDs()
	for i, id := range ids {
		doc, err := c.GetDocument(id)
		if err != nil {
			return fmt.Errorf("failed to get document with id %v: %v", id, err)
		}

		if i > 0 {
			fmt.Fprint(w, "  }, {\n")
		} else {
			fmt.Fprint(w, "{\n")
		}

		// Write the record ID
		fmt.Fprintf(w, "    \"id\": %d,\n", id)

		// Write the vector all on one line
		fmt.Fprintf(w, "    \"vector\": [")
		vector := doc.Vector
		for j, v := range vector {
			if j > 0 {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprintf(w, "%f", v)
		}
		fmt.Fprint(w, "],\n    \"metadata\": ")

		// Write the metadata
		var decodedMetadata interface{}
		if err := json.Unmarshal(doc.Metadata, &decodedMetadata); err != nil {
			return fmt.Errorf("failed to decode metadata for document with id %v: %v", id, err)
		}

		// Create a buffer to hold the JSON for this document
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(decodedMetadata); err != nil {
			return fmt.Errorf("failed to encode document with id %v: %v", id, err)
		}

		// Trim the leading newline
		metadataJSON := buf.Bytes() /*bytes.TrimLeftFunc(buf.Bytes(), func(r rune) bool {
			return r == '\n' || r == ' ' || r == '\t'
		})*/

		// Write the first line without extra indentation
		firstNewline := bytes.Index(metadataJSON, []byte("\n"))
		if firstNewline != -1 {
			fmt.Fprint(w, string(metadataJSON[:firstNewline+1]))
			// Use the custom indenting writer for the rest
			iw := &indentWriter{w: w, prefix: "    ", needIndent: true}
			if _, err := iw.Write(metadataJSON[firstNewline+1:]); err != nil {
				return fmt.Errorf("failed to write document with id %v: %v", id, err)
			}
		} else {
			// If there's only one line, write it directly
			fmt.Fprint(w, string(metadataJSON))
		}
	}

	// Write the closing brackets
	if len(ids) > 0 {
		fmt.Fprint(w, "  }")
	}
	fmt.Fprintln(w, "]\n}")

	return nil
}

func ImportJSON(collectionName string, r io.Reader) error {
	// Create a decoder to read the JSON input
	decoder := json.NewDecoder(r)

	// Read the opening brace
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("failed to read opening brace: %v", err)
	}

	var options CollectionOptions
	var collection *Collection

	// Read the JSON object key by key
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("failed to read object key: %v", err)
		}

		switch key {
		case "collection":
			// Read the collection options
			if err := decoder.Decode(&options); err != nil {
				return fmt.Errorf("failed to decode collection options: %v", err)
			}
			options.Name = collectionName

			// Create the collection
			collection, err = NewCollection(options)
			if err != nil {
				return fmt.Errorf("failed to create collection: %v", err)
			}

		case "records":
			if collection == nil {
				return fmt.Errorf("collection must be defined before records")
			}

			// Read the opening bracket of the records array
			if _, err := decoder.Token(); err != nil {
				return fmt.Errorf("failed to read opening bracket of records array: %v", err)
			}

			// Read and import each record
			for decoder.More() {
				var doc struct {
					ID       uint64          `json:"id"`
					Vector   []float64       `json:"vector"`
					Metadata json.RawMessage `json:"metadata"`
				}

				if err := decoder.Decode(&doc); err != nil {
					return fmt.Errorf("failed to decode document: %v", err)
				}

				// Add the document to the collection
				collection.AddDocument(doc.ID, doc.Vector, doc.Metadata)
			}

			// Read the closing bracket of the records array
			if _, err := decoder.Token(); err != nil {
				return fmt.Errorf("failed to read closing bracket of records array: %v", err)
			}

		default:
			return fmt.Errorf("unexpected key in JSON: %v", key)
		}
	}

	// Read the closing brace
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("failed to read closing brace: %v", err)
	}

	if collection == nil {
		return fmt.Errorf("no collection was created")
	}

	return nil
}

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
				fmt.Printf("[%08d] Could not read Time.\n", at)
				at = start + int(length)
				continue
			}
			fmt.Printf("[%08d] Time: %d, (%s)\n", start, seq, time.Unix(0, int64(seq)*int64(time.Millisecond)).Format(time.RFC3339))

			var lamport uint64
			lamport, at, err = read7Code(buffer, at)
			if err != nil {
				fmt.Printf("[%08d] Could not read Time.\n", at)
				at = start + int(length)
				continue
			}
			fmt.Printf("[%08d] Lamport: %d\n", start, lamport)

			var siteID uint64
			siteID, at, err = read7Code(buffer, at)
			if err != nil {
				fmt.Printf("[%08d] Could not read siteID.\n", at)
				at = start + int(length)
				continue
			}
			fmt.Printf("[%08d] SiteID: %d\n", start, siteID)

			var unused uint64
			unused, at, err = read7Code(buffer, at)
			if err != nil {
				fmt.Printf("[%08d] Could not read unused byte.\n", at)
				at = start + int(length)
				continue
			}
			fmt.Printf("[%08d] Unused: %d\n", start, unused)

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
