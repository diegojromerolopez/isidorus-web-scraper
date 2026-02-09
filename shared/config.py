import os
from dataclasses import dataclass


@dataclass
class Configuration:  # pylint: disable=too-many-instance-attributes
    """
    Application configuration loaded from environment variables.
    """

    base_endpoint_url: str
    s3_endpoint_url: str
    sqs_endpoint_url: str
    dynamodb_endpoint_url: str
    aws_region: str
    aws_access_key_id: str
    aws_secret_access_key: str
    sqs_queue_url: str
    dynamodb_table: str
    database_url: str
    redis_host: str
    redis_port: int

    @classmethod
    def from_env(cls) -> "Configuration":
        """
        Loads configuration from environment variables with defaults.
        """
        base_endpoint = os.getenv("BASE_ENDPOINT_URL", "http://localstack:4566")
        return cls(
            base_endpoint_url=base_endpoint,
            s3_endpoint_url=os.getenv("S3_ENDPOINT_URL", base_endpoint),
            sqs_endpoint_url=os.getenv("SQS_ENDPOINT_URL", base_endpoint),
            dynamodb_endpoint_url=os.getenv("DYNAMODB_ENDPOINT_URL", base_endpoint),
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
        )
