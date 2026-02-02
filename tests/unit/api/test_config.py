import unittest
from unittest.mock import patch

from api.config import Configuration


class TestConfiguration(unittest.TestCase):
    def test_from_env_defaults(self) -> None:
        """
        Test that defaults are used when env vars are missing.
        """
        # Clear specific env vars to force defaults
        with patch.dict("os.environ", {}, clear=True):
            config = Configuration.from_env()

        self.assertEqual(config.aws_endpoint_url, "http://localstack:4566")
        self.assertEqual(config.aws_region, "us-east-1")
        # Validate other defaults...

    def test_from_env_custom(self) -> None:
        """
        Test that env vars override defaults.
        """
        env_vars = {
            "AWS_ENDPOINT_URL": "http://production",
            "AWS_REGION": "eu-west-1",
            "AWS_ACCESS_KEY_ID": "prod-key",
            "AWS_SECRET_ACCESS_KEY": "prod-secret",
            "SQS_QUEUE_URL": "http://sqs-prod",
            "DYNAMODB_TABLE": "prod-table",
            "DATABASE_URL": "postgres://prod",
            "REDIS_HOST": "redis-prod",
            "REDIS_PORT": "1234",
        }
        with patch.dict("os.environ", env_vars):
            config = Configuration.from_env()

        self.assertEqual(config.aws_endpoint_url, "http://production")
        self.assertEqual(config.aws_region, "eu-west-1")
        self.assertEqual(config.redis_port, 1234)
