import hashlib
from datetime import datetime, timezone
from typing import cast

from fastapi import Depends, HTTPException, Security, status
from fastapi.security import APIKeyHeader

from api.clients.dynamodb_client import DynamoDBClient
from api.clients.redis_client import RedisClient
from api.clients.sqs_client import SQSClient
from api.config import Configuration
from api.models import APIKey
from api.repositories.db_repository import DbRepository
from api.repositories.search_repository import SearchRepository
from api.services.db_service import DbService
from api.services.scraper_service import ScraperService
from api.services.search_service import SearchService

config = Configuration.from_env()

API_KEY_HEADER = APIKeyHeader(name="X-API-Key", auto_error=False)


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


def get_search_repository() -> SearchRepository:
    """
    Creates and returns a SearchRepository instance for OpenSearch.
    """
    return SearchRepository(config)


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
    return ScraperService(
        sqs_client,
        redis_client,
        db_repository,
        dynamodb_client,
        config.deletion_queue_url,
    )


def get_db_service(
    repository: DbRepository = Depends(get_db_repository),
) -> DbService:
    """
    Dependency to get the DB service.
    """
    return DbService(repository)


def get_search_service(
    repository: SearchRepository = Depends(get_search_repository),
) -> SearchService:
    """
    Dependency to get the search service.
    """
    return SearchService(repository)


async def get_api_key(
    api_key_header: str | None = Security(API_KEY_HEADER),
    redis_client: RedisClient = Depends(get_redis_client),
) -> APIKey:
    """
    Validates the API key from the header.
    Uses Redis for caching and Postgres as the source of truth.
    """
    if not api_key_header:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="API Key missing",
        )

    # Hash the key to look it up
    hashed_key = hashlib.sha256(api_key_header.encode()).hexdigest()
    cache_key = f"auth:key:{hashed_key}"

    # 1. Try Redis Cache
    cached_val = await redis_client.get(cache_key)
    if cached_val:
        # Format: "name:user_id"
        try:
            name, user_id_str = cached_val.split(":", 1)
            return APIKey(
                name=name,
                user_id=int(user_id_str),
                hashed_key=hashed_key,
                is_active=True,
            )
        except ValueError:
            # If cache is malformed (or old format), fallback to DB
            pass

    # 2. Try Database
    api_key = await APIKey.filter(hashed_key=hashed_key, is_active=True).first()
    if not api_key:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid API Key",
        )

    # Check expiration
    if api_key.expires_at and api_key.expires_at < datetime.now(timezone.utc):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="API Key expired",
        )

    # 3. Cache it for 5 minutes
    # Store as "name:user_id"
    await redis_client.set(cache_key, f"{api_key.name}:{api_key.user_id}", ex=300)

    # Update last_used_at (Asynchronous fire-and-forget or background
    # task would be better)
    # For now, let's just update it periodically or skip to keep it fast
    return cast(APIKey, api_key)
