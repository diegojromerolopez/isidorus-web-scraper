import json
import unittest
from unittest.mock import MagicMock, patch

from workers.image_extractor.clients.sqs_client import SQSClient


class TestSQSClient(unittest.TestCase):
    def setUp(self) -> None:
        self.endpoint_url = "http://localhost:4566"
        self.region = "us-east-1"
        self.access_key = "test"
        self.secret_key = "test"
        self.queue_url = "http://localhost:4566/queue/test-queue"

    @patch("boto3.client")
    def test_init(self, mock_boto: MagicMock) -> None:
        SQSClient(self.endpoint_url, self.region, self.access_key, self.secret_key)
        mock_boto.assert_called_once_with(
            "sqs",
            endpoint_url=self.endpoint_url,
            region_name=self.region,
            aws_access_key_id=self.access_key,
            aws_secret_access_key=self.secret_key,
        )

    @patch("boto3.client")
    def test_receive_messages(self, mock_boto: MagicMock) -> None:
        mock_sqs_client = MagicMock()
        mock_boto.return_value = mock_sqs_client
        mock_sqs_client.receive_message.return_value = {"Messages": [{"Body": "{}"}]}

        client = SQSClient(
            self.endpoint_url, self.region, self.access_key, self.secret_key
        )

        messages = client.receive_messages(self.queue_url)

        mock_sqs_client.receive_message.assert_called_once_with(
            QueueUrl=self.queue_url, MaxNumberOfMessages=1, WaitTimeSeconds=20
        )
        self.assertEqual(len(messages), 1)

    @patch("boto3.client")
    def test_send_message(self, mock_boto: MagicMock) -> None:
        mock_sqs_client = MagicMock()
        mock_boto.return_value = mock_sqs_client

        client = SQSClient(
            self.endpoint_url, self.region, self.access_key, self.secret_key
        )
        message = {"foo": "bar"}

        client.send_message(self.queue_url, message)

        mock_sqs_client.send_message.assert_called_once_with(
            QueueUrl=self.queue_url, MessageBody=json.dumps(message)
        )

    @patch("boto3.client")
    def test_delete_message(self, mock_boto: MagicMock) -> None:
        mock_sqs_client = MagicMock()
        mock_boto.return_value = mock_sqs_client

        client = SQSClient(
            self.endpoint_url, self.region, self.access_key, self.secret_key
        )
        receipt_handle = "handle123"

        client.delete_message(self.queue_url, receipt_handle)

        mock_sqs_client.delete_message.assert_called_once_with(
            QueueUrl=self.queue_url, ReceiptHandle=receipt_handle
        )


if __name__ == "__main__":
    unittest.main()
