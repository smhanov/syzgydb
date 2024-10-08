from typing import List, Dict, Optional
from dataclasses import dataclass, asdict

@dataclass
class Collection:
    collection_name: str
    document_count: int
    dimension_count: int
    quantization: int
    distance_function: str

@dataclass
class Document:
    id: int
    vector: Optional[List[float]] = None
    text: Optional[str] = None
    metadata: Optional[Dict] = None

    def to_dict(self):
        return {k: v for k, v in asdict(self).items() if v is not None}

@dataclass
class SearchResult:
    id: int
    metadata: Dict
    distance: float
