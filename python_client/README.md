# SyzgyDB Python Client

This is the official Python client for interacting with the SyzgyDB vector database REST API. SyzgyDB is a high-performance, embeddable vector database designed for efficient handling of large datasets.

## Installation

You can install the SyzgyDB Python client using pip:

```bash
pip install syzgydb
```

## Quick Start

Here's a quick example of how to use the SyzgyDB Python client:

```python
from syzgydb import SyzgyDBClient, Document

# Initialize the client
client = SyzgyDBClient("http://localhost:8080")

# Create a collection
collection = client.create_collection(
    name="my_collection",
    vector_size=128,
    quantization=64,
    distance_function="cosine"
)

# Insert documents
documents = [
    Document(id=1, vector=[0.1, 0.2, 0.3, ..., 0.5], metadata={"key": "value1"}),
    Document(id=2, text="Example text", metadata={"key": "value2"})
]
client.insert_documents("my_collection", documents)

# Search
results = client.search(
    collection_name="my_collection",
    vector=[0.1, 0.2, 0.3, ..., 0.5],
    k=5
)

for result in results:
    print(f"ID: {result.id}, Distance: {result.distance}, Metadata: {result.metadata}")
```

## Features

- Create and manage collections
- Insert, update, and delete documents
- Perform vector similarity searches
- Support for text-to-vector conversion (when using with Ollama server)
- Flexible search options including k-nearest neighbors and radius search
- Metadata filtering

## API Reference

### SyzgyDBClient

The main class for interacting with SyzgyDB.

#### `__init__(base_url: str)`

Initialize the client with the base URL of your SyzgyDB instance.

#### `create_collection(name: str, vector_size: int, quantization: int, distance_function: str) -> Collection`

Create a new collection.

#### `get_collections() -> List[Collection]`

Get a list of all collections.

#### `get_collection(name: str) -> Collection`

Get details of a specific collection.

#### `delete_collection(name: str) -> Dict`

Delete a collection.

#### `insert_documents(collection_name: str, documents: List[Document]) -> Dict`

Insert documents into a collection.

#### `update_document_metadata(collection_name: str, document_id: int, metadata: Dict) -> Dict`

Update the metadata of a document.

#### `delete_document(collection_name: str, document_id: int) -> Dict`

Delete a document from a collection.

#### `search(collection_name: str, **kwargs) -> List[SearchResult]`

Perform a search in a collection. Supports various search parameters.

#### `get_document_ids(collection_name: str) -> List[int]`

Get all document IDs in a collection.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.
