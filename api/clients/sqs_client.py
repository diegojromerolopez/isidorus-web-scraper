import json
import logging

import boto3

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
        self.client = boto3.client(
            "sqs",
            endpoint_url=endpoint_url,
            region_name=region,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )
        self.queue_url = queue_url

    def send_message(self, message_body: dict) -> bool:
        try:
            self.client.send_message(
                QueueUrl=self.queue_url, MessageBody=json.dumps(message_body)
            )
            return True
        except Exception as e:
            logger.error(f"Failed to send SQS message: {e}")
            raise e
