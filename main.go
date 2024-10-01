package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"
)

const collectionName = "gaussian_collection"

func main() {
	// Define command-line flags
	dump := flag.Bool("dump", false, "Dump the collection and exit")
	points := flag.Int("points", 1000, "Number of points to generate")
	dims := flag.Int("dims", 2, "Number of dimensions for each point")
	resume := flag.Bool("resume", false, "Resume from existing collection")

	// Parse the flags
	flag.Parse()

	if *dump {
		// Dump the collection and exit
		DumpIndex(collectionName)
		return
	}
	fmt.Println("Vectors added to the collection.")
	if !*resume {
		// Delete the existing file if it exists
		if _, err := os.Stat(collectionName); err == nil {
			err = os.Remove(collectionName)
			if err != nil {
				fmt.Printf("Error deleting file: %v\n", err)
				return
			}
		}
	}

	rand.Seed(time.Now().UnixNano())

	// Define collection options
	options := CollectionOptions{
		Name:           collectionName,
		DistanceMethod: Euclidean,
		DimensionCount: *dims,
	}

	fmt.Println("Starting collection creation...")
	collection := NewCollection(options)
	fmt.Println("Collection created.")

	// Number of clusters and vectors
	numClusters := 50
	numVectors := *points

	fmt.Println("Generating cluster centers...")
	clusterCenters := make([][]float64, numClusters)
	if !*resume {
		for i := 0; i < numClusters; i++ {
			center := make([]float64, *dims)
			for d := 0; d < *dims; d++ {
				center[d] = rand.Float64() * 100
			}
			clusterCenters[i] = center
		}

		fmt.Println("Cluster centers generated.")
		fmt.Println("Adding vectors to the collection...")
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
	}

	// Define a search vector
	var searchVector []float64
	if *resume {
		// Use an existing document as the search vector
		doc, err := collection.GetDocument(0) // Assuming ID 0 exists; adjust as needed
		if err != nil {
			fmt.Printf("Error retrieving document: %v\n", err)
			return
		}
		searchVector = doc.Vector
	} else {
		// Use the first cluster center as the search vector
		searchVector = clusterCenters[0]
	}

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
		fmt.Printf("ID: %d, Distance: %.4f, Metadata: %s\n", result.ID, result.Distance, string(result.Metadata))
	}
}
