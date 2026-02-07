import os
import unittest
from unittest.mock import patch

from workers.image_explainer.config import Configuration


class TestConfiguration(unittest.TestCase):
    @patch.dict(
        os.environ,
        {
            "AWS_ENDPOINT_URL": "http://custom-aws:4566",
            "INPUT_QUEUE_URL": "http://queue/input",
            "WRITER_QUEUE_URL": "http://queue/writer",
            "LLM_PROVIDER": "gemini",
        },
        clear=True,
    )
    def test_from_env_custom(self) -> None:
        config = Configuration.from_env()
        self.assertEqual(config.aws_endpoint_url, "http://custom-aws:4566")
        self.assertEqual(config.input_queue_url, "http://queue/input")
        self.assertEqual(config.writer_queue_url, "http://queue/writer")
        self.assertEqual(config.llm_provider, "gemini")

    @patch.dict(os.environ, {}, clear=True)
    def test_from_env_defaults(self) -> None:
        # We need to at least ensure it doesn't crash and uses defaults
        config = Configuration.from_env()
        self.assertEqual(config.aws_endpoint_url, "http://localstack:4566")
        self.assertEqual(config.llm_provider, "openai")
        self.assertEqual(config.images_bucket, "isidorus-images")


if __name__ == "__main__":
    unittest.main()
