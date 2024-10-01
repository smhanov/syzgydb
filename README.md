# Syzgy DB

SyzgyDB is an embeddable vector database written in Go, designed to efficiently handle large datasets by keeping data on disk rather than in memory. This makes SyzgyDB ideal for systems with limited memory resources. It supports various distance metrics and quantization levels, allowing for flexible and efficient vector similarity searches.

## What is a Vector Database?

A vector database is a specialized database designed to store and query high-dimensional vector data. Vectors are numerical representations of data points, often used in machine learning and data science to represent features of objects, such as images, text, or audio. Vector databases enable efficient similarity searches, allowing users to find vectors that are close to a given query vector based on a specified distance metric.

## Applications of Vector Databases

Vector databases are used in a variety of applications, including:

- **Image and Video Retrieval**: Finding similar images or videos based on visual features.
- **Recommendation Systems**: Suggesting products or content based on user preferences and behavior.
- **Natural Language Processing**: Semantic search and document similarity based on text embeddings.
- **Anomaly Detection**: Identifying unusual patterns or outliers in data.
- **Bioinformatics**: Analyzing genetic sequences and protein structures.

## Features

- **Disk-Based Storage**: Operates with minimal memory usage by storing data on disk.
- **Vector Quantization**: Supports multiple quantization levels (4, 8, 16, 32, 64 bits) to optimize storage and performance.
- **Distance Metrics**: Supports Euclidean and Cosine distance calculations for vector similarity.
- **Scalable**: Efficiently handles large datasets with support for adding, updating, and removing documents.
- **Search Capabilities**: Provides nearest neighbor and radius-based search functionalities.

## Installation

To use SyzgyDB in your Go project, you can clone the repository and build the project using the following commands:

```bash
git clone https://github.com/smhanov/syzgydb.git
cd syzgy
go build
```

## Usage

### Creating a Collection

To create a new collection, define the collection options and initialize the collection:

```go
options := CollectionOptions{
    Name:           "example.dat",
    DistanceMethod: Euclidean, // or Cosine
    DimensionCount: 128,       // Number of dimensions for each vector
    Quantization:   64,        // Quantization level (4, 8, 16, 32, 64)
}

collection := NewCollection(options)
```

### Adding Documents

Add documents to the collection by specifying an ID, vector, and optional metadata:

```go
vector := []float64{0.1, 0.2, 0.3, ..., 0.128} // Example vector
metadata := []byte("example metadata")

collection.AddDocument(1, vector, metadata)
```

### Searching

Perform a search to find similar vectors using either nearest neighbor or radius-based search:

```go
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
```

#### Using a Filter Function

You can also apply a filter function during the search to include only documents that meet certain criteria. The filter function takes a document's ID and metadata as arguments and returns a boolean indicating whether the document should be included in the search results.

```go
searchVector := []float64{0.1, 0.2, 0.3, ..., 0.128} // Example search vector

// Define a filter function to exclude documents with odd IDs
filterFn := func(id uint64, metadata []byte) bool {
    return id%2 == 0 // Include only documents with even IDs
}

// Search with a filter function
args := SearchArgs{
    Vector:   searchVector,
    MaxCount: 5, // Return top 5 results
    Filter:   filterFn,
}

results := collection.Search(args)
```

This allows you to customize the search process by excluding certain documents based on their IDs or metadata, providing more control over the search results.

### Updating and Removing Documents

Update the metadata of an existing document or remove a document from the collection:

```go
// Update document metadata
err := collection.UpdateDocument(1, []byte("updated metadata"))

// Remove a document
err = collection.removeDocument(1)
```

### Dumping the Collection

To dump the collection for inspection or backup, use the `DumpIndex` function:

```go
DumpIndex("example_collection")
```

## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue to discuss improvements or report bugs.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
