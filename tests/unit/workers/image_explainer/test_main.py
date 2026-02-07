import json
import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from workers.image_explainer.main import main


# pylint: disable=too-many-arguments,too-many-positional-arguments
# pylint: disable=broad-exception-caught,unused-argument
class TestMain(unittest.IsolatedAsyncioTestCase):
    @patch("workers.image_explainer.main.Configuration")
    @patch("workers.image_explainer.main.SQSClient")
    @patch("workers.image_explainer.main.S3Client")
    @patch("workers.image_explainer.main.ExplainerService")
    @patch("workers.image_explainer.main.asyncio.sleep")
    async def test_main_success(
        self,
        mock_sleep: MagicMock,
        mock_service_cls: MagicMock,
        mock_s3_cls: MagicMock,
        mock_sqs_cls: MagicMock,
        mock_config_cls: MagicMock,
    ) -> None:
        # Prevent infinite loop by making sleep raise an exception on second call
        mock_sleep.side_effect = [None, Exception("Break Loop")]

        # Configuration mock
        mock_config = MagicMock()
        mock_config.input_queue_url = "http://queue/input"
        mock_config.writer_queue_url = "http://queue/writer"
        mock_config_cls.from_env.return_value = mock_config

        # Client mocks
        mock_sqs = AsyncMock()
        mock_sqs_cls.create.return_value = mock_sqs

        # Mock service to be AsyncMock
        mock_service = AsyncMock()
        mock_service_cls.return_value = mock_service

        # Simulate one message and then empty list
        mock_sqs.receive_messages.side_effect = [
            [{"Body": json.dumps({"test": "data"}), "ReceiptHandle": "abc"}],
            [],
        ]

        # Run main and catch the "Break Loop" exception if it reaches the sleep
        try:
            await main()
        except Exception as e:
            if str(e) != "Break Loop":
                raise e

        # Verify initializations
        mock_config_cls.from_env.assert_called_once()
        mock_sqs_cls.create.assert_called_once_with(mock_config)
        mock_s3_cls.create.assert_called_once_with(mock_config)

        # Verify service call
        mock_service.process_message.assert_called_once()
        mock_sqs.delete_message.assert_called_once_with("http://queue/input", "abc")

    @patch("workers.image_explainer.main.Configuration")
    @patch("workers.image_explainer.main.SQSClient")
    @patch("workers.image_explainer.main.S3Client")
    @patch("workers.image_explainer.main.ExplainerService")
    @patch("workers.image_explainer.main.asyncio.sleep")
    async def test_main_polling_error(
        self,
        mock_sleep: MagicMock,
        mock_service_cls: MagicMock,
        mock_s3_cls: MagicMock,
        mock_sqs_cls: MagicMock,
        mock_config_cls: MagicMock,
    ) -> None:
        # Prevent infinite loop
        mock_sleep.side_effect = [None, Exception("Break Loop")]

        mock_config = MagicMock()
        mock_config.input_queue_url = "http://queue/input"
        mock_config.writer_queue_url = "http://queue/writer"
        mock_config_cls.from_env.return_value = mock_config

        mock_sqs = AsyncMock()
        mock_sqs_cls.create.return_value = mock_sqs

        # Raise exception on first call, empty on second
        mock_sqs.receive_messages.side_effect = [Exception("Polling Error"), []]

        try:
            await main()
        except Exception as e:
            if str(e) != "Break Loop":
                raise e

        # Should have called sleep once due to polling error
        mock_sleep.assert_any_call(5)

    @patch("workers.image_explainer.main.Configuration")
    async def test_main_missing_config(self, mock_config_cls: MagicMock) -> None:
        mock_config = MagicMock()
        mock_config.input_queue_url = None  # Missing
        mock_config_cls.from_env.return_value = mock_config

        await main()

        # Should return early and not create clients
        mock_config_cls.from_env.assert_called_once()


if __name__ == "__main__":
    unittest.main()
