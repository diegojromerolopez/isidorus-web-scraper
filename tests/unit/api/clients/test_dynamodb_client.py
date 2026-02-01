import unittest
from typing import Any
from unittest.mock import MagicMock, patch

from api.clients.dynamodb_client import DynamoDBClient


class TestDynamoDBClient(unittest.TestCase):
    def setUp(self) -> None:
        self.endpoint_url = "http://localhost:4566"
        self.region = "us-east-1"
        self.access_key = "test"
        self.secret_key = "test"
        self.table_name = "test-table"

    @patch("boto3.resource")
    def test_init(self, mock_boto3_resource: Any) -> None:
        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )
        mock_boto3_resource.assert_called_once_with(
            "dynamodb",
            endpoint_url=self.endpoint_url,
            region_name=self.region,
            aws_access_key_id=self.access_key,
            aws_secret_access_key=self.secret_key,
        )
        mock_boto3_resource.return_value.Table.assert_called_once_with(self.table_name)
        self.assertEqual(
            client.table, mock_boto3_resource.return_value.Table.return_value
        )

    @patch("boto3.resource")
    def test_put_item_success(self, mock_boto3_resource: Any) -> None:
        mock_table = MagicMock()
        mock_boto3_resource.return_value.Table.return_value = mock_table

        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        item = {"id": "1", "data": "value"}
        result = client.put_item(item)

        self.assertTrue(result)
        mock_table.put_item.assert_called_once_with(Item=item)

    @patch("boto3.resource")
    def test_put_item_failure(self, mock_boto3_resource: Any) -> None:
        mock_table = MagicMock()
        mock_table.put_item.side_effect = Exception("DynamoDB error")
        mock_boto3_resource.return_value.Table.return_value = mock_table

        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        with self.assertRaisesRegex(Exception, "DynamoDB error"):
            client.put_item({"id": "1"})

    @patch("boto3.resource")
    def test_get_item_success(self, mock_boto3_resource: Any) -> None:
        mock_table = MagicMock()
        expected_item = {"id": "1", "data": "value"}
        mock_table.get_item.return_value = {"Item": expected_item}
        mock_boto3_resource.return_value.Table.return_value = mock_table

        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        result = client.get_item({"id": "1"})
        self.assertEqual(result, expected_item)
        mock_table.get_item.assert_called_once_with(Key={"id": "1"})

    @patch("boto3.resource")
    def test_get_item_not_found(self, mock_boto3_resource: Any) -> None:
        mock_table = MagicMock()
        mock_table.get_item.return_value = {}  # No "Item" key
        mock_boto3_resource.return_value.Table.return_value = mock_table

        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        result = client.get_item({"id": "1"})
        self.assertIsNone(result)

    @patch("boto3.resource")
    def test_get_item_failure(self, mock_boto3_resource: Any) -> None:
        mock_table = MagicMock()
        mock_table.get_item.side_effect = Exception("DynamoDB error")
        mock_boto3_resource.return_value.Table.return_value = mock_table

        client = DynamoDBClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.table_name,
        )

        with self.assertRaisesRegex(Exception, "DynamoDB error"):
            client.get_item({"id": "1"})
