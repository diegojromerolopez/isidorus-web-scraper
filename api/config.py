import os
from dataclasses import dataclass

from shared.config import Configuration as BaseConfiguration


@dataclass
class Configuration(BaseConfiguration):
    """
    API-specific configuration.
    """

    deletion_queue_url: str

    @classmethod
    def from_env(cls) -> "Configuration":
        """
        Loads API configuration from environment variables.
        """
        return cls(
            # Base fields
            aws_endpoint_url=os.getenv("AWS_ENDPOINT_URL", "http://localstack:4566"),
            aws_region=os.getenv("AWS_REGION", "us-east-1"),
            aws_access_key_id=os.getenv("AWS_ACCESS_KEY_ID", "test"),
            aws_secret_access_key=os.getenv("AWS_SECRET_ACCESS_KEY", "test"),
            sqs_queue_url=os.getenv(
                "SQS_QUEUE_URL",
                "http://localstack:4566/000000000000/scraper-queue",
            ),
            dynamodb_table=os.getenv("DYNAMODB_TABLE", "scraping_jobs"),
            database_url=os.getenv(
                "DATABASE_URL",
                "postgres://postgres:postgres@postgres:5432/isidorus",
            ),
            redis_host=os.getenv("REDIS_HOST", "redis"),
            redis_port=int(os.getenv("REDIS_PORT", "6379")),
            # API fields
            deletion_queue_url=os.getenv(
                "DELETION_QUEUE_URL",
                "http://localstack:4566/000000000000/deletion-queue",
            ),
        )


__all__ = ["Configuration"]
