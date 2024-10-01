package main

import (
	"container/heap"
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	"sort"
	"sync"
)

type DistanceIndex struct {
	distance float64
	index    uint64
}

type ApproxHeap []DistanceIndex

func (h ApproxHeap) Len() int           { return len(h) }
func (h ApproxHeap) Less(i, j int) bool { return h[i].distance < h[j].distance }
func (h ApproxHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *ApproxHeap) Push(x interface{}) {
	*h = append(*h, x.(DistanceIndex))
}

func (h *ApproxHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type ResultHeap []SearchResult

func (h ResultHeap) Len() int           { return len(h) }
func (h ResultHeap) Less(i, j int) bool { return h[i].Distance > h[j].Distance }
func (h ResultHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *ResultHeap) Push(x interface{}) {
	*h = append(*h, x.(SearchResult))
}

func (h *ResultHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

const (
	Euclidean = iota
	Cosine
)

type Collection struct {
	CollectionOptions
	memfile       *memfile
	pivotsManager PivotsManager
	mutex         sync.Mutex
}

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
	doc := decodeDocument(id, data, c.DimensionCount)
	return doc, nil
}

func (c *Collection) getRandomID() (uint64, error) {

	if len(c.memfile.idOffsets) == 0 {
		return 0, errors.New("no documents in the collection")
	}

	// Create a slice of IDs
	ids := make([]uint64, 0, len(c.memfile.idOffsets))
	for id := range c.memfile.idOffsets {
		ids = append(ids, id)
	}

	// Select a random ID
	randomIndex := rand.Intn(len(ids))
	return ids[randomIndex], nil
}

// iterateDocuments applies a function to each document in the collection.
func (c *Collection) iterateDocuments(fn func(doc *Document)) {
	for id := range c.memfile.idOffsets {
		data, err := c.memfile.readRecord(id)
		if err != nil {
			continue
		}
		doc := decodeDocument(id, data, c.DimensionCount)
		fn(doc)
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

func (c *Collection) Search(args SearchArgs) SearchResults {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if args.MaxCount > 0 {
		return c.searchNearestNeighbours(args)
	} else if args.Radius > 0 {
		return c.searchRadius(args)
	}

	return SearchResults{}
}

func (c *Collection) searchRadius(args SearchArgs) SearchResults {
	results := []SearchResult{}
	pointsSearched := 0

	// Calculate distances from the target to each pivot
	// Calculate distances to pivots
	distances := make([]float64, len(c.pivotsManager.pivots))
	for i, pivot := range c.pivotsManager.pivots {
		distances[i] = c.pivotsManager.distanceFn(args.Vector, pivot.Vector)
	}
	for i, pivot := range c.pivotsManager.pivots {
		dist := c.pivotsManager.distanceFn(args.Vector, pivot.Vector)
		pointsSearched++
		if dist <= args.Radius {
			results = append(results, SearchResult{ID: pivot.ID, Metadata: pivot.Metadata, Distance: dist})
		}
		distances[i] = dist
	}

	// Iterate over all points
	for id := range c.memfile.idOffsets {
		if c.pivotsManager.isPivot(id) {
			continue
		}

		minDistance := c.pivotsManager.approxDistance(args.Vector, id)

		if minDistance <= args.Radius {
			data, err := c.memfile.readRecord(id)
			if err != nil {
				continue
			}

			doc := decodeDocument(data, c.DimensionCount)
			actualDistance := c.pivotsManager.distanceFn(args.Vector, doc.Vector)
			pointsSearched++
			if actualDistance <= args.Radius {
				results = append(results, SearchResult{ID: doc.ID, Metadata: doc.Metadata, Distance: actualDistance})
			}
		}
	}

	// Sort results by distance
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	return SearchResults{
		Results:         results,
		PercentSearched: float64(pointsSearched) / float64(len(c.memfile.idOffsets)) * 100,
	}
}

func (c *Collection) searchNearestNeighbours(args SearchArgs) SearchResults {
	if args.MaxCount <= 0 {
		return SearchResults{}
	}

	pointsSearched := 0

	// Initialize heaps
	approxHeap := &ApproxHeap{}
	heap.Init(approxHeap)

	resultsHeap := &ResultHeap{}
	heap.Init(resultsHeap)

	// Calculate distances to pivots
	distances := make([]float64, len(c.pivotsManager.pivots))
	for i, pivot := range c.pivotsManager.pivots {
		distances[i] = c.pivotsManager.distanceFn(args.Vector, pivot.Vector)
		pointsSearched++
	}

	// Populate the approximate heap
	for id := range c.memfile.idOffsets {
		if c.pivotsManager.isPivot(id) {
			continue
		}

		minDistance := c.pivotsManager.approxDistance(args.Vector, id)
		heap.Push(approxHeap, DistanceIndex{distance: minDistance, index: id})
	}

	// Process the approximate heap
	for approxHeap.Len() > 0 {
		item := heap.Pop(approxHeap).(DistanceIndex)

		if resultsHeap.Len() == args.MaxCount && item.distance >= (*resultsHeap)[0].Distance {
			break
		}

		data, err := c.memfile.readRecord(item.index)
		if err != nil {
			continue
		}

		pointsSearched++

		doc := decodeDocument(data, c.DimensionCount)
		distance := c.pivotsManager.distanceFn(args.Vector, doc.Vector)

		if resultsHeap.Len() < args.MaxCount {
			heap.Push(resultsHeap, SearchResult{ID: doc.ID, Metadata: doc.Metadata, Distance: distance})
		} else if distance < (*resultsHeap)[0].Distance {
			heap.Pop(resultsHeap)
			heap.Push(resultsHeap, SearchResult{ID: doc.ID, Metadata: doc.Metadata, Distance: distance})
		}
	}

	// Collect results
	results := make([]SearchResult, resultsHeap.Len())
	for i := len(results) - 1; i >= 0; i-- {
		results[i] = heap.Pop(resultsHeap).(SearchResult)
	}

	return SearchResults{
		Results:         results,
		PercentSearched: float64(pointsSearched) / float64(len(c.memfile.idOffsets)) * 100,
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

func (c *Collection) AddDocument(id uint64, vector []float64, metadata []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	doc := &Document{
		Vector:   vector,
		Metadata: metadata,
	}

	numDocs := len(c.memfile.idOffsets)

	// Calculate the desired number of pivots using a logarithmic function
	desiredPivots := int(math.Log2(float64(numDocs+1) - 7))

	// Manage pivots
	c.pivotsManager.ensurePivots(c, desiredPivots)

	// Encode the document
	encodedData := encodeDocument(doc)

	// Add or update the document in the memfile
	c.memfile.addRecord(id, encodedData)

	c.pivotsManager.pointAdded(doc)
}

func (c *Collection) removeDocument(id uint64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.pivotsManager.pointRemoved(id)

	// Remove the document from the memfile
	return c.memfile.deleteRecord(id)
}

func (c *Collection) UpdateDocument(id uint64, newMetadata []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Read the existing record
	data, err := c.memfile.readRecord(id)
	if err != nil {
		return err
	}

	// Decode the existing document
	doc := decodeDocument(data, c.DimensionCount)

	// Update the metadata
	doc.Metadata = newMetadata

	// Encode the updated document
	encodedData := encodeDocument(doc)

	// Update the document in the memfile
	c.memfile.addRecord(id, encodedData)

	return nil
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

// 4 bytes: version
// 4 bytes: length of the header
// 1 byte: distance method
// 4 bytes: number of dimensions
const headerSize = 13

func NewCollection(options CollectionOptions) *Collection {
	distanceFn := euclideanDistance
	if options.DistanceMethod == Cosine {
		distanceFn = cosineDistance
	}
	c := &Collection{
		CollectionOptions: options,
		pivotsManager:     *newPivotsManager(distanceFn), // Use newPivotsManager
	}

	header := make([]byte, headerSize)

	// Fill in the header
	binary.BigEndian.PutUint32(header[0:], 1)                  // version
	binary.BigEndian.PutUint32(header[4:], uint32(headerSize)) // length of the header
	header[8] = byte(options.DistanceMethod)
	binary.BigEndian.PutUint32(header[9:], uint32(options.DimensionCount))

	var err error
	c.memfile, err = createMemFile(c.Name, header)
	if err != nil {
		panic(err)
	}

	return c
}

func encodeDocument(doc *Document) []byte {
	// n bytes: vector
	// 4 bytes: length of metadata
	// n bytes: metadata

	dimensions := len(doc.Vector)

	docSize := dimensions*8 + 4 + len(doc.Metadata)
	data := make([]byte, docSize)


	// Encode the floating point vector to the data slice
	vectorOffset := 0
	for i, v := range doc.Vector {
		binary.BigEndian.PutUint64(data[vectorOffset+i*8:], math.Float64bits(v))
	}

	// Encode the metadata length after the vector
	metadataLengthOffset := 8 + dimensions*8
	binary.BigEndian.PutUint32(data[metadataLengthOffset:], uint32(len(doc.Metadata)))

	// Encode the metadata
	metadataOffset := metadataLengthOffset + 4
	copy(data[metadataOffset:], doc.Metadata)

	return data
}

func decodeDocument(id uint64, data []byte, dimensions int) *Document {

	// Use the passed dimensions to decode the vector
	vector := make([]float64, dimensions)
	vectorOffset := 0
	for i := range vector {
		vector[i] = math.Float64frombits(binary.BigEndian.Uint64(data[vectorOffset+i*8:]))
	}

	// Decode the metadata length after the vector
	metadataLengthOffset := vectorOffset + dimensions*8
	metadataLength := binary.BigEndian.Uint32(data[metadataLengthOffset:])

	// Decode the metadata
	metadataOffset := metadataLengthOffset + 4
	metadata := make([]byte, metadataLength)
	copy(metadata, data[metadataOffset:])

	return &Document{
		ID:       id,
		Vector:   vector,
		Metadata: metadata,
	}
}
