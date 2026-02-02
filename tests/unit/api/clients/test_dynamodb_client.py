import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from api.clients.dynamodb_client import DynamoDBClient


class TestDynamoDBClient(unittest.IsolatedAsyncioTestCase):
    async def asyncSetUp(self) -> None:
        self.endpoint_url = "http://localhost:4566"
        self.region = "us-east-1"
        self.access_key = "test"
        self.secret_key = "test"
        self.table_name = "test-table"

    @patch("api.clients.dynamodb_client.aioboto3.Session")
    async def test_init(self, mock_session_cls: MagicMock) -> None:
        mock_session = MagicMock()
        mock_session_cls.return_value = mock_session

        _ = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )
        mock_session_cls.assert_called_once()

    @patch("api.clients.dynamodb_client.aioboto3.Session")
    async def test_put_item_success(self, mock_session_cls: MagicMock) -> None:
        mock_table = AsyncMock()
        mock_dynamodb = AsyncMock()
        mock_dynamodb.Table.return_value = mock_table

        mock_resource_cm = MagicMock()
        mock_resource_cm.__aenter__.return_value = mock_dynamodb
        mock_resource_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.resource.return_value = mock_resource_cm
        mock_session_cls.return_value = mock_session

        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        item = {"id": "1", "data": "value"}
        result = await client.put_item(item)

        self.assertTrue(result)
        mock_table.put_item.assert_called_once_with(Item=item)
        mock_session.resource.assert_called_once()

    @patch("api.clients.dynamodb_client.aioboto3.Session")
    async def test_put_item_failure(self, mock_session_cls: MagicMock) -> None:
        mock_resource_cm = MagicMock()
        mock_resource_cm.__aenter__.side_effect = Exception("DynamoDB error")
        mock_resource_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.resource.return_value = mock_resource_cm
        mock_session_cls.return_value = mock_session

        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        with self.assertRaisesRegex(Exception, "DynamoDB error"):
            await client.put_item({"id": "1"})

    @patch("api.clients.dynamodb_client.aioboto3.Session")
    async def test_get_item_success(self, mock_session_cls: MagicMock) -> None:
        mock_table = AsyncMock()
        expected_item = {"id": "1", "data": "value"}
        mock_table.get_item.return_value = {"Item": expected_item}

        mock_dynamodb = AsyncMock()
        mock_dynamodb.Table.return_value = mock_table

        mock_resource_cm = MagicMock()
        mock_resource_cm.__aenter__.return_value = mock_dynamodb
        mock_resource_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.resource.return_value = mock_resource_cm
        mock_session_cls.return_value = mock_session

        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        result = await client.get_item({"id": "1"})
        self.assertEqual(result, expected_item)
        mock_table.get_item.assert_called_once_with(Key={"id": "1"})

    @patch("api.clients.dynamodb_client.aioboto3.Session")
    async def test_get_item_not_found(self, mock_session_cls: MagicMock) -> None:
        mock_table = AsyncMock()
        mock_table.get_item.return_value = {}  # No Item

        mock_dynamodb = AsyncMock()
        mock_dynamodb.Table.return_value = mock_table

        mock_resource_cm = MagicMock()
        mock_resource_cm.__aenter__.return_value = mock_dynamodb
        mock_resource_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.resource.return_value = mock_resource_cm
        mock_session_cls.return_value = mock_session

        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        result = await client.get_item({"id": "1"})
        self.assertIsNone(result)

    @patch("api.clients.dynamodb_client.aioboto3.Session")
    async def test_get_item_failure(self, mock_session_cls: MagicMock) -> None:
        mock_resource_cm = MagicMock()
        mock_resource_cm.__aenter__.side_effect = Exception("DynamoDB error")
        mock_resource_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.resource.return_value = mock_resource_cm
        mock_session_cls.return_value = mock_session

        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        with self.assertRaisesRegex(Exception, "DynamoDB error"):
            await client.get_item({"id": "1"})


if __name__ == "__main__":
    unittest.main()
