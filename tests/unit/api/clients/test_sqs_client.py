import json
import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from api.clients.sqs_client import SQSClient


class TestSQSClient(unittest.IsolatedAsyncioTestCase):
    # pylint: disable=duplicate-code
    async def asyncSetUp(self) -> None:
        self.endpoint_url = "http://localhost:4566"
        self.region = "us-east-1"
        self.access_key = "test"
        self.secret_key = "test"
        self.queue_url = "http://localhost:4566/queue/test-queue"

    @patch("api.clients.sqs_client.aioboto3.Session")
    async def test_init(self, mock_session_cls: MagicMock) -> None:
        mock_session = MagicMock()
        mock_session_cls.return_value = mock_session

        _ = SQSClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.queue_url,
        )

        # Verify Session was instantiated
        mock_session_cls.assert_called_once()

    @patch("api.clients.sqs_client.aioboto3.Session")
    async def test_send_message_success(self, mock_session_cls: MagicMock) -> None:
        # 1. Setup Mock Session and Client
        mock_sqs_client = AsyncMock()
        mock_client_cm = MagicMock()
        mock_client_cm.__aenter__.return_value = mock_sqs_client
        mock_client_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.client.return_value = mock_client_cm
        mock_session_cls.return_value = mock_session

        # 2. Init Client (uses mock session)
        client = SQSClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.queue_url,
        )

        # 3. Execute
        message = {"foo": "bar"}
        result = await client.send_message(message)

        # 4. Verify
        self.assertTrue(result)
        mock_sqs_client.send_message.assert_called_once_with(
            QueueUrl=self.queue_url, MessageBody=json.dumps(message)
        )
        mock_session.client.assert_called_once_with(
            "sqs",
            endpoint_url=self.endpoint_url,
            region_name=self.region,
            aws_access_key_id=self.access_key,
            aws_secret_access_key=self.secret_key,
        )

    @patch("api.clients.sqs_client.aioboto3.Session")
    async def test_send_message_error(self, mock_session_cls: MagicMock) -> None:
        # 1. Setup Mock to raise error
        mock_client_cm = MagicMock()
        mock_client_cm.__aenter__.side_effect = Exception("SQS Error")
        mock_client_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.client.return_value = mock_client_cm
        mock_session_cls.return_value = mock_session

        # 2. Init Client
        client = SQSClient(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
            self.queue_url,
        )

        # 3. Execute & Verify
        message = {"foo": "bar"}
        with self.assertRaisesRegex(Exception, "SQS Error"):
            await client.send_message(message)


if __name__ == "__main__":
    unittest.main()
