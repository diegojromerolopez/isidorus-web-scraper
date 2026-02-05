import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from api.repositories.db_repository import DbRepository


class TestDbRepository(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        self.repo = DbRepository()

    @patch("api.models.Scraping.create", new_callable=AsyncMock)
    async def test_create_scraping(self, mock_create: AsyncMock) -> None:
        mock_scraping = MagicMock()
        mock_scraping.id = 123
        mock_create.return_value = mock_scraping

        result = await self.repo.create_scraping("http://url.com")
        self.assertEqual(result, 123)
        mock_create.assert_called_once_with(url="http://url.com", user_id=None)

    @patch("api.models.Scraping.get_or_none", new_callable=AsyncMock)
    async def test_delete_scraping(self, mock_get: AsyncMock) -> None:
        mock_scraping = AsyncMock()
        mock_get.return_value = mock_scraping

        result = await self.repo.delete_scraping(123)
        self.assertTrue(result)
        mock_get.assert_called_once_with(id=123)
        mock_scraping.delete.assert_called_once()

    @patch("api.models.PageImage.filter")
    async def test_get_scraping_s3_paths(self, mock_filter: MagicMock) -> None:
        mock_qs = MagicMock()
        mock_filter.return_value = mock_qs
        mock_qs.values_list = AsyncMock(return_value=["s3://b/k1", None, "s3://b/k2"])

        paths = await self.repo.get_scraping_s3_paths(123)
        self.assertEqual(paths, ["s3://b/k1", "s3://b/k2"])
        mock_filter.assert_called_once_with(scraping_id=123)

    @patch("api.models.Scraping.get_or_none", new_callable=AsyncMock)
    async def test_get_scraping(self, mock_get: AsyncMock) -> None:
        mock_scraping = MagicMock()
        mock_scraping.id = 123
        mock_scraping.url = "http://url.com"
        mock_scraping.status = "PENDING"
        # Dates...
        mock_get.return_value = mock_scraping

        result = await self.repo.get_scraping(123)
        self.assertIsNotNone(result)
        if result:
            self.assertEqual(result["id"], 123)
        mock_get.assert_called_once_with(id=123)

    @patch("api.models.Scraping.get_or_none", new_callable=AsyncMock)
    async def test_get_scraping_not_found(self, mock_get: AsyncMock) -> None:
        mock_get.return_value = None
        result = await self.repo.get_scraping(999)
        self.assertIsNone(result)

    @patch("api.models.ScrapedPage.filter")
    async def test_get_scraping_results(self, mock_filter: MagicMock) -> None:
        mock_qs = MagicMock()
        mock_filter.return_value = mock_qs
        mock_order = MagicMock()
        mock_qs.order_by.return_value = mock_order

        page1 = MagicMock()
        page1.url = "http://site1.com"
        page1.summary = "sum"
        image1 = MagicMock()
        image1.image_url = "http://img.com"
        image1.explanation = "desc"
        page1.images = [image1]

        # prefetch_related is awaited and returns the list of pages
        mock_order.prefetch_related = AsyncMock(return_value=[page1])

        results = await self.repo.get_scraping_results(123)

        self.assertEqual(len(results), 1)
        self.assertEqual(results[0]["url"], "http://site1.com")
        self.assertEqual(results[0]["summary"], "sum")
        self.assertEqual(results[0]["images"][0]["url"], "http://img.com")

        mock_filter.assert_called_once_with(scraping_id=123)


if __name__ == "__main__":
    unittest.main()
