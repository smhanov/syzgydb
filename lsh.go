package syzgydb

import (
	"fmt"
	"container/heap"
	"math"
	"math/rand"
)

type HashFunction struct {
	RandomVector []float64
	Offset       float64
	W            float64 // Bucket width
}

func (table *LSHTable) MultiprobeQuery(vector []float64) *PriorityQueue {
	// Hash the input vector
	initialKey := table.Hash(vector)

	// Create a priority queue and add the initial key with priority 0
	pq := &PriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &PriorityItem{Key: initialKey, Priority: 0})

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

	visited := make(map[string]bool)

	for pq.Len() > 0 {
		item := heap.Pop(pq).(*PriorityItem)
		key := item.Key

		// Avoid processing the same key multiple times
		if visited[key] {
			continue
		}
		visited[key] = true


		// Generate neighboring keys and add them to the priority queue
		for neighborKey := range table.Buckets {
			if !visited[neighborKey] {
				distance := calculateDistance(initialKey, neighborKey)
				heap.Push(pq, &PriorityItem{Key: neighborKey, Priority: distance})
			}
		}
	}

	return pq
}

type PriorityItem struct {
	Key      string
	Priority float64
}

type PriorityQueue []*PriorityItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Priority < pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*PriorityItem)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

func NewHashFunction(dim int, w float64) *HashFunction {
	randVec := make([]float64, dim)
	for i := range randVec {
		randVec[i] = rand.NormFloat64()
	}
	offset := rand.Float64() * w
	return &HashFunction{
		RandomVector: randVec,
		Offset:       offset,
		W:            w,
	}
}

func (hf *HashFunction) Hash(vector []float64) int {
	dotProduct := 0.0
	for i, v := range vector {
		dotProduct += v * hf.RandomVector[i]
	}
	return int(math.Floor((dotProduct + hf.Offset) / hf.W))
}

type LSHTable struct {
	HashFunctions []*HashFunction
	Buckets       map[string][]uint64
}

func NewLSHTable(numHashFunctions, dim int, w float64) *LSHTable {
	hashFuncs := make([]*HashFunction, numHashFunctions)
	for i := 0; i < numHashFunctions; i++ {
		hashFuncs[i] = NewHashFunction(dim, w)
	}
	return &LSHTable{
		HashFunctions: hashFuncs,
		Buckets:       make(map[string][]uint64), // Initialize the map here
	}
}

func (table *LSHTable) Hash(vector []float64) string {
	hashes := make([]int, len(table.HashFunctions))
	for i, hf := range table.HashFunctions {
		hashes[i] = hf.Hash(vector)
	}
	return fmt.Sprintf("%v", hashes)
}

func (table *LSHTable) AddPoint(docid uint64, vector []float64) {
	key := table.Hash(vector)
	table.Buckets[key] = append(table.Buckets[key], docid)
}

func (table *LSHTable) removePoint(docid uint64, vector []float64) {
	key := table.Hash(vector)
	// search for the docid in the bucket and remove it
	for i, id := range table.Buckets[key] {
		if id == docid {
			table.Buckets[key] = append(table.Buckets[key][:i], table.Buckets[key][i+1:]...)
			break
		}
	}
}

func (table *LSHTable) Query(vector []float64) []uint64 {
	key := table.Hash(vector)

	return table.Buckets[key]
}
