import unittest
from unittest.mock import AsyncMock

from api.services.scraper_service import ScraperService


class TestScraperService(unittest.IsolatedAsyncioTestCase):
    async def test_start_scraping(self) -> None:
        mock_sqs_client = AsyncMock()
        mock_redis_client = AsyncMock()
        mock_db_repository = AsyncMock()

        mock_db_repository.create_scraping.return_value = 123

        service = ScraperService(mock_sqs_client, mock_redis_client, mock_db_repository)

        url = "http://example.com"
        depth = 2

        scraping_id = await service.start_scraping(url, depth)

        self.assertEqual(scraping_id, 123)

        # Verify DB Global ID Creation
        mock_db_repository.create_scraping.assert_called_once_with(url)

        # Verify Redis Initialization
        mock_redis_client.set.assert_called_once_with("scrape:123:pending", 1)

        # Verify SQS interaction
        mock_sqs_client.send_message.assert_called_once()
        call_msg = mock_sqs_client.send_message.call_args[0][0]

        self.assertEqual(call_msg["url"], url)
        self.assertEqual(call_msg["depth"], depth)
        self.assertEqual(call_msg["scraping_id"], 123)

    async def test_start_scraping_with_dynamodb(self) -> None:
        mock_sqs_client = AsyncMock()
        mock_redis_client = AsyncMock()
        mock_db_repository = AsyncMock()
        mock_dynamodb_client = AsyncMock()

        mock_db_repository.create_scraping.return_value = 123

        service = ScraperService(
            mock_sqs_client, mock_redis_client, mock_db_repository, mock_dynamodb_client
        )

        url = "http://example.com"
        depth = 2

        await service.start_scraping(url, depth)

        # Verify DynamoDB Log
        mock_dynamodb_client.put_item.assert_called_once_with(
            {
                "job_id": "123",
                "url": url,
                "depth": depth,
                "status": "PENDING",
            }
        )


if __name__ == "__main__":
    unittest.main()
