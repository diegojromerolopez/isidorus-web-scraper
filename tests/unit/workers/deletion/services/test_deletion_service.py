import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from workers.deletion.services.deletion_service import DeletionService


class TestDeletionService(
    unittest.IsolatedAsyncioTestCase
):  # pylint: disable=protected-access
    def setUp(self) -> None:
        self.mock_dynamodb = AsyncMock()
        self.mock_s3 = AsyncMock()
        self.service = DeletionService(
            dynamodb_client=self.mock_dynamodb,
            s3_client=self.mock_s3,
            images_bucket="test-bucket",
            batch_size=2,
            s3_batch_size=2,
        )

    @patch("api.models.Scraping.get_or_none", new_callable=AsyncMock)
    @patch("api.models.PageImage.filter")
    @patch("api.models.PageTerm.filter")
    @patch("api.models.PageLink.filter")
    @patch("api.models.ScrapedPage.filter")
    async def test_cleanup_scraping_success(
        self,
        mock_page_filter: MagicMock,
        mock_link_filter: MagicMock,
        mock_term_filter: MagicMock,
        mock_image_filter: MagicMock,
        mock_scraping_get: AsyncMock,
    ) -> None:
        # 1. Mock S3 cleanup
        # pylint: disable=too-many-arguments,too-many-positional-arguments
        mock_image_qs = MagicMock()
        mock_image_filter.return_value = mock_image_qs
        # Two pages in first batch, one has s3_path, one None
        # One page in second batch
        mock_values_list = AsyncMock()
        mock_values_list.side_effect = [
            ["s3://test-bucket/k1", None],
            ["s3://test-bucket/k2"],
            [],
        ]
        mock_image_qs.offset.return_value.limit.return_value.values_list = (
            mock_values_list
        )

        # 2. Mock relational deletions
        mock_term_values = AsyncMock(side_effect=[[1], []])
        mock_term_filter.return_value.limit.return_value.values_list = mock_term_values

        mock_link_values = AsyncMock(side_effect=[[2], []])
        mock_link_filter.return_value.limit.return_value.values_list = mock_link_values

        # mock_image_filter is reused for relational deletion loop
        # We need to extend the side effect of the existing mock_values_list
        # But wait, mock_values_list is for offset().limit().values_list() (S3)
        # relational delete uses limit().values_list() (no offset)

        mock_image_rel_values = AsyncMock(side_effect=[[3], []])
        mock_image_qs.limit.return_value.values_list = mock_image_rel_values

        mock_page_values = AsyncMock(side_effect=[[4], []])
        mock_page_filter.return_value.limit.return_value.values_list = mock_page_values

        mock_scraping = AsyncMock()
        mock_scraping_get.return_value = mock_scraping

        # Ensure delete() is awaitable
        mock_term_filter.return_value.delete = AsyncMock()
        mock_link_filter.return_value.delete = AsyncMock()
        mock_image_qs.delete = AsyncMock()
        mock_page_filter.return_value.delete = AsyncMock()

        # Run cleanup
        await self.service.cleanup_scraping(123)

        # Verify S3 deletions
        self.assertEqual(self.mock_s3.delete_objects.call_count, 2)

        # Verify DynamoDB deletion
        self.mock_dynamodb.delete_item.assert_called_once_with({"scraping_id": "123"})

        # Verify Scraping record deletion
        mock_scraping.delete.assert_called_once()

    @patch("api.models.Scraping.get_or_none", new_callable=AsyncMock)
    async def test_cleanup_scraping_not_found(
        self, mock_scraping_get: AsyncMock
    ) -> None:
        mock_scraping_get.return_value = None
        # Should still try to cleanup other things
        with patch.object(
            self.service, "_DeletionService__cleanup_s3_objects", new_callable=AsyncMock
        ) as mock_s3_cleanup:  # type: ignore[attr-defined]
            with patch.object(
                self.service,
                "_DeletionService__cleanup_relational_data",
                new_callable=AsyncMock,
            ) as mock_db_cleanup:  # type: ignore[attr-defined]
                await self.service.cleanup_scraping(123)
                mock_s3_cleanup.assert_called_once()
                mock_db_cleanup.assert_called_once()

        self.mock_dynamodb.delete_item.assert_called_once()

    async def test_cleanup_scraping_error(self) -> None:
        self.mock_dynamodb.delete_item.side_effect = Exception("DB Error")
        with self.assertRaises(Exception):  # noqa: B017
            await self.service.cleanup_scraping(123)

    @patch("api.models.PageImage.filter")
    async def test_cleanup_s3_objects_varied_paths(
        self, mock_filter: MagicMock
    ) -> None:
        mock_qs = MagicMock()
        mock_filter.return_value = mock_qs

        mock_values = AsyncMock()
        mock_values.side_effect = [
            ["s3://bucket/key", "invalid-path", "s3://key-only", None],
            [],
        ]
        mock_qs.offset.return_value.limit.return_value.values_list = mock_values

        await self.service._DeletionService__cleanup_s3_objects(123)  # type: ignore[attr-defined]  # noqa: E501  # pylint: disable=line-too-long
        self.mock_s3.delete_objects.assert_called_once_with(
            "test-bucket", ["key", "key-only"]
        )

    @patch("api.models.PageImage.filter")
    async def test_cleanup_s3_objects_empty(self, mock_filter: MagicMock) -> None:
        mock_qs = MagicMock()
        mock_filter.return_value = mock_qs
        mock_qs.offset.return_value.limit.return_value.values_list = AsyncMock(
            return_value=[]
        )

        await self.service._DeletionService__cleanup_s3_objects(123)  # type: ignore[attr-defined]  # noqa: E501  # pylint: disable=line-too-long
        self.mock_s3.delete_objects.assert_not_called()

    @patch("api.models.PageTerm.filter")
    async def test_batch_delete_multiple_batches(self, mock_filter: MagicMock) -> None:
        mock_qs = MagicMock()
        mock_filter.return_value = mock_qs
        # Two batches, second batch smaller than batch_size (which is 2)
        mock_values = AsyncMock(side_effect=[[1, 2], [3], []])
        mock_qs.limit.return_value.values_list = mock_values

        # Ensure delete() is awaitable
        mock_qs.delete = AsyncMock()

        from api import models as api_models  # pylint: disable=import-outside-toplevel

        await self.service._DeletionService__batch_delete(api_models.PageTerm, 123)  # type: ignore[attr-defined]  # noqa: E501  # pylint: disable=line-too-long

        # Verify filter called with ids
        # filter called:
        # 1. filter(scraping_id=...).limit()...values_list() -> 3 times
        # (2 batches + 1 empty)
        # 2. filter(id__in=...).delete() -> 2 times
        # Total filter calls = 5?
        # check call_count of mock_filter
        self.assertEqual(mock_filter.call_count, 4)
