from typing import Any

from api.repositories.db_repository import DbRepository


class DbService:
    def __init__(self, data_repo: DbRepository):
        self.data_repo = data_repo

    async def search_websites(self, term: str) -> list[str]:
        return await self.data_repo.find_websites_by_term(term)

    async def get_website_terms(self, website_url: str) -> list[dict[str, Any]]:
        return await self.data_repo.find_terms_by_website(website_url)
