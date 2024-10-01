package main

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
