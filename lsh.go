package syzgydb

import (
	"fmt"
	"math"
	"math/rand"
)

type HashFunction struct {
	RandomVector []float64
	Offset       float64
	W            float64 // Bucket width
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
