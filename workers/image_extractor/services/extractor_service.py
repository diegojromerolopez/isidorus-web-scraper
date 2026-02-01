import json
import logging
import uuid

import httpx

from workers.image_extractor.clients.s3_client import S3Client
from workers.image_extractor.clients.sqs_client import SQSClient
from workers.image_extractor.services.explainer_factory import ExplainerFactory

logger = logging.getLogger(__name__)


class ExtractorService:
    def __init__(
        self,
        sqs_client: SQSClient,
        s3_client: S3Client,
        writer_queue_url: str,
        images_bucket: str,
        llm_provider: str = "openai",
    ):
        self.sqs_client = sqs_client
        self.s3_client = s3_client
        self.writer_queue_url = writer_queue_url
        self.images_bucket = images_bucket
        self.llm = ExplainerFactory.get_explainer(llm_provider)

    async def process_message(self, message_body: str) -> None:
        try:
            body = json.loads(message_body)
            image_url = body.get("url")
            original_url = body.get("original_url")
            scraping_id = body.get("scraping_id")

            if not image_url:
                logger.warning("No image URL in message")
                return

            if not scraping_id:
                logger.warning("No scraping_id in message")
                return

            logger.info(f"Processing image: {image_url} for scraping {scraping_id}")

            # 1. Download image
            s3_path = None
            try:
                async with httpx.AsyncClient() as client:
                    resp = await client.get(image_url, timeout=10.0)
                    if resp.status_code == 200:
                        s3_path = self._upload_image_to_s3(
                            resp.content, image_url, scraping_id
                        )
            except Exception as e:
                logger.error(f"Failed to handle image persistence to S3: {e}")

            # 3. Get Explanation via LangChain
            explanation = ExplainerFactory.explain_image(self.llm, image_url)
            logger.info(f"Generated explanation for {image_url}")

            # Send to Writer
            writer_msg = {
                "type": "image_explanation",
                "url": image_url,
                "original_url": original_url,
                "page_url": original_url,
                "scraping_id": scraping_id,
                "explanation": explanation,
                "s3_path": s3_path,
            }

            self.sqs_client.send_message(self.writer_queue_url, writer_msg)
            logger.info(f"Sent extraction info for {image_url} to writer queue")

        except Exception as e:
            logger.error(f"Error processing image extraction: {e}")

    def _upload_image_to_s3(
        self, content: bytes, image_url: str, scraping_id: int
    ) -> str | None:
        """Uploads image bytes to S3 and returns the S3 path."""
        try:
            ext = image_url.split(".")[-1].split("?")[0] or "bin"
            s3_key = f"{scraping_id}/{uuid.uuid4()}.{ext}"
            s3_path = self.s3_client.upload_bytes(content, self.images_bucket, s3_key)
            logger.info(f"Uploaded image to {s3_path}")
            return s3_path
        except Exception as e:
            logger.error(f"Internal S3 upload failure: {e}")
            return None
