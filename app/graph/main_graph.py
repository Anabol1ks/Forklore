from __future__ import annotations

from typing import Any, Callable, Dict, List, TypedDict

from app.core import exceptions
from app.core.config import settings
from app.core.logging import logger
from app.domain.enums import GenerationMode, KnowledgeType
from app.domain.models.entities import GenerationContext, KnowledgeUnit, PipelineMeta, ReviewReport
from app.services.extraction_service import ExtractionService
from app.services.review_service import ReviewService
from app.storage.repository import Repository
from app.tools.flashcard_generator import FlashcardGeneratorTool, RandomQuestionGeneratorTool


GENERATION_RULES = [
    "One flashcard must test exactly one idea.",
    "Front must be a real question and end with a question mark.",
    "Back must be short, exact, and not copy the source fragment.",
    "Question type must depend on knowledge unit type.",
    "Cards must be self-contained and understandable without neighboring cards.",
    "Drop weak, duplicated, or context-dependent cards.",
]

class PipelineState(TypedDict, total=False):
    note_text: str
    count: int

    raw_note: str
    normalized_note: str
    mode: GenerationMode
    meta: PipelineMeta
    knowledge_units: List[KnowledgeUnit]
    generation_context: GenerationContext
    tool: Any
    generated_items: List[Any]
    review_report: ReviewReport
    review_notes: List[str]
    response: Dict[str, Any]


class GraphDependencies:
    def __init__(
        self,
        extraction_service: ExtractionService,
        review_service: ReviewService,
        repository: Repository,
        flashcard_tool: FlashcardGeneratorTool,
        question_tool: RandomQuestionGeneratorTool,
    ) -> None:
        self.extraction_service = extraction_service
        self.review_service = review_service
        self.repository = repository
        self.flashcard_tool = flashcard_tool
        self.question_tool = question_tool


def normalize_note_text(note: str) -> str:
    lines = []
    previous_blank = False
    for raw_line in str(note).replace("\r\n", "\n").replace("\r", "\n").split("\n"):
        line = " ".join(raw_line.split())
        if not line:
            if not previous_blank:
                lines.append("")
            previous_blank = True
            continue
        lines.append(line)
        previous_blank = False
    return "\n".join(lines).strip()


def _topic_is_weak(topic: str) -> bool:
    cleaned = (topic or "").strip().lower()
    if not cleaned:
        return True
    if len(cleaned) > 60:
        return True
    if len(cleaned.split()) > 7:
        return True
    if "=" in cleaned or "≈" in cleaned:
        return True
    if cleaned.startswith(("если ", "в этом ", "это ", "например ", "что ")):
        return True
    if cleaned in {"формула", "уравнение", "обозначение"}:
        return True
    return False


def _unit_is_usable(unit: KnowledgeUnit) -> bool:
    if not unit.content or not unit.source_fragment or not unit.topic:
        return False
    if len(unit.content) < 25:
        return False
    if unit.content[0].islower():
        return False
    if _topic_is_weak(unit.topic):
        return False
    if unit.confidence < 0.45 or unit.importance < 0.4:
        return False
    return True


def _sort_units(units: List[KnowledgeUnit]) -> List[KnowledgeUnit]:
    return sorted(
        units,
        key=lambda unit: (
            unit.importance + unit.confidence,
            len(unit.topic),
            -len(unit.content),
        ),
        reverse=True,
    )


class GenerationGraph:
    def __init__(self, deps: GraphDependencies) -> None:
        self.deps = deps
        self._steps: List[Callable[[PipelineState], PipelineState]] = [
            self._receive_input,
            self._validate_input,
            self._normalize_note,
            self._extract_knowledge,
            self._prepare_generation_context,
            self._select_tool,
            self._run_tool,
            self._postprocess_items,
            self._review_output,
            self._format_response,
            self._save_result,
        ]

    def run(self, payload: Dict[str, Any]) -> Dict[str, Any]:
        state: PipelineState = dict(payload)
        for step in self._steps:
            state = step(state)
        return state.get("response", {})

    # Pipeline steps -----------------------------------------------------
    def _receive_input(self, state: PipelineState) -> PipelineState:
        logger.info("receive_input")
        note_text = state.get("note_text") or ""
        mode = state.get("mode")
        count = int(state.get("count", 10))
        meta = PipelineMeta(count=count, mode=mode)
        return {
            **state,
            "raw_note": str(note_text),
            "mode": mode,
            "meta": meta,
        }

    def _validate_input(self, state: PipelineState) -> PipelineState:
        logger.info("validate_input")
        raw_note = state.get("raw_note", "").strip()
        mode = state.get("mode")
        meta = state.get("meta")
        count = meta.count if meta else 0
        if not raw_note:
            raise exceptions.EmptyNoteError("Note text is empty")
        if len(raw_note) < settings.min_note_length:
            raise exceptions.NoteTooShortError("Note text is too short")
        if mode not in (GenerationMode.FLASHCARDS, GenerationMode.RANDOM_QUESTIONS):
            raise exceptions.InvalidModeError(f"Unsupported mode: {mode}")
        if not 1 <= count <= 50:
            raise exceptions.ValidationError("count must be between 1 and 50")
        state["raw_note"] = raw_note
        return state

    def _normalize_note(self, state: PipelineState) -> PipelineState:
        logger.info("normalize_note")
        state["normalized_note"] = normalize_note_text(state["raw_note"])
        return state

    def _extract_knowledge(self, state: PipelineState) -> PipelineState:
        logger.info("extract_knowledge")
        note = state.get("normalized_note", state.get("raw_note", ""))
        state["knowledge_units"] = self.deps.extraction_service.extract(note)
        return state

    def _prepare_generation_context(self, state: PipelineState) -> PipelineState:
        logger.info("prepare_generation_context")
        meta = state["meta"]
        raw_units = state.get("knowledge_units") or []
        usable_units = _sort_units([unit for unit in raw_units if _unit_is_usable(unit)])
        state["knowledge_units"] = usable_units
        state["generation_context"] = GenerationContext(
            mode=state["mode"],
            note_text=state.get("normalized_note", state.get("raw_note", "")),
            target_count=meta.count,
            knowledge_units=usable_units,
            generation_rules=GENERATION_RULES,
        )
        return state

    def _select_tool(self, state: PipelineState) -> PipelineState:
        logger.info("select_tool")
        mode = state["mode"]
        if mode == GenerationMode.FLASHCARDS:
            state["tool"] = self.deps.flashcard_tool
        elif mode == GenerationMode.RANDOM_QUESTIONS:
            state["tool"] = self.deps.question_tool
        else:
            raise exceptions.InvalidModeError(f"Unsupported mode: {mode}")
        return state

    def _run_tool(self, state: PipelineState) -> PipelineState:
        logger.info("run_tool")
        tool = state["tool"]
        ctx = state["generation_context"]
        state["generated_items"] = tool.run(ctx)
        return state

    def _postprocess_items(self, state: PipelineState) -> PipelineState:
        logger.info("postprocess_items")
        items = state.get("generated_items") or []
        mode = state["mode"]
        if mode == GenerationMode.FLASHCARDS:
            filtered_items, notes = self.deps.review_service.filter_flashcards(items)  # type: ignore[arg-type]
        else:
            filtered_items, notes = self.deps.review_service.filter_questions(items)  # type: ignore[arg-type]
        state["generated_items"] = filtered_items
        state["review_notes"] = notes
        return state

    def _review_output(self, state: PipelineState) -> PipelineState:
        logger.info("review_output")
        items = state.get("generated_items") or []
        mode = state["mode"]
        if mode == GenerationMode.FLASHCARDS:
            report = self.deps.review_service.review_flashcards(items)  # type: ignore[arg-type]
        else:
            report = self.deps.review_service.review_questions(items)  # type: ignore[arg-type]
        report.notes.extend(state.get("review_notes") or [])
        state["review_report"] = report
        return state

    def _format_response(self, state: PipelineState) -> PipelineState:
        logger.info("format_response")
        meta = state["meta"]
        state["response"] = {
            "mode": state["mode"],
            "items": state.get("generated_items") or [],
            "review": state.get("review_report"),
            "meta": {"count": meta.count},
        }
        return state

    def _save_result(self, state: PipelineState) -> PipelineState:
        logger.info("save_result")
        self.deps.repository.save(state.get("response"))
        return state
