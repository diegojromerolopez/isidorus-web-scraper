import unittest
from unittest.mock import AsyncMock

from api.services.scraper_service import ScraperService


class TestScraperService(unittest.IsolatedAsyncioTestCase):
    async def test_start_scraping(self) -> None:
        mock_sqs_client = AsyncMock()
        mock_redis_client = AsyncMock()
        mock_db_repository = AsyncMock()

        mock_db_repository.create_scraping.return_value = 123

        service = ScraperService(
            mock_sqs_client,
            mock_redis_client,
            mock_db_repository,
            deletion_queue_url="http://deletion-q",
        )

        url = "http://example.com"
        depth = 2

        scraping_id = await service.start_scraping(url, depth)

        self.assertEqual(scraping_id, 123)

        # Verify DB Global ID Creation
        mock_db_repository.create_scraping.assert_called_once_with(url, None)

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
            mock_sqs_client,
            mock_redis_client,
            mock_db_repository,
            mock_dynamodb_client,
            AsyncMock(),
        )

        url = "http://example.com"
        depth = 2

        await service.start_scraping(url, depth)

        # Verify DynamoDB Log
        mock_dynamodb_client.put_item.assert_called_once()
        put_item_call = mock_dynamodb_client.put_item.call_args[0][0]
        self.assertEqual(put_item_call["scraping_id"], "123")
        self.assertEqual(put_item_call["url"], url)
        self.assertEqual(put_item_call["status"], "PENDING")
        self.assertIn("created_at", put_item_call)

    async def test_get_scraping_status(self) -> None:
        mock_sqs_client = AsyncMock()
        mock_redis_client = AsyncMock()
        mock_db_repository = AsyncMock()
        mock_dynamodb_client = AsyncMock()

        expected_pg = {
            "id": 123,
            "url": "http://example.com",
        }
        mock_db_repository.get_scraping.return_value = expected_pg

        expected_dynamo = {
            "status": "COMPLETED",
            "created_at": "2024-01-01T00:00:00",
            "completed_at": "2024-01-01T01:00:00",
        }
        mock_dynamodb_client.get_item.return_value = expected_dynamo

        service = ScraperService(
            mock_sqs_client,
            mock_redis_client,
            mock_db_repository,
            mock_dynamodb_client,
            AsyncMock(),
        )

        result = await service.get_full_scraping(123)

        expected_merged = {
            **expected_pg,
            **expected_dynamo,
            "depth": 1,
            "links_count": 0,
            "pages": None,
        }
        self.assertEqual(result, expected_merged)
        mock_db_repository.get_scraping.assert_called_once_with(123)
        mock_dynamodb_client.get_item.assert_called_once_with({"scraping_id": "123"})

    async def test_get_scraping_results(self) -> None:
        mock_sqs_client = AsyncMock()
        mock_redis_client = AsyncMock()
        mock_db_repository = AsyncMock()

        expected_results = [
            {"url": "http://example.com/page1", "term_count": 5},
            {"url": "http://example.com/page2", "term_count": 3},
        ]
        mock_db_repository.get_scraping_results.return_value = expected_results

        service = ScraperService(
            mock_sqs_client,
            mock_redis_client,
            mock_db_repository,
            deletion_queue_url="http://deletion-q",
        )

        result = await service.get_scraping_results(123)

        self.assertEqual(result, expected_results)
        mock_db_repository.get_scraping_results.assert_called_once_with(123)

    async def test_get_scraping_status_not_found(self) -> None:
        mock_db_repository = AsyncMock()
        mock_db_repository.get_scraping.return_value = None

        service = ScraperService(AsyncMock(), AsyncMock(), mock_db_repository)
        result = await service.get_full_scraping(999)
        self.assertIsNone(result)

    async def test_get_scraping_status_no_dynamo(self) -> None:
        mock_db_repository = AsyncMock()
        mock_db_repository.get_scraping.return_value = {
            "id": 123,
            "url": "http://x.com",
        }

        service = ScraperService(
            AsyncMock(),
            AsyncMock(),
            mock_db_repository,
            dynamodb_client=None,
            deletion_queue_url="http://deletion-q",
        )
        result = await service.get_full_scraping(123)
        self.assertIsNotNone(result)
        assert result is not None
        self.assertEqual(result["status"], "UNKNOWN")

    async def test_get_scraping_status_dynamo_no_item(self) -> None:
        mock_db_repository = AsyncMock()
        mock_dynamodb_client = AsyncMock()

        mock_db_repository.get_scraping.return_value = {
            "id": 123,
            "url": "http://x.com",
        }
        mock_dynamodb_client.get_item.return_value = None

        service = ScraperService(
            AsyncMock(),
            AsyncMock(),
            mock_db_repository,
            mock_dynamodb_client,
            deletion_queue_url="http://deletion-q",
        )
        result = await service.get_full_scraping(123)
        self.assertIsNotNone(result)
        assert result is not None
        self.assertEqual(result["status"], "UNKNOWN")

    async def test_delete_scraping_success(self) -> None:
        mock_sqs_client = AsyncMock()
        mock_db_repository = AsyncMock()
        mock_db_repository.get_scraping.return_value = {
            "id": 123,
            "url": "http://x.com",
            "user_id": 1,
        }

        service = ScraperService(
            mock_sqs_client,
            AsyncMock(),
            mock_db_repository,
            deletion_queue_url="http://deletion-q",
        )

        result = await service.delete_scraping(123, 1)

        self.assertTrue(result)
        mock_sqs_client.send_message.assert_called_once_with(
            {"scraping_id": 123}, queue_url="http://deletion-q"
        )

    async def test_delete_scraping_not_found(self) -> None:
        mock_db_repository = AsyncMock()
        mock_db_repository.get_scraping.return_value = None

        service = ScraperService(AsyncMock(), AsyncMock(), mock_db_repository)

        with self.assertRaises(Exception) as cm:
            await service.delete_scraping(999, 1)
        self.assertIn("not found", str(cm.exception))

    async def test_delete_scraping_not_authorized(self) -> None:
        mock_db_repository = AsyncMock()
        mock_db_repository.get_scraping.return_value = {
            "id": 123,
            "url": "http://x.com",
            "user_id": 1,
        }

        service = ScraperService(AsyncMock(), AsyncMock(), mock_db_repository)

        with self.assertRaises(Exception) as cm:
            await service.delete_scraping(123, 2)
        self.assertIn("not authorized", str(cm.exception))

    async def test_enqueue_deletion(self) -> None:
        mock_sqs_client = AsyncMock()
        service = ScraperService(
            mock_sqs_client,
            AsyncMock(),
            AsyncMock(),
            deletion_queue_url="http://deletion-q",
        )

        await service.enqueue_deletion(123)

        mock_sqs_client.send_message.assert_called_once_with(
            {"scraping_id": 123}, queue_url="http://deletion-q"
        )


if __name__ == "__main__":
    unittest.main()
