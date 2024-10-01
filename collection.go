package main

import (
	"encoding/binary"
	"math"
)

// Constants for euclidean distance or cosine similarity
const (
	Euclidean = iota
	Cosine
)

type Collection struct {
	CollectionOptions
	memfile *memfile
}

func (c *Collection) addDocument(id uint64, vector []float64, metadata []byte) {
	doc := &Document{
		ID:       id,
		Vector:   vector,
		Metadata: metadata,
	}

	// Encode the document
	encodedData := encodeDocument(doc)

	// Add or update the document in the memfile
	c.memfile.addRecord(id, encodedData)
}

func (c *Collection) removeDocument(id uint64) error {
	// Remove the document from the memfile
	return c.memfile.deleteRecord(id)
}

type CollectionOptions struct {
	Name           string
	DistanceMethod int
	DimensionCount int
}

type Document struct {
	ID       uint64
	Vector   []float64
	Metadata []byte
}

type SearchResult struct {
	ID       uint64
	Metadata []byte
	Distance float64
}

type SearchResults struct {
	Results []SearchResult

	// percentage of database searched
	PercentSearched float64
}

type SearchArgs struct {
	Vector []float64
	Filter FilterFn

	// for nearest neighbour search
	MaxCount int

	// for radius search
	Radius float64
}

type FilterFn func(id uint64, metadata []byte) bool

// one byte: version
// 1 byte: distance method
// 8 bytes: number of dimensions
const headerSize = 1

func NewCollection(options CollectionOptions) *Collection {
	c := &Collection{
		CollectionOptions: options,
	}

	header := make([]byte, headerSize)

	// Fill in the header
	header[0] = 1 // version
	header[1] = byte(options.DistanceMethod)
	binary.LittleEndian.PutUint64(header[2:], uint64(options.DimensionCount))

	var err error
	c.memfile, err = createMemFile(c.Name, header)
	if err != nil {
		panic(err)
	}

	return c
}

func encodeDocument(doc *Document) []byte {
	// 8 bytes: document ID
	// 4 bytes: length of vector
	// 4 bytes: length of metadata
	// n bytes: vector
	// n bytes: metadata

	docSize := 8 + 4 + 4 + len(doc.Vector)*8 + len(doc.Metadata)
	data := make([]byte, docSize)

	binary.LittleEndian.PutUint64(data[0:], doc.ID)
	binary.LittleEndian.PutUint32(data[8:], uint32(len(doc.Vector)))
	binary.LittleEndian.PutUint32(data[12:], uint32(len(doc.Metadata)))

	// Encode the floating point vector to the data slice
	vectorOffset := 16
	for i, v := range doc.Vector {
		binary.LittleEndian.PutUint64(data[vectorOffset+i*8:], math.Float64bits(v))
	}

	// Encode the metadata
	metadataOffset := vectorOffset + len(doc.Vector)*8
	copy(data[metadataOffset:], doc.Metadata)

	return data
}

func decodeDocument(data []byte) *Document {
	// Decode the document ID
	id := binary.LittleEndian.Uint64(data[0:])

	// Decode the length of the vector
	vectorLength := binary.LittleEndian.Uint32(data[8:])

	// Decode the length of the metadata
	metadataLength := binary.LittleEndian.Uint32(data[12:])

	// Decode the vector
	vector := make([]float64, vectorLength)
	vectorOffset := 16
	for i := range vector {
		vector[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[vectorOffset+i*8:]))
	}

	// Decode the metadata
	metadataOffset := vectorOffset + int(vectorLength)*8
	metadata := make([]byte, metadataLength)
	copy(metadata, data[metadataOffset:])

	return &Document{
		ID:       id,
		Vector:   vector,
		Metadata: metadata,
	}
}
