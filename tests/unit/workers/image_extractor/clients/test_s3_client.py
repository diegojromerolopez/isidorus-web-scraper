import unittest
from unittest.mock import MagicMock, patch

from botocore.exceptions import ClientError

from workers.image_extractor.clients.s3_client import S3Client


class TestS3Client(unittest.TestCase):
    def setUp(self) -> None:
        self.mock_boto_client = MagicMock()
        with patch(
            "workers.image_extractor.clients.s3_client.boto3.client",
            return_value=self.mock_boto_client,
        ):
            self.client = S3Client(
                endpoint_url="http://localhost:4566",
                region_name="us-east-1",
                access_key="test",
                secret_key="test",
            )

    def test_init(self) -> None:
        """Test S3Client initialization"""
        with patch(
            "workers.image_extractor.clients.s3_client.boto3.client"
        ) as mock_boto:
            client = S3Client(
                endpoint_url="http://test:4566",
                region_name="eu-west-1",
                access_key="key123",
                secret_key="secret123",
            )
            mock_boto.assert_called_once_with(
                "s3",
                endpoint_url="http://test:4566",
                region_name="eu-west-1",
                aws_access_key_id="key123",
                aws_secret_access_key="secret123",
            )
            self.assertIsNotNone(client.client)

    def test_upload_bytes_success(self) -> None:
        """Test successful upload of bytes to S3"""
        test_data = b"test image data"
        bucket = "test-bucket"
        key = "test-key.png"

        result = self.client.upload_bytes(test_data, bucket, key)

        self.assertEqual(result, f"s3://{bucket}/{key}")
        self.mock_boto_client.put_object.assert_called_once_with(
            Body=test_data,
            Bucket=bucket,
            Key=key,
            ContentType="application/octet-stream",
        )

    def test_upload_bytes_client_error(self) -> None:
        """Test upload failure with ClientError"""
        test_data = b"test image data"
        bucket = "test-bucket"
        key = "test-key.png"

        error_response = {
            "Error": {"Code": "NoSuchBucket", "Message": "Bucket not found"}
        }
        self.mock_boto_client.put_object.side_effect = ClientError(
            error_response, "PutObject"
        )

        with self.assertRaises(ClientError):
            self.client.upload_bytes(test_data, bucket, key)

        self.mock_boto_client.put_object.assert_called_once_with(
            Body=test_data,
            Bucket=bucket,
            Key=key,
            ContentType="application/octet-stream",
        )


if __name__ == "__main__":
    unittest.main()
