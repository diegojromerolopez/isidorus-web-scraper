import json
import logging
from typing import Any

import aioboto3  # type: ignore

from shared.config import Configuration

logger = logging.getLogger(__name__)


class SQSClient:
    # pylint: disable=too-few-public-methods
    def __init__(  # pylint: disable=too-many-arguments,too-many-positional-arguments
        self,
        endpoint_url: str | None,
        region: str,
        access_key: str | None,
        secret_key: str | None,
        queue_url: str | None,
    ) -> None:
        self.__endpoint_url = endpoint_url
        self.__region = region
        self.__access_key = access_key
        self.__secret_key = secret_key
        self.__queue_url = queue_url
        self.__session = aioboto3.Session()

    @staticmethod
    def create(config: "Configuration") -> "SQSClient":  # type: ignore
        """
        Creates an SQSClient instance from the configuration.
        """
        return SQSClient(
            endpoint_url=config.aws_endpoint_url,
            region=config.aws_region,
            access_key=config.aws_access_key_id,
            secret_key=config.aws_secret_access_key,
            queue_url=config.sqs_queue_url,
        )

    async def send_message(
        self, message_body: dict, queue_url: str | None = None
    ) -> bool:
        try:
            target_queue = queue_url or self.__queue_url
            async with self.__session.client(
                "sqs",
                endpoint_url=self.__endpoint_url,
                region_name=self.__region,
                aws_access_key_id=self.__access_key,
                aws_secret_access_key=self.__secret_key,
            ) as client:
                await client.send_message(
                    QueueUrl=target_queue, MessageBody=json.dumps(message_body)
                )
                return True
        except Exception as e:
            logger.error("Failed to send SQS message: %s", e)
            raise e

    async def receive_messages(
        self, queue_url: str, max_messages: int = 1, wait_time: int = 20
    ) -> list[dict[str, Any]]:
        """
        Receives messages from the SQS queue.
        """
        try:
            async with self.__session.client(
                "sqs",
                endpoint_url=self.__endpoint_url,
                region_name=self.__region,
                aws_access_key_id=self.__access_key,
                aws_secret_access_key=self.__secret_key,
            ) as client:
                response = await client.receive_message(
                    QueueUrl=queue_url,
                    MaxNumberOfMessages=max_messages,
                    WaitTimeSeconds=wait_time,
                )
                messages: list[dict[str, Any]] = response.get("Messages", [])
                return messages
        except Exception as e:
            logger.error("Failed to receive SQS messages: %s", e)
            return []

    async def delete_message(self, queue_url: str, receipt_handle: str) -> bool:
        """
        Deletes a message from the SQS queue.
        """
        try:
            async with self.__session.client(
                "sqs",
                endpoint_url=self.__endpoint_url,
                region_name=self.__region,
                aws_access_key_id=self.__access_key,
                aws_secret_access_key=self.__secret_key,
            ) as client:
                await client.delete_message(
                    QueueUrl=queue_url, ReceiptHandle=receipt_handle
                )
                return True
        except Exception as e:
            logger.error("Failed to delete SQS message: %s", e)
            return False
