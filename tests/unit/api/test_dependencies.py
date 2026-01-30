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
        self.assertEqual(client.queue_url, "http://sqs")

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
