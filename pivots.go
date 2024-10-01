package main

import (
	"errors"
	"math"
)

// PivotsManager manages the list of pivots and their distances
type PivotsManager struct {
	// Pivot documents, in the order they were selected
	pivots []*Document

	// Map from a point ID to the distances to each pivot.
	// The key is the point ID and the value is a slice of distances to each pivot.
	// The distances are in the order specified in pivotIDs
	distances map[uint64][]float64
}

// pointAdded calculates the distance to each pivot and updates the distances map if the point doesn't already exist.
func (pm *PivotsManager) pointAdded(doc *Document) {
    // Check if the point already exists in the distances map
    if _, exists := pm.distances[doc.ID]; exists {
        return
    }

    // Calculate the distance to each pivot
    distances := make([]float64, len(pm.pivots))
    for i, pivot := range pm.pivots {
        distances[i] = CalculateDistance(doc.Vector, pivot.Vector)
    }

    // Add the entry to the distances map
    pm.distances[doc.ID] = distances
}

func NewPivotsManager() *PivotsManager {
	return &PivotsManager{
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

	// Set the pivot to the random document
	pm.pivots = []*Document{doc}

	// Use iterateDocuments to fill in distance information in the distances map
	c.iterateDocuments(func(d *Document) {
		distance := CalculateDistance(d.Vector, doc.Vector)
		pm.distances[d.ID] = []float64{distance}
	})

	return nil
}

// SelectPivotWithMinVariance selects a pivot with minimum variance of distances to other pivots
func (pm *PivotsManager) SelectPivotWithMinVariance(c *Collection) error {
	if len(pm.distances) == 0 {
		return errors.New("no distances available to calculate variance")
	}

	var minVarianceID uint64
	minVariance := math.MaxFloat64

	// Find the point ID with the minimum variance of distances
	for id, dists := range pm.distances {
		variance := calculateVariance(dists)
		if variance < minVariance {
			minVariance = variance
			minVarianceID = id
		}
	}

	// Retrieve the document with the minimum variance ID
	doc, err := c.getDocument(minVarianceID)
	if err != nil {
		return err
	}

	// Update the distances map with actual distances to all other points
	c.iterateDocuments(func(d *Document) {
		distance := CalculateDistance(doc.Vector, d.Vector)
		pm.distances[d.ID] = append(pm.distances[d.ID], distance)
	})

	// Set the new pivot
	pm.pivots = append(pm.pivots, doc)

	return nil
}

// ensurePivots ensures that the number of pivots is at least the desired number
func (pm *PivotsManager) ensurePivots(c *Collection, desiredPivots int) {
	if len(pm.pivots) >= desiredPivots {
		return nil
	}

	if len(pm.pivots) == 0 {
		pm.SelectInitialPivot(c)
		return
	}

	pm.SelectPivotWithMinVariance(c)
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
