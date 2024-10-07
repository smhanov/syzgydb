#!/usr/bin/env python3
import csv
import requests

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

    def get_info(self, collection_name):
        url = f"{self.server_address}/api/v1/collections/{collection_name}"
        response = requests.get(url)
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

    def insert_records(self, collection_name, records):
        url = f"{self.server_address}/api/v1/collections/{collection_name}/records"
        print("Send to ", url)
        response = requests.post(url, json=records)
        if response.status_code != 200:
            print(f"HTTP Error: {response.status_code} - {response.text}")
        return response.json()

def processTweets():
    name = "tweets2"
    # Get collection info
    collection_info = client.get_info(name)
    document_count = collection_info.get("document_count", 0)


    #print("Creating collection...")
    #print(client.create_collection("tweets2", 768, 8, "cosine"))

    # Read the csv file "training.1600000.processed.noemoticon.csv"
    # Collect all tweets from the 6th column in the collection

    batch_size = 1
    records = []
    with open("training.1600000.processed.noemoticon.csv", mode='r', encoding='utf-8') as file:
        csv_reader = csv.reader(file)
        
        # Skip rows based on document_count
        for _ in range(document_count):
            next(csv_reader, None)
        for row in csv_reader:
            # Extract the tweet from the 6th column (index 5)
            tweet_text = row[5]
            
            # Create a record for the tweet
            record_id = csv_reader.line_num  # or any other unique identifier
            record = {
                "id": record_id,
                "text": tweet_text,
                "metadata": {"text": tweet_text}
            }
            records.append(record)

            # Insert records in batches of 100
            if len(records) == batch_size:
                print(f"Inserting batch of {batch_size} records...")
                response = client.insert_records("tweets2", records)
                print(response)
                records = []  # Clear the batch

    # Insert any remaining records
    if records:
        print(f"Inserting final batch of {len(records)} records...")
        response = client.insert_records("tweets2", records)
        print(response)
            
            
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

