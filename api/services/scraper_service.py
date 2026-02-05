from datetime import datetime, timezone
from typing import TypedDict, cast

from api.clients.dynamodb_client import DynamoDBClient
from api.clients.redis_client import RedisClient
from api.clients.sqs_client import SQSClient
from api.repositories.db_repository import (
    DbRepository,
    ScrapedPageRecord,
    ScrapingRecord,
)


class ScrapingNotFoundError(Exception):
    """Exception raised when a scraping is not found."""


class NotAuthorizedError(Exception):
    """Exception raised when a user is not authorized to perform an action."""


class ScrapingMetadata(TypedDict):
    status: str
    created_at: str | None
    completed_at: str | None
    depth: int
    links_count: int
    pages: list[ScrapedPageRecord] | None


class FullScrapingRecord(ScrapingRecord, ScrapingMetadata):
    pass


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
            now = datetime.now(timezone.utc).isoformat()
            await self.dynamodb_client.put_item(
                {
                    "scraping_id": str(scraping_id),
                    "url": url,
                    "depth": depth,
                    "status": "PENDING",
                    "links_count": 0,
                    "created_at": now,
                }
            )

        # Send first message to Scraper Queue
        message = {
            "url": url,
            "depth": depth,
            "scraping_id": scraping_id,
            "user_id": user_id,
        }
        await self.sqs_client.send_message(message)

        return scraping_id

    async def get_full_scraping(self, scraping_id: int) -> FullScrapingRecord | None:
        """
        Retrieves the status of a scraping session, using DynamoDB as
        the source of truth for the scraping metadata (status, timestamps).
        """
        # 1. Get identity from Postgres
        id_record = await self.db_repository.get_scraping(scraping_id)
        if not id_record:
            return None

        # 2. Get scraping metadata from DynamoDB
        scraping_metadata: ScrapingMetadata = {
            "status": "UNKNOWN",
            "created_at": None,
            "completed_at": None,
            "depth": 1,
            "links_count": 0,
            "pages": None,
        }
        if self.dynamodb_client:
            item = await self.dynamodb_client.get_item(
                {"scraping_id": str(scraping_id)}
            )
            if item:
                scraping_metadata = {
                    "status": item.get("status", "UNKNOWN"),
                    "created_at": item.get("created_at"),
                    "completed_at": item.get("completed_at"),
                    "depth": int(item.get("depth", 1)),
                    "links_count": int(item.get("links_count", 0)),
                    "pages": None,
                }

        # 3. Merge
        full_scraping_record = {**id_record, **scraping_metadata}
        return cast(FullScrapingRecord, full_scraping_record)

    async def get_full_scrapings(
        self, user_id: int, offset: int = 0, limit: int = 10
    ) -> tuple[list[FullScrapingRecord], int]:
        """
        Retrieves all scrapings for a user, merging data from Postgres and DynamoDB.
        """
        scrapings, total = await self.db_repository.get_scrapings(
            user_id, offset, limit
        )

        merged_scrapings: list[FullScrapingRecord] = []
        for scraping in scrapings:
            merged_scraping: FullScrapingRecord = cast(FullScrapingRecord, scraping)
            sid = str(scraping["id"])
            links_count = 0
            status = "UNKNOWN"

            if self.dynamodb_client:
                item = await self.dynamodb_client.get_item({"scraping_id": sid})
                if item:
                    links_count = int(item.get("links_count", 0))
                    status = item.get("status", "UNKNOWN")
                    merged_scraping = cast(
                        FullScrapingRecord,
                        {
                            **scraping,
                            **{
                                "status": status,
                                "created_at": item.get("created_at"),
                                "completed_at": item.get("completed_at"),
                                "depth": int(item.get("depth", 1)),
                                "links_count": links_count,
                                "pages": None,
                            },
                        },
                    )

            merged_scrapings.append(merged_scraping)

        return merged_scrapings, total

    async def get_scraping_results(self, scraping_id: int) -> list[ScrapedPageRecord]:
        """
        Retrieves the results of a scraping session.
        """
        return await self.db_repository.get_scraping_results(scraping_id)

    async def delete_scraping(self, scraping_id: int, user_id: int) -> bool:
        """
        Initiates the deletion of a scraping job.
        Verifies existence and ownership before enqueuing to the deletion worker.
        """
        scraping_record = await self.db_repository.get_scraping(scraping_id)
        if not scraping_record:
            raise ScrapingNotFoundError(f"Scraping with ID {scraping_id} not found")

        if scraping_record["user_id"] != user_id:
            raise NotAuthorizedError(
                f"User {user_id} is not authorized to delete scraping {scraping_id}"
            )

        return await self.enqueue_deletion(scraping_id)

    async def enqueue_deletion(self, scraping_id: int) -> bool:
        """
        Sends a message to the deletion queue to handle full deletion asynchronously.
        """
        if not self.deletion_queue_url:
            return False

        message = {"scraping_id": scraping_id}
        await self.sqs_client.send_message(message, queue_url=self.deletion_queue_url)
        return True
