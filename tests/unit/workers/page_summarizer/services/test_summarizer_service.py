import json
import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from workers.page_summarizer.services.summarizer_service import SummarizerService


@patch("workers.page_summarizer.services.summarizer_service.SummarizerFactory")
class TestSummarizerService(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        self.mock_sqs = AsyncMock()
        self.writer_queue = "writer-queue"
        self.api_key = "test-key"

    async def test_init(self, mock_factory: MagicMock) -> None:
        _ = SummarizerService(
            sqs_client=self.mock_sqs,
            writer_queue_url=self.writer_queue,
            llm_provider="openai",
            llm_api_key=self.api_key,
        )
        mock_factory.get_llm.assert_called_with("openai", self.api_key)

    async def test_process_message_success(self, mock_factory: MagicMock) -> None:
        # Setup message
        msg_body = json.dumps(
            {
                "scraping_id": 123,
                "user_id": 1,
                "url": "http://example.com",
                "content": "some text content",
            }
        )

        mock_factory.summarize_text = AsyncMock(return_value="Summary")

        # Test both queues
        indexer_queue = "indexer-queue"
        service = SummarizerService(
            self.mock_sqs, self.writer_queue, indexer_queue_url=indexer_queue
        )

        await service.process_message(msg_body)

        mock_factory.summarize_text.assert_called()

        # Should be called twice: once for writer, once for indexer
        self.assertEqual(self.mock_sqs.send_message.call_count, 2)

        # Check Writer Message
        writer_args = self.mock_sqs.send_message.call_args_list[0][0]
        self.assertEqual(writer_args[0]["type"], "page_summary")
        self.assertEqual(writer_args[0]["summary"], "Summary")
        self.assertEqual(writer_args[1], self.writer_queue)

    async def test_process_message_missing_fields(
        self, _mock_factory: MagicMock
    ) -> None:
        msg_body = json.dumps(
            {
                "url": "http://example.com"
                # Missing scraping_id and content
            }
        )

        service = SummarizerService(self.mock_sqs, self.writer_queue)
        await service.process_message(msg_body)

        self.mock_sqs.send_message.assert_not_called()

    async def test_process_message_json_error(self, _mock_factory: MagicMock) -> None:
        service = SummarizerService(self.mock_sqs, self.writer_queue)
        await service.process_message("invalid json")
        self.mock_sqs.send_message.assert_not_called()

    async def test_process_message_exception(self, mock_factory: MagicMock) -> None:
        msg_body = json.dumps(
            {"scraping_id": 123, "url": "http://example.com", "content": "text"}
        )
        mock_factory.summarize_text = AsyncMock(side_effect=Exception("Boom"))

        service = SummarizerService(self.mock_sqs, self.writer_queue)
        await service.process_message(msg_body)

        # Should catch exception and log error, not crash
        self.mock_sqs.send_message.assert_not_called()
