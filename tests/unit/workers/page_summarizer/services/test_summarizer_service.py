import unittest
from unittest.mock import AsyncMock

from workers.page_summarizer.services.summarizer_service import SummarizerService


class TestSummarizerService(unittest.IsolatedAsyncioTestCase):
    async def asyncSetUp(self) -> None:
        self.mock_sqs = AsyncMock()
        self.writer_queue = "writer-queue"
        self.service = SummarizerService(
            sqs_client=self.mock_sqs,
            writer_queue_url=self.writer_queue,
            llm_provider="mock",
        )

    async def test_process_message_valid(self) -> None:
        body = (
            '{"scraping_id": 1, "page_id": 2, '
            '"url": "http://example.com", "content": "some content"}'
        )

        await self.service.process_message(body)

        # Verify LLM was called (implicitly by checking SQS message content)
        # Verify SQS message sent
        self.mock_sqs.send_message.assert_called_once()
        call_args = self.mock_sqs.send_message.call_args
        msg = call_args[0][0]

        self.assertEqual(msg["type"], "page_summary")
        self.assertEqual(msg["scraping_id"], 1)
        self.assertEqual(msg["page_id"], 2)
        self.assertEqual(msg["url"], "http://example.com")
        self.assertEqual(msg["summary"], "Mocked summary for testing")

    async def test_process_message_missing_fields(self) -> None:
        body = '{"url": "http://example.com"}'
        await self.service.process_message(body)
        self.mock_sqs.send_message.assert_not_called()

    async def test_process_message_invalid_json(self) -> None:
        body = "invalid-json"
        await self.service.process_message(body)
        self.mock_sqs.send_message.assert_not_called()
