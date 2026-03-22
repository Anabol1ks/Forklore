from typing import Any
from .repository import Repository

class InMemoryRepository(Repository):
    def __init__(self) -> None:
        self._storage: list[Any] = []

    def save(self, data: Any) -> None:
        self._storage.append(data)

    def list_all(self) -> list[Any]:
        return list(self._storage)
