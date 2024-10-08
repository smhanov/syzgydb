[![Go Reference](https://pkg.go.dev/badge/github.com/smhanov/syzgydb.svg)](https://pkg.go.dev/github.com/smhanov/syzgydb)

# Syzgy DB

![image](https://github.com/user-attachments/assets/f8cc7b60-1fd0-4319-a607-b8d3269a288d)

## Introduction

SyzgyDB is a high-performance, embeddable vector database designed for applications requiring efficient handling of large datasets. Written in Go, it leverages disk-based storage to minimize memory usage, making it ideal for systems with limited resources. SyzgyDB supports a range of distance metrics, including Euclidean and Cosine, and offers multiple quantization levels to optimize storage and search performance.

With built-in integration for the Ollama server, SyzgyDB can automatically generate vector embeddings from text and images, simplifying the process of adding and querying data. This makes it well-suited for use cases such as image and video retrieval, recommendation systems, natural language processing, anomaly detection, and bioinformatics. With its RESTful API, SyzgyDB provides easy integration and management of collections and records, enabling developers to perform fast and flexible vector similarity searches.


## Features

* **Disk-Based Storage**: Operates with minimal memory usage by storing data on disk.
* **Automatic Embedding Generation**: Seamlessly integrates with the Ollama server to generate vector embeddings from text and images, reducing the need for manual preprocessing.
* **Vector Quantization**: Supports multiple quantization levels (4, 8, 16, 32, 64 bits) to optimize storage and performance.
* **Distance Metrics**: Supports Euclidean and Cosine distance calculations for vector similarity.
* **Scalable**: Efficiently handles large datasets with support for adding, updating, and removing documents.
* **Search Capabilities**: Provides nearest neighbor and radius-based search functionalities.


## Running with Docker

```bash
docker run -p 8080:8080 -v /path/to/your/data:/data smhanov/syzgydb
```

This command will:

1. Pull the `smhanov/syzgydb` image from Docker Hub.
2. Map port 8080 of the container to port 8080 on your host machine.
3. Map the `/data` directory inside the container to `/path/to/your/data` on your host system, ensuring that your data is persisted outside the container.


## Configuration

The configuration settings can be specified on the command line, using an environment variable, or in a file /etc/syzgydb.conf.


| **Configuration Setting** | **Description** | **Default Value** |
|---------------------------|-----------------|-------------------|
| `DATA_FOLDER`             | Specifies where the persistent files are kept. | `./data` (command line) or `/data` (Docker) |
| `OLLAMA_SERVER`           | The optional Ollama server used to create embeddings. | `localhost:11434` |
| `TEXT_MODEL`              | The name of the text embedding model to use with Ollama. | `all-minilm` (384 dimensions) |
| `IMAGE_MODEL`             | The name of the image embedding model to use with Ollama. | `minicpm-v` |

## RESTful API

SyzgyDB provides a RESTful API for managing collections and records. Below are the available endpoints and example `curl` requests.

### Collections API

A collection is a database, and you can create them and get information about them.

#### Create a Collection

 **Endpoint**: `POST /api/v1/collections`
 **Description**: Creates a new collection with specified parameters.
 **Request Body** (JSON):
  ```json
  {
    "name": "collection_name",
    "vector_size": 128,
    "quantization": 64,
    "distance_function": "cosine"
  }
  ```
 **Example `curl`**:
  ```bash
  curl -X POST http://localhost:8080/api/v1/collections -H "Content-Type: application/json" -d '{"name":"collection_name","vector_size":128,"quantization":64,"distance_function":"cosine"}'
  ```

#### Drop a Collection

 **Endpoint**: `DELETE /api/v1/collections/{collection_name}`
 **Description**: Deletes the specified collection.
 **Example `curl`**:
  ```bash
  curl -X DELETE http://localhost:8080/api/v1/collections/collection_name
  ```

#### Get Collection Info

 **Endpoint**: `GET /api/v1/collections/{collection_name}`
 **Description**: Retrieves information about a collection.
 **Example `curl`**:
  ```bash
  curl -X GET http://localhost:8080/api/v1/collections/collection_name
  ```

### Data API

#### Insert / update records

 **Endpoint**: `POST /api/v1/collections/{collection_name}/records`
 **Description**: Inserts multiple records into a collection. Overwrites if the ID exists. You can provide either a `vector` or a `text` field for each record. If a `text` field is provided, the server will automatically generate the vector embedding using the Ollama server. If an image field is provided, it should be in base64 format.
 **Request Body** (JSON):
  ```json
  [
    {
      "id": 1234567890,
      "text": "example text", // Optional: Provide text to generate vector
      "vector": [0.1, 0.2, ..., 0.5], // Optional: Directly provide a vector
      "metadata": {
        "key1": "value1",
        "key2": "value2"
      }
    },
    {
      "id": 1234567891,
      "text": "another example text",
      "metadata": {
        "key1": "value3"
      }
    }
  ]
  ```
 **Example `curl`**:
  ```bash
  curl -X POST http://localhost:8080/api/v1/collections/collection_name/records -H "Content-Type: application/json" -d '[{"id":1234567890,"vector":[0.1,0.2,0.3,0.4,0.5],"metadata":{"key1":"value1","key2":"value2"}},{"id":1234567891,"text":"example text","metadata":{"key1":"value1","key2":"value2"}}]'
  ```


#### Update a Record's Metadata

 **Endpoint**: `PUT /api/v1/collections/{collection_name}/records/{id}/metadata`
 **Description**: Updates metadata for a record.
 **Request Body** (JSON):
  ```json
  {
    "metadata": {
      "key1": "new_value1",
      "key3": "value3"
    }
  }
  ```
 **Example `curl`**:
  ```bash
  curl -X PUT http://localhost:8080/api/v1/collections/collection_name/records/1234567890/metadata -H "Content-Type: application/json" -d '{"metadata":{"key1":"new_value1","key3":"value3"}}'
  ```

#### Delete a Record

 **Endpoint**: `DELETE /api/v1/collections/{collection_name}/records/{id}`
 **Description**: Deletes a record.
 **Example `curl`**:
  ```bash
  curl -X DELETE http://localhost:8080/api/v1/collections/collection_name/records/1234567890
  ```

#### Get All Document IDs

 **Endpoint**: `GET /api/v1/collections/{collection_name}/ids`
 **Description**: Retrieves a JSON array of all document IDs in the specified collection.
 **Example `curl`**:
  ```bash
  curl -X GET http://localhost:8080/api/v1/collections/collection_name/ids
  ```

#### Search Records

 **Endpoint**: `POST /api/v1/collections/{collection_name}/search`
 **Description**: Searches for records based on the provided criteria. If no search parameters are provided, it lists all records in the collection, allowing pagination with `limit` and `offset`.

 **Request Body** (JSON):
  ```json
  {
    "vector": [0.1, 0.2, 0.3, ..., 0.5], // Optional: Provide a vector for similarity search
    "text": "example text",              // Optional: Provide text to generate vector for search
    "k": 5,                              // Optional: Number of nearest neighbors to return
    "radius": 0,                       // Optional: Radius for range search
    "limit": 0,                         // Optional: Maximum number of records to return
    "offset": 0,                         // Optional: Number of records to skip for pagination
    "precision": "",                 // Optional: Set to "exact" for exhaustive search
    "filter": "age >= 18 AND status == 'active'" // Optional: Query filter expression
  }
  ```

 **Parameters Explanation**:
  - **`vector`**: A numerical array representing the query vector. Used for similarity searches. If provided, the search will be based on this vector.
  - **`text`**: A string input that will be converted into a vector using the Ollama server. This is an alternative to providing a `vector` directly.
  - **`k`**: Specifies the number of nearest neighbors to return. Used when performing a k-nearest neighbor search.
  - **`radius`**: Defines the radius for a range search. All records within this distance from the query vector will be returned.
  - **`limit`**: Limits the number of records returned in the response. Useful for paginating results.
  - **`offset`**: Skips the specified number of records before starting to return results. Used in conjunction with `limit` for pagination.
  - **`precision`**: Specifies the search precision. Defaults to "medium". Set to "exact" to perform an exhaustive search of all points.
  - **`filter`**: A string containing a query filter expression. This allows for additional filtering of results based on metadata fields.

 **Example `curl`**:
  ```bash
  curl -X POST http://localhost:8080/api/v1/collections/collection_name/search -H "Content-Type: application/json" -d '{"vector":[0.1,0.2,0.3,0.4,0.5],"k":5,"limit":10,"offset":0,"filter":"age >= 18 AND status == \"active\""}'
  ```

 **Usage Scenarios**:
  - **List All Records**: Call the endpoint with no parameters to list all records, using `limit` and `offset` to paginate.
  - **Text-Based Search**: Provide a `text` parameter to perform a search based on the text's vector representation.
  - **Vector-Based Search**: Use the `vector` parameter for direct vector similarity searches.
  - **Range Query**: Specify a `radius` to perform a range query, returning all records within the specified distance.
  - **K-Nearest Neighbors**: Use the `k` parameter to find the top `k` nearest records to the query vector.
  - **Filtered Search**: Use the `filter` parameter to apply additional constraints based on metadata fields.

## Usage in a Go Project

You don't need to use the docker or REST api. You can build it right in to your go project. Here's how.

```go 
    import "github.com/smhanov/syzgydb"
```

### Creating a Collection

To create a new collection, define the collection options and initialize the collection:

```go
options := syzgydb.CollectionOptions{
    Name:           "example.dat",
    DistanceMethod: syzgydb.Euclidean, // or Cosine
    DimensionCount: 128,       // Number of dimensions for each vector
    Quantization:   64,        // Quantization level (4, 8, 16, 32, 64)
}

collection := syzgydb.NewCollection(options)
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
args := syzgydb.SearchArgs{
    Vector:   searchVector,
    K: 5, // Return top 5 results
}

results := collection.Search(args)

// Radius-based search
args = syzgydb.SearchArgs{
    Vector: searchVector,
    Radius: 0.5, // Search within a radius of 0.5
}

results = collection.Search(args)
```

#### Using a Filter Function

You can apply a filter function during the search to include only documents that meet certain criteria. There are two ways to create a filter function:

1. Using a custom function:

```go
filterFn := func(id uint64, metadata []byte) bool {
    return id%2 == 0 // Include only documents with even IDs
}

args := syzgydb.SearchArgs{
    Vector:   searchVector,
    K: 5, // Return top 5 results
    Filter:   filterFn,
}

results := collection.Search(args)
```

2. Using the `BuildFilter` method with a query string:

```go
queryString := `age >= 18 AND status == \"active\"`
filterFn, err := syzgydb.BuildFilter(queryString)
if err != nil {
    log.Fatalf("Error building filter: %v", err)
}

args := syzgydb.SearchArgs{
    Vector:   searchVector,
    K: 5, // Return top 5 results
    Filter:   filterFn,
}

results := collection.Search(args)
```

The `BuildFilter` method allows you to create a filter function from a query string using the Query Filter Language described in this document. This provides a flexible way to filter search results based on metadata fields without writing custom Go code for each filter.

### Updating and Removing Documents

Update the metadata of an existing document or remove a document from the collection:

```go
// Update document metadata
err := collection.UpdateDocument(1, []byte("updated metadata"))

// Remove a document
err = collection.RemoveDocument(1)
```

### Dumping the Collection

To dump the collection for inspection or backup, use the `DumpIndex` function:

```go
syzgydb.DumpIndex("example.dat")
```

## Query Filter Language

SyzgyDB supports a powerful query filter language that allows you to filter search results based on metadata fields. This language can be used in the `filter` parameter of the search API.

### Basic Syntax

- **Field Comparison**: `field_name operator value`
  - Example: `age >= 18`

- **Logical Operations**: Combine conditions using `AND`, `OR`, `NOT`
  - Example: `(age >= 18 AND status == "active") OR role == "admin"`

- **Parentheses**: Use to group conditions and control evaluation order
  - Example: `(status == "active" AND age >= 18) OR role == "admin"`

### Supported Operators

- **Comparison**: `==`, `!=`, `>`, `<`, `>=`, `<=`
- **String Operations**: `CONTAINS`, `STARTS_WITH`, `ENDS_WITH`, `MATCHES` (regex)
- **Existence**: `EXISTS`, `DOES NOT EXIST`
- **Array Operations**: `IN`, `NOT IN`

### Functions

- `field.length`: Returns the length of a string or array


### Examples

1. **Basic Comparison**:
   ```
   age >= 18 AND status == "active"
   ```

2. **String Operations**:
   ```
   name STARTS_WITH "John" AND email ENDS_WITH "@example.com"
   ```

3. **Array Operations**:
   ```
   status IN ["important", "urgent"] 
   ```

4. **Nested Fields**:
   ```
   user.profile.verified == true AND user.friends.length > 5
   ```

5. **Complex Query**:
   ```
   (status == "active" AND age >= 18) OR (role == "admin" AND NOT (department == "IT"))
   ```


## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue to discuss improvements or report bugs.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.