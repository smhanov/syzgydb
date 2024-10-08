#!/bin/bash

# Set server address
SERVER_ADDRESS="http://localhost:8080"

# Drop the collection if it exists
echo "Dropping existing collection..."
curl -X DELETE "$SERVER_ADDRESS/api/v1/collections/test_collection"
echo "Creating collection..."
curl -X POST "$SERVER_ADDRESS/api/v1/collections" -H "Content-Type: application/json" -d '{
  "name": "test_collection",
  "vector_size": 384,
  "quantization": 64,
  "distance_function": "cosine"
}'

# Add records using the Text field
echo "Adding records..."
curl -X POST "$SERVER_ADDRESS/api/v1/collections/test_collection/records" -H "Content-Type: application/json" -d '{
  "id": 1,
  "text": "This is the first test record",
  "metadata": {
    "category": "test"
  }
}'

curl -X POST "$SERVER_ADDRESS/api/v1/collections/test_collection/records" -H "Content-Type: application/json" -d '{
  "id": 2,
  "text": "This is the second test record",
  "metadata": {
    "category": "test"
  }
}'

curl -X POST "$SERVER_ADDRESS/api/v1/collections/test_collection/records" -H "Content-Type: application/json" -d '{
  "id": 3,
  "text": "This is the third test record",
  "metadata": {
    "category": "test"
  }
}'

echo "Test records added successfully."

# Search and list all records
echo "Listing all records..."
curl -X GET "$SERVER_ADDRESS/api/v1/collections/test_collection/search?offset=0&limit=10&include_vectors=false" -H "Content-Type: application/json" -d '{}'
