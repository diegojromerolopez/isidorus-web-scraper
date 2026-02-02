import json
import unittest
from unittest.mock import AsyncMock, MagicMock

from api.clients.sqs_client import SQSClient


class TestSQSClient(unittest.IsolatedAsyncioTestCase):
    async def asyncSetUp(self) -> None:
        self.endpoint_url = "http://localhost:4566"
        self.region = "us-east-1"
        self.access_key = "test"
        self.secret_key = "test"
        self.queue_url = "http://localhost:4566/queue/test-queue"

    async def test_init(self) -> None:
        client = SQSClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.queue_url,
        )
        self.assertIsNotNone(client._SQSClient__session)  # type: ignore[attr-defined]

    async def test_send_message_success(self) -> None:
        client = SQSClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.queue_url,
        )

        # Mock the async context manager for session.client
        mock_sqs_client = AsyncMock()

        mock_client_cm = MagicMock()
        mock_client_cm.__aenter__.return_value = mock_sqs_client
        mock_client_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.client.return_value = mock_client_cm
        client._SQSClient__session = mock_session  # type: ignore[attr-defined]

        message = {"foo": "bar"}
        result = await client.send_message(message)

        self.assertTrue(result)
        mock_sqs_client.send_message.assert_called_once_with(
            QueueUrl=self.queue_url, MessageBody=json.dumps(message)
        )

    async def test_send_message_error(self) -> None:
        client = SQSClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.queue_url,
        )

        # Mock to raise exception on __aenter__
        mock_client_cm = MagicMock()
        mock_client_cm.__aenter__.side_effect = Exception("SQS Error")
        mock_client_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.client.return_value = mock_client_cm
        client._SQSClient__session = mock_session  # type: ignore[attr-defined]

        message = {"foo": "bar"}

        with self.assertRaisesRegex(Exception, "SQS Error"):
            await client.send_message(message)


if __name__ == "__main__":
    unittest.main()
