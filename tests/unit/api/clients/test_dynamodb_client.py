import unittest
from unittest.mock import AsyncMock, MagicMock

from api.clients.dynamodb_client import DynamoDBClient


class TestDynamoDBClient(unittest.IsolatedAsyncioTestCase):
    async def asyncSetUp(self) -> None:
        self.endpoint_url = "http://localhost:4566"
        self.region = "us-east-1"
        self.access_key = "test"
        self.secret_key = "test"
        self.table_name = "test-table"

    async def test_init(self) -> None:
        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )
        self.assertIsNotNone(
            client._DynamoDBClient__session  # type: ignore[attr-defined]
        )

    async def test_put_item_success(self) -> None:
        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        mock_table = AsyncMock()

        mock_dynamodb = AsyncMock()
        mock_dynamodb.Table.return_value = mock_table

        # Resource CM should be MagicMock, but __aenter__ returns awaitable (AsyncMock)
        mock_resource_cm = MagicMock()
        mock_resource_cm.__aenter__.return_value = mock_dynamodb
        mock_resource_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.resource.return_value = mock_resource_cm
        client._DynamoDBClient__session = mock_session  # type: ignore[attr-defined]

        item = {"id": "1", "data": "value"}
        result = await client.put_item(item)

        self.assertTrue(result)
        mock_table.put_item.assert_called_once_with(Item=item)

    async def test_put_item_failure(self) -> None:
        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        mock_resource_cm = MagicMock()
        mock_resource_cm.__aenter__.side_effect = Exception("DynamoDB error")
        mock_resource_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.resource.return_value = mock_resource_cm
        client._DynamoDBClient__session = mock_session  # type: ignore[attr-defined]

        with self.assertRaisesRegex(Exception, "DynamoDB error"):
            await client.put_item({"id": "1"})

    async def test_get_item_success(self) -> None:
        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

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
        client._DynamoDBClient__session = mock_session  # type: ignore[attr-defined]

        result = await client.get_item({"id": "1"})
        self.assertEqual(result, expected_item)
        mock_table.get_item.assert_called_once_with(Key={"id": "1"})

    async def test_get_item_not_found(self) -> None:
        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        mock_table = AsyncMock()
        mock_table.get_item.return_value = {}  # No Item

        mock_dynamodb = AsyncMock()
        mock_dynamodb.Table.return_value = mock_table

        mock_resource_cm = MagicMock()
        mock_resource_cm.__aenter__.return_value = mock_dynamodb
        mock_resource_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.resource.return_value = mock_resource_cm
        client._DynamoDBClient__session = mock_session  # type: ignore[attr-defined]

        result = await client.get_item({"id": "1"})
        self.assertIsNone(result)

    async def test_get_item_failure(self) -> None:
        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        mock_resource_cm = MagicMock()
        mock_resource_cm.__aenter__.side_effect = Exception("DynamoDB error")
        mock_resource_cm.__aexit__.return_value = None

        client._DynamoDBClient__session = MagicMock()  # type: ignore[attr-defined]
        mock_session = MagicMock()
        mock_session.resource.return_value = mock_resource_cm
        client._DynamoDBClient__session = mock_session  # type: ignore[attr-defined]

        with self.assertRaisesRegex(Exception, "DynamoDB error"):
            await client.get_item({"id": "1"})


if __name__ == "__main__":
    unittest.main()
