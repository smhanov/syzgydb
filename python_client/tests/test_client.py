import unittest
from unittest.mock import patch, MagicMock
from syzgy import SyzgyClient, Collection, Document, SearchResult

class TestSyzgyClient(unittest.TestCase):
    def setUp(self):
        self.client = SyzgyClient("http://localhost:8080")

    @patch('requests.request')
    def test_create_collection(self, mock_request):
        mock_response = MagicMock()
        mock_response.status_code = 201
        mock_response.json.return_value = {
            "name": "test_collection",
            "document_count": 0,
            "dimension_count": 128,
            "quantization": 8,
            "distance_function": "cosine"
        }
        mock_request.return_value = mock_response

        collection = self.client.create_collection("test_collection", 128, 8, "cosine")
        self.assertIsInstance(collection, Collection)
        self.assertEqual(collection.name, "test_collection")

    # Add more tests for other methods...

if __name__ == '__main__':
    unittest.main()
