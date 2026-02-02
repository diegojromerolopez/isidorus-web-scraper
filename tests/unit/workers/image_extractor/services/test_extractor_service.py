import json
import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from workers.image_extractor.services.extractor_service import ExtractorService


class TestExtractorService(unittest.IsolatedAsyncioTestCase):
    # pylint: disable=protected-access
    async def asyncSetUp(self) -> None:
        self.mock_sqs_client = MagicMock()
        self.mock_s3_client = MagicMock()
        self.writer_queue = "http://writer"
        self.images_bucket = "test-bucket"

        # Patch ExplainerFactory.get_explainer in setUp to avoid actual LLM init
        with patch(
            "workers.image_extractor.services.extractor_service"
            ".ExplainerFactory.get_explainer"
        ) as mock_get:
            self.mock_llm = MagicMock()
            mock_get.return_value = self.mock_llm
            self.service = ExtractorService(
                self.mock_sqs_client,
                self.mock_s3_client,
                self.writer_queue,
                self.images_bucket,
            )

    @patch("workers.image_extractor.services.extractor_service.httpx.AsyncClient")
    @patch(
        "workers.image_extractor.services.extractor_service"
        ".ExplainerFactory.explain_image"
    )
    async def test_process_message_success(
        self, mock_explain: MagicMock, mock_async_client_cls: MagicMock
    ) -> None:
        # Mock httpx AsyncClient
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_resp.content = b"fake-image-bytes"

        mock_client = AsyncMock()
        mock_client.get.return_value = mock_resp
        mock_client.__aenter__.return_value = mock_client
        mock_client.__aexit__.return_value = None

        mock_async_client_cls.return_value = mock_client

        # Mock S3 upload
        self.mock_s3_client.upload_bytes.return_value = "s3://test-bucket/key.jpg"

        # Mock Explanation
        mock_explain.return_value = "A beautiful landscape"

        message_body = json.dumps(
            {
                "url": "http://img.com/a.jpg",
                "original_url": "http://page.com",
                "scraping_id": 123,
            }
        )

        await self.service.process_message(message_body)

        self.mock_sqs_client.send_message.assert_called_once()
        call_args = self.mock_sqs_client.send_message.call_args
        queue_url, sent_msg = call_args[0]

        self.assertEqual(queue_url, self.writer_queue)
        self.assertEqual(sent_msg["explanation"], "A beautiful landscape")
        self.assertEqual(sent_msg["s3_path"], "s3://test-bucket/key.jpg")
        self.assertEqual(sent_msg["scraping_id"], 123)

    async def test_process_message_no_url(self) -> None:
        await self.service.process_message(json.dumps({}))
        self.mock_sqs_client.send_message.assert_not_called()

    async def test_process_message_invalid_json(self) -> None:
        await self.service.process_message("invalid json")
        self.mock_sqs_client.send_message.assert_not_called()

    async def test_process_message_upload_failure(self) -> None:
        # Mock S3 upload failure
        self.mock_s3_client.upload_bytes.side_effect = Exception("S3 error")
        # We also need to mock explain_image to avoid it running if we reach that point
        # But if upload fails, s3_path is None, and we still proceed to explain/send?
        # Looking at code: Yes, it logs error for upload, then proceeds to explanation.

        # We need to mock the LLM related calls since they will be called
        self.mock_llm.invoke = MagicMock(return_value="Explanation")
        with patch(
            "workers.image_extractor.services.extractor_service"
            ".ExplainerFactory.explain_image",
            return_value="Explanation",
        ):
            # Mock httpx to succeed so we reach upload
            mock_resp = MagicMock()
            mock_resp.status_code = 200
            mock_resp.content = b"fake-image-bytes"

            mock_client = AsyncMock()
            mock_client.get.return_value = mock_resp
            mock_client.__aenter__.return_value = mock_client
            mock_client.__aexit__.return_value = None

            with patch(
                "workers.image_extractor.services.extractor_service.httpx.AsyncClient",
                return_value=mock_client,
            ):
                message_body = json.dumps(
                    {
                        "url": "http://img.com/a.jpg",
                        "original_url": "http://page.com",
                        "scraping_id": 123,
                    }
                )
                await self.service.process_message(message_body)

        # Verify SQS message has s3_path=None
        self.mock_sqs_client.send_message.assert_called_once()
        call_args = self.mock_sqs_client.send_message.call_args
        _, sent_msg = call_args[0]
        self.assertIsNone(sent_msg["s3_path"])
        self.assertEqual(sent_msg["explanation"], "Explanation")


if __name__ == "__main__":
    unittest.main()
