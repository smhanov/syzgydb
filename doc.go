/*
Package syzygy provides an embeddable vector database written in Go, designed to efficiently handle large datasets by keeping data on disk rather than in memory. This makes SyzygyDB ideal for systems with limited memory resources.

# What is a Vector Database?

A vector database is a specialized database designed to store and query high-dimensional vector data. Vectors are numerical representations of data points, often used in machine learning and data science to represent features of objects, such as images, text, or audio. Vector databases enable efficient similarity searches, allowing users to find vectors that are close to a given query vector based on a specified distance metric.

# Features

- Disk-Based Storage: Operates with minimal memory usage by storing data on disk.
- Vector Quantization: Supports multiple quantization levels (4, 8, 16, 32, 64 bits) to optimize storage and performance.
- Distance Metrics: Supports Euclidean and Cosine distance calculations for vector similarity.
- Scalable: Efficiently handles large datasets with support for adding, updating, and removing documents.
- Search Capabilities: Provides nearest neighbor and radius-based search functionalities.

# Usage

## Creating a Collection

To create a new collection, define the collection options and initialize the collection:

	options := CollectionOptions{
	    Name:           "example_collection",
	    DistanceMethod: Euclidean, // or Cosine
	    DimensionCount: 128,       // Number of dimensions for each vector
	    Quantization:   64,        // Quantization level (4, 8, 16, 32, 64)
	}

	collection := NewCollection(options)

## Adding Documents

Add documents to the collection by specifying an ID, vector, and optional metadata:

	vector := []float64{0.1, 0.2, 0.3, ..., 0.128} // Example vector
	metadata := []byte("example metadata")

	collection.AddDocument(1, vector, metadata)

## Searching

Perform a search to find similar vectors using either nearest neighbor or radius-based search:

	searchVector := []float64{0.1, 0.2, 0.3, ..., 0.128} // Example search vector

	// Nearest neighbor search
	args := SearchArgs{
	    Vector:   searchVector,
	    MaxCount: 5, // Return top 5 results
	}

	results := collection.Search(args)

	// Radius-based search
	args = SearchArgs{
	    Vector: searchVector,
	    Radius: 0.5, // Search within a radius of 0.5
	}

	results = collection.Search(args)

## Updating and Removing Documents

Update the metadata of an existing document or remove a document from the collection:

	// Update document metadata
	err := collection.UpdateDocument(1, []byte("updated metadata"))

	// Remove a document
	err = collection.removeDocument(1)

## Dumping the Collection

To dump the collection for inspection or backup, use the DumpIndex function:

	DumpIndex("example_collection")

# Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue to discuss improvements or report bugs.

# License

This project is licensed under the MIT License. See the LICENSE file for details.
*/
package syzygy
