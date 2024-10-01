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

	// fill in the header here

	var err error
	c.memfile, err = createMemFile(c.Name, header)
	if err != nil {
		panic(err)
	}

	return c
}
