import logging
from typing import Any, cast

import boto3

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
        self.dynamodb = boto3.resource(
            "dynamodb",
            endpoint_url=endpoint_url,
            region_name=region,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )
        self.table = self.dynamodb.Table(table_name)

    def put_item(self, item: dict) -> bool:
        try:
            self.table.put_item(Item=item)
            return True
        except Exception as e:
            logger.error(f"Failed to put item to DynamoDB: {e}")
            raise e

    def get_item(self, key: dict) -> dict[Any, Any] | None:
        try:
            response = self.table.get_item(Key=key)
            item = response.get("Item")
            return cast(dict[Any, Any], item) if item else None
        except Exception as e:
            logger.error(f"Failed to get item from DynamoDB: {e}")
            raise e
