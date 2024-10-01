package main

import (
	"math"
	"math/rand"
)



// PivotsManager manages the list of pivots and their distances
type PivotsManager struct {
	// Ids of the pivots in the arrays
	pivotIDs uint64[]

	// Map from a point ID to the distances to each pivot.
	// The key is the point ID and the value is a slice of distances to each pivot.
	// The pivots are in the order specified in pivotIDs
	pivots map]uint64][]float64
}

// AddPivot adds a new pivot to the manager
func (pm *PivotsManager) AddPivot(vector []float64) {
	pm.Pivots = append(pm.Pivots, Pivot{Vector: vector})
}

// CalculateDistance calculates the Euclidean distance between two vectors
func CalculateDistance(vec1, vec2 []float64) float64 {
	sum := 0.0
	for i := range vec1 {
		diff := vec1[i] - vec2[i]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

// SelectInitialPivot selects an initial random pivot
func (pm *PivotsManager) SelectInitialPivot(vectors [][]float64) []float64 {
	return vectors[rand.Intn(len(vectors))]
}

// SelectFarthestPoint selects the point farthest from the given vector
func (pm *PivotsManager) SelectFarthestPoint(vectors [][]float64, reference []float64) []float64 {
	var farthest []float64
	maxDistance := -1.0

	for _, vec := range vectors {
		distance := CalculateDistance(vec, reference)
		if distance > maxDistance {
			maxDistance = distance
			farthest = vec
		}
	}

	return farthest
}

// SelectPivotWithMinVariance selects a pivot with minimum variance of distances to other pivots
func (pm *PivotsManager) SelectPivotWithMinVariance(vectors [][]float64) []float64 {
	var bestPivot []float64
	minVariance := math.MaxFloat64

	for _, vec := range vectors {
		var distances []float64
		for _, pivot := range pm.Pivots {
			distances = append(distances, CalculateDistance(vec, pivot.Vector))
		}

		variance := calculateVariance(distances)
		if variance < minVariance {
			minVariance = variance
			bestPivot = vec
		}
	}

	return bestPivot
}

// calculateVariance calculates the variance of a slice of float64
func calculateVariance(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	mean := 0.0
	for _, value := range data {
		mean += value
	}
	mean /= float64(len(data))

	variance := 0.0
	for _, value := range data {
		diff := value - mean
		variance += diff * diff
	}
	return variance / float64(len(data))
}
