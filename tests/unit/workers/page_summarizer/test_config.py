import os
import unittest

from workers.page_summarizer.config import Configuration


class TestConfiguration(unittest.TestCase):
    def test_from_env_defaults(self) -> None:
        # Clear env vars that might affect the test
        vars_to_clear = [
            "AWS_ENDPOINT_URL",
            "AWS_REGION",
            "AWS_ACCESS_KEY_ID",
            "AWS_SECRET_ACCESS_KEY",
            "SQS_QUEUE_URL",
            "DYNAMODB_TABLE",
            "DATABASE_URL",
            "REDIS_HOST",
            "REDIS_PORT",
            "INPUT_QUEUE_URL",
            "WRITER_QUEUE_URL",
            "LLM_PROVIDER",
            "LLM_API_KEY",
        ]
        old_values = {v: os.environ.get(v) for v in vars_to_clear}
        for v in vars_to_clear:
            if v in os.environ:
                del os.environ[v]

        try:
            config = Configuration.from_env()
            self.assertEqual(config.aws_endpoint_url, "http://localstack:4566")
            self.assertEqual(config.input_queue_url, "")
            self.assertEqual(config.llm_provider, "openai")
        finally:
            # Restore
            for v, val in old_values.items():
                if val is not None:
                    os.environ[v] = val

    def test_from_env_custom(self) -> None:
        os.environ["INPUT_QUEUE_URL"] = "http://custom-input"
        os.environ["REDIS_PORT"] = "1234"
        try:
            config = Configuration.from_env()
            self.assertEqual(config.input_queue_url, "http://custom-input")
            self.assertEqual(config.redis_port, 1234)
        finally:
            del os.environ["INPUT_QUEUE_URL"]
            del os.environ["REDIS_PORT"]
