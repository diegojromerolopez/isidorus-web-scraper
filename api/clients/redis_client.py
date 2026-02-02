from typing import Any, cast

import redis.asyncio as redis  # type: ignore

from api.config import Configuration


class RedisClient:
    def __init__(self, host: str, port: int, db: int = 0):
        self.__client = redis.Redis(host=host, port=port, db=db)

    @staticmethod
    def create(config: Configuration) -> "RedisClient":
        """
        Creates a RedisClient instance from the configuration.
        """
        return RedisClient(host=config.redis_host, port=config.redis_port)

    async def set(self, key: str, value: Any) -> None:
        await self.__client.set(key, value)

    async def get(self, key: str) -> str | None:
        value = await self.__client.get(key)
        if value is None:
            return None
        return value.decode("utf-8")

    async def incr(self, key: str, amount: int = 1) -> int:
        return cast(int, await self.__client.incrby(key, amount))

    async def decr(self, key: str, amount: int = 1) -> int:
        return cast(int, await self.__client.decrby(key, amount))
