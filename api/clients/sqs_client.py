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
        self.endpoint_url = endpoint_url
        self.region = region
        self.access_key = access_key
        self.secret_key = secret_key
        self.queue_url = queue_url
        self.session = aioboto3.Session()

    async def send_message(self, message_body: dict) -> bool:
        try:
            async with self.session.client(
                "sqs",
                endpoint_url=self.endpoint_url,
                region_name=self.region,
                aws_access_key_id=self.access_key,
                aws_secret_access_key=self.secret_key,
            ) as client:
                await client.send_message(
                    QueueUrl=self.queue_url, MessageBody=json.dumps(message_body)
                )
                return True
        except Exception as e:
            logger.error(f"Failed to send SQS message: {e}")
            raise e
