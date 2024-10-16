package syzgydb

import (
	"container/heap"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/smhanov/syzgydb/query"
	"github.com/smhanov/syzgydb/replication"
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
	Name string `json:"name"`

	// DistanceMethod specifies the method used to calculate distances between vectors.
	// It can be either Euclidean or Cosine.
	DistanceMethod int `json:"distance_method"`

	// DimensionCount is the number of dimensions for each vector in the collection.
	DimensionCount int `json:"dimension_count"`

	// Quantization specifies the bit-level quantization for storing vectors.
	// Supported values are 4, 8, 16, 32, and 64, with 64 as the default.
	Quantization int `json:"quantization"`

	// FileMode specifies the mode for opening the memfile.
	FileMode FileMode `json:"file_mode"`

	NodeID    uint64                `json:"node_id"`
	Timestamp replication.Timestamp `json:"timestamp"`
}

type ICollection interface {
	AddDocument(id uint64, vector []float64, metadata []byte) error
	Close() error
	ComputeStats() CollectionStats
	GetAllIDs() []uint64
	GetDocument(id uint64) (*Document, error)
	GetDocumentCount() int
	GetOptions() CollectionOptions
	RemoveDocument(id uint64) error
	Search(args SearchArgs) SearchResults
	UpdateDocument(id uint64, newMetadata []byte) error
}

// GetDocumentCount returns the total number of documents in the collection.
//
// This method provides a quick way to determine the size of the collection
// by returning the count of document IDs stored in the memfile.
func (c *Collection) GetDocumentCount() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Use spanfile to count records
	_, numRecords := c.spanfile.GetStats()
	return numRecords
}

/*
ComputeStats gathers and returns statistics about the collection.
It returns a CollectionStats object filled with the relevant statistics.
*/
func (c *Collection) ComputeStats() CollectionStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Calculate the storage size
	storageSize, documentCount := c.spanfile.GetStats()

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
	mutex    sync.RWMutex // Change from sync.Mutex to sync.RWMutex
	distance func([]float64, []float64) float64
}

// BuildFilter compiles the query into a filter function that can be used with SearchArgs.
func BuildFilter(queryIn string) (FilterFn, error) {
	fn, err := query.FilterFunctionFromQuery(queryIn)
	if err != nil {
		return nil, err
	}

	return func(id uint64, metadata []byte) bool {
		pass, err := fn(metadata)
		if err != nil {
			log.Printf("Error applying filter to document %d: %v", id, err)
			return false
		}
		return pass
	}, nil
}

/*
NewCollection creates a new Collection with the specified options.
It initializes the collection's memory file and pivots manager.
*/
func NewCollection(options CollectionOptions) (*Collection, error) {
	// Check if the file exists
	fileExists := false
	if options.FileMode != CreateAndOverwrite {
		if fileInfo, err := os.Stat(options.Name); err == nil {
			if fileInfo.Size() > 0 {
				fileExists = true
			}
		}
	}

	if options.Timestamp.IsZero() {
		options.Timestamp = replication.Now()
	}

	// Open or create the memory-mapped file with the specified mode
	spanFile, err := OpenFile(options.Name, options.FileMode)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}

	if fileExists {
		// Read the header to get the collection options
		header, err := spanFile.ReadRecord("")
		if err != nil {
			return nil, fmt.Errorf("failed to read header: %v", err)
		}

		if len(header.DataStreams) == 0 {
			return nil, fmt.Errorf("no datastreams found in header")
		}

		// Decode the collection options from the header
		err = json.Unmarshal(header.DataStreams[0].Data, &options)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal options: %v", err)
		}
	} else {
		if options.Quantization == 0 {
			options.Quantization = 64
		}

		// Write the options to a JSON string and save it to the spanFile as record with id ""
		// as datastream 0
		optionsData, err := json.Marshal(options)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal options: %v", err)
		}

		dataStreams := []DataStream{
			{StreamID: 0, Data: optionsData},
		}
		err = spanFile.WriteRecord("", dataStreams, options.NodeID, options.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to write options: %v", err)
		}
	}

	// Determine the distance function
	var distanceFunc func([]float64, []float64) float64
	switch options.DistanceMethod {
	case Euclidean:
		distanceFunc = euclideanDistance
	case Cosine:
		distanceFunc = angularDistance
	default:
		return nil, fmt.Errorf("unsupported distance method")
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
		err := c.spanfile.IterateRecords(func(recordID string, sr *SpanReader) error {
			id, err := strconv.ParseUint(recordID, 10, 64)
			if err != nil {
				return nil
			}
			doc := c.decodeDocument(sr, id)
			c.lshTree.addPoint(id, doc.Vector)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to iterate records: %v", err)
		}
	}

	return c, nil
}

// GetOptions returns the collection options used to create the collection.
func (c *Collection) GetOptions() CollectionOptions {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.CollectionOptions
}

func (c *Collection) GetName() string {
	name := c.CollectionOptions.Name
	base := name[strings.LastIndex(name, "/")+1:]
	return base[:strings.LastIndex(base, ".")]
}

/*
GetAllIDs returns a sorted list of all document IDs in the collection.
*/
func (c *Collection) GetAllIDs() []uint64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

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

func (c *Collection) getLatestVectorClock() *replication.VectorClock {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.spanfile.getLatestVectorClock().Clone()
}

/*
ComputeAverageDistance calculates the average distance between random pairs of documents in the collection.
It returns the average distance or 0.0 if there are fewer than two documents or if the sample size is non-positive.
*/
func (c *Collection) computeAverageDistance(samples int) float64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

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

// AddRecordDirect adds a record directly to the collection with the specified ID, metadata, datastreams, and timestamp.
func (c *Collection) AddRecordDirect(id uint64, dataStreams []DataStream, nodeID uint64, timestamp replication.Timestamp) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Find the vector datastream (assuming it's always StreamID 1)
	var vectorData []byte
	for _, ds := range dataStreams {
		if ds.StreamID == 1 {
			vectorData = ds.Data
			break
		}
	}

	if vectorData == nil {
		return fmt.Errorf("vector data not found in datastreams")
	}

	// Decode the vector
	vector := decodeVector(vectorData, c.DimensionCount, c.Quantization)

	// Use the common function to add the record
	return c.addRecordCommon(id, vector, dataStreams, nodeID, timestamp)
}

/*
AddDocument adds a new document to the collection with the specified ID, vector, and metadata.
It manages pivots and encodes the document for storage.
*/
func (c *Collection) AddDocument(id uint64, vector []float64, metadata []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if the vector size matches the expected dimensions
	if len(vector) != c.DimensionCount {
		return fmt.Errorf("vector size does not match the expected number of dimensions: expected %d, got %d", c.DimensionCount, len(vector))
	}

	// Encode the vector
	encodedVector := encodeVector(vector, c.Quantization)

	// Prepare datastreams
	dataStreams := []DataStream{
		{StreamID: 0, Data: metadata},
		{StreamID: 1, Data: encodedVector},
	}

	// Use the common function to add the record
	return c.addRecordCommon(id, vector, dataStreams, 0, c.spanfile.NextTimestamp())
}

func (c *Collection) addRecordCommon(id uint64, vector []float64, dataStreams []DataStream, nodeID uint64, timestamp replication.Timestamp) error {
	// Write to spanfile
	err := c.spanfile.WriteRecord(fmt.Sprintf("%d", id), dataStreams, nodeID, timestamp)
	if err != nil {
		return fmt.Errorf("failed to write record: %v", err)
	}

	// Add the document's vector to the LSH table
	c.lshTree.addPoint(id, vector)

	return nil
}

func encodeVector(vector []float64, quantization int) []byte {
	doc := &Document{Vector: vector}
	return encodeDocument(doc, quantization)
}

/*
GetDocument retrieves a document from the collection by its ID.
It returns the document or an error if the document is not found.
*/
func (c *Collection) GetDocument(id uint64) (*Document, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

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

	// Get the next timestamp
	timestamp := c.spanfile.NextTimestamp()

	dataStreams := []DataStream{
		{StreamID: 0, Data: newMetadata},
		{StreamID: 1, Data: span.DataStreams[1].Data},
	}
	err = c.spanfile.WriteRecord(fmt.Sprintf("%d", id), dataStreams, 0, timestamp)
	if err != nil {
		return err
	}

	return nil
}

func (c *Collection) removeDocumentDirect(id uint64, nodeID uint64, timeStamp replication.Timestamp) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	idStr := fmt.Sprintf("%d", id)
	// Remove the document's vector from the LSH table
	doc, err := c.getDocument(id)
	if err == nil {
		c.lshTree.removePoint(id, doc.Vector)
	}
	return c.spanfile.RemoveRecord(idStr, nodeID, timeStamp)
}

func (c *Collection) RemoveDocument(id uint64) error {
	return c.removeDocumentDirect(id, 0, c.spanfile.NextTimestamp())
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
	c.mutex.RLock()
	defer c.mutex.RUnlock()
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
			// heuristic: we want the index to stop looking only if we don't get any better ones
			// for a while.
			if (*resultsPQ)[0].Priority >= distance {
				return PointAccepted
			}
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

	_, numRecords := c.spanfile.GetStats()

	return SearchResults{
		Results:         results,
		PercentSearched: float64(pointsSearched) / float64(numRecords) * 100,
	}
}

func encodeDocument(doc *Document, quantization int) []byte {
	dimensions := len(doc.Vector)

	vectorSize := getVectorSize(quantization, dimensions)

	docSize := vectorSize
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

	return data
}
func (c *Collection) decodeDocument(sr *SpanReader, id uint64) *Document {
	data, err := sr.getStream(1)
	if err != nil {
		log.Panicf("Failed to read vector data of doc %d: %v", id, err)
	}

	vector := decodeVector(data, c.DimensionCount, c.Quantization)

	metadata, err := sr.getStream(0)
	if err != nil {
		log.Panic("Failed to read metadata")
	}

	metadataCopy := make([]byte, len(metadata))
	copy(metadataCopy, metadata)

	return &Document{
		ID:       id,
		Vector:   vector,
		Metadata: metadataCopy,
	}
}

func decodeVector(data []byte, dimensions int, quantization int) []float64 {
	vector := make([]float64, dimensions)

	for i := range vector {
		var quantizedValue uint64
		switch quantization {
		case 4:
			if i%2 == 0 {
				quantizedValue = uint64(data[i/2] >> 4)
			} else {
				quantizedValue = uint64(data[i/2] & 0x0F)
			}
		case 8:
			quantizedValue = uint64(data[i])
		case 16:
			quantizedValue = uint64(binary.BigEndian.Uint16(data[i*2:]))
		case 32:
			quantizedValue = uint64(binary.BigEndian.Uint32(data[i*4:]))
		case 64:
			quantizedValue = binary.BigEndian.Uint64(data[i*8:])
		}

		vector[i] = dequantize(quantizedValue, quantization)
	}

	return vector
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

// EncodeVector encodes a float64 vector to a byte slice
func EncodeVector(vector []float64, quantization int) []byte {
	return encodeVector(vector, quantization)
}

// DecodeVector decodes a byte slice to a float64 vector
func DecodeVector(data []byte, dimensions int, quantization int) []float64 {
	return decodeVector(data, dimensions, quantization)
}

type searchIndex interface {
	addPoint(docid uint64, vector []float64)
	removePoint(docid uint64, vector []float64)
	search(vector []float64, callback func(docid uint64) int)
}
