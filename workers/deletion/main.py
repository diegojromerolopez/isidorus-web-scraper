import asyncio
import json
import logging
import os
import signal
from tortoise import Tortoise

from shared.clients.s3_client import S3Client
from api.clients.dynamodb_client import DynamoDBClient
from shared.clients.sqs_client import SQSClient
from workers.deletion.config import Configuration
from workers.deletion.services.deletion_service import DeletionService

# Logging configuration
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

async def init_db(database_url: str):
    await Tortoise.init(
        db_url=database_url,
        modules={"models": ["api.models"]},
    )

async def main(stop_event: asyncio.Event | None = None):
    # Configuration
    config = Configuration.from_env()

    if not config.input_queue_url:
        logger.error("INPUT_QUEUE_URL must be set")
        return

    await init_db(config.database_url)

    sqs_client = SQSClient(
        endpoint_url=config.aws_endpoint_url,
        region=config.aws_region,
        access_key=config.aws_access_key_id,
        secret_key=config.aws_secret_access_key,
        queue_url=config.input_queue_url,
    )

    dynamodb_client = DynamoDBClient(
        endpoint_url=config.aws_endpoint_url,
        region=config.aws_region,
        access_key=config.aws_access_key_id,
        secret_key=config.aws_secret_access_key,
        table_name=config.dynamodb_table,
    )

    s3_client = S3Client(
        endpoint_url=config.aws_endpoint_url,
        region_name=config.aws_region,
        access_key=config.aws_access_key_id,
        secret_key=config.aws_secret_access_key,
    )

    deletion_service = DeletionService(
        dynamodb_client=dynamodb_client,
        s3_client=s3_client,
        images_bucket=config.images_bucket,
    )

    logger.info("Deletion worker started. Listening for deletion requests...")

    if stop_event is None:
        stop_event = asyncio.Event()

    def signal_handler():
        logger.info("Shutdown signal received")
        if stop_event:
            stop_event.set()

    try:
        loop = asyncio.get_running_loop()
        for sig in (signal.SIGINT, signal.SIGTERM):
            loop.add_signal_handler(sig, signal_handler)
    except NotImplementedError:
        # Signal handlers not supported on some platforms (e.g. Windows)
        pass

    while not stop_event.is_set():
        try:
            messages = await sqs_client.receive_messages(config.input_queue_url, max_messages=1, wait_time=5)
            for msg in messages:
                try:
                    body = json.loads(msg["Body"])
                    scraping_id = body.get("scraping_id")
                    if scraping_id:
                        await deletion_service.cleanup_scraping(scraping_id)
                    
                    await sqs_client.delete_message(config.input_queue_url, msg["ReceiptHandle"])
                except Exception as e:
                    logger.error(f"Error processing message: {e}")
        except Exception as e:
            logger.error(f"Error receiving messages: {e}")
            await asyncio.sleep(5)

    await Tortoise.close_connections()

if __name__ == "__main__":
    asyncio.run(main())
