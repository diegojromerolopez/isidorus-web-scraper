import json
import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from shared.clients.s3_client import S3Client
from shared.clients.sqs_client import SQSClient
from workers.image_extractor.services.extractor_service import ExtractorService


class TestExtractorService(unittest.IsolatedAsyncioTestCase):
    # pylint: disable=protected-access
    async def asyncSetUp(self) -> None:
        self.mock_sqs = MagicMock(spec=SQSClient)
        self.mock_s3 = MagicMock(spec=S3Client)
        # S3 methods are now async
        self.mock_s3.upload_bytes = AsyncMock(return_value="s3://test/image.jpg")

        self.writer_queue = "writer-queue"
        self.images_bucket = "images-bucket"

        # Patch ExplainerFactory.get_explainer in setUp to avoid actual LLM init
        with patch(
            "workers.image_extractor.services.extractor_service"
            ".ExplainerFactory.get_explainer"
        ) as mock_get:
            self.mock_llm = MagicMock()
            mock_get.return_value = self.mock_llm
            self.mock_llm = MagicMock()
            mock_get.return_value = self.mock_llm
            self.service = ExtractorService(
                self.mock_sqs,
                self.mock_s3,
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
        self.mock_s3.upload_bytes = AsyncMock(return_value="s3://test-bucket/key.jpg")

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

        self.mock_sqs.send_message.assert_called_once()
        call_args = self.mock_sqs.send_message.call_args
        sent_msg, queue_url = call_args[0]

        self.assertEqual(queue_url, self.writer_queue)
        self.assertEqual(sent_msg["explanation"], "A beautiful landscape")
        self.assertEqual(sent_msg["s3_path"], "s3://test-bucket/key.jpg")
        self.assertEqual(sent_msg["scraping_id"], 123)

    async def test_process_message_no_url(self) -> None:
        await self.service.process_message(json.dumps({}))
        self.mock_sqs.send_message.assert_not_called()

    async def test_process_message_invalid_json(self) -> None:
        await self.service.process_message("invalid json")
        self.mock_sqs.send_message.assert_not_called()

    async def test_process_message_upload_failure(self) -> None:
        # Mock S3 upload failure
        self.mock_s3.upload_bytes = AsyncMock(side_effect=Exception("S3 error"))
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
        self.mock_sqs.send_message.assert_called_once()
        call_args = self.mock_sqs.send_message.call_args
        sent_msg, _ = call_args[0]
        self.assertIsNone(sent_msg["s3_path"])
        self.assertEqual(sent_msg["explanation"], "Explanation")

    async def test_process_message_download_exception(self) -> None:
        """Test handling of HTTP download exception."""
        with patch(
            "workers.image_extractor.services.extractor_service.httpx.AsyncClient"
        ) as mock_client_cls:
            mock_client = AsyncMock()
            mock_client.get.side_effect = Exception("Network Error")
            mock_client.__aenter__.return_value = mock_client
            mock_client.__aexit__.return_value = None
            mock_client_cls.return_value = mock_client

            # Mock LLM to avoid actual calls
            with patch(
                "workers.image_extractor.services.extractor_service."
                "ExplainerFactory.explain_image",
                return_value="Explanation",
            ):
                message_body = json.dumps(
                    {"url": "http://img.com/a.jpg", "scraping_id": 123}
                )
                await self.service.process_message(message_body)

        # S3 path should be None, but explanation still proceeds
        self.mock_sqs.send_message.assert_called()
        sent_msg = self.mock_sqs.send_message.call_args[0][0]
        self.assertIsNone(sent_msg["s3_path"])

    async def test_process_message_download_non_200(self) -> None:
        """Test handling of non-200 HTTP response."""
        with patch(
            "workers.image_extractor.services.extractor_service.httpx.AsyncClient"
        ) as mock_client_cls:
            mock_resp = MagicMock()
            mock_resp.status_code = 404

            mock_client = AsyncMock()
            mock_client.get.return_value = mock_resp
            mock_client.__aenter__.return_value = mock_client
            mock_client.__aexit__.return_value = None
            mock_client_cls.return_value = mock_client

            # Mock LLM
            with patch(
                "workers.image_extractor.services.extractor_service."
                "ExplainerFactory.explain_image",
                return_value="Explanation",
            ):
                message_body = json.dumps(
                    {"url": "http://img.com/a.jpg", "scraping_id": 123}
                )
                await self.service.process_message(message_body)

        self.mock_s3.upload_bytes.assert_not_called()
        self.mock_sqs.send_message.assert_called()

    async def test_process_message_no_scraping_id(self) -> None:
        """Test handling of missing scraping_id."""
        message_body = json.dumps({"url": "http://img.com/a.jpg"})
        await self.service.process_message(message_body)
        self.mock_sqs.send_message.assert_not_called()


if __name__ == "__main__":
    unittest.main()
