from abc import ABC, abstractmethod
from typing import Any

class Repository(ABC):
    @abstractmethod
    def save(self, data: Any) -> None:
        ...

    @abstractmethod
    def list_all(self) -> list[Any]:
        ...
