from typing import TypedDict

from api.repositories.search_repository import SearchRepository


class SearchPageResult(TypedDict):
    url: str
    scraping_id: int
    created_at: str | None
    highlights: list[str]


class SearchService:  # pylint: disable=too-few-public-methods
    def __init__(self, repository: SearchRepository):
        self.__repository = repository

    async def search_pages(
        self, query_term: str, user_id: int
    ) -> list[SearchPageResult]:
        """
        Searches for pages containing the query term, scoped by user_id.
        """
        query = {
            "query": {
                "bool": {
                    "must": [
                        {
                            "multi_match": {
                                "query": query_term,
                                "fields": ["content", "summary"],
                            }
                        },
                        {"term": {"user_id": user_id}},
                    ]
                }
            },
            "highlight": {"fields": {"content": {}, "summary": {}}},
        }

        response = await self.__repository.search(index="scraped_pages", body=query)

        results: list[SearchPageResult] = []
        for hit in response["hits"]["hits"]:
            source = hit["_source"]
            highlights = []
            if "highlight" in hit:
                for field in hit["highlight"]:
                    highlights.extend(hit["highlight"][field])

            results.append(
                {
                    "url": source["url"],
                    "scraping_id": source["scraping_id"],
                    "created_at": source.get("created_at", ""),
                    "highlights": highlights[:3],  # Limit to 3 snippets
                }
            )

        return results
