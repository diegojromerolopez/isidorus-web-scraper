from typing import Any, cast

import redis  # type: ignore


class RedisClient:
    def __init__(self, host: str, port: int, db: int = 0):
        self.client = redis.Redis(host=host, port=port, db=db)

    def set(self, key: str, value: Any) -> None:
        self.client.set(key, value)

    def get(self, key: str) -> str | None:
        value = self.client.get(key)
        if value is None:
            return None
        return value.decode("utf-8")

    def incr(self, key: str, amount: int = 1) -> int:
        return cast(int, self.client.incrby(key, amount))

    def decr(self, key: str, amount: int = 1) -> int:
        return cast(int, self.client.decrby(key, amount))
