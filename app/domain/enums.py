from enum import Enum


class GenerationMode(str, Enum):
    FLASHCARDS = "flashcards"
    RANDOM_QUESTIONS = "random_questions"


class KnowledgeType(str, Enum):
    TERM = "term"
    DEFINITION = "definition"
    CONCEPT = "concept"
    FACT = "fact"
    RELATION = "relation"
    EXAMPLE = "example"
    LIST_ITEM = "list_item"
    PROCESS_STEP = "process_step"
