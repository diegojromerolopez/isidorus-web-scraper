import asyncio
import logging
import os
import time

from workers.image_extractor.clients.s3_client import S3Client
from workers.image_extractor.clients.sqs_client import SQSClient
from workers.image_extractor.services.extractor_service import ExtractorService

# Logging setup
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def main() -> None:
    logger.info("Image Extractor Worker started (Refactor)")

    # Configuration
    AWS_ENDPOINT_URL = os.getenv("AWS_ENDPOINT_URL")
    AWS_REGION = os.getenv("AWS_REGION", "us-east-1")
    AWS_ACCESS_KEY_ID = os.getenv("AWS_ACCESS_KEY_ID")
    AWS_SECRET_ACCESS_KEY = os.getenv("AWS_SECRET_ACCESS_KEY")
    INPUT_QUEUE_URL = os.getenv("INPUT_QUEUE_URL") or ""
    WRITER_QUEUE_URL = os.getenv("WRITER_QUEUE_URL") or ""
    IMAGES_BUCKET = os.getenv("IMAGES_BUCKET") or "isidorus-images"
    LLM_PROVIDER = os.getenv("LLM_PROVIDER", "openai")

    if not INPUT_QUEUE_URL or not WRITER_QUEUE_URL:
        logger.error("INPUT_QUEUE_URL and WRITER_QUEUE_URL must be set")
        return

    # Dependency Injection
    sqs_client = SQSClient(
        endpoint_url=AWS_ENDPOINT_URL,
        region=AWS_REGION,
        access_key=AWS_ACCESS_KEY_ID or "",
        secret_key=AWS_SECRET_ACCESS_KEY or "",
    )

    s3_client = S3Client(
        endpoint_url=AWS_ENDPOINT_URL,
        region_name=AWS_REGION,
        access_key=AWS_ACCESS_KEY_ID or "",
        secret_key=AWS_SECRET_ACCESS_KEY or "",
    )

    extractor_service = ExtractorService(
        sqs_client=sqs_client,
        s3_client=s3_client,
        writer_queue_url=WRITER_QUEUE_URL,
        images_bucket=IMAGES_BUCKET,
        llm_provider=LLM_PROVIDER,
    )

    # Main Loop
    while True:
        try:
            messages = sqs_client.receive_messages(INPUT_QUEUE_URL)
            for message in messages:
                await extractor_service.process_message(message["Body"])

                # Delete message
                sqs_client.delete_message(INPUT_QUEUE_URL, message["ReceiptHandle"])

        except Exception as e:
            logger.error(f"Polling error: {e}")
            time.sleep(5)


if __name__ == "__main__":  # pragma: no cover
    asyncio.run(main())
