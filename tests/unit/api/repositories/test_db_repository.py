import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from api.repositories.db_repository import DbRepository


class TestDbRepository(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        self.repo = DbRepository()

    @patch("api.models.ScrapedPage.filter")
    async def test_find_websites_by_term(self, mock_filter: MagicMock) -> None:
        # filter(...).distinct().values_list(...)
        mock_qs = MagicMock()
        mock_filter.return_value = mock_qs
        mock_distinct = MagicMock()
        mock_qs.distinct.return_value = mock_distinct

        # values_list should be awaited?
        # In Tortoise, filter() returns QuerySet. Await happens on execution
        # method or await queryset. But values_list() returns ValuesListQuery
        # which is awaitable. So we mock the return of values_list logic.

        # Actually repo code: await models.ScrapedPage.filter(
        # terms__term=term).distinct().values_list("url", flat=True)
        # So values_list returns an awaitable that yields result.

        mock_values_list = AsyncMock(
            return_value=["http://site1.com", "http://site2.com"]
        )
        mock_distinct.values_list.return_value = (
            mock_values_list()
        )  # Wait, it calls it and awaits result.
        # Correct mock: values_list.return_value must be an Awaitable
        # that returns the list.

        # Actually simplest way:
        mock_distinct.values_list = AsyncMock(
            return_value=["http://site1.com", "http://site2.com"]
        )

        websites = await self.repo.find_websites_by_term("test")

        self.assertEqual(len(websites), 2)
        self.assertEqual(websites[0], "http://site1.com")
        mock_filter.assert_called_once_with(terms__term="test")

    @patch("api.models.PageTerm.filter")
    async def test_find_terms_by_website(self, mock_filter: MagicMock) -> None:
        # await models.PageTerm.filter(...).values("term", "frequency")
        mock_qs = MagicMock()
        mock_filter.return_value = mock_qs

        mock_qs.values = AsyncMock(
            return_value=[
                {"term": "term1", "frequency": 5},
                {"term": "term2", "frequency": 3},
            ]
        )

        terms = await self.repo.find_terms_by_website("http://site.com")

        self.assertEqual(len(terms), 2)
        self.assertEqual(terms[0]["term"], "term1")
        self.assertEqual(terms[0]["occurrence"], 5)
        mock_filter.assert_called_once_with(page__url="http://site.com")

    @patch("api.models.Scraping.create", new_callable=AsyncMock)
    async def test_create_scraping(self, mock_create: AsyncMock) -> None:
        mock_scraping = MagicMock()
        mock_scraping.id = 123
        mock_create.return_value = mock_scraping

        result = await self.repo.create_scraping("http://url.com")
        self.assertEqual(result, 123)
        mock_create.assert_called_once_with(url="http://url.com", status="PENDING")

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
    async def test_get_scrape_results(self, mock_filter: MagicMock) -> None:
        # Mock QuerySet returned by filter()
        mock_qs = MagicMock()
        mock_filter.return_value = mock_qs

        # Mock count() - awaitable
        mock_qs.count = AsyncMock(return_value=1)

        # Mock chaining: filter().order_by().prefetch_related()
        mock_order = MagicMock()
        mock_qs.order_by.return_value = mock_order

        # Mock result data
        page1 = MagicMock()
        page1.url = "http://site1.com"
        term1 = MagicMock()
        term1.term = "t1"
        term1.frequency = 10
        page1.terms = [term1]
        
        image1 = MagicMock()
        image1.image_url = "http://img.com"
        image1.explanation = "desc"
        page1.images = [image1]

        # prefetch_related is awaited and returns the list of pages
        mock_order.prefetch_related = AsyncMock(return_value=[page1])

        results = await self.repo.get_scrape_results(123)

        self.assertEqual(len(results), 1)
        self.assertEqual(results[0]["url"], "http://site1.com")
        self.assertEqual(results[0]["terms"][0]["term"], "t1")
        self.assertEqual(results[0]["images"][0]["url"], "http://img.com")
        
        # Verify calls
        # Note: filter is called twice. Once for count, once for fetching.
        # Ideally we check call args or count.
        self.assertEqual(mock_filter.call_count, 2)
        mock_qs.count.assert_called_once()
    
    @patch("api.models.ScrapedPage.filter")
    async def test_get_scrape_results_empty(self, mock_filter: MagicMock) -> None:
        # Test optimization: count() == 0 returns []
        mock_qs = MagicMock()
        mock_filter.return_value = mock_qs
        mock_qs.count = AsyncMock(return_value=0)

        results = await self.repo.get_scrape_results(123)

        self.assertEqual(results, [])
        mock_qs.count.assert_called_once()
        # Ensure chain was NOT called
        mock_qs.order_by.assert_not_called()


if __name__ == "__main__":
    unittest.main()
