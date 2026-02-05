from typing import Any

from api.clients.dynamodb_client import DynamoDBClient
from api.clients.redis_client import RedisClient
from api.clients.sqs_client import SQSClient
from api.repositories.db_repository import DbRepository


class ScraperService:
    def __init__(  # pylint: disable=too-many-arguments,too-many-positional-arguments
        self,
        sqs_client: SQSClient,
        redis_client: RedisClient,
        db_repository: DbRepository,
        dynamodb_client: DynamoDBClient | None = None,
        deletion_queue_url: str | None = None,
    ):
        self.sqs_client = sqs_client
        self.redis_client = redis_client
        self.db_repository = db_repository
        self.dynamodb_client = dynamodb_client
        self.deletion_queue_url = deletion_queue_url

    async def start_scraping(
        self, url: str, depth: int, user_id: int | None = None
    ) -> int:
        """
        Starts a new scraping job.
        """
        scraping_id = await self.db_repository.create_scraping(url, user_id)

        # Initial Redis Key for Distributed Completion Tracking
        pending_key = f"scrape:{scraping_id}:pending"
        await self.redis_client.set(pending_key, 1)

        # Log to DynamoDB if client is available
        if self.dynamodb_client:
            from datetime import (  # pylint: disable=import-outside-toplevel
                datetime,
                timezone,
            )

            now = datetime.now(timezone.utc).isoformat()
            await self.dynamodb_client.put_item(
                {
                    "scraping_id": str(scraping_id),
                    "url": url,
                    "depth": depth,
                    "status": "PENDING",
                    "created_at": now,
                }
            )

        # Send first message to Scraper Queue
        message = {
            "url": url,
            "depth": depth,
            "scraping_id": scraping_id,
        }
        await self.sqs_client.send_message(message)

        return scraping_id

    async def get_scraping_status(self, scraping_id: int) -> dict[str, Any] | None:
        """
        Retrieves the status of a scraping session, using DynamoDB as
        the source of truth for lifecycle state (status, timestamps).
        """
        # 1. Get identity from Postgres
        scraping_pg = await self.db_repository.get_scraping(scraping_id)
        if not scraping_pg:
            return None

        # 2. Get status and timestamps from DynamoDB
        status_data: dict[str, Any] = {
            "status": "UNKNOWN",
            "created_at": None,
            "completed_at": None,
        }
        if self.dynamodb_client:
            item = await self.dynamodb_client.get_item(
                {"scraping_id": str(scraping_id)}
            )
            if item:
                status_data = {
                    "status": item.get("status", "UNKNOWN"),
                    "created_at": item.get("created_at"),
                    "completed_at": item.get("completed_at"),
                    "depth": int(item.get("depth", 1)),
                }

        # 3. Merge
        return {**scraping_pg, **status_data}

    async def get_scraping_results(self, scraping_id: int) -> list[dict[str, Any]]:
        """
        Retrieves the results of a scraping session.
        """
        return await self.db_repository.get_scrape_results(scraping_id)

    async def enqueue_deletion(self, scraping_id: int) -> bool:
        """
        Sends a message to the deletion queue to handle full deletion asynchronously.
        """
        if not self.deletion_queue_url:
            return False

        message = {"scraping_id": scraping_id}
        await self.sqs_client.send_message(message, queue_url=self.deletion_queue_url)
        return True
