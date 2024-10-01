package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	// Set random seed for reproducibility
	rand.Seed(time.Now().UnixNano())

	// Define collection options
	options := CollectionOptions{
		Name:           "gaussian_collection",
		DistanceMethod: Euclidean,
		DimensionCount: 3,
	}

	// Create a new collection
	collection := NewCollection(options)

	// Number of clusters and vectors
	numClusters := 50
	numVectors := 100000

	// Generate random cluster centers
	clusterCenters := make([][]float64, numClusters)
	for i := 0; i < numClusters; i++ {
		clusterCenters[i] = []float64{
			rand.Float64() * 100,
			rand.Float64() * 100,
			rand.Float64() * 100,
		}
	}

	// Add vectors to the collection
	for i := 0; i < numVectors; i++ {
		// Select a random cluster center
		center := clusterCenters[rand.Intn(numClusters)]

		// Generate a vector around the cluster center with Gaussian noise
		vector := []float64{
			center[0] + rand.NormFloat64(),
			center[1] + rand.NormFloat64(),
			center[2] + rand.NormFloat64(),
		}

		// Add the vector to the collection
		collection.AddDocument(uint64(i), vector, []byte(fmt.Sprintf("metadata_%d", i)))
	}

	// Define a search vector (e.g., the first cluster center)
	searchVector := clusterCenters[0]

	// Define search arguments
	args := SearchArgs{
		Vector:   searchVector,
		MaxCount: 10, // Limit to top 10 results
	}

	// Time the search operation
	startTime := time.Now()
	results := collection.Search(args)
	duration := time.Since(startTime)

	// Output the search results
	fmt.Printf("Search completed in %v\n", duration)
	fmt.Printf("Percent of space searched: %.2f%%\n", results.PercentSearched)
	fmt.Printf("Top %d results:\n", len(results.Results))
	for _, result := range results.Results {
		fmt.Printf("ID: %d, Distance: %.4f\n", result.ID, result.Distance)
	}
}
