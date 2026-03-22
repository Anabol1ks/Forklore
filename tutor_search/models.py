from pydantic import BaseModel
from uuid import UUID
from datetime import datetime
from typing import Optional, List

class TutorIndexItem(BaseModel):
    tutor_id: UUID
    full_name: str
    bio: Optional[str] = None
    university: Optional[str] = None
    course: Optional[str] = None          # курс (1, 2, ...)
    subjects: List[str] = []              # предметы
    topics: List[str] = []                # темы
    is_active: bool = True
    updated_at: datetime

class SearchHit(BaseModel):
    tutor_id: UUID
    full_name: str
    university: Optional[str] = None
    course: Optional[str] = None
    subjects: List[str] = []
    topics: List[str] = []
    snippet: Optional[str] = None
    rank: float

class SearchRequest(BaseModel):
    raw_query: str
    limit: int = 20
    offset: int = 0

class SearchResponse(BaseModel):
    total: int
    results: List[SearchHit]

class SearchParamsData(BaseModel):
    query: str           
    university: Optional[str] = None
    course: Optional[str] = None
    subjects: Optional[List[str]] = None
    topics: Optional[List[str]] = None
    limit: int = 20
    offset: int = 0

class UniversityItem(BaseModel):
    id: str
    name: str
    synonyms: List[str] = []

class SubjectItem(BaseModel):
    id: str
    name: str
    synonyms: List[str] = []

class TopicItem(BaseModel):
    id: str
    name: str
    subject_id: Optional[str] = None   # связь с предметом

class DictionariesResponse(BaseModel):
    universities: List[UniversityItem]
    subjects: List[SubjectItem]
    topics: List[TopicItem]