import unittest
from unittest.mock import AsyncMock, patch

from api.clients.redis_client import RedisClient


class TestRedisClient(unittest.IsolatedAsyncioTestCase):
    # pylint: disable=protected-access
    async def asyncSetUp(self) -> None:
        self.mock_redis = AsyncMock()
        # Mock class Redis to return our AsyncMock instance
        self.patcher = patch(
            "api.clients.redis_client.redis.Redis", return_value=self.mock_redis
        )
        self.mock_redis_class = self.patcher.start()
        self.client = RedisClient(host="localhost", port=6379, db=0)

    async def asyncTearDown(self) -> None:
        self.patcher.stop()

    async def test_init(self) -> None:
        """Test RedisClient initialization"""
        with patch("api.clients.redis_client.redis.Redis") as mock_redis_val:
            client = RedisClient(host="testhost", port=1234, db=5)
            mock_redis_val.assert_called_with(host="testhost", port=1234, db=5)
            self.assertIsNotNone(
                client._RedisClient__client  # type: ignore[attr-defined]
            )

    async def test_set(self) -> None:
        """Test set operation"""
        await self.client.set("key1", "value1")
        self.mock_redis.set.assert_called_once_with("key1", "value1", ex=None)

    async def test_set_with_expiration(self) -> None:
        """Test set operation with expiration"""
        await self.client.set("key2", "value2", ex=60)
        self.mock_redis.set.assert_called_once_with("key2", "value2", ex=60)

    async def test_get_with_value(self) -> None:
        """Test get operation when value exists"""
        self.mock_redis.get.return_value = b"test_value"
        result = await self.client.get("key1")
        self.assertEqual(result, "test_value")
        self.mock_redis.get.assert_called_once_with("key1")

    async def test_get_with_none(self) -> None:
        """Test get operation when value is None"""
        self.mock_redis.get.return_value = None
        result = await self.client.get("key1")
        self.assertIsNone(result)
        self.mock_redis.get.assert_called_once_with("key1")

    async def test_incr_default(self) -> None:
        """Test increment with default amount"""
        self.mock_redis.incrby.return_value = 5
        result = await self.client.incr("counter")
        self.assertEqual(result, 5)
        self.mock_redis.incrby.assert_called_once_with("counter", 1)

    async def test_incr_custom_amount(self) -> None:
        """Test increment with custom amount"""
        self.mock_redis.incrby.return_value = 10
        result = await self.client.incr("counter", 5)
        self.assertEqual(result, 10)
        self.mock_redis.incrby.assert_called_once_with("counter", 5)

    async def test_decr_default(self) -> None:
        """Test decrement with default amount"""
        self.mock_redis.decrby.return_value = 3
        result = await self.client.decr("counter")
        self.assertEqual(result, 3)
        self.mock_redis.decrby.assert_called_once_with("counter", 1)

    async def test_decr_custom_amount(self) -> None:
        """Test decrement with custom amount"""
        self.mock_redis.decrby.return_value = 0
        result = await self.client.decr("counter", 3)
        self.assertEqual(result, 0)
        self.mock_redis.decrby.assert_called_once_with("counter", 3)


if __name__ == "__main__":
    unittest.main()
