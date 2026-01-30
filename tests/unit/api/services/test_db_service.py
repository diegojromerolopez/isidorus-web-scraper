import unittest
from unittest.mock import AsyncMock

from api.services.db_service import DbService


class TestDbService(unittest.IsolatedAsyncioTestCase):
    async def test_search_websites(self) -> None:
        mock_repo = AsyncMock()
        mock_repo.find_websites_by_term.return_value = ["http://site1.com"]

        service = DbService(mock_repo)
        result = await service.search_websites("test")

        self.assertEqual(result, ["http://site1.com"])
        mock_repo.find_websites_by_term.assert_called_once_with("test")

    async def test_get_website_terms(self) -> None:
        mock_repo = AsyncMock()
        mock_repo.find_terms_by_website.return_value = [{"term": "t1", "occurrence": 1}]

        service = DbService(mock_repo)
        result = await service.get_website_terms("http://site.com")

        self.assertEqual(len(result), 1)
        self.assertEqual(result[0]["term"], "t1")
        mock_repo.find_terms_by_website.assert_called_once_with("http://site.com")


if __name__ == "__main__":
    unittest.main()
