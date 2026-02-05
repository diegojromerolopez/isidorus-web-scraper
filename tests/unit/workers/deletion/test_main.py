import unittest
import json
import asyncio
from unittest.mock import AsyncMock, MagicMock, patch
from workers.deletion.main import main

class TestDeletionMain(unittest.IsolatedAsyncioTestCase):
    @patch("workers.deletion.main.Configuration")
    @patch("workers.deletion.main.SQSClient")
    @patch("workers.deletion.main.DynamoDBClient")
    @patch("workers.deletion.main.S3Client")
    @patch("workers.deletion.main.DeletionService")
    @patch("workers.deletion.main.Tortoise")
    async def test_main_loop(self, mock_tortoise, mock_service_cls, mock_s3_cls, mock_dynamo_cls, mock_sqs_cls, mock_config_cls):
        # Mock setup
        mock_config = MagicMock()
        mock_config.input_queue_url = "http://test-queue"
        mock_config.database_url = "sqlite://:memory:"
        mock_config.aws_endpoint_url = "http://test"
        mock_config.aws_region = "us-east-1"
        mock_config.aws_access_key_id = "test"
        mock_config.aws_secret_access_key = "test"
        mock_config.dynamodb_table = "test-table"
        mock_config.images_bucket = "test-bucket"
        mock_config_cls.from_env.return_value = mock_config
        
        mock_tortoise.init = AsyncMock()
        mock_tortoise.close_connections = AsyncMock()

        mock_stop_event = MagicMock()
        mock_stop_event.is_set.side_effect = [False, True] # Run once then stop
        
        mock_sqs = AsyncMock()
        mock_sqs_cls.return_value = mock_sqs
        mock_sqs.receive_messages.return_value = [
            {"Body": json.dumps({"scraping_id": 123}), "ReceiptHandle": "abc"}
        ]
        
        mock_service = AsyncMock()
        mock_service_cls.return_value = mock_service
        
        # Run main
        await main(stop_event=mock_stop_event)

        # Verify
        mock_service.cleanup_scraping.assert_called_with(123)
        mock_sqs.delete_message.assert_called_with("http://test-queue", "abc")
        mock_tortoise.init.assert_called()
        mock_tortoise.close_connections.assert_called()

    @patch("workers.deletion.main.Configuration")
    @patch("workers.deletion.main.logger")
    async def test_main_no_queue_url(self, mock_logger, mock_config_cls):
        mock_config = MagicMock()
        mock_config.input_queue_url = ""
        mock_config_cls.from_env.return_value = mock_config
        await main()
        mock_logger.error.assert_called_with("INPUT_QUEUE_URL must be set")

    @patch("workers.deletion.main.Configuration")
    @patch("workers.deletion.main.SQSClient")
    @patch("workers.deletion.main.logger")
    async def test_main_loop_error(self, mock_logger, mock_sqs_cls, mock_config_cls):
        mock_config = MagicMock()
        mock_config.input_queue_url = "http://test-queue"
        mock_config.database_url = "sqlite://:memory:"
        mock_config.aws_endpoint_url = "http://test"
        mock_config.aws_region = "us-east-1"
        mock_config.aws_access_key_id = "test"
        mock_config.aws_secret_access_key = "test"
        mock_config.dynamodb_table = "test-table"
        mock_config.images_bucket = "test-bucket"
        mock_config_cls.from_env.return_value = mock_config
        
        mock_sqs = AsyncMock()
        mock_sqs_cls.return_value = mock_sqs
        mock_sqs.receive_messages.side_effect = Exception("SQS Error")
        
        mock_stop_event = MagicMock()
        mock_stop_event.is_set.side_effect = [False, True]
        
        # Mock sleep to avoid waiting
        with patch("workers.deletion.main.asyncio.sleep", new_callable=AsyncMock):
            await main(stop_event=mock_stop_event)
        
        mock_logger.error.assert_any_call("Error receiving messages: SQS Error")

    @patch("workers.deletion.main.Configuration")
    @patch("workers.deletion.main.SQSClient")
    @patch("workers.deletion.main.logger")
    @patch("workers.deletion.main.DeletionService")
    @patch("workers.deletion.main.DynamoDBClient")
    @patch("workers.deletion.main.S3Client")
    @patch("workers.deletion.main.Tortoise")
    async def test_main_loop_processing_error(self, mock_tortoise, mock_s3_cls, mock_dynamo_cls, mock_service_cls, mock_logger, mock_sqs_cls, mock_config_cls):
        mock_config = MagicMock()
        mock_config.input_queue_url = "http://test-queue"
        mock_config.database_url = "sqlite://:memory:"
        mock_config.aws_endpoint_url = "http://test"
        mock_config.aws_region = "us-east-1"
        mock_config.aws_access_key_id = "test"
        mock_config.aws_secret_access_key = "test"
        mock_config.dynamodb_table = "test-table"
        mock_config.images_bucket = "test-bucket"
        mock_config_cls.from_env.return_value = mock_config
        
        mock_tortoise.init = AsyncMock()
        mock_tortoise.close_connections = AsyncMock()
        
        mock_sqs = AsyncMock()
        mock_sqs_cls.return_value = mock_sqs
        mock_sqs.receive_messages.return_value = [{"Body": "invalid json", "ReceiptHandle": "abc"}]
        
        mock_stop_event = MagicMock()
        mock_stop_event.is_set.side_effect = [False, True]
        
        await main(stop_event=mock_stop_event)
        
        mock_logger.error.assert_any_call("Error processing message: Expecting value: line 1 column 1 (char 0)")
