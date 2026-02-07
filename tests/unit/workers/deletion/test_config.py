import os
import unittest
from unittest.mock import patch

from workers.deletion.config import Configuration


class TestDeletionConfig(unittest.TestCase):
    @patch.dict(
        os.environ,
        {
            "AWS_ENDPOINT_URL": "http://test-aws",
            "AWS_REGION": "us-west-2",
            "AWS_ACCESS_KEY_ID": "key",
            "AWS_SECRET_ACCESS_KEY": "secret",
            "SQS_QUEUE_URL": "http://sqs",
            "DYNAMODB_TABLE": "table",
            "DATABASE_URL": "postgres://db",
            "REDIS_HOST": "redis-host",
            "REDIS_PORT": "6380",
            "INPUT_QUEUE_URL": "http://input",
            "IMAGES_BUCKET": "bucket",
            "OPENSEARCH_URL": "http://os",
        },
    )
    def test_from_env(self) -> None:
        config = Configuration.from_env()
        self.assertEqual(config.aws_endpoint_url, "http://test-aws")
        self.assertEqual(config.aws_region, "us-west-2")
        self.assertEqual(config.aws_access_key_id, "key")
        self.assertEqual(config.aws_secret_access_key, "secret")
        self.assertEqual(config.sqs_queue_url, "http://sqs")
        self.assertEqual(config.dynamodb_table, "table")
        self.assertEqual(config.database_url, "postgres://db")
        self.assertEqual(config.redis_host, "redis-host")
        self.assertEqual(config.redis_port, 6380)
        self.assertEqual(config.input_queue_url, "http://input")
        self.assertEqual(config.images_bucket, "bucket")
        self.assertEqual(config.opensearch_url, "http://os")
