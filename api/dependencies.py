from fastapi import Depends

from api.clients.dynamodb_client import DynamoDBClient
from api.clients.redis_client import RedisClient
from api.clients.sqs_client import SQSClient
from api.config import Configuration
from api.repositories.db_repository import DbRepository
from api.services.db_service import DbService
from api.services.scraper_service import ScraperService

config = Configuration.from_env()


def get_sqs_client() -> SQSClient:
    """
    Creates and returns an SQSClient instance configured with environment variables.
    """
    return SQSClient.create(config)


def get_dynamodb_client() -> DynamoDBClient:
    """
    Creates and returns a DynamoDBClient instance configured with environment variables.
    """
    return DynamoDBClient.create(config)


def get_db_repository() -> DbRepository:
    """
    Creates and returns a DbRepository instance connected to the configured database.
    """
    return DbRepository()


def get_redis_client() -> RedisClient:
    """
    Dependency to get the Redis client.
    """
    return RedisClient.create(config)


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
