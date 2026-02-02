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


class TestDependencies(unittest.TestCase):
    @patch("api.dependencies.AWS_ENDPOINT_URL", "http://localstack")
    @patch("api.dependencies.AWS_REGION", "us-east-1")
    @patch("api.dependencies.AWS_ACCESS_KEY_ID", "key")
    @patch("api.dependencies.AWS_SECRET_ACCESS_KEY", "secret")
    @patch("api.dependencies.SQS_QUEUE_URL", "http://sqs")
    def test_get_sqs_client(self) -> None:
        client = get_sqs_client()
        self.assertIsInstance(client, SQSClient)

    @patch("api.dependencies.DATABASE_URL", "postgresql://user:pass@host/db")
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

    @patch("api.dependencies.DATABASE_URL", None)
    def test_get_db_repository_no_url(self) -> None:
        with self.assertRaises(ValueError):
            get_db_repository()

    @patch("api.dependencies.REDIS_HOST", "localhost")
    @patch("api.dependencies.REDIS_PORT", 6379)
    def test_get_redis_client(self) -> None:
        from api.clients.redis_client import RedisClient
        from api.dependencies import get_redis_client

        client = get_redis_client()
        self.assertIsInstance(client, RedisClient)

    @patch("api.dependencies.AWS_ENDPOINT_URL", "http://localstack")
    @patch("api.dependencies.AWS_REGION", "us-east-1")
    @patch("api.dependencies.AWS_ACCESS_KEY_ID", "key")
    @patch("api.dependencies.AWS_SECRET_ACCESS_KEY", "secret")
    @patch("api.dependencies.DYNAMODB_TABLE", "jobs")
    def test_get_dynamodb_client(self) -> None:
        from api.clients.dynamodb_client import DynamoDBClient
        from api.dependencies import get_dynamodb_client

        client = get_dynamodb_client()
        self.assertIsInstance(client, DynamoDBClient)
