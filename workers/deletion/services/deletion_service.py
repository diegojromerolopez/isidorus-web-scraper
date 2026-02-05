import logging
from typing import Any

from opensearchpy import AsyncOpenSearch

from api import models as api_models
from api.clients.dynamodb_client import DynamoDBClient
from shared.clients.s3_client import S3Client

logger = logging.getLogger(__name__)


class DeletionService:  # pylint: disable=too-few-public-methods
    def __init__(
        self,
        dynamodb_client: DynamoDBClient,
        s3_client: S3Client,
        os_client: AsyncOpenSearch,
        images_bucket: str,
        batch_size: int = 5000,
        s3_batch_size: int = 1000,
    ):
        self.__dynamodb_client = dynamodb_client
        self.__s3_client = s3_client
        self.__os_client = os_client
        self.__images_bucket = images_bucket
        self.__batch_size = batch_size
        self.__s3_batch_size = s3_batch_size

    async def cleanup_scraping(self, scraping_id: int) -> bool:
        """
        Orchestrates the full deletion of a scraping job.
        """
        logger.info("Starting cleanup for scraping_id: %s", scraping_id)

        try:
            # 1. Delete S3 Objects first (we need the paths from DB)
            await self.__cleanup_s3_objects(scraping_id)

            # 2. Delete OpenSearch Data
            await self.__cleanup_opensearch_data(scraping_id)

            # 3. Delete Relational Data in batches
            await self.__cleanup_relational_data(scraping_id)

            # 3. Delete from DynamoDB
            await self.__dynamodb_client.delete_item({"scraping_id": str(scraping_id)})

            # 4. Final SQL deletion of the Scraping record
            scraping = await api_models.Scraping.get_or_none(id=scraping_id)
            if scraping:
                await scraping.delete()

            logger.info("Successfully cleaned up scraping_id: %s", scraping_id)
            return True
        except Exception as e:
            logger.error("Failed to cleanup scraping_id %s: %s", scraping_id, e)
            raise e

    async def __cleanup_s3_objects(self, scraping_id: int) -> None:
        """
        Fetches and deletes S3 objects associated with the scraping.
        """
        # Fetch S3 paths in chunks
        offset = 0
        while True:
            images = (
                await api_models.PageImage.filter(scraping_id=scraping_id)
                .offset(offset)
                .limit(self.__s3_batch_size)
                .values_list("s3_path", flat=True)
            )
            if not images:
                break

            keys = []
            for path in images:
                if path and path.startswith("s3://"):
                    parts = path[5:].split("/", 1)
                    if len(parts) == 2:
                        # parts[0] is bucket, parts[1] is key
                        keys.append(parts[1])
                    elif len(parts) == 1:
                        keys.append(parts[0])

            if keys:
                await self.__s3_client.delete_objects(self.__images_bucket, keys)

            if len(images) < self.__s3_batch_size:
                break
            offset += self.__s3_batch_size

    async def __cleanup_relational_data(self, scraping_id: int) -> None:
        """
        Deletes related records in batches to avoid locking issues.
        """
        # Order of deletion is important for foreign keys if not cascaded,
        # but here we use manual batching for performance.

        # 2. Delete PageLinks
        await self.__batch_delete(api_models.PageLink, scraping_id)

        # 3. Delete PageImages
        await self.__batch_delete(api_models.PageImage, scraping_id)

        # 4. Delete ScrapedPages
        # Note: ScrapedPage is the parent of the above.
        await self.__batch_delete(api_models.ScrapedPage, scraping_id)

    async def __batch_delete(self, model_class: Any, scraping_id: int) -> None:
        """
        Helper to delete records in batches.
        """
        while True:
            # Get IDs for the next batch
            ids = (
                await model_class.filter(scraping_id=scraping_id)
                .limit(self.__batch_size)
                .values_list("id", flat=True)
            )
            if not ids:
                break

            # Delete by ID in a single query
            await model_class.filter(id__in=ids).delete()

            if len(ids) < self.__batch_size:
                break
            # We don't need offset because records are being deleted

    async def __cleanup_opensearch_data(self, scraping_id: int) -> None:
        """
        Deletes documents from OpenSearch associated with the scraping.
        """
        logger.info("Cleaning up OpenSearch for scraping_id: %s", scraping_id)
        try:
            query = {"query": {"term": {"scraping_id": scraping_id}}}
            await self.__os_client.delete_by_query(
                index="scraped_pages",
                body=query,
                wait_for_completion=True,
                refresh=True,
            )
            logger.info("OpenSearch cleanup finished for scraping_id: %s", scraping_id)
        except Exception as e:
            # We log but don't fail the whole cleanup if OpenSearch fails
            # This is to avoid leaving inconsistent state in DB/S3
            logger.error(
                "Failed to cleanup OpenSearch for scraping_id %s: %s", scraping_id, e
            )
