import unittest
from unittest.mock import MagicMock, patch

from api.clients.sqs_client import SQSClient
from api.dependencies import (
    get_db_repository,
    get_db_service,
    get_scraper_service,
    get_sqs_client,
)
from api.repositories.db_repository import DbRepository


# pylint: disable=import-outside-toplevel
# pylint: disable=import-outside-toplevel
class TestDependencies(unittest.IsolatedAsyncioTestCase):
    @patch("shared.clients.sqs_client.aioboto3.Session")
    def test_get_sqs_client(self, _mock_session: MagicMock) -> None:
        client = get_sqs_client()
        self.assertIsInstance(client, SQSClient)

    def test_get_db_repository(self) -> None:
        repo = get_db_repository()
        self.assertIsInstance(repo, DbRepository)

    def test_get_scraper_service(self) -> None:
        mock_sqs = MagicMock(spec=SQSClient)
        service = get_scraper_service(mock_sqs)
        self.assertEqual(service.sqs_client, mock_sqs)

    def test_get_db_service(self) -> None:
        mock_repo = MagicMock(spec=DbRepository)
        service = get_db_service(mock_repo)
        self.assertEqual(service.data_repo, mock_repo)

    @patch("api.clients.redis_client.redis.Redis")
    def test_get_redis_client(self, _mock_redis: MagicMock) -> None:
        from api.clients.redis_client import RedisClient
        from api.dependencies import get_redis_client

        client = get_redis_client()
        self.assertIsInstance(client, RedisClient)

    @patch("api.clients.dynamodb_client.aioboto3.Session")
    def test_get_dynamodb_client(self, _mock_session: MagicMock) -> None:
        from api.clients.dynamodb_client import DynamoDBClient
        from api.dependencies import get_dynamodb_client

        client = get_dynamodb_client()
        self.assertIsInstance(client, DynamoDBClient)
