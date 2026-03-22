import json
from fastapi import APIRouter, HTTPException, Query, Request
from app.core import exceptions
from app.domain.enums import GenerationMode
from app.graph.main_graph import GenerationGraph, GraphDependencies
from pathlib import Path
from app.services.extraction_service import ExtractionService
from app.services.llm_client import LLMClient
from app.services.review_service import ReviewService
from app.storage.in_memory_repository import InMemoryRepository
from app.tools.flashcard_generator import FlashcardGeneratorTool, RandomQuestionGeneratorTool

router = APIRouter()

# Initialize dependencies once per process
prompt_dir = Path(__file__).resolve().parents[2] / "prompts"
llm_client = LLMClient()
extraction_service = ExtractionService(llm_client, str(prompt_dir / "extract_knowledge.txt"))
review_service = ReviewService()
repository = InMemoryRepository()
flashcard_tool = FlashcardGeneratorTool(llm_client, str(prompt_dir / "generate_flashcards.txt"))
question_tool = RandomQuestionGeneratorTool()

deps = GraphDependencies(
    extraction_service=extraction_service,
    review_service=review_service,
    repository=repository,
    flashcard_tool=flashcard_tool,
    question_tool=question_tool,
)

graph = GenerationGraph(deps)


@router.post(
    "/generate-text",
    openapi_extra={
        "requestBody": {
            "required": True,
            "content": {
                "text/plain": {
                    "schema": {
                        "type": "string",
                        "description": "Raw note text (can be multi-line).",
                    }
                }
            },
        }
    },
)
async def generate_text(
    request: Request,
    mode: GenerationMode = Query(..., description="flashcards | random_questions"),
    count: int = Query(10, ge=1, le=50),
):
    """
    Same generation pipeline, but accepts the note as raw text/plain body.

    This is convenient for Swagger UI: you can paste a multi-line note without JSON escaping.
    """
    try:
        raw = await request.body()
        decoded = raw.decode("utf-8", errors="replace").strip("\ufeff")
        # If a client accidentally sends a JSON string (e.g. "\"...\""), accept it too.
        if decoded.startswith('"') and decoded.endswith('"'):
            try:
                decoded = json.loads(decoded)
            except Exception:
                pass

        payload = {
            "note_text": decoded,
            "mode": mode,
            "count": count,
        }
        return graph.run(payload)
    except exceptions.PipelineError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:  # pragma: no cover
        raise HTTPException(status_code=500, detail="Internal server error") from e
