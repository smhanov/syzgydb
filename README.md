# Syzgy DB
![image](https://github.com/user-attachments/assets/f8cc7b60-1fd0-4319-a607-b8d3269a288d)


SyzgyDB is a high-performance, embeddable vector database designed for applications requiring efficient handling of large datasets. Written in Go, it leverages disk-based storage to minimize memory usage, making it ideal for systems with limited resources. SyzgyDB supports a range of distance metrics, including Euclidean and Cosine, and offers multiple quantization levels to optimize storage and search performance. It is well-suited for use cases such as image and video retrieval, recommendation systems, natural language processing, anomaly detection, and bioinformatics. With its RESTful API, SyzgyDB provides easy integration and management of collections and records, enabling developers to perform fast and flexible vector similarity searches.


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

## Running with Docker

You can run SyzgyDB using Docker to simplify the setup process. Follow these steps to build and run the Docker container:

### Build the Docker Image

First, ensure you have Docker installed on your system. Then, build the Docker image using the following command:

```bash
docker build -t syzgydb .
```

This command will create a Docker image named `syzgydb` using the Dockerfile in the repository.

### Run the Docker Container

Once the image is built, you can run the Docker container with the following command:

```bash
docker run -p 8080:8080 syzgydb
```

This command will start the SyzgyDB server inside a Docker container and map port 8080 of the container to port 8080 on your host machine. You can access the server's API at `http://localhost:8080`.

### Stopping the Docker Container

To stop the running Docker container, you can use the `docker ps` command to find the container ID and then stop it using:

```bash
docker stop <container_id>
```

Replace `<container_id>` with the actual ID of the running container.

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

## RESTful API

SyzgyDB provides a RESTful API for managing collections and records. Below are the available endpoints and example `curl` requests.

### Collections API

#### Create a Collection

- **Endpoint**: `POST /api/v1/collections`
- **Description**: Creates a new collection with specified parameters.
- **Request Body** (JSON):
  ```json
  {
    "name": "collection_name",
    "vector_size": 128,
    "quantization": 64,
    "distance_function": "cosine"
  }
  ```
- **Example `curl`**:
  ```bash
  curl -X POST http://localhost:8080/api/v1/collections -H "Content-Type: application/json" -d '{"name":"collection_name","vector_size":128,"quantization":64,"distance_function":"cosine"}'
  ```

#### Drop a Collection

- **Endpoint**: `DELETE /api/v1/collections/{collection_name}`
- **Description**: Deletes the specified collection.
- **Example `curl`**:
  ```bash
  curl -X DELETE http://localhost:8080/api/v1/collections/collection_name
  ```

#### Get Collection Info

- **Endpoint**: `GET /api/v1/collections/{collection_name}`
- **Description**: Retrieves information about a collection.
- **Example `curl`**:
  ```bash
  curl -X GET http://localhost:8080/api/v1/collections/collection_name
  ```

### Data API

#### Insert One Record

- **Endpoint**: `POST /api/v1/collections/{collection_name}/records`
- **Description**: Inserts a record into a collection. Overwrites if the ID exists. You can provide either a `vector` or a `text` field. If a `text` field is provided, the server will automatically generate the vector embedding using the Ollama server.
- **Request Body** (JSON):
  ```json
  {
    "id": 1234567890,
    "text": "example text", // Optional: Provide text to generate vector
    "vector": [0.1, 0.2, ..., 0.5], // Optional: Directly provide a vector
    "metadata": {
      "key1": "value1",
      "key2": "value2"
    }
  }
  ```
- **Example `curl`**:
  ```bash
  # Using a vector
  curl -X POST http://localhost:8080/api/v1/collections/collection_name/records -H "Content-Type: application/json" -d '{"id":1234567890,"vector":[0.1,0.2,0.3,0.4,0.5],"metadata":{"key1":"value1","key2":"value2"}}'

  # Using text to generate a vector
  curl -X POST http://localhost:8080/api/v1/collections/collection_name/records -H "Content-Type: application/json" -d '{"id":1234567891,"text":"example text","metadata":{"key1":"value1","key2":"value2"}}'
  ```

### Explanation

- **Text Field**: The `text` field can be used as an alternative to the `vector` field. When provided, the server will use the Ollama server to generate the vector embedding.
- **Automatic Embedding**: This feature allows users to submit raw text, which is then converted into a vector representation, simplifying the process of adding records to the database.

### Summary

This update to the `README.md` provides clear instructions on how to use the text-to-vector conversion feature, making it easier for users to understand and utilize this functionality in their applications.

#### Update a Record's Metadata

- **Endpoint**: `PUT /api/v1/collections/{collection_name}/records/{id}/metadata`
- **Description**: Updates metadata for a record.
- **Request Body** (JSON):
  ```json
  {
    "metadata": {
      "key1": "new_value1",
      "key3": "value3"
    }
  }
  ```
- **Example `curl`**:
  ```bash
  curl -X PUT http://localhost:8080/api/v1/collections/collection_name/records/1234567890/metadata -H "Content-Type: application/json" -d '{"metadata":{"key1":"new_value1","key3":"value3"}}'
  ```

#### Delete a Record

- **Endpoint**: `DELETE /api/v1/collections/{collection_name}/records/{id}`
- **Description**: Deletes a record.
- **Example `curl`**:
  ```bash
  curl -X DELETE http://localhost:8080/api/v1/collections/collection_name/records/1234567890
  ```

#### Search Records

- **Endpoint**: `GET /api/v1/collections/{collection_name}/search`
- **Description**: Searches for records based on criteria.
- **Example `curl`**:
  ```bash
  curl -X GET "http://localhost:8080/api/v1/collections/collection_name/search?offset=0&limit=10&include_vectors=false&radius=0.5&k=5" -H "Content-Type: application/json" -d '{"vector":[0.1,0.2,0.3,0.4,0.5]}'
  ```

## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue to discuss improvements or report bugs.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
