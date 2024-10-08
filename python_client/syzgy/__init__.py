import requests
from typing import List, Dict, Union, Optional
from .exceptions import SyzgyException
from .models import Collection, Document, SearchResult

class SyzgyClient:
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip('/')

    def _request(self, method: str, endpoint: str, **kwargs) -> Dict:
        url = f"{self.base_url}{endpoint}"
        response = requests.request(method, url, **kwargs)
        if response.status_code >= 400:
            raise SyzgyException(f"HTTP {response.status_code}: {response.text}")
        return response.json()

    def create_collection(self, name: str, vector_size: int, quantization: int, distance_function: str) -> Collection:
        data = {
            "name": name,
            "vector_size": vector_size,
            "quantization": quantization,
            "distance_function": distance_function
        }
        result = self._request("POST", "/api/v1/collections", json=data)
        return Collection(**result)

    def get_collections(self) -> List[Collection]:
        result = self._request("GET", "/api/v1/collections")
        return [Collection(**collection) for collection in result]

    def get_collection(self, name: str) -> Collection:
        result = self._request("GET", f"/api/v1/collections/{name}")
        return Collection(**result)

    def delete_collection(self, name: str) -> Dict:
        return self._request("DELETE", f"/api/v1/collections/{name}")

    def insert_documents(self, collection_name: str, documents: List[Document]) -> Dict:
        data = [doc.to_dict() for doc in documents]
        return self._request("POST", f"/api/v1/collections/{collection_name}/records", json=data)

    def update_document_metadata(self, collection_name: str, document_id: int, metadata: Dict) -> Dict:
        data = {"metadata": metadata}
        return self._request("PUT", f"/api/v1/collections/{collection_name}/records/{document_id}/metadata", json=data)

    def delete_document(self, collection_name: str, document_id: int) -> Dict:
        return self._request("DELETE", f"/api/v1/collections/{collection_name}/records/{document_id}")

    def search(self, collection_name: str, vector: Optional[List[float]] = None, text: Optional[str] = None,
               k: Optional[int] = None, radius: Optional[float] = None, limit: Optional[int] = None,
               offset: Optional[int] = None, precision: Optional[str] = None,
               filter: Optional[str] = None) -> List[SearchResult]:
        data = {
            "vector": vector,
            "text": text,
            "k": k,
            "radius": radius,
            "limit": limit,
            "offset": offset,
            "precision": precision,
            "filter": filter
        }
        data = {k: v for k, v in data.items() if v is not None}
        result = self._request("POST", f"/api/v1/collections/{collection_name}/search", json=data)
        return [SearchResult(**item) for item in result["results"]]

    def get_document_ids(self, collection_name: str) -> List[int]:
        result = self._request("GET", f"/api/v1/collections/{collection_name}/ids")
        return result
