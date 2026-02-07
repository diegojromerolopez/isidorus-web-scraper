import base64
import json
import logging

from shared.clients.s3_client import S3Client
from shared.clients.sqs_client import SQSClient
from workers.image_explainer.services.explainer_factory import ExplainerFactory

logger = logging.getLogger(__name__)


class ExplainerService:
    # pylint: disable=too-few-public-methods,too-many-arguments
    # pylint: disable=too-many-positional-arguments,too-many-locals
    def __init__(
        self,
        sqs_client: SQSClient,
        s3_client: S3Client,
        writer_queue_url: str,
        llm_provider: str,
        llm_api_key: str | None = None,
    ):
        self.__sqs_client = sqs_client
        self.__s3_client = s3_client
        self.__writer_queue_url = writer_queue_url
        self.__llm = ExplainerFactory.get_explainer(llm_provider, llm_api_key)

    async def process_message(self, message_body: str) -> None:
        """
        Processes a message from the image-explainer-queue.
        1. Downloads image from S3.
        2. Generates explanation using LLM.
        3. Sends explanation to writer-queue.
        """
        try:
            body = json.loads(message_body)
            s3_path = body.get("s3_path")
            scraping_id = body.get("scraping_id")
            image_url = body.get("image_url")
            original_url = body.get("original_url")

            if not s3_path:
                logger.warning("No s3_path in message")
                return

            logger.info("Explaining image from S3: %s", s3_path)

            # Extract bucket and key from s3://bucket/key
            path_parts = s3_path.replace("s3://", "").split("/", 1)
            if len(path_parts) != 2:
                logger.error("Invalid S3 path format: %s", s3_path)
                return

            bucket, key = path_parts

            # 1. Download image from S3
            image_bytes = await self.__s3_client.download_bytes(bucket, key)
            if not image_bytes:
                logger.error("Failed to download image from S3: %s", s3_path)
                return

            # 2. Generate explanation
            # We pass a data URL to the explainer factory
            base64_image = base64.b64encode(image_bytes).decode("utf-8")
            data_url = f"data:image/jpeg;base64,{base64_image}"
            explanation = await ExplainerFactory.explain_image(self.__llm, data_url)

            # 3. Send to Writer
            writer_msg = {
                "type": "image_explanation",
                "url": image_url,
                "original_url": original_url,
                "page_url": original_url,
                "scraping_id": scraping_id,
                "s3_path": s3_path,
                "explanation": explanation,
            }

            await self.__sqs_client.send_message(writer_msg, self.__writer_queue_url)
            logger.info("Sent image explanation to writer queue: %s", image_url)

        except Exception as e:  # pylint: disable=broad-exception-caught
            logger.error("Error processing image explanation: %s", e)
