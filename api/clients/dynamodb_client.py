import logging
from typing import Any, cast

import aioboto3  # type: ignore

logger = logging.getLogger(__name__)


class DynamoDBClient:
    def __init__(
        self,
        endpoint_url: str | None,
        region: str,
        access_key: str | None,
        secret_key: str | None,
        table_name: str,
    ) -> None:
        self.endpoint_url = endpoint_url
        self.region = region
        self.access_key = access_key
        self.secret_key = secret_key
        self.table_name = table_name
        self.session = aioboto3.Session()

    async def put_item(self, item: dict) -> bool:
        try:
            async with self.session.resource(
                "dynamodb",
                endpoint_url=self.endpoint_url,
                region_name=self.region,
                aws_access_key_id=self.access_key,
                aws_secret_access_key=self.secret_key,
            ) as dynamodb:
                table = await dynamodb.Table(self.table_name)
                await table.put_item(Item=item)
                return True
        except Exception as e:
            logger.error(f"Failed to put item to DynamoDB: {e}")
            raise e

    async def get_item(self, key: dict) -> dict[Any, Any] | None:
        try:
            async with self.session.resource(
                "dynamodb",
                endpoint_url=self.endpoint_url,
                region_name=self.region,
                aws_access_key_id=self.access_key,
                aws_secret_access_key=self.secret_key,
            ) as dynamodb:
                table = await dynamodb.Table(self.table_name)
                response = await table.get_item(Key=key)
                item = response.get("Item")
                return cast(dict[Any, Any], item) if item else None
        except Exception as e:
            logger.error(f"Failed to get item from DynamoDB: {e}")
            raise e
