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

const minBucketsToSearch = 2

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

	// Overwrite any existing database
	Create bool
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
	storageSize := c.memfile.Len()

	// Calculate the average distance
	averageDistance := c.computeAverageDistance(100) // Example: use 100 samples

	// Determine the distance method as a string
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
	Offset int
	Limit  int
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
	memfile  *memfile
	lshTable *lshTable // Add this line
	mutex    sync.Mutex
	distance func([]float64, []float64) float64 // Add this line
}

/*
NewCollection creates a new Collection with the specified options.
It initializes the collection's memory file and pivots manager.
*/
func NewCollection(options CollectionOptions) *Collection {

	if options.Create {
		// Remove the existing file if it exists
		if _, err := os.Stat(options.Name); err == nil {
			os.Remove(options.Name)
		}
	}

	// Define the header size and create a buffer to read it
	header := make([]byte, headerSize)

	// Check if the file exists
	fileExists := false
	if _, err := os.Stat(options.Name); err == nil {
		fileExists = true
	}

	// Open or create the memory-mapped file
	var err error
	var memFile *memfile
	if fileExists {
		// Open the existing file and read the header
		memFile, err = createMemFile(options.Name, header)
		if err != nil {
			panic(err)
		}

		// Read the header from the file
		if _, err := memFile.ReadAt(header, 0); err != nil {
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

		// Create a new file and write the header
		memFile, err = createMemFile(options.Name, header)
		if err != nil {
			panic(err)
		}

	}

	// Determine the distance function
	var distanceFunc func([]float64, []float64) float64
	switch options.DistanceMethod {
	case Euclidean:
		distanceFunc = euclideanDistance
	case Cosine:
		distanceFunc = cosineDistance
	default:
		panic("Unsupported distance method")
	}
	lshTable := newLSHTable(10, options.DimensionCount, 4.0) // Example parameters

	c := &Collection{
		CollectionOptions: options,
		memfile:           memFile,
		lshTable:          lshTable,
		distance:          distanceFunc,
	}

	// If the file exists, iterate through all existing documents and add them to the LSH table
	if fileExists {
		for id := range memFile.idOffsets {
			data, err := memFile.readRecord(id)
			if err != nil {
				continue
			}
			doc := c.decodeDocument(data, id)
			c.lshTable.addPoint(id, doc.Vector)
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

	ids := make([]uint64, 0, len(c.memfile.idOffsets))
	for id := range c.memfile.idOffsets {
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	return ids
}

/*
ComputeAverageDistance calculates the average distance between random pairs of documents in the collection.
It returns the average distance or 0.0 if there are fewer than two documents or if the sample size is non-positive.
*/
func (c *Collection) computeAverageDistance(samples int) float64 {
	if len(c.memfile.idOffsets) < 2 || samples <= 0 {
		return 0.0
	}

	totalDistance := 0.0
	count := 0

	// Create a slice of all document IDs
	ids := make([]uint64, 0, len(c.memfile.idOffsets))
	for id := range c.memfile.idOffsets {
		ids = append(ids, id)
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

	if c.memfile != nil {
		err := c.memfile.Sync()
		if err != nil {
			return err
		}
		err = c.memfile.Close()
		if err != nil {
			return err
		}
		c.memfile = nil
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

	// Add or update the document in the memfile
	c.memfile.addRecord(id, encodedData)

	// Add the document's vector to the LSH table
	c.lshTable.addPoint(id, vector)
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
	// Read the record from the memfile
	data, err := c.memfile.readRecord(id)
	if err != nil {
		return nil, err
	}

	// Decode the document
	doc := c.decodeDocument(data, id)
	return doc, nil
}

/*
UpdateDocument updates the metadata of an existing document in the collection.
It returns an error if the document is not found.
*/
func (c *Collection) UpdateDocument(id uint64, newMetadata []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Read the existing record
	data, err := c.memfile.readRecord(id)
	if err != nil {
		return err
	}

	// Decode the existing document
	doc := c.decodeDocument(data, id)

	// Update the metadata
	doc.Metadata = newMetadata

	// Encode the updated document
	encodedData := encodeDocument(doc, c.Quantization)

	// Update the document in the memfile
	c.memfile.addRecord(id, encodedData)

	return nil
}

func (c *Collection) removeDocument(id uint64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Remove the document's vector from the LSH table
	doc, err := c.getDocument(id)
	if err == nil {
		c.lshTable.removePoint(id, doc.Vector)
	}
	return c.memfile.deleteRecord(id)
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
	if args.K == 0 && args.Radius == 0 {
		// Collect all document IDs
		ids := make([]uint64, 0, len(c.memfile.idOffsets))
		for id := range c.memfile.idOffsets {
			ids = append(ids, id)
		}

		// Sort IDs
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

		// Apply offset and limit
		start := args.Offset
		if start > len(ids) {
			start = len(ids)
		}
		end := start + args.Limit
		if end > len(ids) {
			end = len(ids)
		}

		// Collect results
		results := make([]SearchResult, 0, end-start)
		for _, id := range ids[start:end] {
			doc, err := c.getDocument(id)
			if err != nil {
				continue
			}
			results = append(results, SearchResult{
				ID:       doc.ID,
				Metadata: doc.Metadata,
				Distance: 0, // Distance is not applicable here
			})
		}

		return SearchResults{
			Results:         results,
			PercentSearched: 100.0, // All records are considered
		}
	}

	if len(args.Vector) != c.DimensionCount {
		log.Panicf("vector size does not match the expected number of dimensions: expected %d, got %d", c.DimensionCount, len(args.Vector))
	}

	if args.K > 0 {
		return c.searchNearestNeighbours(args)
	} else if args.Radius > 0 {
		return c.searchRadius(args)
	}

	return SearchResults{}
}

func (c *Collection) searchRadius(args SearchArgs) SearchResults {
	results := []SearchResult{}
	pointsSearched := 0
	bucketsSearched := 0 // Declare bucketsSearched here

	// Use LSH to get a priority queue of candidate buckets
	pq := c.lshTable.multiprobeQuery(args.Vector)

	// Process candidates from the priority queue
	for pq.Len() > 0 {
		item := heap.Pop(pq).(*priorityItem)
		bucketKey := item.Key
		numAdded := 0

		// Process each document in the current bucket
		for _, id := range c.lshTable.Buckets[bucketKey] {
			data, err := c.memfile.readRecord(id)
			if err != nil {
				continue
			}

			pointsSearched++

			doc := c.decodeDocument(data, id)

			// Apply filter function if provided
			if args.Filter != nil && !args.Filter(doc.ID, doc.Metadata) {
				continue
			}

			// Calculate the distance between the search vector and the document vector
			distance := c.distance(args.Vector, doc.Vector) // Use the configured distance function

			// Check if the distance is within the specified radius
			if distance <= args.Radius {
				results = append(results, SearchResult{ID: doc.ID, Metadata: doc.Metadata, Distance: distance})
				numAdded++
			}
		}

		bucketsSearched++

		// Stop if no points were added from this bucket and we've searched at least the minimum number of buckets
		if numAdded == 0 && bucketsSearched >= minBucketsToSearch {
			break
		}
	}

	return SearchResults{
		Results:         results,
		PercentSearched: float64(pointsSearched) / float64(len(c.memfile.idOffsets)) * 100,
	}
}

func (c *Collection) searchNearestNeighbours(args SearchArgs) SearchResults {
	if args.K <= 0 {
		return SearchResults{}
	}

	// Use LSH to get a priority queue of candidate buckets
	pq := c.lshTable.multiprobeQuery(args.Vector)

	resultsPQ := &resultPriorityQueue{}
	heap.Init(resultsPQ)
	pointsSearched := 0
	bucketsSearched := 0

	// Process candidates from the priority queue
	for pq.Len() > 0 {
		item := heap.Pop(pq).(*priorityItem)
		bucketKey := item.Key
		numAdded := 0
		// Process each document in the current bucket
		for _, id := range c.lshTable.Buckets[bucketKey] {
			data, err := c.memfile.readRecord(id)
			if err != nil {
				continue
			}

			pointsSearched++

			doc := c.decodeDocument(data, id)

			// Apply filter function if provided
			if args.Filter != nil && !args.Filter(doc.ID, doc.Metadata) {
				continue
			}

			distance := c.distance(args.Vector, doc.Vector) // Use the configured distance function

			// Add to results priority queue if closer than the current farthest
			if resultsPQ.Len() < args.K || distance < (*resultsPQ)[0].Priority {
				numAdded++
				heap.Push(resultsPQ, &resultItem{
					SearchResult: SearchResult{ID: doc.ID, Metadata: doc.Metadata, Distance: distance},
					Priority:     distance,
				})
				if resultsPQ.Len() > args.K {
					heap.Pop(resultsPQ) // Remove the farthest point
				}
			}
		}
		bucketsSearched++

		// Stop if no points were added from this bucket and we've searched at least the minimum number of buckets
		if numAdded == 0 && bucketsSearched >= minBucketsToSearch {
			break
		}
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

// Helper function to compare two vectors for equality
func equalVectors(vec1, vec2 []float64) bool {
	if len(vec1) != len(vec2) {
		return false
	}
	for i := range vec1 {
		if vec1[i] != vec2[i] {
			return false
		}
	}
	return true
}

func euclideanDistance(vec1, vec2 []float64) float64 {
	sum := 0.0
	for i := range vec1 {
		diff := vec1[i] - vec2[i]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

func cosineDistance(vec1, vec2 []float64) float64 {
	dotProduct := 0.0
	magnitude1 := 0.0
	magnitude2 := 0.0
	for i := range vec1 {
		dotProduct += vec1[i] * vec2[i]
		magnitude1 += vec1[i] * vec1[i]
		magnitude2 += vec2[i] * vec2[i]
	}
	if magnitude1 == 0 || magnitude2 == 0 {
		return 1.0 // Return max distance if one vector is zero
	}
	return 1.0 - (dotProduct / (math.Sqrt(magnitude1) * math.Sqrt(magnitude2)))
}
