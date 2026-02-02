import unittest
from unittest.mock import AsyncMock

from fastapi.testclient import TestClient

from api.dependencies import get_api_key, get_db_service, get_scraper_service
from api.main import app
from api.models import APIKey


class TestMain(unittest.TestCase):
    def setUp(self) -> None:
        self.client = TestClient(app)
        self.mock_scraper_service = AsyncMock()
        self.mock_db_service = AsyncMock()
        self.mock_api_key = APIKey(name="test-key", is_active=True)

        # Override dependencies
        app.dependency_overrides[get_scraper_service] = (
            lambda: self.mock_scraper_service
        )
        app.dependency_overrides[get_db_service] = lambda: self.mock_db_service
        app.dependency_overrides[get_api_key] = lambda: self.mock_api_key

    def tearDown(self) -> None:
        app.dependency_overrides = {}

    def test_scrape_success(self) -> None:
        self.mock_scraper_service.start_scraping.return_value = 123
        response = self.client.post(
            "/scrape", json={"url": "http://example.com", "depth": 2}
        )

        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json(), {"scraping_id": 123})
        self.mock_scraper_service.start_scraping.assert_called_once_with(
            "http://example.com", 2
        )

    def test_scrape_error(self) -> None:
        self.mock_scraper_service.start_scraping.side_effect = Exception("SQS Error")
        response = self.client.post(
            "/scrape", json={"url": "http://example.com", "depth": 2}
        )

        self.assertEqual(response.status_code, 500)
        self.assertIn("SQS Error", response.json()["detail"])

    def test_search_success(self) -> None:
        self.mock_db_service.search_websites.return_value = ["http://site1.com"]
        response = self.client.get("/search?t=test")

        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json(), {"websites": ["http://site1.com"]})
        self.mock_db_service.search_websites.assert_called_once_with("test")

    def test_search_missing_param(self) -> None:
        # FastAPI handles missing params if not optional,
        # but here we added check in code too
        response = self.client.get("/search?t=")
        self.assertEqual(response.status_code, 400)

    def test_search_error(self) -> None:
        self.mock_db_service.search_websites.side_effect = Exception("DB Error")
        response = self.client.get("/search?t=test")
        self.assertEqual(response.status_code, 500)

    def test_terms_success(self) -> None:
        self.mock_db_service.get_website_terms.return_value = [
            {"term": "word", "occurrence": 5}
        ]
        response = self.client.get("/terms?w=http://site.com")

        self.assertEqual(response.status_code, 200)
        self.assertEqual(
            response.json(), {"terms": [{"term": "word", "occurrence": 5}]}
        )
        self.mock_db_service.get_website_terms.assert_called_once_with(
            "http://site.com"
        )

    def test_terms_missing_param(self) -> None:
        response = self.client.get("/terms?w=")
        self.assertEqual(response.status_code, 400)

    def test_terms_error(self) -> None:
        self.mock_db_service.get_website_terms.side_effect = Exception("DB Error")
        response = self.client.get("/terms?w=http://site.com")
        self.assertEqual(response.status_code, 500)

    def test_get_scrape_status_pending(self) -> None:
        self.mock_scraper_service.get_scraping_status.return_value = {
            "status": "PENDING",
            "id": 123,
        }
        response = self.client.get("/scrape?scraping_id=123")

        self.assertEqual(response.status_code, 200)
        # Check basic structure
        self.assertEqual(response.json()["status"], "PENDING")
        self.assertEqual(response.json()["scraping"]["id"], 123)
        self.assertNotIn("data", response.json())
        self.mock_scraper_service.get_scraping_status.assert_called_once_with(123)

    def test_get_scrape_status_completed(self) -> None:
        self.mock_scraper_service.get_scraping_status.return_value = {
            "status": "COMPLETED",
            "id": 123,
        }
        self.mock_scraper_service.get_scraping_results.return_value = [
            {"url": "http://foo.com", "terms": []}
        ]

        response = self.client.get("/scrape?scraping_id=123")

        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()["status"], "COMPLETED")
        self.assertEqual(
            response.json()["data"], [{"url": "http://foo.com", "terms": []}]
        )

    def test_get_scrape_status_not_found(self) -> None:
        self.mock_scraper_service.get_scraping_status.return_value = None
        response = self.client.get("/scrape?scraping_id=999")
        self.assertEqual(response.status_code, 404)
        self.assertEqual(response.json()["detail"], "Scraping not found")

    def test_get_scrape_status_error(self) -> None:
        self.mock_scraper_service.get_scraping_status.side_effect = Exception(
            "Unexpected"
        )
        response = self.client.get("/scrape?scraping_id=123")
        self.assertEqual(response.status_code, 500)
        self.assertIn("Unexpected", response.json()["detail"])
