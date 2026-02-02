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
    aws_endpoint_url = os.getenv("AWS_ENDPOINT_URL")
    aws_region = os.getenv("AWS_REGION", "us-east-1")
    aws_access_key_id = os.getenv("AWS_ACCESS_KEY_ID")
    aws_secret_access_key = os.getenv("AWS_SECRET_ACCESS_KEY")
    input_queue_url = os.getenv("INPUT_QUEUE_URL") or ""
    writer_queue_url = os.getenv("WRITER_QUEUE_URL") or ""
    images_bucket = os.getenv("IMAGES_BUCKET") or "isidorus-images"
    llm_provider = os.getenv("LLM_PROVIDER", "openai")

    if not input_queue_url or not writer_queue_url:
        logger.error("INPUT_QUEUE_URL and WRITER_QUEUE_URL must be set")
        return

    # Dependency Injection
    sqs_client = SQSClient(
        endpoint_url=aws_endpoint_url,
        region=aws_region,
        access_key=aws_access_key_id or "",
        secret_key=aws_secret_access_key or "",
    )

    s3_client = S3Client(
        endpoint_url=aws_endpoint_url,
        region_name=aws_region,
        access_key=aws_access_key_id or "",
        secret_key=aws_secret_access_key or "",
    )

    extractor_service = ExtractorService(
        sqs_client=sqs_client,
        s3_client=s3_client,
        writer_queue_url=writer_queue_url,
        images_bucket=images_bucket,
        llm_provider=llm_provider,
    )

    # Main Loop
    while True:
        try:
            messages = sqs_client.receive_messages(input_queue_url)
            for message in messages:
                await extractor_service.process_message(message["Body"])

                # Delete message
                sqs_client.delete_message(input_queue_url, message["ReceiptHandle"])

        except Exception as e:  # pylint: disable=broad-exception-caught
            logger.error("Polling error: %s", e)
            time.sleep(5)


if __name__ == "__main__":  # pragma: no cover
    asyncio.run(main())
