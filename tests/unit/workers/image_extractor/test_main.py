import os
import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from workers.image_extractor.main import main


class TestMain(unittest.IsolatedAsyncioTestCase):
    @patch.dict(os.environ, {}, clear=True)
    @patch("workers.image_extractor.main.logger")
    async def test_main_missing_env_vars(self, mock_logger: MagicMock) -> None:
        await main()
        mock_logger.error.assert_called_with(
            "INPUT_QUEUE_URL and WRITER_QUEUE_URL must be set"
        )

    @patch.dict(
        os.environ,
        {
            "AWS_ENDPOINT_URL": "http://localhost:4566",
            "INPUT_QUEUE_URL": "input-queue",
            "WRITER_QUEUE_URL": "writer-queue",
        },
    )
    @patch("workers.image_extractor.main.SQSClient")
    @patch("workers.image_extractor.main.S3Client")
    @patch("workers.image_extractor.main.ExtractorService")
    @patch("workers.image_extractor.main.time.sleep")
    async def test_main_loop_success(
        self,
        _mock_sleep: MagicMock,
        mock_service_cls: MagicMock,
        _mock_s3_cls: MagicMock,
        mock_sqs_cls: MagicMock,
    ) -> None:
        # Mock SQS Client
        mock_sqs_instance = MagicMock()
        mock_sqs_cls.return_value = mock_sqs_instance

        # Mock Service with AsyncMock for process_message
        mock_service_instance = MagicMock()
        mock_service_instance.process_message = AsyncMock()
        mock_service_cls.return_value = mock_service_instance

        # Setup receive_messages to return one batch then exit
        mock_sqs_instance.receive_messages.side_effect = [
            [{"Body": "{}", "ReceiptHandle": "abc"}],
            SystemExit("Exit Test"),
        ]

        try:
            await main()
        except SystemExit:
            pass

        # Verify processing
        mock_service_instance.process_message.assert_called_once_with("{}")
        mock_sqs_instance.delete_message.assert_called_once_with("input-queue", "abc")

    @patch.dict(
        os.environ,
        {
            "AWS_ENDPOINT_URL": "http://localhost:4566",
            "INPUT_QUEUE_URL": "input-queue",
            "WRITER_QUEUE_URL": "writer-queue",
        },
    )
    @patch("workers.image_extractor.main.SQSClient")
    @patch("workers.image_extractor.main.S3Client")
    @patch("workers.image_extractor.main.ExtractorService")
    @patch("workers.image_extractor.main.logger")
    @patch("workers.image_extractor.main.time.sleep")  # Mock sleep to be fast
    async def test_main_loop_exception(
        self,
        mock_sleep: MagicMock,
        mock_logger: MagicMock,
        _mock_service_cls: MagicMock,
        _mock_s3_cls: MagicMock,
        mock_sqs_cls: MagicMock,
    ) -> None:
        mock_sqs_instance = MagicMock()
        mock_sqs_cls.return_value = mock_sqs_instance

        # Side effect: Raise normal exception (caught), then SystemExit (uncaught)
        mock_sqs_instance.receive_messages.side_effect = [
            Exception("SQS Error"),
            SystemExit("Exit Test"),
        ]

        try:
            await main()
        except SystemExit:
            pass

        mock_logger.error.assert_called_with("Polling error: SQS Error")
        mock_sleep.assert_called_once_with(5)


if __name__ == "__main__":
    unittest.main()
