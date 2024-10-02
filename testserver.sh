#!/bin/bash

# Set server address
SERVER_ADDRESS="http://localhost:8080"

# Create a new collection
echo "Creating collection..."
curl -X POST "$SERVER_ADDRESS/api/v1/collections" -H "Content-Type: application/json" -d '{
  "name": "test_collection",
  "vector_size": 128,
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
