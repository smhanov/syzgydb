package syzgydb

import (
	"container/heap"
	"encoding/binary"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"sync"
)

const (
	StopSearch    = iota // Indicates to stop the search due to an error
	PointAccepted        // Indicates the point was accepted and is better
	PointChecked         // Indicates the point was checked unnecessarily
	PointIgnored         // no action taken; pretend point did not exist
)

const useTree = true

/*
CollectionOptions defines the configuration options for creating a Collection.
*/
type CollectionOptions struct {
	// Name is the identifier for the collection.
	Name string

	// DistanceMethod specifies the method used to calculate distances between vectors.
	// It can be either Euclidean or Cosine.
	DistanceMethod int

	// DimensionCount is the number of dimensions for each vector in the collection.
	DimensionCount int

	// Quantization specifies the bit-level quantization for storing vectors.
	// Supported values are 4, 8, 16, 32, and 64, with 64 as the default.
	Quantization int

	// FileMode specifies the mode for opening the memfile.
	FileMode FileMode
}

// GetDocumentCount returns the total number of documents in the collection.
//
// This method provides a quick way to determine the size of the collection
// by returning the count of document IDs stored in the memfile.
func (c *Collection) GetDocumentCount() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Use spanfile to count records
	_, numRecords := c.spanfile.GetStats()
	return numRecords
}

/*
ComputeStats gathers and returns statistics about the collection.
It returns a CollectionStats object filled with the relevant statistics.
*/
func (c *Collection) ComputeStats() CollectionStats {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Calculate the number of documents
	documentCount := len(c.memfile.idOffsets)

	// Calculate the storage size
	storageSize, _ := c.spanfile.GetStats()

	// Calculate the average distance
	averageDistance := c.computeAverageDistance(100) // Example: use 100 samples

	var distanceMethod string
	switch c.DistanceMethod {
	case Euclidean:
		distanceMethod = "euclidean"
	case Cosine:
		distanceMethod = "cosine"
	default:
		distanceMethod = "unknown"
	}

	// Create and return the CollectionStats
	return CollectionStats{
		DocumentCount:   documentCount,
		DimensionCount:  c.DimensionCount,
		Quantization:    c.Quantization,
		DistanceMethod:  distanceMethod,
		StorageSize:     int64(storageSize),
		AverageDistance: averageDistance,
	}
}

/*
Document represents a single document in the collection, consisting of an ID, vector, and metadata.
*/
type Document struct {
	// ID is the unique identifier for the document.
	ID uint64

	// Vector is the numerical representation of the document.
	Vector []float64

	// Metadata is additional information associated with the document.
	Metadata []byte
}

/*
SearchResult represents a single result from a search operation, including the document ID, metadata, and distance.
*/
type SearchResult struct {
	// ID is the unique identifier of the document in the search result.
	ID uint64

	// Metadata is the associated metadata of the document in the search result.
	Metadata []byte

	// Distance is the calculated distance from the search vector to the document vector.
	Distance float64
}

/*
SearchResults contains the results of a search operation, including the list of results and the percentage of the database searched.
*/
type SearchResults struct {
	// Results is a slice of SearchResult containing the documents that matched the search criteria.
	Results []SearchResult

	// PercentSearched indicates the percentage of the database that was searched to obtain the results.
	PercentSearched float64
}

/*
SearchArgs defines the arguments for performing a search in the collection.
*/
type SearchArgs struct {
	// Vector is the search vector used to find similar documents.
	Vector []float64

	// Filter is an optional function to filter documents based on their ID and metadata.
	Filter FilterFn

	// K specifies the maximum number of nearest neighbors to return.
	K int

	// Radius specifies the maximum distance for radius-based search.
	Radius float64

	// when MaxCount and Radius are both 0 we will return all the documents in order of id.
	// These specify the offset and limit
	Offset    int
	Limit     int
	Precision string
}

/*
Contains statistics about the collection
*/
type CollectionStats struct {
	// Number of documents in the collection
	DocumentCount int `json:"document_count"`

	// Number of dimensions in each document vector
	DimensionCount int `json:"dimension_count"`

	// Quantization level used for storing vectors
	Quantization int `json:"quantization"`

	// Distance method used for calculating distances
	// cosine or euclidean
	DistanceMethod string `json:"distance_method"`

	// Storage on disk used by the collection
	StorageSize int64 `json:"storage_size"`

	// Average distance between random pairs of documents
	AverageDistance float64 `json:"average_distance"`
}

type FilterFn func(id uint64, metadata []byte) bool

// 4 bytes: version
// 4 bytes: length of the header
// 1 byte: distance method
// 4 bytes: number of dimensions
const headerSize = 14 // Update the header size to 14

const (
	Euclidean = iota
	Cosine
)

/*
Collection represents a collection of documents, supporting operations such as adding, updating, removing, and searching documents.
*/
type Collection struct {
	CollectionOptions
	spanfile *SpanFile // Change from memfile to spanfile
	index    searchIndex
	lshTree  *lshTree
	mutex    sync.Mutex
	distance func([]float64, []float64) float64
}

/*
NewCollection creates a new Collection with the specified options.
It initializes the collection's memory file and pivots manager.
*/
func NewCollection(options CollectionOptions) *Collection {

	// Define the header size and create a buffer to read it
	header := make([]byte, headerSize)

	// Check if the file exists
	fileExists := false
	if options.FileMode != CreateAndOverwrite {
		if _, err := os.Stat(options.Name); err == nil {
			fileExists = true
		}
	}

	// Open or create the memory-mapped file with the specified mode
	var err error
	var spanFile *SpanFile
	spanFile, err = OpenFile(options.Name, OpenOptions{CreateIfNotExists: options.FileMode == CreateIfNotExists})
	if err != nil {
		panic(err)
	}

	if fileExists {
		// Read the header from the file
		if _, err := spanFile.file.ReadAt(header, 0); err != nil {
			panic(err)
		}

		// Extract the values from the header
		options.DistanceMethod = int(header[8])
		options.DimensionCount = int(binary.BigEndian.Uint32(header[9:]))
		options.Quantization = int(header[13])
	} else {
		if options.Quantization == 0 {
			options.Quantization = 64
		}

		// Fill in the header
		binary.BigEndian.PutUint32(header[0:], 1)                  // version
		binary.BigEndian.PutUint32(header[4:], uint32(headerSize)) // length of the header
		header[8] = byte(options.DistanceMethod)
		binary.BigEndian.PutUint32(header[9:], uint32(options.DimensionCount))
		header[13] = byte(options.Quantization)

		// Write the header to the new file
		if _, err := spanFile.file.WriteAt(header, 0); err != nil {
			panic(err)
		}
		spanFile.file.Sync()
	}

	// Determine the distance function
	var distanceFunc func([]float64, []float64) float64
	switch options.DistanceMethod {
	case Euclidean:
		distanceFunc = euclideanDistance
	case Cosine:
		distanceFunc = angularDistance
	default:
		panic("Unsupported distance method")
	}

	c := &Collection{
		CollectionOptions: options,
		spanfile:          spanFile,
		distance:          distanceFunc,
	}

	if useTree {
		lshTree := newLSHTree(c, 100, 5)
		c.index = lshTree
		c.lshTree = lshTree
	}

	// If the file exists, iterate through all existing documents and add them to the LSH table
	if fileExists {
		for id := range memFile.idOffsets {
			data, err := memFile.readRecord(id)
			if err != nil {
				continue
			}
			doc := c.decodeDocument(data, id)
			c.lshTree.addPoint(id, doc.Vector)
		}
	}

	return c
}

/*
GetAllIDs returns a sorted list of all document IDs in the collection.
*/
func (c *Collection) GetAllIDs() []uint64 {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var ids []uint64
	c.spanfile.IterateRecords(func(recordID string, sr *SpanReader) error {
		id, err := strconv.ParseUint(recordID, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
		return nil
	})

	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	return ids
}

/*
ComputeAverageDistance calculates the average distance between random pairs of documents in the collection.
It returns the average distance or 0.0 if there are fewer than two documents or if the sample size is non-positive.
*/
func (c *Collection) computeAverageDistance(samples int) float64 {
	if samples <= 0 {
		return 0.0
	}

	totalDistance := 0.0
	count := 0

	var ids []uint64
	c.spanfile.IterateRecords(func(recordID string, sr *SpanReader) error {
		id, err := strconv.ParseUint(recordID, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
		return nil
	})

	if len(ids) < 2 {
		return 0.0
	}

	// Perform up to 'samples' comparisons
	for i := 0; i < samples; i++ {
		// Randomly select two different IDs
		id1 := ids[rand.Intn(len(ids))]
		id2 := ids[rand.Intn(len(ids))]
		if id1 == id2 {
			continue // Ensure the points are different
		}

		// Retrieve the documents
		doc1, err1 := c.getDocument(id1)
		doc2, err2 := c.getDocument(id2)
		if err1 != nil || err2 != nil {
			continue
		}

		// Calculate the distance between the two vectors
		distance := c.distance(doc1.Vector, doc2.Vector)
		totalDistance += distance
		count++
	}

	if count == 0 {
		return 0.0
	}

	// Return the average distance
	return totalDistance / float64(count)
}

/*
Close closes the memfile associated with the collection.

Returns:
- An error if the memfile cannot be closed.
*/
func (c *Collection) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.spanfile != nil {
		err := c.spanfile.Close()
		if err != nil {
			return err
		}
		c.spanfile = nil
	}

	return nil
}

/*
AddDocument adds a new document to the collection with the specified ID, vector, and metadata.
It manages pivots and encodes the document for storage.
*/
func (c *Collection) AddDocument(id uint64, vector []float64, metadata []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if the vector size matches the expected dimensions
	if len(vector) != c.DimensionCount {
		log.Panicf("vector size does not match the expected number of dimensions: expected %d, got %d", c.DimensionCount, len(vector))
	}

	doc := &Document{
		Vector:   vector,
		Metadata: metadata,
		ID:       id,
	}

	// Encode the document
	encodedData := encodeDocument(doc, c.Quantization)

	// Write to spanfile
	dataStreams := []DataStream{
		{StreamID: 0, Data: metadata},
		{StreamID: 1, Data: encodedVector},
	}
	err := c.spanfile.WriteRecord(fmt.Sprintf("%d", id), dataStreams)
	if err != nil {
		log.Panicf("Failed to write record: %v", err)
	}

	// Add the document's vector to the LSH table
	c.lshTree.addPoint(id, vector)
}

/*
GetDocument retrieves a document from the collection by its ID.
It returns the document or an error if the document is not found.
*/
func (c *Collection) GetDocument(id uint64) (*Document, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.getDocument(id)
}

func (c *Collection) getDocument(id uint64) (*Document, error) {
	span, err := c.spanfile.ReadRecord(fmt.Sprintf("%d", id))
	if err != nil {
		return nil, err
	}

	metadata := span.DataStreams[0].Data
	vector := decodeVector(span.DataStreams[1].Data, c.DimensionCount, c.Quantization)

	return &Document{
		ID:       id,
		Vector:   vector,
		Metadata: metadata,
	}, nil
}

/*
UpdateDocument updates the metadata of an existing document in the collection.
It returns an error if the document is not found.
*/
func (c *Collection) UpdateDocument(id uint64, newMetadata []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	span, err := c.spanfile.ReadRecord(fmt.Sprintf("%d", id))
	if err != nil {
		return err
	}

	vector := decodeVector(span.DataStreams[1].Data, c.DimensionCount, c.Quantization)

	dataStreams := []DataStream{
		{StreamID: 0, Data: newMetadata},
		{StreamID: 1, Data: span.DataStreams[1].Data},
	}
	err = c.spanfile.WriteRecord(fmt.Sprintf("%d", id), dataStreams)
	if err != nil {
		return err
	}

	return nil
}

func (c *Collection) removeDocument(id uint64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Remove the document's vector from the LSH table
	doc, err := c.getDocument(id)
	if err == nil {
		c.lshTree.removePoint(id, doc.Vector)
	}
	return c.spanfile.RemoveRecord(fmt.Sprintf("%d", id))
}

// iterateDocuments applies a function to each document in the collection.
/*
func (c *Collection) iterateDocuments(fn func(doc *Document)) {
	for id := range c.memfile.idOffsets {
		data, err := c.memfile.readRecord(id)
		if err != nil {
			continue
		}
		doc := c.decodeDocument(data, id)
		fn(doc)
	}
}*/

type resultItem struct {
	SearchResult
	Priority float64
}

type resultPriorityQueue []*resultItem

func (pq resultPriorityQueue) Len() int { return len(pq) }

func (pq resultPriorityQueue) Less(i, j int) bool {
	return pq[i].Priority > pq[j].Priority // Max-heap based on distance
}

func (pq resultPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *resultPriorityQueue) Push(x interface{}) {
	item := x.(*resultItem)
	*pq = append(*pq, item)
}

func (pq *resultPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

/*
Search returns the search results, including the list of matching documents and the percentage of the database searched.
*/
func (c *Collection) Search(args SearchArgs) SearchResults {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// Default precision to "medium" if not set
	if args.Precision == "" {
		args.Precision = "medium"
	}

	resultsPQ := &resultPriorityQueue{}
	heap.Init(resultsPQ)
	pointsSearched := 0

	consider := func(docid uint64) int {
		doc, err := c.getDocument(docid)
		if err != nil {
			return StopSearch
		}

		pointsSearched++

		// Apply filter function if provided
		if args.Filter != nil && !args.Filter(doc.ID, doc.Metadata) {
			return PointIgnored
		}

		distance := c.distance(args.Vector, doc.Vector)

		if args.Radius > 0 && distance <= args.Radius {
			heap.Push(resultsPQ, &resultItem{
				SearchResult: SearchResult{ID: doc.ID, Metadata: doc.Metadata, Distance: distance},
				Priority:     distance,
			})
			return PointAccepted
		} else if args.K > 0 {
			if resultsPQ.Len() <= args.K {
				if resultsPQ.Len() < args.K || (*resultsPQ)[0].Priority > distance {
					heap.Push(resultsPQ, &resultItem{
						SearchResult: SearchResult{ID: doc.ID, Metadata: doc.Metadata, Distance: distance},
						Priority:     distance,
					})
					if resultsPQ.Len() > args.K {
						heap.Pop(resultsPQ)
					}
					return PointAccepted
				}
			}
		} else if args.K == 0 && args.Radius == 0 {
			// Exhaustive search: add all results
			heap.Push(resultsPQ, &resultItem{
				SearchResult: SearchResult{ID: doc.ID, Metadata: doc.Metadata, Distance: distance},
				Priority:     distance,
			})
			return PointAccepted
		}
		return PointChecked
	}

	if args.Radius == 0 && args.K == 0 || args.Precision == "exact" {
		// Exhaustive search: consider all documents
		c.spanfile.IterateRecords(func(recordID string, sr *SpanReader) error {
			id, err := strconv.ParseUint(recordID, 10, 64)
			if err == nil {
				consider(id)
			}
			return nil
		})
	} else {
		c.index.search(args.Vector, consider)
	}

	// Extract results from the priority queue
	results := make([]SearchResult, resultsPQ.Len())
	for i := len(results) - 1; i >= 0; i-- {
		results[i] = heap.Pop(resultsPQ).(*resultItem).SearchResult
	}

	return SearchResults{
		Results:         results,
		PercentSearched: float64(pointsSearched) / float64(len(c.memfile.idOffsets)) * 100,
	}
}

func encodeDocument(doc *Document, quantization int) []byte {
	dimensions := len(doc.Vector)

	vectorSize := getVectorSize(quantization, dimensions)

	docSize := vectorSize + 4 + len(doc.Metadata)
	data := make([]byte, docSize)

	// Encode the vector
	vectorOffset := 0
	for i, v := range doc.Vector {
		quantizedValue := quantize(v, quantization)
		switch quantization {
		case 4:
			if i%2 == 0 {
				data[vectorOffset+i/2] = byte(quantizedValue << 4)
			} else {
				data[vectorOffset+i/2] |= byte(quantizedValue & 0x0F)
			}
		case 8:
			data[vectorOffset+i] = byte(quantizedValue)
		case 16:
			binary.BigEndian.PutUint16(data[vectorOffset+i*2:], uint16(quantizedValue))
		case 32:
			binary.BigEndian.PutUint32(data[vectorOffset+i*4:], uint32(quantizedValue))
		case 64:
			binary.BigEndian.PutUint64(data[vectorOffset+i*8:], quantizedValue)
		}
	}

	// Encode the metadata length after the vector
	metadataLengthOffset := vectorOffset + vectorSize
	binary.BigEndian.PutUint32(data[metadataLengthOffset:], uint32(len(doc.Metadata)))

	// Encode the metadata
	metadataOffset := metadataLengthOffset + 4
	copy(data[metadataOffset:], doc.Metadata)

	return data
}

func (c *Collection) decodeDocument(data []byte, id uint64) *Document {
	dimensions := c.DimensionCount
	quantization := c.Quantization
	vector := make([]float64, dimensions)
	vectorOffset := 0

	for i := range vector {
		var quantizedValue uint64
		switch quantization {
		case 4:
			if i%2 == 0 {
				quantizedValue = uint64(data[vectorOffset+i/2] >> 4)
			} else {
				quantizedValue = uint64(data[vectorOffset+i/2] & 0x0F)
			}
		case 8:
			quantizedValue = uint64(data[vectorOffset+i])
		case 16:
			quantizedValue = uint64(binary.BigEndian.Uint16(data[vectorOffset+i*2:]))
		case 32:
			quantizedValue = uint64(binary.BigEndian.Uint32(data[vectorOffset+i*4:]))
		case 64:
			quantizedValue = binary.BigEndian.Uint64(data[vectorOffset+i*8:])
		}

		vector[i] = dequantize(quantizedValue, quantization)
	}

	// Decode the metadata length after the vector
	metadataLengthOffset := vectorOffset + getVectorSize(quantization, dimensions)
	metadataLength := binary.BigEndian.Uint32(data[metadataLengthOffset:])

	// Decode the metadata
	metadataOffset := metadataLengthOffset + 4

	return &Document{
		ID:       id,
		Vector:   vector,
		Metadata: data[metadataOffset : metadataOffset+int(metadataLength)],
	}
}

func getVectorSize(quantization int, dimensions int) int {
	switch quantization {
	case 4:
		return (dimensions + 1) / 2
	case 8:
		return dimensions
	case 16:
		return dimensions * 2
	case 32:
		return dimensions * 4
	case 64:
		return dimensions * 8
	default:
		panic("Unsupported quantization level")
	}
}
func euclideanDistance(vec1, vec2 []float64) float64 {
	sum := 0.0
	for i := range vec1 {
		diff := vec1[i] - vec2[i]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

func angularDistance(vec1, vec2 []float64) float64 {
	dotProduct, magnitude1, magnitude2 := 0.0, 0.0, 0.0
	for i := range vec1 {
		dotProduct += vec1[i] * vec2[i]
		magnitude1 += vec1[i] * vec1[i]
		magnitude2 += vec2[i] * vec2[i]
	}
	if magnitude1 == 0 || magnitude2 == 0 {
		return 1.0 // Return max distance if one vector is zero
	}
	return math.Acos(dotProduct/(math.Sqrt(magnitude1)*math.Sqrt(magnitude2))) / math.Pi
}

type searchIndex interface {
	addPoint(docid uint64, vector []float64)
	removePoint(docid uint64, vector []float64)
	search(vector []float64, callback func(docid uint64) int)
}
