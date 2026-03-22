from __future__ import annotations

import itertools
import json
import re
from pathlib import Path
from typing import Iterable, List

from app.domain.enums import KnowledgeType
from app.domain.models.entities import KnowledgeUnit
from app.services.llm_client import LLMClient


class ExtractionService:
    def __init__(self, llm_client: LLMClient, prompt_path: str) -> None:
        self.llm = llm_client
        self.prompt_path = prompt_path
        self._prompt_template = Path(prompt_path).read_text(encoding="utf-8")

    def _extract_json_array(self, text: str) -> list[dict]:
        match = re.search(r"```(?:json)?\s*(\[.*?\])\s*```", text or "", flags=re.DOTALL | re.IGNORECASE)
        if match:
            text = match.group(1)
        text = (text or "").strip()
        if not text.startswith("[") and "[" in text and "]" in text:
            text = text[text.find("[") : text.rfind("]") + 1]
        try:
            data = json.loads(text)
        except Exception:
            return []
        if isinstance(data, list):
            return [item for item in data if isinstance(item, dict)]
        return []

    def _normalize_note(self, note: str) -> str:
        lines: list[str] = []
        previous_blank = False
        for raw_line in note.replace("\r\n", "\n").replace("\r", "\n").split("\n"):
            line = " ".join(raw_line.split())
            if not line:
                if not previous_blank:
                    lines.append("")
                previous_blank = True
                continue
            lines.append(line)
            previous_blank = False
        return "\n".join(lines).strip()

    def _strip_marker(self, text: str) -> str:
        text = re.sub(r"^\s*\d+[).]\s*", "", text)
        text = re.sub(r"^\s*[-*•]\s*", "", text)
        return text.strip()

    def _is_heading(self, line: str) -> bool:
        if "\n" in line:
            return False
        stripped = self._strip_marker(line).rstrip(":")
        if not stripped or len(stripped) > 80:
            return False
        if re.search(r"[.!?]", stripped):
            return False
        words = stripped.split()
        if not 1 <= len(words) <= 8:
            return False
        return stripped[0].isupper()

    def _looks_like_comparison_heading(self, text: str) -> bool:
        return bool(re.match(r"^(?:сравнение|различия|отличия)\b", self._strip_marker(text), flags=re.IGNORECASE))

    def _normalize_topic(self, text: str) -> str:
        topic = self._strip_marker(text)
        topic = re.sub(r"\s+", " ", topic)
        topic = topic.strip(" .,:;!?\"'()[]{}")
        topic = topic.lower()
        comparison_match = re.match(
            r"^(?:сравнение|различия|отличия)\s+(.+?)(?:\s+(?:по|признак|признаки|критерий|критерии|таблица)\b|$)",
            topic,
        )
        if comparison_match:
            topic = comparison_match.group(1).strip()
        topic = re.sub(r"\b(?:определение|таблица|схема)\b", "", topic)
        topic = re.sub(r"\bпризнак\b$", "", topic).strip()
        if topic.startswith("главная идея "):
            topic = topic.replace("главная идея ", "", 1).strip()
        if topic.startswith("основные типы "):
            topic = topic.replace("основные типы ", "", 1).strip()
        tokens = topic.split()
        if tokens:
            deduped: list[str] = []
            seen: set[str] = set()
            for token in tokens:
                if token in {"и", "или"} or token not in seen:
                    deduped.append(token)
                    seen.add(token)
            topic = " ".join(deduped)
        return topic[:70]

    def _looks_like_formula_topic(self, text: str) -> bool:
        normalized = self._normalize_topic(text)
        if not normalized:
            return False
        if "=" in normalized or "≈" in normalized:
            return True
        if re.search(r"\b[a-zа-я]\w*\s*/\s*[a-zа-я]\w*\b", normalized, flags=re.IGNORECASE):
            return True
        return False

    def _canonical_topic(self, topic: str, context_topic: str | None = None) -> str:
        normalized = self._normalize_topic(topic)
        context = self._normalize_topic(context_topic or "")
        comparison = self._comparison_topic(topic) or self._comparison_topic(context)
        if comparison:
            normalized = comparison
        if not normalized:
            return context
        if not context or normalized == context or context in normalized:
            return normalized
        if normalized in {"главная идея", "идея"}:
            return context
        if normalized in {"пример", "примеры"}:
            return context
        if normalized in {"формула", "уравнение", "обозначение"} and context:
            return context
        if normalized in {"структура данных", "этап", "шаг", "процесс"} and context:
            return f"{normalized} ({context})"[:70]
        return normalized

    def _comparison_topic(self, text: str | None) -> str:
        normalized = self._normalize_topic(text or "")
        if not normalized:
            return ""
        if " и " in normalized and len(normalized.split()) <= 6:
            return normalized
        match = re.match(
            r"^(?:сравнение|различия|отличия)\s+(.+?)(?:\s+(?:по|признак|признаки|критерий|критерии|таблица)\b|$)",
            self._strip_marker(text or "").lower(),
        )
        if match:
            return self._normalize_topic(match.group(1))
        return ""

    def _comparison_entities(self, topic: str) -> tuple[str, str] | None:
        normalized = self._normalize_topic(topic)
        if " и " not in normalized:
            return None
        left, right = [part.strip() for part in normalized.split(" и ", 1)]
        if not left or not right:
            return None
        return left, right

    def _join_items(self, items: list[str]) -> str:
        cleaned = [self._strip_marker(item).strip(" ,;:.") for item in items if self._strip_marker(item).strip(" ,;:.")]
        if not cleaned:
            return ""
        if len(cleaned) == 1:
            return cleaned[0]
        if len(cleaned) == 2:
            return f"{cleaned[0]} и {cleaned[1]}"
        return f"{', '.join(cleaned[:-1])} и {cleaned[-1]}"

    def _looks_like_value_start(self, token: str) -> bool:
        stripped = token.strip("«»\"'()")
        if not stripped:
            return False
        return bool(re.match(r"^[A-ZА-ЯЁ0-9]", stripped))

    def _comparison_value_split(self, tokens: list[str]) -> int | None:
        best_idx: int | None = None
        best_score = float("-inf")
        for idx in range(1, len(tokens)):
            left = tokens[:idx]
            right = tokens[idx:]
            if not left or not right:
                continue
            score = 0.0
            if self._looks_like_value_start(tokens[idx]):
                score += 2.5
            if len(left) <= 4:
                score += 1.0
            if len(right) <= 6:
                score += 1.0
            if len(left) > 6 or len(right) > 6:
                score -= 1.5
            score -= abs(len(left) - len(right)) * 0.2
            if score > best_score:
                best_score = score
                best_idx = idx
        return best_idx

    def _build_list_block_unit(
        self,
        lines: list[str],
        context_topic: str | None,
        counter: Iterable[int],
    ) -> list[KnowledgeUnit]:
        topic = self._canonical_topic(context_topic or "")
        items = [self._strip_marker(line).rstrip(".") for line in lines if self._strip_marker(line) and ":" not in line]
        if not topic or len(items) < 2:
            return []
        content = f"{topic.capitalize()} включает {self._join_items(items)}"
        unit = self._build_unit(
            idx=next(counter),
            ktype=KnowledgeType.LIST_ITEM,
            topic=topic,
            content=content,
            source_fragment=" ".join(lines),
            importance=0.82,
            confidence=0.8,
        )
        return [unit] if unit else []

    def _comparison_row_from_line(self, line: str, comparison_topic: str) -> tuple[str, str, str] | None:
        tokens = [token.strip(" ,;:.") for token in self._strip_marker(line).split() if token.strip(" ,;:.")]
        if len(tokens) < 3:
            return None
        best: tuple[str, str, str] | None = None
        best_score = float("-inf")

        for attr_len in range(1, min(3, len(tokens) - 2) + 1):
            attr_tokens = tokens[:attr_len]
            if attr_tokens[0].lower() in {"сравнение", "различия", "отличия", "признак", "признаки"}:
                continue
            remaining = tokens[attr_len:]
            split_idx = self._comparison_value_split(remaining)
            if split_idx is None:
                continue
            left_tokens = remaining[:split_idx]
            right_tokens = remaining[split_idx:]
            attr = " ".join(attr_tokens)
            left = " ".join(left_tokens)
            right = " ".join(right_tokens)
            if not attr or not left or not right:
                continue
            score = 0.0
            if attr_len == 1:
                score += 1.0
            if attr_len >= 2 and attr_tokens[-1][:1].islower():
                score += 1.5
            if self._looks_like_value_start(right_tokens[0]):
                score += 1.0
            if comparison_topic and attr.lower() in self._normalize_topic(comparison_topic):
                score -= 1.0
            score -= abs(len(left_tokens) - len(right_tokens)) * 0.25
            if score > best_score:
                best_score = score
                best = (attr, left, right)
        return best

    def _comparison_rows_from_flat_text(self, text: str, comparison_topic: str) -> list[tuple[str, str, str]]:
        normalized = self._strip_marker(text)
        match = re.match(
            r"^(?:сравнение|различия|отличия)\s+.+?\s+признак\s+\S+\s+\S+\s+(.+)$",
            normalized,
            flags=re.IGNORECASE,
        )
        if not match:
            return []
        tokens = [token.strip(" ,;:.") for token in match.group(1).split() if token.strip(" ,;:.")]
        rows: list[tuple[str, str, str]] = []
        while len(tokens) >= 3 and len(rows) < 4:
            attr, left, right = tokens[0], tokens[1], tokens[2]
            rows.append((attr, left, right))
            tokens = tokens[3:]
        return rows

    def _build_comparison_units(
        self,
        lines: list[str],
        context_topic: str | None,
        counter: Iterable[int],
    ) -> list[KnowledgeUnit]:
        cleaned_lines = [self._strip_marker(line) for line in lines if self._strip_marker(line)]
        heading_line = cleaned_lines[0] if cleaned_lines and self._looks_like_comparison_heading(cleaned_lines[0]) else ""
        row_lines = cleaned_lines[1:] if heading_line else cleaned_lines
        comparison_topic = (
            self._comparison_topic(heading_line)
            or self._comparison_topic(context_topic or "")
            or self._comparison_topic(" ".join(cleaned_lines))
        )
        entities = self._comparison_entities(comparison_topic)
        if not comparison_topic or not entities:
            return []

        units: list[KnowledgeUnit] = []
        row_labels: list[str] = []
        rows: list[tuple[str, str, str]] = []

        for line in row_lines:
            parsed = self._comparison_row_from_line(line, comparison_topic)
            if parsed:
                rows.append(parsed)

        if not rows and cleaned_lines:
            rows.extend(self._comparison_rows_from_flat_text(" ".join(cleaned_lines), comparison_topic))

        for attr, left_value, right_value in rows[:4]:
            row_labels.append(attr.lower())
            content = f"По признаку «{attr.lower()}»: у {entities[0]} — {left_value.lower()}, у {entities[1]} — {right_value.lower()}"
            unit = self._build_unit(
                idx=next(counter),
                ktype=KnowledgeType.RELATION,
                topic=comparison_topic,
                content=content,
                source_fragment=f"{attr} {left_value} {right_value}",
                importance=0.86,
                confidence=0.82,
            )
            if unit:
                units.append(unit)

        if row_labels:
            summary = self._build_unit(
                idx=next(counter),
                ktype=KnowledgeType.LIST_ITEM,
                topic=comparison_topic,
                content=f"{comparison_topic.capitalize()} сравнивают по признакам: {self._join_items(row_labels)}",
                source_fragment=" ".join(cleaned_lines),
                importance=0.8,
                confidence=0.78,
            )
            if summary:
                units.insert(0, summary)

        return units

    def _split_sentences(self, text: str) -> list[str]:
        parts = re.split(r"(?<=[.!?])\s+", text)
        return [part.strip() for part in parts if part and part.strip()]

    def _split_list_items(self, text: str) -> list[str]:
        if ";" in text:
            parts = [part.strip(" ,;") for part in text.split(";")]
            return [part for part in parts if part]
        if re.search(r"\b(?:включает|состоит из|содержит)\b", text.lower()):
            return [text.strip()]
        comma_parts = [part.strip(" ,") for part in text.split(",")]
        if len(comma_parts) > 2 and all(3 <= len(part.split()) <= 12 for part in comma_parts):
            return [part for part in comma_parts if part]
        return [text.strip()]

    def _looks_like_list(self, text: str) -> bool:
        lowered = text.lower()
        return bool(
            re.search(r"\b(?:включает|состоит из|содержит|типы|элементы|части)\b", lowered)
            or lowered.count(",") >= 2
            or ";" in lowered
        )

    def _looks_like_example(self, text: str) -> bool:
        lowered = text.lower()
        return lowered.startswith("пример") or "например" in lowered or "служит примером" in lowered

    def _looks_like_process_step(self, text: str) -> bool:
        lowered = text.lower()
        return bool(
            lowered.startswith(("шаг", "этап", "сначала", "затем", "после этого"))
            or re.search(r"\bэтап\b", lowered)
        )

    def _looks_like_relation(self, text: str) -> bool:
        lowered = text.lower()
        return bool(
            re.search(r"\b(?:связан|отличается|зависит|включает|состоит из|содержит|делится на)\b", lowered)
        )

    def _definition_match(self, text: str) -> re.Match[str] | None:
        patterns = (
            r"^([^:=]{2,80}?)\s*[—-]\s*это\s+(.+)$",
            r"^([^:=]{2,80}?)\s*[—-]\s+(.+)$",
            r"^(.{2,80}?)\s+это\s+(.+)$",
            r"^(.{2,80}?)\s+is\s+(.+)$",
        )
        for pattern in patterns:
            match = re.match(pattern, text, flags=re.IGNORECASE)
            if match:
                return match
        return None

    def _clean_fragment(self, text: str) -> str:
        fragment = re.sub(r"\s+", " ", text).strip(" ,;")
        if fragment and fragment[-1] not in ".!?":
            fragment = f"{fragment}."
        return fragment

    def _clean_answer_clause(self, text: str) -> str:
        clause = re.sub(r"^(?:например|пример:?)\s*", "", text.strip(), flags=re.IGNORECASE)
        clause = clause.strip(" ,;")
        return self._clean_fragment(clause)

    def _strip_leading_clause(self, text: str) -> str:
        stripped = self._strip_marker(text)
        conditional = re.match(r"^если\s+[^,]+,\s+(.+)$", stripped, flags=re.IGNORECASE)
        if conditional:
            return conditional.group(1).strip()
        dated = re.match(
            r"^(?:\d{1,2}\s+[А-Яа-яё]+\s+\d{4}\s+года|[Вв]\s+\d{4}(?:-\d{4})?\s+год(?:у|ах))\s+(.+)$",
            stripped,
            flags=re.IGNORECASE,
        )
        if dated:
            return dated.group(1).strip()
        return stripped

    def _looks_like_predicate_start(self, token: str) -> bool:
        lowered = token.lower().strip("«»\"'()")
        if not lowered:
            return False
        direct_matches = {
            "есть",
            "был",
            "была",
            "было",
            "были",
            "может",
            "равен",
            "равна",
            "равно",
            "равны",
            "направлен",
            "направлена",
            "направлено",
            "направлены",
            "созваны",
            "созвана",
            "созвано",
            "созван",
            "принята",
            "принят",
            "принято",
            "приняты",
            "свергнуто",
            "упразднена",
            "казнен",
            "достиг",
            "достигла",
            "достигло",
            "достигли",
            "одинаковы",
            "одинакова",
            "одинаково",
            "одинаков",
        }
        if lowered in direct_matches:
            return True
        return bool(
            re.match(
                r".*(?:ет|ют|ут|ит|ят|ется|ются|ится|ятся|ался|алась|алось|ались|ился|илась|илось|ились|ал|ала|ало|али|ил|ила|ило|или)$",
                lowered,
                flags=re.IGNORECASE,
            )
        )

    def _subject_topic_from_text(self, text: str) -> str:
        stripped = self._strip_leading_clause(text)
        if not stripped:
            return ""
        passive_match = re.match(
            r"^(?:был[аио]?|были|произошл[аои]?|созван[аоы]?|принят[аоы]?|свергнут[аоы]?|упразднен[аоы]?|казнен)\s+(.+)$",
            stripped,
            flags=re.IGNORECASE,
        )
        if passive_match:
            return self._normalize_topic(" ".join(passive_match.group(1).split()[:4]))

        words = re.findall(r"[A-Za-zА-Яа-яЁё0-9+-]+", stripped)
        if len(words) < 2:
            return ""
        for idx in range(1, min(5, len(words))):
            if self._looks_like_predicate_start(words[idx]):
                if idx > 1 and words[idx - 1].lower() in {"почти", "обычно", "часто"}:
                    subject = " ".join(words[: idx - 1])
                    normalized = self._normalize_topic(subject)
                    if normalized in {"он", "она", "оно", "они"}:
                        return ""
                    return normalized
                subject = " ".join(words[:idx])
                normalized = self._normalize_topic(subject)
                if normalized in {"он", "она", "оно", "они"}:
                    return ""
                return normalized
        return ""

    def _infer_topic_from_text(self, text: str, context_topic: str | None = None) -> str:
        definition = self._definition_match(text)
        if definition:
            return self._canonical_topic(definition.group(1), context_topic)
        label_match = re.match(r"^([^:]{2,80}):\s+(.+)$", text)
        if label_match:
            return self._canonical_topic(label_match.group(1), context_topic)
        subject_topic = self._subject_topic_from_text(text)
        if subject_topic and not self._looks_like_formula_topic(subject_topic):
            return self._canonical_topic(subject_topic, context_topic)
        stripped = self._strip_leading_clause(text)
        words = [word for word in re.findall(r"[A-Za-zА-Яа-я0-9-]+", stripped) if len(word) > 2]
        if words:
            return self._normalize_topic(" ".join(words[:4]))
        return self._normalize_topic(context_topic or "конспект")

    def _build_unit(
        self,
        *,
        idx: int,
        ktype: KnowledgeType,
        topic: str,
        content: str,
        source_fragment: str,
        importance: float = 0.75,
        confidence: float = 0.75,
    ) -> KnowledgeUnit | None:
        normalized_topic = self._normalize_topic(topic)
        normalized_content = self._clean_fragment(content)
        normalized_source = self._clean_fragment(source_fragment)
        if not normalized_topic or len(normalized_content) < 25:
            return None
        if normalized_content[0].islower():
            return None
        if self._looks_like_formula_topic(normalized_topic):
            return None
        # Drop meta "table of contents" units that tend to produce junk flashcards.
        if normalized_topic.startswith(("конспект", "шпаргалка", "содержание", "план")):
            return None
        return KnowledgeUnit(
            id=f"ku_{idx}",
            type=ktype,
            topic=normalized_topic,
            content=normalized_content,
            source_fragment=normalized_source,
            importance=max(0.0, min(1.0, importance)),
            confidence=max(0.0, min(1.0, confidence)),
        )

    def _units_from_labelled_fact(
        self,
        label: str,
        body: str,
        context_topic: str | None,
        counter: Iterable[int],
    ) -> list[KnowledgeUnit]:
        topic = self._canonical_topic(label, context_topic)
        if not topic:
            topic = self._canonical_topic(context_topic or label)

        if self._looks_like_example(body):
            content = f"Пример для {topic} — {self._clean_answer_clause(body)}"
            unit = self._build_unit(
                idx=next(counter),
                ktype=KnowledgeType.EXAMPLE,
                topic=topic,
                content=content,
                source_fragment=f"{label}: {body}",
                importance=0.7,
                confidence=0.7,
            )
            return [unit] if unit else []

        if self._looks_like_list(body):
            content = f"{topic.capitalize()} включает {self._clean_answer_clause(body).rstrip('.')}"
            unit = self._build_unit(
                idx=next(counter),
                ktype=KnowledgeType.LIST_ITEM,
                topic=topic,
                content=content,
                source_fragment=f"{label}: {body}",
                importance=0.78,
                confidence=0.78,
            )
            return [unit] if unit else []

        if label.lower().startswith("главная идея"):
            base_topic = self._canonical_topic(context_topic or "главная идея")
            content = f"Главная идея темы «{base_topic}» состоит в том, что {self._clean_answer_clause(body).rstrip('.')}"
            unit = self._build_unit(
                idx=next(counter),
                ktype=KnowledgeType.CONCEPT,
                topic=base_topic,
                content=content,
                source_fragment=f"{label}: {body}",
                importance=0.85,
                confidence=0.8,
            )
            return [unit] if unit else []

        content = f"{label}: {self._clean_answer_clause(body)}"
        unit = self._build_unit(
            idx=next(counter),
            ktype=KnowledgeType.FACT,
            topic=topic,
            content=content,
            source_fragment=f"{label}: {body}",
            importance=0.72,
            confidence=0.72,
        )
        return [unit] if unit else []

    def _units_from_sentence(
        self,
        sentence: str,
        context_topic: str | None,
        counter: Iterable[int],
    ) -> list[KnowledgeUnit]:
        stripped = self._strip_marker(sentence)
        if len(stripped) < 20:
            return []

        label_match = re.match(r"^([^:]{2,80}):\s+(.+)$", stripped)
        if label_match:
            return self._units_from_labelled_fact(label_match.group(1), label_match.group(2), context_topic, counter)

        definition = self._definition_match(stripped)
        if definition:
            raw_topic = definition.group(1)
            body = definition.group(2)
            topic = self._canonical_topic(raw_topic, context_topic)
            unit = self._build_unit(
                idx=next(counter),
                ktype=KnowledgeType.DEFINITION if topic else KnowledgeType.CONCEPT,
                topic=topic or self._infer_topic_from_text(stripped, context_topic),
                content=f"{raw_topic.strip()} — это {self._clean_answer_clause(body).rstrip('.')}",
                source_fragment=stripped,
                importance=0.88,
                confidence=0.82,
            )
            return [unit] if unit else []

        if self._looks_like_example(stripped):
            topic = self._canonical_topic(context_topic or self._infer_topic_from_text(stripped))
            content = f"Пример для {topic} — {self._clean_answer_clause(stripped)}"
            unit = self._build_unit(
                idx=next(counter),
                ktype=KnowledgeType.EXAMPLE,
                topic=topic,
                content=content,
                source_fragment=stripped,
                importance=0.68,
                confidence=0.7,
            )
            return [unit] if unit else []

        if self._looks_like_process_step(stripped):
            topic = self._canonical_topic(context_topic or self._infer_topic_from_text(stripped))
            unit = self._build_unit(
                idx=next(counter),
                ktype=KnowledgeType.PROCESS_STEP,
                topic=topic,
                content=stripped,
                source_fragment=stripped,
                importance=0.7,
                confidence=0.72,
            )
            return [unit] if unit else []

        inferred_topic = self._infer_topic_from_text(stripped, context_topic)
        if self._looks_like_relation(stripped):
            topic = inferred_topic or self._canonical_topic(context_topic or self._infer_topic_from_text(stripped))
            unit_type = KnowledgeType.LIST_ITEM if self._looks_like_list(stripped) else KnowledgeType.RELATION
            unit = self._build_unit(
                idx=next(counter),
                ktype=unit_type,
                topic=topic,
                content=stripped,
                source_fragment=stripped,
                importance=0.76,
                confidence=0.75,
            )
            return [unit] if unit else []

        topic = inferred_topic or self._canonical_topic(context_topic or self._infer_topic_from_text(stripped))
        unit = self._build_unit(
            idx=next(counter),
            ktype=KnowledgeType.FACT,
            topic=topic,
            content=stripped,
            source_fragment=stripped,
            importance=0.65,
            confidence=0.68,
        )
        return [unit] if unit else []

    def _extract_paragraph_units(
        self,
        paragraph: str,
        context_topic: str | None,
        counter: Iterable[int],
    ) -> list[KnowledgeUnit]:
        raw_lines = [line for line in paragraph.split("\n") if line.strip()]
        stripped_lines = [self._strip_marker(line) for line in raw_lines if self._strip_marker(line)]
        if not stripped_lines:
            return []

        paragraph_context = context_topic
        if (
            len(stripped_lines) >= 2
            and self._is_heading(stripped_lines[0])
            and (
                re.search(r"[.!?:—-]", stripped_lines[1])
                or self._definition_match(stripped_lines[1])
                or re.match(r"^([^:]{2,80}):\s+.+$", stripped_lines[1])
                or self._looks_like_comparison_heading(stripped_lines[0])
                or (len(stripped_lines) >= 3 and all(not re.search(r"[.!?]", line) for line in stripped_lines[1:]))
            )
        ):
            paragraph_context = self._normalize_topic(stripped_lines[0])
            raw_lines = raw_lines[1:]
            stripped_lines = stripped_lines[1:]
            if not stripped_lines:
                return []

        is_list_block = (
            len(stripped_lines) > 1
            and all(not line.endswith(",") for line in raw_lines)
            and all(len(line) <= 120 for line in stripped_lines)
            and all(not re.search(r"[.!?]", line) for line in stripped_lines)
        )

        candidates: list[str] = []
        if is_list_block:
            comparison_units = self._build_comparison_units(stripped_lines, paragraph_context, counter)
            if comparison_units:
                return comparison_units
            list_block_units = self._build_list_block_unit(stripped_lines, paragraph_context, counter)
            if list_block_units:
                return list_block_units
            candidates.extend(stripped_lines)
        else:
            combined = " ".join(stripped_lines)
            flat_comparison_units = self._build_comparison_units([combined], paragraph_context or combined, counter)
            if flat_comparison_units:
                return flat_comparison_units
            for sentence in self._split_sentences(combined):
                if self._looks_like_list(sentence) and len(sentence) > 170:
                    candidates.extend(self._split_list_items(sentence))
                else:
                    candidates.append(sentence)

        units: list[KnowledgeUnit] = []
        for candidate in candidates:
            units.extend(self._units_from_sentence(candidate, paragraph_context, counter))
        return units

    def _dedupe_units(self, units: list[KnowledgeUnit]) -> list[KnowledgeUnit]:
        unique: list[KnowledgeUnit] = []
        seen = set()
        for unit in units:
            key = (
                unit.type.value,
                self._normalize_topic(unit.topic),
                re.sub(r"\s+", " ", unit.content.lower()),
            )
            if key in seen:
                continue
            unique.append(unit)
            seen.add(key)
        return unique

    def _parse_llm_units(self, items: list[dict], max_units: int) -> List[KnowledgeUnit]:
        units: List[KnowledgeUnit] = []
        counter = itertools.count(1)
        for item in items:
            content = str(item.get("content") or "").strip()
            topic = str(item.get("topic") or "").strip()
            if not content or not topic:
                continue
            raw_type = str(item.get("type") or "fact").strip().lower()
            try:
                ktype = KnowledgeType(raw_type)
            except Exception:
                ktype = KnowledgeType.FACT
            source_fragment = str(item.get("source_fragment") or content).strip()
            importance = float(item.get("importance") or 0.75)
            confidence = float(item.get("confidence") or 0.75)
            unit = self._build_unit(
                idx=next(counter),
                ktype=ktype,
                topic=topic,
                content=content,
                source_fragment=source_fragment,
                importance=importance,
                confidence=confidence,
            )
            if unit:
                units.append(unit)
            if len(units) >= max_units:
                break
        return self._dedupe_units(units)

    def _deterministic_extract(self, note: str, max_units: int) -> List[KnowledgeUnit]:
        normalized = self._normalize_note(note)
        if not normalized:
            return []

        paragraphs = [paragraph.strip() for paragraph in re.split(r"\n{2,}", normalized) if paragraph.strip()]
        counter = itertools.count(1)
        units: list[KnowledgeUnit] = []
        current_heading: str | None = None
        context_topic: str | None = None

        for paragraph in paragraphs:
            if self._is_heading(paragraph):
                current_heading = self._strip_marker(paragraph).rstrip(":")
                context_topic = None
                continue

            paragraph_context = context_topic or self._normalize_topic(current_heading or "")
            extracted = self._extract_paragraph_units(paragraph, paragraph_context, counter)
            for unit in extracted:
                units.append(unit)
                if unit.type in {KnowledgeType.DEFINITION, KnowledgeType.CONCEPT, KnowledgeType.TERM}:
                    context_topic = unit.topic
                if len(units) >= max_units:
                    return self._dedupe_units(units)

        return self._dedupe_units(units)[:max_units]

    def extract(self, note: str, max_units: int = 20) -> List[KnowledgeUnit]:
        note_norm = note.strip()
        if not note_norm:
            return []

        items: list[dict] = []
        try:
            prompt = self._prompt_template.format(max_units=max_units, note_text=note_norm)
            raw = self.llm.generate(prompt, max_tokens=1200)
            items = self._extract_json_array(raw or "")
        except Exception:
            items = []

        parsed_units = self._parse_llm_units(items, max_units)
        if parsed_units:
            return parsed_units
        return self._deterministic_extract(note_norm, max_units)
