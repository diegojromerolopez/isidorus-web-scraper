import unittest
from typing import Any
from unittest.mock import AsyncMock, MagicMock, patch

from api.config import Configuration
from api.repositories.search_repository import SearchRepository


class TestSearchRepository(unittest.IsolatedAsyncioTestCase):
    @patch("api.repositories.search_repository.AsyncOpenSearch")
    async def test_search(self, mock_os_class: MagicMock) -> None:
        mock_client = AsyncMock()
        mock_os_class.return_value = mock_client

        config = MagicMock(spec=Configuration)
        config.opensearch_url = "http://localhost:9200"

        repo = SearchRepository(config)

        mock_client.search.return_value = {"hits": {"hits": []}}

        body: dict[str, Any] = {"query": {"match_all": {}}}
        result = await repo.search(index="test_index", body=body)

        self.assertEqual(result, {"hits": {"hits": []}})
        mock_client.search.assert_called_once_with(index="test_index", body=body)

    @patch("api.repositories.search_repository.AsyncOpenSearch")
    async def test_close(self, mock_os_class: MagicMock) -> None:
        mock_client = AsyncMock()
        mock_os_class.return_value = mock_client

        config = MagicMock(spec=Configuration)
        config.opensearch_url = "http://localhost:9200"
        repo = SearchRepository(config)

        await repo.close()
        mock_client.close.assert_called_once()
