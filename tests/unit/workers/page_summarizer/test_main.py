import asyncio
import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from workers.page_summarizer.config import Configuration
from workers.page_summarizer.main import main


class TestMain(unittest.IsolatedAsyncioTestCase):
    @patch("workers.page_summarizer.main.SQSClient")
    @patch("workers.page_summarizer.main.SummarizerService")
    @patch("workers.page_summarizer.main.Configuration")
    async def test_main_loop_success(
        self, mock_config_cls, mock_service_cls, mock_sqs_cls
    ):
        # Setup Mocks
        mock_config = MagicMock()
        mock_config.input_queue_url = "input"
        mock_config.writer_queue_url = "writer"
        mock_config.llm_provider = "mock"
        mock_config_cls.from_env.return_value = mock_config

        mock_sqs = AsyncMock()
        mock_sqs_cls.create.return_value = mock_sqs

        # Mock message receipt
        mock_sqs.receive_messages.side_effect = [
            [{"Body": "{}", "ReceiptHandle": "handle"}],  # First iteration: 1 message
            Exception("Stop Loop"),  # Second iteration: Stop loop
        ]

        mock_service = AsyncMock()
        mock_service_cls.return_value = mock_service

        # Run Main
        try:
            await main()
        except Exception:
            pass

        # Verify
        mock_sqs.receive_messages.assert_called()
        mock_service.process_message.assert_called()
        mock_sqs.delete_message.assert_called()

    @patch("workers.page_summarizer.main.Configuration")
    async def test_main_missing_config(self, mock_config_cls):
        mock_config = MagicMock()
        mock_config.input_queue_url = ""
        mock_config_cls.from_env.return_value = mock_config

        await main()
        # Should return immediately
