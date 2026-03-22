from __future__ import annotations

import re
from difflib import SequenceMatcher
from typing import List, Tuple

from app.domain.models.entities import Flashcard, Question, ReviewReport


FORBIDDEN_FRONT_PATTERNS = (
    "что важно знать о",
    "расскажи о",
    "что можно сказать о",
)


class ReviewService:
    max_front_chars = 120
    max_back_chars = 200
    max_topic_chars = 60

    def _normalize(self, text: str) -> str:
        return re.sub(r"\s+", " ", (text or "").strip())

    def _similarity(self, left: str, right: str) -> float:
        return SequenceMatcher(None, self._normalize(left).lower(), self._normalize(right).lower()).ratio()

    def _looks_like_formula_topic(self, text: str) -> bool:
        normalized = self._normalize(text).lower()
        if not normalized:
            return False
        if "=" in normalized or "≈" in normalized:
            return True
        if re.search(r"\b[a-zа-я]\w*\s*/\s*[a-zа-я]\w*\b", normalized, flags=re.IGNORECASE):
            return True
        return False

    def _is_clean_source_label(self, topic: str, source_fragment: str) -> bool:
        normalized_topic = self._normalize(topic).lower()
        normalized_source = self._normalize(source_fragment)
        if not normalized_topic or not normalized_source:
            return False
        source_lower = normalized_source.lower()
        if not source_lower.startswith(normalized_topic):
            return False
        tail = normalized_source[len(normalized_topic) :].lstrip()
        return tail.startswith(("-", "—", ":"))

    def _topic_is_weak(self, topic: str, source_fragment: str) -> bool:
        normalized = self._normalize(topic).lower()
        if not normalized:
            return True
        if self._looks_like_formula_topic(normalized):
            return True
        words = normalized.split()
        if len(normalized) > self.max_topic_chars:
            return True
        if len(words) > 7:
            return True
        # "Topic" must not be a question wrapper or a note meta label.
        if normalized.startswith(
            (
                "чем ",
                "что входит",
                "от чего",
                "с чем",
                "на что",
                "какова ",
                "чему ",
                "как вычислить",
                "как записывается",
                "конспект",
                "шпаргалка",
            )
        ):
            return True
        if normalized in {"формула", "уравнение", "обозначение"}:
            return True
        if normalized.startswith(("если ", "в этом ", "это ", "например ", "что ")):
            return True
        if re.search(r"\b\d+[.)]?$", normalized):
            return True
        if len(words) >= 4 and len(set(words)) < len(words):
            return True
        if len(words) >= 3 and words[-1] in {"признак", "определение", "таблица", "схема"}:
            return True
        if normalized.startswith(("сравнение ", "определение ", "таблица ", "схема ")):
            return True
        normalized_source = self._normalize(source_fragment)
        source_start = normalized_source.lower()[: max(len(normalized), 12)]
        if normalized == source_start and len(normalized.split()) >= 4 and not self._is_clean_source_label(normalized, source_fragment):
            tail = normalized_source[len(normalized) :].lstrip(" ,;:-")
            first_tail = tail.split()[0] if tail.split() else ""
            if not first_tail or not first_tail[:1].islower():
                return True
        return False

    def _looks_truncated(self, text: str) -> bool:
        lowered = self._normalize(text).lower()
        if not lowered:
            return True
        if lowered.startswith(("если у нас есть данные о", "в этом подходе", "что важно знать о", "в чем ")):
            return True
        if lowered.endswith((" и", " или", " без", " для", " о", " of", " and")):
            return True
        return False

    def _front_looks_raw(self, front: str) -> bool:
        lowered = self._normalize(front).lower()
        if not lowered:
            return True
        wrapper_patterns = (
            r"^какой\s+факт\s+о\s+.+\s+указан\s+в\s+конспекте\?$",
            r"^какая\s+особенность\s+.+\s+указана\s+в\s+конспекте\?$",
            r"^what fact about '.+' is stated in the notes\?$",
        )
        if any(re.match(pattern, lowered, flags=re.IGNORECASE) for pattern in wrapper_patterns):
            return True
        if re.search(r"что\s+характерно\s+для\s+(?:сравнение|определение|признак)\b", lowered):
            return True
        if re.search(r"какой\s+факт\s+о\s+[«\"]?(?:сравнение|определение|признак)\b", lowered):
            return True
        if re.search(r"[«\"]?(?:сравнение|определение)\s+.+\s+(?:признак|таблица)[»\"]?\?", lowered):
            return True
        if re.search(r"\b(\w+)\s+\1\b", lowered):
            return True
        return False

    def _answer_has_wrapper(self, answer: str) -> bool:
        lowered = self._normalize(answer).lower()
        patterns = (
            r"^(?:верно следующее|факт о [^:]+|особенность в том, что|при сравнении видно, что)\s*[:,-]",
            r"^(?:what is true about|fact about|the key point is that)\b",
        )
        return any(re.match(pattern, lowered, flags=re.IGNORECASE) for pattern in patterns)

    def _is_definition_question(self, text: str) -> bool:
        lowered = self._normalize(text).lower()
        return lowered.startswith(("что такое ", "что означает термин ", "what is "))

    def _definition_question_topic(self, text: str) -> str:
        normalized = self._normalize(text)
        quoted = re.search(r"[«\"]([^»\"]+)[»\"]", normalized)
        if quoted:
            return self._normalize(quoted.group(1))
        match = re.match(r"^(?:что такое|what is)\s+(.+?)\?$", normalized, flags=re.IGNORECASE)
        if match:
            return self._normalize(match.group(1))
        return ""

    def _definition_answer_is_derived(self, front: str, back: str, source: str) -> bool:
        if not self._is_definition_question(front):
            return False
        topic = self._definition_question_topic(front)
        normalized_source = self._normalize(source)
        normalized_back = self._normalize(back)
        if not topic or not normalized_source or not normalized_back:
            return False
        patterns = (
            rf"^[«\"]?{re.escape(topic)}[»\"]?\s*[—-]\s*(?:это\s+)?(.+)$",
            rf"^[«\"]?{re.escape(topic)}[»\"]?\s+это\s+(.+)$",
        )
        for pattern in patterns:
            match = re.match(pattern, normalized_source, flags=re.IGNORECASE)
            if not match:
                continue
            tail = self._normalize(match.group(1))
            variants = {tail, f"Это {tail[0].lower()}{tail[1:]}" if tail else "", f"This is {tail}" if tail else ""}
            if any(variant and self._similarity(normalized_back, variant) >= 0.92 for variant in variants):
                return True
        return False

    def _back_is_too_close_to_source(self, front: str, back: str, source: str, threshold: float = 0.92) -> bool:
        normalized_source = self._normalize(source)
        normalized_back = self._normalize(back)
        if not normalized_source or not normalized_back:
            return False
        if self._similarity(normalized_back, normalized_source) < threshold:
            return False
        if self._definition_answer_is_derived(front, back, source):
            return False
        return True

    def _validate_flashcard(self, card: Flashcard) -> List[str]:
        issues: List[str] = []
        front = self._normalize(card.front)
        back = self._normalize(card.back)
        topic = self._normalize(card.topic)
        source = self._normalize(card.source_fragment)

        if not front:
            issues.append("front is empty")
        if not back:
            issues.append("back is empty")
        if not topic:
            issues.append("topic is empty")
        if front and not front.endswith("?"):
            issues.append("front must end with '?'")
        if front.startswith("Вопрос:") or front.startswith("Question:"):
            issues.append("front contains forbidden prefix")
        if back.startswith("Ответ:") or back.startswith("Answer:"):
            issues.append("back contains forbidden prefix")
        if len(front) > self.max_front_chars:
            issues.append("front too long")
        if len(back) > self.max_back_chars:
            issues.append("back too long")
        if any(pattern in front.lower() for pattern in FORBIDDEN_FRONT_PATTERNS):
            issues.append("front uses forbidden generic template")
        if front.lower().startswith(("в чем ", "в чём ")):
            issues.append("front uses weak 'В чем' template")
        if self._looks_truncated(front.rstrip("?")):
            issues.append("front looks truncated")
        if self._front_looks_raw(front):
            issues.append("front looks like a raw heading rather than a meaningful question")
        if self._looks_truncated(back):
            issues.append("back looks truncated")
        if self._answer_has_wrapper(back):
            issues.append("back contains wrapper prefix")
        if self._topic_is_weak(topic, source):
            issues.append("topic looks like raw or weak text")
        if source and self._back_is_too_close_to_source(front, back, source):
            issues.append("back is too similar to source fragment")
        if source and self._similarity(front, source) >= 0.72:
            issues.append("front is too similar to source fragment")
        if re.search(r"\b(\w+)\s+\1\b", front.lower()):
            issues.append("front contains duplicated words")
        if len(re.split(r"[;:]", back)) > 3:
            issues.append("back is not atomic")
        return issues

    def _validate_question(self, question: Question) -> List[str]:
        issues: List[str] = []
        prompt = self._normalize(question.question)
        expected = self._normalize(question.expected_answer)
        topic = self._normalize(question.topic)
        source = self._normalize(question.source_fragment)
        if not prompt:
            issues.append("question is empty")
        if not expected:
            issues.append("expected answer is empty")
        if not topic:
            issues.append("topic is empty")
        if prompt and not prompt.endswith("?"):
            issues.append("question must end with '?'")
        if len(prompt) > self.max_front_chars:
            issues.append("question too long")
        if len(expected) > self.max_back_chars:
            issues.append("expected answer too long")
        if self._front_looks_raw(prompt):
            issues.append("question looks like a raw heading rather than a meaningful question")
        if self._answer_has_wrapper(expected):
            issues.append("expected answer contains wrapper prefix")
        if self._topic_is_weak(topic, source):
            issues.append("topic looks like raw or weak text")
        if source and self._back_is_too_close_to_source(prompt, expected, source, threshold=0.9):
            issues.append("expected answer is too similar to source fragment")
        return issues

    def _flashcards_are_duplicates(self, left: Flashcard, right: Flashcard) -> bool:
        left_front = self._normalize(left.front).lower()
        right_front = self._normalize(right.front).lower()
        if self._similarity(left_front, right_front) < 0.92:
            return False

        left_back = self._normalize(left.back).lower()
        right_back = self._normalize(right.back).lower()
        if self._similarity(left_back, right_back) >= 0.88:
            return True

        left_topic = self._normalize(left.topic).lower()
        right_topic = self._normalize(right.topic).lower()
        return bool(left_topic and right_topic) and self._similarity(left_topic, right_topic) >= 0.9

    def filter_flashcards(self, cards: List[Flashcard]) -> Tuple[List[Flashcard], List[str]]:
        filtered: List[Flashcard] = []
        notes: List[str] = []
        seen: list[Flashcard] = []
        for card in cards:
            issues = self._validate_flashcard(card)
            if issues:
                notes.append(f"{card.id} dropped: {', '.join(issues)}")
                continue

            if any(self._flashcards_are_duplicates(card, existing) for existing in seen):
                notes.append(f"{card.id} dropped: near-duplicate card")
                continue
            filtered.append(card)
            seen.append(card)
        return filtered, notes

    def filter_questions(self, questions: List[Question]) -> Tuple[List[Question], List[str]]:
        filtered: List[Question] = []
        notes: List[str] = []
        seen: list[str] = []
        for question in questions:
            issues = self._validate_question(question)
            if issues:
                notes.append(f"{question.id} dropped: {', '.join(issues)}")
                continue
            normalized = self._normalize(question.question).lower()
            if any(self._similarity(normalized, existing) >= 0.92 for existing in seen):
                notes.append(f"{question.id} dropped: near-duplicate question")
                continue
            filtered.append(question)
            seen.append(normalized)
        return filtered, notes

    def review_flashcards(self, cards: List[Flashcard]) -> ReviewReport:
        issues: List[str] = []
        duplicates: List[str] = []
        seen_cards: list[Flashcard] = []
        for card in cards:
            card_issues = self._validate_flashcard(card)
            issues.extend([f"Card {card.id}: {issue}" for issue in card_issues])
            if any(self._flashcards_are_duplicates(card, existing) for existing in seen_cards):
                duplicates.append(card.id)
            else:
                seen_cards.append(card)
        if not cards:
            issues.append("No valid flashcards remained after filtering")
        return ReviewReport(
            passed=not issues and not duplicates,
            issues=issues,
            duplicates_found=duplicates,
            notes=[],
        )

    def review_questions(self, questions: List[Question]) -> ReviewReport:
        issues: List[str] = []
        duplicates: List[str] = []
        seen_questions: list[str] = []
        for question in questions:
            question_issues = self._validate_question(question)
            issues.extend([f"Question {question.id}: {issue}" for issue in question_issues])
            normalized = self._normalize(question.question).lower()
            if any(self._similarity(normalized, existing) >= 0.92 for existing in seen_questions):
                duplicates.append(question.id)
            else:
                seen_questions.append(normalized)
        if not questions:
            issues.append("No valid questions remained after filtering")
        return ReviewReport(
            passed=not issues and not duplicates,
            issues=issues,
            duplicates_found=duplicates,
            notes=[],
        )
