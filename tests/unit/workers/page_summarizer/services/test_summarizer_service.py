import json
import unittest
from unittest.mock import AsyncMock, patch

from workers.page_summarizer.services.summarizer_service import SummarizerService


@patch("workers.page_summarizer.services.summarizer_service.SummarizerFactory")
class TestSummarizerService(unittest.IsolatedAsyncioTestCase):
    def setUp(self):
        self.mock_sqs = AsyncMock()
        self.writer_queue = "writer-queue"
        self.api_key = "test-key"

    async def test_init(self, mock_factory):
        _ = SummarizerService(self.mock_sqs, self.writer_queue, "openai", self.api_key)
        mock_factory.get_llm.assert_called_with("openai", self.api_key)

    async def test_process_message_success(self, mock_factory):
        # Setup message
        msg_body = json.dumps(
            {
                "scraping_id": 123,
                "url": "http://example.com",
                "content": "some text content",
            }
        )

        mock_factory.summarize_text.return_value = "Summary"

        service = SummarizerService(self.mock_sqs, self.writer_queue)

        # We need to ensure service._SummarizerService__llm is set (it is by init)

        await service.process_message(msg_body)

        mock_factory.summarize_text.assert_called()
        self.mock_sqs.send_message.assert_called_once()

        # Check args
        args = self.mock_sqs.send_message.call_args[0]
        payload = args[0]
        self.assertEqual(payload["type"], "page_summary")
        self.assertEqual(payload["scraping_id"], 123)
        self.assertEqual(payload["summary"], "Summary")
        self.assertEqual(args[1], self.writer_queue)

    async def test_process_message_with_page_id(self, mock_factory):
        # Setup message
        msg_body = json.dumps(
            {
                "scraping_id": 123,
                "page_id": 456,
                "url": "http://example.com",
                "content": "some text content",
            }
        )

        mock_factory.summarize_text.return_value = "Summary"
        service = SummarizerService(self.mock_sqs, self.writer_queue)
        await service.process_message(msg_body)

        args = self.mock_sqs.send_message.call_args[0]
        payload = args[0]
        self.assertEqual(payload["page_id"], 456)

    async def test_process_message_missing_fields(self, mock_factory):
        msg_body = json.dumps(
            {
                "url": "http://example.com"
                # Missing scraping_id and content
            }
        )

        service = SummarizerService(self.mock_sqs, self.writer_queue)
        await service.process_message(msg_body)

        self.mock_sqs.send_message.assert_not_called()

    async def test_process_message_json_error(self, mock_factory):
        service = SummarizerService(self.mock_sqs, self.writer_queue)
        await service.process_message("invalid json")
        self.mock_sqs.send_message.assert_not_called()

    async def test_process_message_exception(self, mock_factory):
        msg_body = json.dumps(
            {"scraping_id": 123, "url": "http://example.com", "content": "text"}
        )
        mock_factory.summarize_text.side_effect = Exception("Boom")

        service = SummarizerService(self.mock_sqs, self.writer_queue)
        await service.process_message(msg_body)

        # Should catch exception and log error, not crash
        self.mock_sqs.send_message.assert_not_called()
