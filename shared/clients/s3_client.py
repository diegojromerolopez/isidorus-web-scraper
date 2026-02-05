import logging

import aioboto3  # type: ignore
from botocore.exceptions import ClientError  # type: ignore

from shared.config import Configuration

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
        self.__endpoint_url = endpoint_url
        self.__region_name = region_name
        self.__access_key = access_key
        self.__secret_key = secret_key
        self.__session = aioboto3.Session()

    @staticmethod
    def create(config: Configuration) -> "S3Client":
        """
        Creates an S3Client instance from the configuration.
        """
        return S3Client(
            endpoint_url=config.aws_endpoint_url,
            region_name=config.aws_region,
            access_key=config.aws_access_key_id,
            secret_key=config.aws_secret_access_key,
        )

    async def upload_bytes(
        self,
        data: bytes,
        bucket: str,
        key: str,
        content_type: str = "application/octet-stream",
    ) -> str:
        try:
            async with self.__session.client(
                "s3",
                endpoint_url=self.__endpoint_url,
                region_name=self.__region_name,
                aws_access_key_id=self.__access_key,
                aws_secret_access_key=self.__secret_key,
            ) as client:
                await client.put_object(
                    Body=data, Bucket=bucket, Key=key, ContentType=content_type
                )
                return f"s3://{bucket}/{key}"
        except ClientError as e:
            logger.error("Failed to upload to S3: %s", e)
            raise

    async def delete_object(self, bucket: str, key: str) -> bool:
        try:
            async with self.__session.client(
                "s3",
                endpoint_url=self.__endpoint_url,
                region_name=self.__region,
                aws_access_key_id=self.__access_key,
                aws_secret_access_key=self.__secret_key,
            ) as client:
                await client.delete_object(Bucket=bucket, Key=key)
                return True
        except ClientError as e:
            logger.error("Failed to delete from S3: %s", e)
            raise

    async def delete_objects(self, bucket: str, keys: list[str]) -> bool:
        """
        Deletes multiple objects from S3 in a single request (max 1000).
        """
        if not keys:
            return True
        try:
            async with self.__session.client(
                "s3",
                endpoint_url=self.__endpoint_url,
                region_name=self.__region,
                aws_access_key_id=self.__access_key,
                aws_secret_access_key=self.__secret_key,
            ) as client:
                delete_dict = {"Objects": [{"Key": k} for k in keys]}
                await client.delete_objects(Bucket=bucket, Delete=delete_dict)
                return True
        except ClientError as e:
            logger.error("Failed to batch delete from S3: %s", e)
            raise e

