# Syzgy Python Client

This is the official Python client for interacting with the SyzgyDB vector database REST API. SyzgyDB is a high-performance, embeddable vector database designed for efficient handling of large datasets.

## Installation

You can install the SyzgyDB Python client using pip:

```bash
pip install syzgy
```

## Quick Start

Here's a quick example of how to use the SyzgyDB Python client:

```python
from syzgy import SyzgyClient, Document

# Initialize the client
client = SyzgyClient("http://localhost:8080")

# Create a collection
collection = client.create_collection(
    name="my_collection",
    vector_size=5,
    quantization=64,
    distance_function="cosine"
)

# Insert documents
documents = [
    Document(id=1, vector=[0.1, 0.2, 0.3, 0.4, 0.5], metadata={"key": "value1"}),
    Document(id=2, vector=[1, 2, 3, 4, 5], metadata={"key": "value2"})
]
collection.insert_documents(documents)

# Search
results = collection.search(
    vector=[0,0,0,0,0.1],
    k=5
)

for result in results:
    print(f"ID: {result.id}, Distance: {result.distance}, Metadata: {result.metadata}")

# Delete the collection
client.delete_collection("my_collection")
```

This example demonstrates:

1. Initializing the SyzgyClient
2. Creating a collection
3. Inserting documents into the collection
4. Performing a search on the collection
5. Deleting the collection

Note that document insertion, searching, and other collection-specific operations are performed on the Collection object, while collection management (creation, deletion) is done through the SyzgyClient.

## Features

- Create and manage collections
- Insert, update, and delete documents
- Perform vector similarity searches
- Support for text-to-vector conversion (when using with Ollama server)
- Flexible search options including k-nearest neighbors and radius search
- Metadata filtering

## API Reference

### SyzgyClient

The main class for interacting with SyzgyDB.

#### `__init__(base_url: str)`

Initialize the client with the base URL of your Syzgy instance.

#### `create_collection(name: str, vector_size: int, quantization: int, distance_function: str) -> Collection`

Create a new collection.

#### `get_collections() -> List[Collection]`

Get a list of all collections.

#### `get_collection(name: str) -> Collection`

Get details of a specific collection.

#### `delete_collection(name: str) -> Dict`

Delete a collection.

### Collection

Represents a collection in SyzgyDB.

#### `insert_documents(documents: List[Document]) -> Dict`

Insert documents into the collection.

#### `update_document_metadata(document_id: int, metadata: Dict) -> Dict`

Update the metadata of a document.

#### `delete_document(document_id: int) -> Dict`

Delete a document from the collection.

#### `search(**kwargs) -> List[SearchResult]`

Perform a search in the collection. Supports various search parameters:
- `vector`: Optional[List[float]]
- `text`: Optional[str]
- `k`: Optional[int]
- `radius`: Optional[float]
- `limit`: Optional[int]
- `offset`: Optional[int]
- `precision`: Optional[str]
- `filter`: Optional[str]

#### `get_document_ids() -> List[int]`

Get all document IDs in the collection.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.
