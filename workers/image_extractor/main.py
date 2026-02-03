import asyncio
import logging
import time

from shared.clients.s3_client import S3Client
from shared.clients.sqs_client import SQSClient
from workers.image_extractor.config import Configuration
from workers.image_extractor.services.extractor_service import ExtractorService

# Logging setup
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def main() -> None:
    logger.info("Image Extractor Worker started (Refactor)")

    # Configuration
    config = Configuration.from_env()

    if not config.input_queue_url or not config.writer_queue_url:
        logger.error("INPUT_QUEUE_URL and WRITER_QUEUE_URL must be set")
        return

    # Dependency Injection
    sqs_client = SQSClient.create(config)
    s3_client = S3Client.create(config)

    extractor_service = ExtractorService(
        sqs_client=sqs_client,
        s3_client=s3_client,
        writer_queue_url=config.writer_queue_url,
        images_bucket=config.images_bucket,
        llm_provider=config.llm_provider,
        llm_api_key=config.llm_api_key,
    )

    # Main Loop
    while True:
        try:
            # SQSClient.receive_messages is now async
            messages = await sqs_client.receive_messages(config.input_queue_url)
            for message in messages:
                await extractor_service.process_message(message["Body"])

                # Delete message is now async
                await sqs_client.delete_message(
                    config.input_queue_url, message["ReceiptHandle"]
                )

        except Exception as e:  # pylint: disable=broad-exception-caught
            logger.error("Polling error: %s", e)
            time.sleep(5)


if __name__ == "__main__":  # pragma: no cover
    asyncio.run(main())
