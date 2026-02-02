import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from shared.clients.s3_client import S3Client


class TestS3Client(unittest.IsolatedAsyncioTestCase):
    async def asyncSetUp(self) -> None:
        self.endpoint_url = "http://localhost:4566"
        self.region = "us-east-1"
        self.access_key = "test"
        self.secret_key = "test"
        self.bucket = "test-bucket"

    @patch("shared.clients.s3_client.aioboto3.Session")
    async def test_upload_bytes_success(self, mock_session_cls: MagicMock) -> None:
        # Mock Session, Client Context Manager, and S3 Client
        mock_s3_client = AsyncMock()
        mock_client_cm = MagicMock()
        mock_client_cm.__aenter__.return_value = mock_s3_client
        mock_client_cm.__aexit__.return_value = None

        mock_session = MagicMock()
        mock_session.client.return_value = mock_client_cm
        mock_session_cls.return_value = mock_session

        client = S3Client(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
        )

        data = b"some data"
        key = "test.txt"

        result = await client.upload_bytes(data, self.bucket, key)

        self.assertEqual(result, f"s3://{self.bucket}/{key}")
        mock_s3_client.put_object.assert_called_once_with(
            Body=data,
            Bucket=self.bucket,
            Key=key,
            ContentType="application/octet-stream",
        )

    @patch("shared.clients.s3_client.aioboto3.Session")
    async def test_upload_bytes_failure(self, mock_session_cls: MagicMock) -> None:
        # Mock Exception
        mock_client_cm = MagicMock()
        mock_client_cm.__aenter__.side_effect = Exception("S3 Error")

        mock_session = MagicMock()
        mock_session.client.return_value = mock_client_cm
        mock_session_cls.return_value = mock_session

        client = S3Client(
            self.endpoint_url,
            self.region,
            self.access_key,
            self.secret_key,
        )

        with self.assertRaisesRegex(Exception, "S3 Error"):
            await client.upload_bytes(b"data", self.bucket, "key")
