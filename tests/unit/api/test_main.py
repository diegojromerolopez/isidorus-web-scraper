import unittest
from unittest.mock import AsyncMock, MagicMock

from fastapi.testclient import TestClient

from api.dependencies import (
    get_api_key,
    get_db_service,
    get_scraper_service,
    get_search_service,
)
from api.main import app
from api.services.scraper_service import NotAuthorizedError, ScrapingNotFoundError


class TestMain(unittest.TestCase):
    def setUp(self) -> None:
        self.client = TestClient(app)
        self.mock_scraper_service = AsyncMock()
        self.mock_db_service = AsyncMock()
        self.mock_search_service = AsyncMock()
        self.mock_api_key = MagicMock()
        self.mock_api_key.user_id = 1
        self.mock_db_repository = AsyncMock()

        # Override dependencies
        app.dependency_overrides[get_scraper_service] = (
            lambda: self.mock_scraper_service
        )
        app.dependency_overrides[get_db_service] = lambda: self.mock_db_service
        app.dependency_overrides[get_search_service] = lambda: self.mock_search_service
        app.dependency_overrides[get_api_key] = lambda: self.mock_api_key
        from api.dependencies import (  # pylint: disable=import-outside-toplevel
            get_db_repository,
        )

        app.dependency_overrides[get_db_repository] = lambda: self.mock_db_repository

    def tearDown(self) -> None:
        app.dependency_overrides = {}

    def test_health_check(self) -> None:
        response = self.client.get("/health")
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json(), {"status": "ok"})

    def test_scrape_success(self) -> None:
        self.mock_scraper_service.start_scraping.return_value = 123
        response = self.client.post(
            "/scrape", json={"url": "http://example.com", "depth": 2}
        )

        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json(), {"scraping_id": 123})
        self.mock_scraper_service.start_scraping.assert_called_once_with(
            "http://example.com", 2, 1
        )

    def test_scrape_error(self) -> None:
        self.mock_scraper_service.start_scraping.side_effect = Exception("SQS Error")
        response = self.client.post(
            "/scrape", json={"url": "http://example.com", "depth": 2}
        )

        self.assertEqual(response.status_code, 500)
        self.assertIn("SQS Error", response.json()["detail"])

    def test_search_success(self) -> None:
        self.mock_search_service.search_pages.return_value = [
            {
                "url": "http://site1.com",
                "scraping_id": 1,
                "created_at": "2024-01-01",
                "highlights": ["<em>test</em>"],
            }
        ]
        response = self.client.get("/search?t=test")

        self.assertEqual(response.status_code, 200)
        self.assertEqual(len(response.json()["results"]), 1)
        self.assertEqual(response.json()["results"][0]["url"], "http://site1.com")
        self.mock_search_service.search_pages.assert_called_once_with("test", 1)

    def test_search_missing_param(self) -> None:
        response = self.client.get("/search?t=")
        self.assertEqual(response.status_code, 400)

    def test_search_error(self) -> None:
        self.mock_search_service.search_pages.side_effect = Exception("OS Error")
        response = self.client.get("/search?t=test")
        self.assertEqual(response.status_code, 500)
        self.assertIn("OS Error", response.json()["detail"])

    def test_get_scrape_status_pending(self) -> None:
        self.mock_scraper_service.get_full_scraping.return_value = {
            "id": 123,
            "url": "http://foo.com",
            "user_id": 1,
            "status": "PENDING",
            "created_at": None,
            "completed_at": None,
            "depth": 1,
            "links_count": 0,
        }
        response = self.client.get("/scraping/123")

        self.assertEqual(response.status_code, 200)
        # Check basic structure
        self.assertEqual(response.json()["scraping"]["status"], "PENDING")
        self.assertEqual(response.json()["scraping"]["id"], 123)
        self.assertEqual(response.json()["scraping"]["pages"], [])
        self.mock_scraper_service.get_full_scraping.assert_called_once_with(123)

    def test_get_scrape_status_completed(self) -> None:
        self.mock_scraper_service.get_full_scraping.return_value = {
            "id": 123,
            "url": "http://foo.com",
            "user_id": 1,
            "status": "COMPLETED",
            "created_at": None,
            "completed_at": None,
            "depth": 1,
            "links_count": 5,
        }
        self.mock_scraper_service.get_scraping_results.return_value = [
            {"url": "http://foo.com", "images": [], "summary": None}
        ]

        response = self.client.get("/scraping/123")

        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()["scraping"]["status"], "COMPLETED")
        self.assertEqual(
            response.json()["scraping"]["pages"],
            [{"url": "http://foo.com", "images": [], "summary": None}],
        )

    def test_get_scrape_status_not_found(self) -> None:
        self.mock_scraper_service.get_full_scraping.return_value = None
        response = self.client.get("/scraping/999")
        self.assertEqual(response.status_code, 404)
        self.assertEqual(response.json()["detail"], "Scraping not found")

    def test_get_scrape_status_error(self) -> None:
        self.mock_scraper_service.get_full_scraping.side_effect = Exception(
            "Unexpected"
        )
        response = self.client.get("/scraping/123")
        self.assertEqual(response.status_code, 500)
        self.assertIn("Unexpected", response.json()["detail"])

    def test_scrapings_success(self) -> None:
        self.mock_scraper_service.get_full_scrapings.return_value = (
            [
                {
                    "id": 123,
                    "url": "http://foo.com",
                    "user_id": 1,
                    "status": "COMPLETED",
                    "created_at": None,
                    "completed_at": None,
                    "depth": 1,
                    "links_count": 5,
                    "pages": None,
                }
            ],
            1,
        )
        response = self.client.get("/scrapings?page=1&size=10")

        self.assertEqual(response.status_code, 200)
        self.assertEqual(len(response.json()["scrapings"]), 1)
        self.assertEqual(response.json()["meta"]["total"], 1)
        self.mock_scraper_service.get_full_scrapings.assert_called_once_with(
            user_id=1, offset=0, limit=10
        )

    def test_delete_scraping_success(self) -> None:
        self.mock_scraper_service.delete_scraping.return_value = True

        response = self.client.delete("/scraping/123")

        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json(), {"message": "Scraping deletion enqueued"})
        self.mock_scraper_service.delete_scraping.assert_called_once_with(123, 1)

    def test_delete_scraping_not_found(self) -> None:
        self.mock_scraper_service.delete_scraping.side_effect = ScrapingNotFoundError(
            "not found"
        )
        response = self.client.delete("/scraping/999")
        self.assertEqual(response.status_code, 404)
        self.assertEqual(response.json()["detail"], "not found")

    def test_delete_scraping_unauthorized(self) -> None:
        self.mock_scraper_service.delete_scraping.side_effect = NotAuthorizedError(
            "not authorized"
        )
        response = self.client.delete("/scraping/123")
        self.assertEqual(response.status_code, 403)
        self.assertEqual(response.json()["detail"], "not authorized")
