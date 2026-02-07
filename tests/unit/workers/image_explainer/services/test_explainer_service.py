import json
import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from shared.clients.s3_client import S3Client
from shared.clients.sqs_client import SQSClient
from workers.image_explainer.services.explainer_service import ExplainerService


class TestExplainerService(unittest.IsolatedAsyncioTestCase):
    # pylint: disable=protected-access
    async def asyncSetUp(self) -> None:
        self.mock_sqs = MagicMock(spec=SQSClient)
        self.mock_s3 = MagicMock(spec=S3Client)
        self.writer_queue = "writer-queue"
        self.llm_provider = "mock"

        # Initialize service with mocks
        self.service = ExplainerService(
            sqs_client=self.mock_sqs,
            s3_client=self.mock_s3,
            writer_queue_url=self.writer_queue,
            llm_provider=self.llm_provider,
        )

    async def test_process_message_success(self) -> None:
        # Mock S3 download
        self.mock_s3.download_bytes = AsyncMock(return_value=b"fake-image-bytes")
        self.mock_sqs.send_message = AsyncMock()

        message_body = json.dumps(
            {
                "s3_path": "s3://bucket/123/image.jpg",
                "scraping_id": 123,
                "image_url": "http://img.com/a.jpg",
                "original_url": "http://page.com",
            }
        )

        with patch(
            "workers.image_explainer.services.explainer_factory"
            ".ExplainerFactory.explain_image",
            new_callable=AsyncMock,
            return_value="Beautiful landscape",
        ) as mock_explain:
            await self.service.process_message(message_body)

            mock_explain.assert_called_once()
            self.mock_s3.download_bytes.assert_called_once_with(
                "bucket", "123/image.jpg"
            )
            self.mock_sqs.send_message.assert_called_once()

            sent_msg = self.mock_sqs.send_message.call_args[0][0]
            self.assertEqual(sent_msg["explanation"], "Beautiful landscape")
            self.assertEqual(sent_msg["scraping_id"], 123)

    async def test_process_message_no_s3_path(self) -> None:
        await self.service.process_message(json.dumps({}))
        self.mock_sqs.send_message.assert_not_called()

    async def test_process_message_invalid_s3_path(self) -> None:
        message_body = json.dumps({"s3_path": "invalid-path"})
        await self.service.process_message(message_body)
        self.mock_sqs.send_message.assert_not_called()

    async def test_process_message_download_failure(self) -> None:
        self.mock_s3.download_bytes = AsyncMock(return_value=None)

        message_body = json.dumps(
            {
                "s3_path": "s3://bucket/key",
                "scraping_id": 123,
            }
        )

        await self.service.process_message(message_body)
        self.mock_sqs.send_message.assert_not_called()

    async def test_process_message_invalid_json(self) -> None:
        await self.service.process_message("invalid json")
        self.mock_sqs.send_message.assert_not_called()

    async def test_process_message_exception(self) -> None:
        # Broad exception test
        self.mock_s3.download_bytes = AsyncMock(side_effect=Exception("S3 Error"))

        message_body = json.dumps(
            {
                "s3_path": "s3://bucket/key",
                "scraping_id": 123,
            }
        )

        # Should not raise exception (caught internally)
        await self.service.process_message(message_body)
        self.mock_sqs.send_message.assert_not_called()


if __name__ == "__main__":
    unittest.main()
