import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from api.clients.sqs_client import SQSClient
from api.dependencies import (
    get_db_repository,
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
        from api.clients.dynamodb_client import DynamoDBClient
        from api.clients.redis_client import RedisClient

        mock_sqs = MagicMock(spec=SQSClient)
        mock_redis = MagicMock(spec=RedisClient)
        mock_dynamo = MagicMock(spec=DynamoDBClient)
        mock_repo = MagicMock(spec=DbRepository)

        service = get_scraper_service(mock_sqs, mock_redis, mock_dynamo, mock_repo)
        self.assertEqual(service.sqs_client, mock_sqs)
        self.assertEqual(service.db_repository, mock_repo)

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

    async def test_get_api_key_missing(self) -> None:
        from fastapi import HTTPException

        from api.dependencies import get_api_key

        with self.assertRaises(HTTPException) as cm:
            await get_api_key(api_key_header=None, redis_client=MagicMock())
        self.assertEqual(cm.exception.status_code, 401)

    @patch("api.models.APIKey.filter")
    async def test_get_api_key_valid_db(self, mock_filter: MagicMock) -> None:
        import hashlib

        from api.dependencies import get_api_key
        from api.models import APIKey

        mock_redis = AsyncMock()
        mock_redis.get.return_value = None

        key = "test-key"
        hashed = hashlib.sha256(key.encode()).hexdigest()
        mock_api_key = MagicMock(spec=APIKey)
        mock_api_key.name = "Test Key"
        mock_api_key.user_id = 1
        mock_api_key.expires_at = None

        mock_filter.return_value.first = AsyncMock(return_value=mock_api_key)

        result = await get_api_key(api_key_header=key, redis_client=mock_redis)

        self.assertEqual(result, mock_api_key)
        mock_redis.set.assert_called_once()
        mock_filter.assert_called_once_with(hashed_key=hashed, is_active=True)

    async def test_get_api_key_valid_cache(self) -> None:
        from api.dependencies import get_api_key

        mock_redis = AsyncMock()
        mock_redis.get.return_value = "Cached Name:1"

        result = await get_api_key(api_key_header="some-key", redis_client=mock_redis)
        self.assertEqual(result.name, "Cached Name")
        self.assertEqual(result.user_id, 1)
        self.assertTrue(result.is_active)

    @patch("api.models.APIKey.filter")
    async def test_get_api_key_invalid(self, mock_filter: MagicMock) -> None:
        from fastapi import HTTPException

        from api.dependencies import get_api_key

        mock_redis = AsyncMock()
        mock_redis.get.return_value = None
        mock_filter.return_value.first = AsyncMock(return_value=None)

        with self.assertRaises(HTTPException) as cm:
            await get_api_key(api_key_header="invalid", redis_client=mock_redis)
        self.assertEqual(cm.exception.status_code, 401)
        self.assertEqual(cm.exception.detail, "Invalid API Key")

    @patch("api.models.APIKey.filter")
    async def test_get_api_key_expired(self, mock_filter: MagicMock) -> None:
        from datetime import datetime, timedelta, timezone

        from fastapi import HTTPException

        from api.dependencies import get_api_key
        from api.models import APIKey

        mock_redis = AsyncMock()
        mock_redis.get.return_value = None

        mock_api_key = MagicMock(spec=APIKey)
        mock_api_key.expires_at = datetime.now(timezone.utc) - timedelta(days=1)
        mock_api_key.user_id = 1

        mock_filter.return_value.first = AsyncMock(return_value=mock_api_key)

        with self.assertRaises(HTTPException) as cm:
            await get_api_key(api_key_header="expired", redis_client=mock_redis)
        self.assertEqual(cm.exception.status_code, 401)
        self.assertEqual(cm.exception.detail, "API Key expired")

    @patch("api.models.APIKey.filter")
    async def test_get_api_key_inactive(self, mock_filter: MagicMock) -> None:
        from fastapi import HTTPException

        from api.dependencies import get_api_key

        mock_redis = AsyncMock()
        mock_redis.get.return_value = None
        # Inactive key will not be found because we filter by is_active=True
        mock_filter.return_value.first = AsyncMock(return_value=None)

        with self.assertRaises(HTTPException) as cm:
            await get_api_key(api_key_header="inactive", redis_client=mock_redis)
        self.assertEqual(cm.exception.status_code, 401)
        self.assertEqual(cm.exception.detail, "Invalid API Key")

    @patch("api.models.APIKey.filter")
    async def test_get_api_key_future_expiry(self, mock_filter: MagicMock) -> None:
        from datetime import datetime, timedelta, timezone

        from api.dependencies import get_api_key
        from api.models import APIKey

        mock_redis = AsyncMock()
        mock_redis.get.return_value = None

        mock_api_key = MagicMock(spec=APIKey)
        mock_api_key.name = "Future Key"
        mock_api_key.user_id = 1
        mock_api_key.expires_at = datetime.now(timezone.utc) + timedelta(days=1)

        mock_filter.return_value.first = AsyncMock(return_value=mock_api_key)

        result = await get_api_key(api_key_header="future", redis_client=mock_redis)
        self.assertEqual(result, mock_api_key)
        self.assertEqual(result.user_id, 1)
