from typing import Any, cast

from opensearchpy import AsyncOpenSearch  # pylint: disable=import-error

from api.config import Configuration


class SearchRepository:
    def __init__(self, config: Configuration):
        self.__client = AsyncOpenSearch(
            hosts=[config.opensearch_url],
            http_compress=True,
            use_ssl=False,
            verify_certs=False,
            ssl_assert_hostname=False,
            ssl_show_warn=False,
        )

    async def search(self, index: str, body: dict[str, Any]) -> dict[str, Any]:
        """
        Executes a search query against OpenSearch.
        """
        try:
            return cast(
                dict[str, Any],
                await self.__client.search(index=index, body=body),
            )
        finally:
            # Note: We might want to keep the client open if this is a singleton
            # but for now we follow the existing pattern of closing it or
            # managing its lifecycle. In FastAPI, we usually close on shutdown.
            pass

    async def close(self) -> None:
        """
        Closes the OpenSearch client.
        """
        await self.__client.close()
