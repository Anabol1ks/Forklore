from typing import List, Optional

from pydantic import BaseModel, Field

from app.domain.enums import GenerationMode, KnowledgeType


class KnowledgeUnit(BaseModel):
    id: str
    type: KnowledgeType
    topic: str
    content: str
    source_fragment: str
    importance: float = Field(ge=0.0, le=1.0)
    confidence: float = Field(ge=0.0, le=1.0)


class Flashcard(BaseModel):
    id: str
    front: str
    back: str
    topic: str
    source_fragment: str


class Question(BaseModel):
    id: str
    question: str
    expected_answer: str
    topic: str
    source_fragment: str


class GenerationContext(BaseModel):
    mode: GenerationMode
    note_text: str = ""
    target_count: int
    knowledge_units: List[KnowledgeUnit]
    generation_rules: List[str] = Field(default_factory=list)


class ReviewReport(BaseModel):
    passed: bool
    issues: List[str] = Field(default_factory=list)
    duplicates_found: List[str] = Field(default_factory=list)
    notes: List[str] = Field(default_factory=list)


class PipelineMeta(BaseModel):
    count: int
    mode: GenerationMode
    request_id: Optional[str] = None
