import requests
import json

class SyzgyDBClient:
    def __init__(self, server_address):
        self.server_address = server_address

    def create_collection(self, name, vector_size, quantization, distance_function):
        url = f"{self.server_address}/api/v1/collections"
        data = {
            "name": name,
            "vector_size": vector_size,
            "quantization": quantization,
            "distance_function": distance_function
        }
        response = requests.post(url, json=data)
        return response.json()

    def delete_collection(self, name):
        url = f"{self.server_address}/api/v1/collections/{name}"
        response = requests.delete(url)
        return response.json()

    def insert_record(self, collection_name, record_id, text=None, vector=None, metadata=None):
        url = f"{self.server_address}/api/v1/collections/{collection_name}/records"
        data = {
            "id": record_id,
            "text": text,
            "vector": vector,
            "metadata": metadata or {}
        }
        response = requests.post(url, json=data)
        return response.json()

    def search_records(self, collection_name, vector=None, text=None, offset=0, limit=10, include_vectors=False):
        url = f"{self.server_address}/api/v1/collections/{collection_name}/search"
        params = {
            "offset": offset,
            "limit": limit,
            "include_vectors": str(include_vectors).lower()
        }
        data = {
            "vector": vector,
            "text": text
        }
        response = requests.get(url, params=params, json=data)
        return response.json()

# Example usage
if __name__ == "__main__":
    client = SyzgyDBClient("http://localhost:8080")

    # Create a collection
    print("Creating collection...")
    print(client.create_collection("test_collection", 384, 64, "cosine"))

    # Insert records
    print("Inserting records...")
    print(client.insert_record("test_collection", 1, text="This is the first test record", metadata={"category": "test"}))
    print(client.insert_record("test_collection", 2, text="This is the second test record", metadata={"category": "test"}))
    print(client.insert_record("test_collection", 3, text="This is the third test record", metadata={"category": "test"}))

    # Search records
    print("Searching records...")
    print(client.search_records("test_collection", text="test record"))
