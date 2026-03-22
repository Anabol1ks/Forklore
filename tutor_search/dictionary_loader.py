import os
import logging
import httpx
from typing import List, Optional
from models import UniversityItem, SubjectItem, TopicItem

logger = logging.getLogger(__name__)

class DictionaryLoader:
    def __init__(self, go_api_url: Optional[str] = None):
        self.go_api_url = go_api_url or os.getenv("GO_API_URL")
        self.universities: List[UniversityItem] = []
        self.subjects: List[SubjectItem] = []
        self.topics: List[TopicItem] = []

    async def load(self):
        try:
            async with httpx.AsyncClient() as client:
                # Загружаем университеты
                resp = await client.get(f"{self.go_api_url}/api/v1/dictionaries/universities", timeout=5.0)
                if resp.status_code == 200:
                    data = resp.json()
                    self.universities = [UniversityItem(**u) for u in data.get("universities", [])]
                else:
                    raise Exception(f"HTTP {resp.status_code}")

                # Предметы
                resp = await client.get(f"{self.go_api_url}/api/v1/dictionaries/subjects", timeout=5.0)
                if resp.status_code == 200:
                    data = resp.json()
                    self.subjects = [SubjectItem(**s) for s in data.get("subjects", [])]

                # Темы
                resp = await client.get(f"{self.go_api_url}/api/v1/dictionaries/topics", timeout=5.0)
                if resp.status_code == 200:
                    data = resp.json()
                    self.topics = [TopicItem(**t) for t in data.get("topics", [])]

            logger.info(f"Loaded dictionaries: {len(self.universities)} unis, {len(self.subjects)} subjects, {len(self.topics)} topics")
        except Exception as e:
            logger.warning(f"Could not load dictionaries from Go: {e}, using static fallback")


    def get_all_university_names(self) -> List[str]:
        names = []
        for uni in self.universities:
            names.append(uni.name)
            names.extend(uni.synonyms)
        return list(set(names))

    def get_all_subject_names(self) -> List[str]:
        return [s.name for s in self.subjects] + [syn for s in self.subjects for syn in s.synonyms]

    def get_all_topic_names(self) -> List[str]:
        return [t.name for t in self.topics]