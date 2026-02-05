import unittest
from unittest.mock import AsyncMock

from api.services.search_service import SearchService


class TestSearchService(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        self.mock_repository = AsyncMock()
        self.service = SearchService(self.mock_repository)

    async def test_search_pages_success(self) -> None:
        # Mock OpenSearch response
        self.mock_repository.search.return_value = {
            "hits": {
                "hits": [
                    {
                        "_source": {
                            "url": "http://example.com",
                            "scraping_id": 123,
                            "created_at": "2026-02-05T20:00:00Z",
                        },
                        "highlight": {
                            "content": ["<em>test</em> snippet"],
                            "summary": ["<em>test</em> summary"],
                        },
                    }
                ]
            }
        }

        results = await self.service.search_pages("test", 1)

        self.assertEqual(len(results), 1)
        self.assertEqual(results[0]["url"], "http://example.com")
        self.assertEqual(results[0]["scraping_id"], 123)
        self.assertEqual(results[0]["created_at"], "2026-02-05T20:00:00Z")
        self.assertEqual(len(results[0]["highlights"]), 2)

        self.mock_repository.search.assert_called_once()
        call_args = self.mock_repository.search.call_args
        self.assertEqual(call_args.kwargs["index"], "scraped_pages")
        query = call_args.kwargs["body"]
        self.assertEqual(
            query["query"]["bool"]["must"][0]["multi_match"]["query"], "test"
        )
        self.assertEqual(query["query"]["bool"]["must"][1]["term"]["user_id"], 1)

    async def test_search_pages_no_highlights(self) -> None:
        self.mock_repository.search.return_value = {
            "hits": {
                "hits": [{"_source": {"url": "http://example.com", "scraping_id": 123}}]
            }
        }

        results = await self.service.search_pages("test", 1)
        self.assertEqual(len(results), 1)
        self.assertEqual(results[0]["highlights"], [])

    async def test_search_pages_limit_highlights(self) -> None:
        self.mock_repository.search.return_value = {
            "hits": {
                "hits": [
                    {
                        "_source": {"url": "http://example.com", "scraping_id": 123},
                        "highlight": {"content": ["h1", "h2", "h3", "h4"]},
                    }
                ]
            }
        }

        results = await self.service.search_pages("test", 1)
        self.assertEqual(len(results[0]["highlights"]), 3)
