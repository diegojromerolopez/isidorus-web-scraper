import logging
from typing import Any, cast

import aioboto3  # type: ignore

from api.config import Configuration

logger = logging.getLogger(__name__)


class DynamoDBClient:
    def __init__(  # pylint: disable=too-many-arguments,too-many-positional-arguments
        self,
        endpoint_url: str | None,
        region: str,
        access_key: str | None,
        secret_key: str | None,
        table_name: str,
    ) -> None:
        self.__endpoint_url = endpoint_url
        self.__region = region
        self.__access_key = access_key
        self.__secret_key = secret_key
        self.__table_name = table_name
        self.__session = aioboto3.Session()

    @staticmethod
    def create(config: Configuration) -> "DynamoDBClient":
        """
        Creates a DynamoDBClient instance from the configuration.
        """
        return DynamoDBClient(
            endpoint_url=config.aws_endpoint_url,
            region=config.aws_region,
            access_key=config.aws_access_key_id,
            secret_key=config.aws_secret_access_key,
            table_name=config.dynamodb_table,
        )

    async def put_item(self, item: dict) -> bool:
        try:
            async with self.__session.resource(
                "dynamodb",
                endpoint_url=self.__endpoint_url,
                region_name=self.__region,
                aws_access_key_id=self.__access_key,
                aws_secret_access_key=self.__secret_key,
            ) as dynamodb:
                table = await dynamodb.Table(self.__table_name)
                await table.put_item(Item=item)
                return True
        except Exception as e:
            logger.error("Failed to put item: %s", e)
            raise e

    async def get_item(self, key: dict) -> dict[Any, Any] | None:
        try:
            async with self.__session.resource(
                "dynamodb",
                endpoint_url=self.__endpoint_url,
                region_name=self.__region,
                aws_access_key_id=self.__access_key,
                aws_secret_access_key=self.__secret_key,
            ) as dynamodb:
                table = await dynamodb.Table(self.__table_name)
                response = await table.get_item(Key=key)
                item = response.get("Item")
                return cast(dict[Any, Any], item) if item else None
        except Exception as e:
            logger.error("Failed to get item from DynamoDB: %s", e)
            raise e

    async def delete_item(self, key: dict) -> bool:
        try:
            async with self.__session.resource(
                "dynamodb",
                endpoint_url=self.__endpoint_url,
                region_name=self.__region,
                aws_access_key_id=self.__access_key,
                aws_secret_access_key=self.__secret_key,
            ) as dynamodb:
                table = await dynamodb.Table(self.__table_name)
                await table.delete_item(Key=key)
                return True
        except Exception as e:
            logger.error("Failed to delete item from DynamoDB: %s", e)
            raise e
