import json
import unittest
from unittest.mock import MagicMock, patch

from api.clients.sqs_client import SQSClient


class TestSQSClient(unittest.TestCase):
    def setUp(self) -> None:
        self.endpoint_url = "http://localhost:4566"
        self.region = "us-east-1"
        self.access_key = "test"
        self.secret_key = "test"
        self.queue_url = "http://localhost:4566/queue/test-queue"

    @patch("boto3.client")
    def test_init(self, mock_boto_client: MagicMock) -> None:
        client = SQSClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.queue_url,
        )
        mock_boto_client.assert_called_once_with(
            "sqs",
            endpoint_url=self.endpoint_url,
            region_name=self.region,
            aws_access_key_id=self.access_key,
            aws_secret_access_key=self.secret_key,
        )
        self.assertEqual(client.queue_url, self.queue_url)

    @patch("boto3.client")
    def test_send_message_success(self, mock_boto_client: MagicMock) -> None:
        # Setup mock
        mock_sqs = MagicMock()
        mock_boto_client.return_value = mock_sqs

        client = SQSClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.queue_url,
        )
        message = {"foo": "bar"}

        result = client.send_message(message)

        self.assertTrue(result)
        mock_sqs.send_message.assert_called_once_with(
            QueueUrl=self.queue_url, MessageBody=json.dumps(message)
        )

    @patch("boto3.client")
    def test_send_message_error(self, mock_boto_client: MagicMock) -> None:
        # Setup mock
        mock_sqs = MagicMock()
        mock_boto_client.return_value = mock_sqs
        mock_sqs.send_message.side_effect = Exception("SQS Error")

        client = SQSClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.queue_url,
        )
        message = {"foo": "bar"}

        with self.assertRaises(Exception):  # noqa: B017
            client.send_message(message)


if __name__ == "__main__":
    unittest.main()
