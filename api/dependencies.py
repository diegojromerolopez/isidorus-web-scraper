import os

from fastapi import Depends

from api.clients.dynamodb_client import DynamoDBClient
from api.clients.redis_client import RedisClient
from api.clients.sqs_client import SQSClient
from api.repositories.db_repository import DbRepository
from api.services.db_service import DbService
from api.services.scraper_service import ScraperService

# Configuration
AWS_ENDPOINT_URL = os.getenv("AWS_ENDPOINT_URL", "http://localstack:4566")
AWS_REGION = os.getenv("AWS_REGION", "us-east-1")
AWS_ACCESS_KEY_ID = os.getenv("AWS_ACCESS_KEY_ID", "test")
AWS_SECRET_ACCESS_KEY = os.getenv("AWS_SECRET_ACCESS_KEY", "test")
SQS_QUEUE_URL = os.getenv(
    "SQS_QUEUE_URL", "http://localstack:4566/000000000000/scraper-queue"
)
DYNAMODB_TABLE = os.getenv("DYNAMODB_TABLE", "scraping_jobs")
DATABASE_URL = os.getenv(
    "DATABASE_URL", "postgres://postgres:postgres@postgres:5432/isidorus"
)
REDIS_HOST = os.getenv("REDIS_HOST", "redis")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))


def get_sqs_client() -> SQSClient:
    """
    Creates and returns an SQSClient instance configured with environment variables.
    """
    return SQSClient(
        endpoint_url=AWS_ENDPOINT_URL,
        region=AWS_REGION,
        access_key=AWS_ACCESS_KEY_ID,
        secret_key=AWS_SECRET_ACCESS_KEY,
        queue_url=SQS_QUEUE_URL,
    )


def get_dynamodb_client() -> DynamoDBClient:
    """
    Creates and returns a DynamoDBClient instance configured with environment variables.
    """
    return DynamoDBClient(
        endpoint_url=AWS_ENDPOINT_URL,
        region=AWS_REGION,
        access_key=AWS_ACCESS_KEY_ID,
        secret_key=AWS_SECRET_ACCESS_KEY,
        table_name=DYNAMODB_TABLE,
    )


def get_db_repository() -> DbRepository:
    """
    Creates and returns a DbRepository instance connected to the configured database.
    """
    if DATABASE_URL is None:
        raise ValueError("DATABASE_URL must be set")
    return DbRepository()


def get_redis_client() -> RedisClient:
    """
    Dependency to get the Redis client.
    """
    return RedisClient(host=REDIS_HOST, port=REDIS_PORT)


def get_scraper_service(
    sqs_client: SQSClient = Depends(get_sqs_client),
    redis_client: RedisClient = Depends(get_redis_client),
    dynamodb_client: DynamoDBClient = Depends(get_dynamodb_client),
    db_repository: DbRepository = Depends(get_db_repository),
) -> ScraperService:
    """
    Dependency provider for ScraperService.
    Requires SQSClient, RedisClient, DynamoDBClient, DbRepository.
    """
    return ScraperService(sqs_client, redis_client, db_repository, dynamodb_client)


def get_db_service(
    repository: DbRepository = Depends(get_db_repository),
) -> DbService:
    """
    Dependency to get the DB service.
    """
    return DbService(repository)
