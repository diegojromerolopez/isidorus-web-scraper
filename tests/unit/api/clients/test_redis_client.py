import unittest
from unittest.mock import MagicMock, patch

from api.clients.redis_client import RedisClient


class TestRedisClient(unittest.TestCase):
    def setUp(self) -> None:
        self.mock_redis = MagicMock()
        with patch(
            "api.clients.redis_client.redis.Redis", return_value=self.mock_redis
        ):
            self.client = RedisClient(host="localhost", port=6379, db=0)

    def test_init(self) -> None:
        """Test RedisClient initialization"""
        with patch("api.clients.redis_client.redis.Redis") as mock_redis_class:
            client = RedisClient(host="testhost", port=1234, db=5)
            mock_redis_class.assert_called_once_with(host="testhost", port=1234, db=5)
            self.assertIsNotNone(client.client)

    def test_set(self) -> None:
        """Test set operation"""
        self.client.set("key1", "value1")
        self.mock_redis.set.assert_called_once_with("key1", "value1")

    def test_get_with_value(self) -> None:
        """Test get operation when value exists"""
        self.mock_redis.get.return_value = b"test_value"
        result = self.client.get("key1")
        self.assertEqual(result, "test_value")
        self.mock_redis.get.assert_called_once_with("key1")

    def test_get_with_none(self) -> None:
        """Test get operation when value is None"""
        self.mock_redis.get.return_value = None
        result = self.client.get("key1")
        self.assertIsNone(result)
        self.mock_redis.get.assert_called_once_with("key1")

    def test_incr_default(self) -> None:
        """Test increment with default amount"""
        self.mock_redis.incrby.return_value = 5
        result = self.client.incr("counter")
        self.assertEqual(result, 5)
        self.mock_redis.incrby.assert_called_once_with("counter", 1)

    def test_incr_custom_amount(self) -> None:
        """Test increment with custom amount"""
        self.mock_redis.incrby.return_value = 10
        result = self.client.incr("counter", 5)
        self.assertEqual(result, 10)
        self.mock_redis.incrby.assert_called_once_with("counter", 5)

    def test_decr_default(self) -> None:
        """Test decrement with default amount"""
        self.mock_redis.decrby.return_value = 3
        result = self.client.decr("counter")
        self.assertEqual(result, 3)
        self.mock_redis.decrby.assert_called_once_with("counter", 1)

    def test_decr_custom_amount(self) -> None:
        """Test decrement with custom amount"""
        self.mock_redis.decrby.return_value = 0
        result = self.client.decr("counter", 3)
        self.assertEqual(result, 0)
        self.mock_redis.decrby.assert_called_once_with("counter", 3)


if __name__ == "__main__":
    unittest.main()
