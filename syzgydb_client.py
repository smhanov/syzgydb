#!/usr/bin/env python3
import csv
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

    def search_records(self, collection_name, vector=None, text=None, offset=0, limit=0, include_vectors=False, k=0):
        url = f"{self.server_address}/api/v1/collections/{collection_name}/search"
        data = {
            "offset": offset,
            "limit": limit,
            "include_vectors": str(include_vectors).lower(),
        }
        if k > 0:
            data["k"] = k

        if vector is not None:
            data["vector"] = vector
        if text is not None:
            data["text"] = text
        
        response = requests.post(url, json=data)
        return response.json()

def processTweets():
    # Create a collection
    print("Creating collection...")
    print(client.delete_collection("tweets"))

    print(client.create_collection("tweets", 384, 64, "cosine"))

    # Read the csv file "training.1600000.processed.noemoticon.csv"
    # insert all tweets from the 6th column in the collection

    
    with open("training.1600000.processed.noemoticon.csv", mode='r', encoding='utf-8') as file:
        csv_reader = csv.reader(file)
        
        # Iterate over each row in the CSV
        i =0
        for row in csv_reader:
            # Extract the tweet from the 6th column (index 5)
            tweet_text = row[5]
            
            # Insert the tweet into the collection
            # Assuming each tweet has a unique ID, you can use the row index as the ID
            record_id = csv_reader.line_num  # or any other unique identifier
            print(client.insert_record("tweets", record_id, text=tweet_text, metadata={"text": tweet_text}))
            i += 1
            if i == 1000:
                break
            
            # Example usage
if __name__ == "__main__":
    client = SyzgyDBClient("http://localhost:8080")
    processTweets()
    if 0:
        # Create a collection
        print("Creating collection...")
        print(client.delete_collection("pycollection"))

        print(client.create_collection("pycollection", 384, 64, "cosine"))

        # Insert records
        print("Inserting records...")
        print(client.insert_record("pycollection", 1, text="This is the first test record", metadata={"category": "test"}))
        print(client.insert_record("pycollection", 2, text="This is the second test record", metadata={"category": "test"}))
        print(client.insert_record("pycollection", 3, text="This is the third test record", metadata={"category": "test"}))

        # Search records
        print("Searching records...")
        print(client.search_records("pycollection", text="test record", k=2))

