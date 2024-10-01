package main

import (
	"errors"
	"fmt"
	"math"
)

// Brute force is used for small collections
const pivotThreshold = 100

// PivotsManager manages the list of pivots and their distances
type PivotsManager struct {
	// Pivot documents, in the order they were selected
	pivots []*Document

	// Map from a point ID to the distances to each pivot.
	// The key is the point ID and the value is a slice of distances to each pivot.
	// The distances are in the order specified in pivotIDs
	distances map[uint64][]float64

	distanceFn distanceFn
}

type distanceFn func(vec1, vec2 []float64) float64

// approxDistance calculates the approximate minimum distance of a point from a target document using the triangle inequality.
func (pm *PivotsManager) approxDistance(target *Document, id uint64) float64 {

	// if there are no pivots, return max float64
	if len(pm.pivots) == 0 {
		return math.MaxFloat64
	}

	// Check if the point ID exists in the distances map
	dists, exists := pm.distances[id]
	if !exists {
		fmt.Printf("Point ID not found: %d\n", id) // Debug print
		panic(errors.New("point ID not found in distances map"))
	}

	// Calculate the distance of each pivot from the target
	targetPivotDistances := make([]float64, len(pm.pivots))
	for i, pivot := range pm.pivots {
		targetPivotDistances[i] = pm.distanceFn(target.Vector, pivot.Vector)
	}

	// Use the triangle inequality to compute the minimum possible distance
	minPossibleDistance := math.MaxFloat64
	for i, pivotDistance := range dists {
		possibleDistance := math.Abs(targetPivotDistances[i] - pivotDistance)
		if possibleDistance < minPossibleDistance {
			minPossibleDistance = possibleDistance
		}
	}

	return minPossibleDistance
}

// pointRemoved removes a point from the distances map and updates pivots if necessary.
func (pm *PivotsManager) pointRemoved(docID uint64) {
	delete(pm.distances, docID)

	// Check if the point is a pivot
	pivotIndex := -1
	for i, pivot := range pm.pivots {
		if pivot.ID == docID {
			pivotIndex = i
			break
		}
	}

	// If the point is a pivot, remove it from the pivots array
	if pivotIndex != -1 {
		pm.pivots = append(pm.pivots[:pivotIndex], pm.pivots[pivotIndex+1:]...)

		// Remove the corresponding entry from each entry in the distances map
		for id, dists := range pm.distances {
			if pivotIndex < len(dists) {
				pm.distances[id] = append(dists[:pivotIndex], dists[pivotIndex+1:]...)
			}
		}
	}
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
		distances[i] = pm.distanceFn(doc.Vector, pivot.Vector)
	}

	// Add the entry to the distances map
	pm.distances[doc.ID] = distances
}

func newPivotsManager(distanceFn distanceFn) *PivotsManager {
	return &PivotsManager{
		pivots:     []*Document{},
		distances:  make(map[uint64][]float64), // Initialize the map
		distanceFn: distanceFn,
	}
}

// CalculateDistance calculates the Euclidean distance between two vectors
func CalculateDistance(vec1, vec2 []float64, method int) float64 {
	switch method {
	case Euclidean:
		return euclideanDistance(vec1, vec2)
	case Cosine:
		return cosineDistance(vec1, vec2)
	default:
		panic("unsupported distance method")
	}
}

func (pm *PivotsManager) SelectInitialPivot(c *Collection) error {
	// Step 1: Select a random point
	randomID, err := c.getRandomID()
	if err != nil {
		return err
	}

	// Get the document associated with the random ID
	randomDoc, err := c.getDocument(randomID)
	if err != nil {
		return err
	}

	// Step 2: Find the point farthest from the random point
	var firstPivot *Document
	maxDistance := -1.0
	c.iterateDocuments(func(d *Document) {
		distance := pm.distanceFn(randomDoc.Vector, d.Vector)
		if distance > maxDistance {
			maxDistance = distance
			firstPivot = d
		}
	})

	// Step 3: Find the point farthest from the first pivot
	var secondPivot *Document
	maxDistance = -1.0
	c.iterateDocuments(func(d *Document) {
		distance := pm.distanceFn(firstPivot.Vector, d.Vector)
		pm.distances[d.ID] = []float64{distance}
		if distance > maxDistance {
			maxDistance = distance
			secondPivot = d
		}
	})

	// find the distances to the second pivot
	c.iterateDocuments(func(d *Document) {
		distance := pm.distanceFn(secondPivot.Vector, d.Vector)
		pm.distances[d.ID] = append(pm.distances[d.ID], distance)
	})

	// Set the pivots
	pm.pivots = []*Document{firstPivot, secondPivot}
	return nil
}

// SelectPivotWithMinVariance selects a pivot with minimum variance of distances to other pivots
func (pm *PivotsManager) SelectPivotWithMinVariance(c *Collection) error {
	if len(pm.pivots) == 1 {
		// If there is only one pivot, find the point farthest from the existing pivot
		existingPivot := pm.pivots[0]
		var farthestDoc *Document
		maxDistance := -1.0

		c.iterateDocuments(func(d *Document) {
			distance := pm.distanceFn(existingPivot.Vector, d.Vector)
			if distance > maxDistance {
				maxDistance = distance
				farthestDoc = d
			}
		})

		if farthestDoc != nil {
			// Set the new pivot
			pm.pivots = append(pm.pivots, farthestDoc)

			// Update the distances map for all documents
			c.iterateDocuments(func(d *Document) {
				pm.pointAdded(d)
			})
		}

		return nil
	}

	// Existing logic for selecting pivot with minimum variance
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
		distance := pm.distanceFn(doc.Vector, d.Vector)
		pm.distances[d.ID] = append(pm.distances[d.ID], distance)
	})

	// Set the new pivot
	pm.pivots = append(pm.pivots, doc)

	return nil
}

// ensurePivots ensures that the number of pivots is at least the desired number
func (pm *PivotsManager) ensurePivots(c *Collection, desiredPivots int) {
	for len(pm.pivots) < desiredPivots {
		if len(pm.pivots) == 0 {
			pm.SelectInitialPivot(c)
			return
		}

		pm.SelectPivotWithMinVariance(c)
	}
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
