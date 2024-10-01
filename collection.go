package main

import "encoding/binary"

// Constants for euclidean distance or cosine similarity
const (
	Euclidean = iota
	Cosine
)

type Collection struct {
	CollectionOptions
	memfile *memfile
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

func EncodeDocument(doc *Document) []byte {
	// 8 bytes: document ID
	// 4 bytes: length of vector
	// 4 bytes: length of metadata
	// n bytes: vector
	// n bytes: metadata

	docSize := 8 + 8 + 8 + len(doc.Vector) + len(doc.Metadata)
	data := make([]byte, docSize)

	binary.LittleEndian.PutUint64(data[0:], doc.ID)
	binary.LittleEndian.PutUint32(data[8:], uint32(len(doc.Vector)))
	binary.LittleEndian.PutUint32(data[12:], uint32(len(doc.Metadata)))

	// encode the floating point vector to the data slice

	return data
}
