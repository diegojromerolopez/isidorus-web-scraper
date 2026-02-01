import json
from typing import Any, cast

import boto3


class SQSClient:
    def __init__(
        self,
        endpoint_url: str | None,
        region: str,
        access_key: str | None,
        secret_key: str | None,
    ) -> None:
        self.client = boto3.client(
            "sqs",
            endpoint_url=endpoint_url,
            region_name=region,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )

    def receive_messages(
        self, queue_url: str, max_messages: int = 1, waitTime: int = 20
    ) -> list[dict[str, Any]]:
        response = self.client.receive_message(
            QueueUrl=queue_url,
            MaxNumberOfMessages=max_messages,
            WaitTimeSeconds=waitTime,
        )
        return cast(list[dict[str, Any]], response.get("Messages", []))

    def delete_message(self, queue_url: str, receipt_handle: str) -> None:
        self.client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)

    def send_message(self, queue_url: str, message_body: dict[str, Any]) -> None:
        self.client.send_message(
            QueueUrl=queue_url, MessageBody=json.dumps(message_body)
        )
