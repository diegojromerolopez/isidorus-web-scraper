import logging

import boto3
from botocore.exceptions import ClientError  # type: ignore

logger = logging.getLogger(__name__)


class S3Client:
    # pylint: disable=too-few-public-methods
    def __init__(
        self,
        endpoint_url: str | None = None,
        region_name: str = "us-east-1",
        access_key: str | None = None,
        secret_key: str | None = None,
    ):
        self.client = boto3.client(
            "s3",
            endpoint_url=endpoint_url,
            region_name=region_name,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )

    def upload_bytes(
        self,
        data: bytes,
        bucket: str,
        key: str,
        content_type: str = "application/octet-stream",
    ) -> str:
        try:
            self.client.put_object(
                Body=data, Bucket=bucket, Key=key, ContentType=content_type
            )
            return f"s3://{bucket}/{key}"
        except ClientError as e:
            logger.error("Failed to upload to S3: %s", e)
            raise
