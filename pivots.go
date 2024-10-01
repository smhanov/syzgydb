package main

import (
	"math"
)

// PivotsManager manages the list of pivots and their distances
type PivotsManager struct {
	// Ids of the pivots in the arrays
	pivotIDs []uint64

	// Map from a point ID to the distances to each pivot.
	// The key is the point ID and the value is a slice of distances to each pivot.
	// The distances are in the order specified in pivotIDs
	distances map[uint64][]float64
}

func NewPivotsManager() *PivotsManager {
	return &PivotsManager{
		pivotIDs:  []uint64{},
		distances: make(map[uint64][]float64),
	}
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

func (pm *PivotsManager) SelectInitialPivot(c *Collection) error {
	// Get a random document ID
	randomID, err := c.getRandomID()
	if err != nil {
		return err
	}

	// Get the document associated with the random ID
	doc, err := c.getDocument(randomID)
	if err != nil {
		return err
	}

	// Set the pivotIDs to the random document ID
	pm.pivotIDs = []uint64{randomID}

	// Use iterateDocuments to fill in distance information in the pivots map
	c.iterateDocuments(func(d *Document) {
		distance := CalculateDistance(d.Vector, doc.Vector)
		pm.distances[d.ID] = []float64{distance}
	})

	return nil
}

// SelectPivotWithMinVariance selects a pivot with minimum variance of distances to other pivots
func (pm *PivotsManager) SelectPivotWithMinVariance(vectors map[uint64][]float64) uint64 {
	var bestPivotID uint64
	minVariance := math.MaxFloat64

	for _, vec := range vectors {
		var distances []float64
		for _, pivotID := range pm.pivotIDs {
			distances = append(distances, CalculateDistance(vec, pm.distances[pivotID]))
		}

		variance := calculateVariance(distances)
		if variance < minVariance {
			minVariance = variance
			bestPivotID = id
		}
	}

	return bestPivotID
}

// ensurePivots ensures that the number of pivots is at least the desired number
func (pm *PivotsManager) ensurePivots(c *Collection, desiredPivots int) error {
	if len(pm.pivotIDs) >= desiredPivots {
		return nil
	}

	if len(pm.pivotIDs) == 0 {
		return pm.SelectInitialPivot(c)
	}

	// Select new pivot based on variance
	newPivotID := pm.SelectPivotWithMinVariance(c)
	newPivot, err := c.getDocument(newPivotID)
	if err != nil {
		return err
	}

	return pm.ensurePivots(c, desiredPivots)
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
