from typing import Any

from api.clients.dynamodb_client import DynamoDBClient
from api.clients.redis_client import RedisClient
from api.clients.sqs_client import SQSClient
from api.repositories.db_repository import DbRepository


class ScraperService:
    def __init__(
        self,
        sqs_client: SQSClient,
        redis_client: RedisClient,
        db_repository: DbRepository,
        dynamodb_client: DynamoDBClient | None = None,
    ):
        self.sqs_client = sqs_client
        self.redis_client = redis_client
        self.db_repository = db_repository
        self.dynamodb_client = dynamodb_client

    async def start_scraping(self, url: str, depth: int) -> int:
        """
        Starts a new scraping job.
        """
        scraping_id = await self.db_repository.create_scraping(url)

        # Initial Redis Key for Distributed Completion Tracking
        pending_key = f"scrape:{scraping_id}:pending"
        self.redis_client.set(pending_key, 1)

        # Log to DynamoDB if client is available
        if self.dynamodb_client:
            self.dynamodb_client.put_item(
                {
                    "job_id": str(scraping_id),
                    "url": url,
                    "depth": depth,
                    "status": "PENDING",
                }
            )

        # Send first message to Scraper Queue
        message = {
            "url": url,
            "depth": depth,
            "scraping_id": scraping_id,
        }
        self.sqs_client.send_message(message)

        return scraping_id

    async def get_scraping_status(self, scraping_id: int) -> dict[str, Any] | None:
        """
        Retrieves the status of a scraping session.
        """
        return await self.db_repository.get_scraping(scraping_id)

    async def get_scraping_results(self, scraping_id: int) -> list[dict[str, Any]]:
        """
        Retrieves the results of a scraping session.
        """
        return await self.db_repository.get_scrape_results(scraping_id)
