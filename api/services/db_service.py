from api.repositories.db_repository import (
    DbRepository,
    ScrapingRecord,
    TermOccurrence,
)


class DbService:
    def __init__(self, db_repo: DbRepository) -> None:
        self.__db_repo = db_repo

    async def search_websites(self, term: str) -> list[str]:
        return await self.__db_repo.find_websites_by_term(term)

    async def get_website_terms(self, website_url: str) -> list[TermOccurrence]:
        return await self.__db_repo.find_terms_by_website(website_url)

    async def get_scrapings(
        self, user_id: int, offset: int, limit: int
    ) -> tuple[list[ScrapingRecord], int]:
        return await self.__db_repo.get_scrapings(user_id, offset, limit)
