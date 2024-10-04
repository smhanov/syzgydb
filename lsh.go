package syzgydb

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand"
)

type hashFunction struct {
	RandomVector []float64
	Offset       float64
	W            float64 // Bucket width
}

type priorityItem struct {
	Key      string
	Priority float64
}

type priorityQueue []*priorityItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].Priority < pq[j].Priority
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue) Push(x interface{}) {
	item := x.(*priorityItem)
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

func newHashFunction(dim int, w float64) *hashFunction {
	randVec := make([]float64, dim)
	for i := range randVec {
		randVec[i] = rand.NormFloat64()
	}
	offset := rand.Float64() * w
	return &hashFunction{
		RandomVector: randVec,
		Offset:       offset,
		W:            w,
	}
}

func (hf *hashFunction) hash(vector []float64) int {
	dotProduct := 0.0
	for i, v := range vector {
		dotProduct += v * hf.RandomVector[i]
	}
	return int(math.Floor((dotProduct + hf.Offset) / hf.W))
}

type lshTable struct {
	HashFunctions []*hashFunction
	Buckets       map[string][]uint64
}

func newLSHTable(numHashFunctions, dim int, w float64) *lshTable {
	hashFuncs := make([]*hashFunction, numHashFunctions)
	for i := 0; i < numHashFunctions; i++ {
		hashFuncs[i] = newHashFunction(dim, w)
	}
	return &lshTable{
		HashFunctions: hashFuncs,
		Buckets:       make(map[string][]uint64), // Initialize the map here
	}
}

func (table *lshTable) hash(vector []float64) string {
	hashes := make([]int, len(table.HashFunctions))
	for i, hf := range table.HashFunctions {
		hashes[i] = hf.hash(vector)
	}
	return fmt.Sprintf("%v", hashes)
}

func (table *lshTable) addPoint(docid uint64, vector []float64) {
	key := table.hash(vector)
	table.Buckets[key] = append(table.Buckets[key], docid)
}

func (table *lshTable) removePoint(docid uint64, vector []float64) {
	key := table.hash(vector)
	// search for the docid in the bucket and remove it
	for i, id := range table.Buckets[key] {
		if id == docid {
			table.Buckets[key] = append(table.Buckets[key][:i], table.Buckets[key][i+1:]...)
			break
		}
	}
}

func (table *lshTable) search(vector []float64, callback func(docid uint64) bool) {
	pq := table.multiprobeQuery(vector)
	for pq.Len() > 0 {
		item := heap.Pop(pq).(*priorityItem)
		bucketKey := item.Key
		for _, id := range table.Buckets[bucketKey] {
			if callback(id) {
				return
			}
		}
	}
}

func (table *lshTable) multiprobeQuery(vector []float64) *priorityQueue {
	// Hash the input vector
	initialKey := table.hash(vector)

	// Function to calculate Euclidean distance between two hash keys
	calculateDistance := func(key1, key2 string) float64 {
		var dist float64
		var hash1, hash2 []int
		fmt.Sscanf(key1, "%v", &hash1)
		fmt.Sscanf(key2, "%v", &hash2)
		for i := range hash1 {
			diff := float64(hash1[i] - hash2[i])
			dist += diff * diff
		}
		return math.Sqrt(dist)
	}

	// Create a priority queue and add all buckets to it so they can be searched
	// in order.
	pq := &priorityQueue{}
	heap.Init(pq)

	// Generate neighboring keys and add them to the priority queue
	for neighborKey := range table.Buckets {
		distance := calculateDistance(initialKey, neighborKey)
		heap.Push(pq, &priorityItem{Key: neighborKey, Priority: distance})
	}

	return pq
}
