import json
import logging
import uuid

import httpx

from shared.clients.s3_client import S3Client
from shared.clients.sqs_client import SQSClient
from workers.image_extractor.services.explainer_factory import ExplainerFactory

logger = logging.getLogger(__name__)


class ExtractorService:
    # pylint: disable=too-few-public-methods
    def __init__(
        self,
        sqs_client: SQSClient,
        s3_client: S3Client,
        writer_queue_url: str,
        images_bucket: str,
        llm_provider: str = "openai",
        llm_api_key: str | None = None,
    ):  # pylint: disable=too-many-arguments,too-many-positional-arguments
        self.__sqs_client = sqs_client
        self.__s3_client = s3_client
        self.__writer_queue_url = writer_queue_url
        self.__images_bucket = images_bucket
        self.__llm = ExplainerFactory.get_explainer(llm_provider, llm_api_key)

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

            logger.info("Processing image: %s for scraping %s", image_url, scraping_id)

            # 1. Download image
            s3_path = None
            try:
                async with httpx.AsyncClient() as client:
                    resp = await client.get(image_url, timeout=10.0)
                    if resp.status_code == 200:
                        # Upload to S3
                        s3_path = await self.__upload_image_to_s3(
                            resp.content, image_url, scraping_id
                        )
            except Exception as e:  # pylint: disable=broad-exception-caught
                logger.error("Failed to handle image persistence to S3: %s", e)

            # 3. Get Explanation via LangChain
            explanation = ExplainerFactory.explain_image(self.__llm, image_url)
            logger.info("Generated explanation for %s", image_url)

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

            await self.__sqs_client.send_message(writer_msg, self.__writer_queue_url)
            logger.info("Sent extraction info for %s to writer queue", image_url)

        except Exception as e:  # pylint: disable=broad-exception-caught
            # Catch-all to prevent worker crash on single message failure
            logger.error("Error processing image extraction: %s", e)

    async def __upload_image_to_s3(
        self, content: bytes, image_url: str, scraping_id: int
    ) -> str | None:
        try:
            ext = image_url.split(".")[-1].split("?")[0] or "bin"
            s3_key = f"{scraping_id}/{uuid.uuid4()}.{ext}"
            s3_path = await self.__s3_client.upload_bytes(
                content, self.__images_bucket, s3_key, "image/jpeg"
            )
            logger.info("Uploaded image to %s", s3_path)
            return s3_path
        except Exception as e:  # pylint: disable=broad-exception-caught
            logger.error("Internal S3 upload failure: %s", e)
            return None
