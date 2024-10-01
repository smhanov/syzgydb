package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"
)

func main() {
	// Define command-line flags
	points := flag.Int("points", 1000, "Number of points to generate")
	dims := flag.Int("dims", 2, "Number of dimensions for each point")

	// Parse the flags
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	// Define collection options
	options := CollectionOptions{
		Name:           "gaussian_collection",
		DistanceMethod: Euclidean,
		DimensionCount: *dims,
	}

	// Create a new collection
	collection := NewCollection(options)

	// Number of clusters and vectors
	numClusters := 50
	numVectors := *points

	// Generate random cluster centers
	clusterCenters := make([][]float64, numClusters)
	for i := 0; i < numClusters; i++ {
		center := make([]float64, *dims)
		for d := 0; d < *dims; d++ {
			center[d] = rand.Float64() * 100
		}
		clusterCenters[i] = center
	}

	// Add vectors to the collection
	for i := 0; i < numVectors; i++ {
		// Select a random cluster center
		center := clusterCenters[rand.Intn(numClusters)]

		// Generate a vector around the cluster center with Gaussian noise
		vector := make([]float64, *dims)
		for d := 0; d < *dims; d++ {
			vector[d] = center[d] + rand.NormFloat64()
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
