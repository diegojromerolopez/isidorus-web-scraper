import json
import logging

import aioboto3  # type: ignore

logger = logging.getLogger(__name__)


class SQSClient:
    def __init__(
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

    async def send_message(self, message_body: dict) -> bool:
        try:
            async with self.__session.client(
                "sqs",
                endpoint_url=self.__endpoint_url,
                region_name=self.__region,
                aws_access_key_id=self.__access_key,
                aws_secret_access_key=self.__secret_key,
            ) as client:
                await client.send_message(
                    QueueUrl=self.__queue_url, MessageBody=json.dumps(message_body)
                )
                return True
        except Exception as e:
            logger.error(f"Failed to send SQS message: {e}")
            raise e
