from api.repositories.db_repository import (
    DbRepository,
    ScrapingRecord,
)


class DbService:  # pylint: disable=too-few-public-methods
    def __init__(self, db_repo: DbRepository) -> None:
        self.__db_repo = db_repo

    async def get_scrapings(
        self, user_id: int, offset: int, limit: int
    ) -> tuple[list[ScrapingRecord], int]:
        return await self.__db_repo.get_scrapings(user_id, offset, limit)
