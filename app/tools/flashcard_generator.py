from __future__ import annotations

import json
import re
from difflib import SequenceMatcher
from pathlib import Path
from typing import List

from app.domain.enums import KnowledgeType
from app.domain.models.entities import Flashcard, GenerationContext, KnowledgeUnit, Question
from app.services.llm_client import LLMClient


class _GenerationToolMixin:
    max_front_chars = 120
    max_back_chars = 200

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

    def _normalize_text(self, text: str) -> str:
        return re.sub(r"\s+", " ", (text or "").strip())

    def _normalize_topic(self, topic: str) -> str:
        cleaned = self._normalize_text(topic).strip(" .,:;!?\"'()[]{}").lower()
        cleaned = re.sub(r"\s+", " ", cleaned)
        return cleaned[:60]

    def _looks_like_formula_topic(self, topic: str) -> bool:
        normalized = self._normalize_topic(topic)
        if not normalized:
            return False
        if "=" in normalized or "≈" in normalized:
            return True
        if re.search(r"\b[a-zа-я]\w*\s*/\s*[a-zа-я]\w*\b", normalized, flags=re.IGNORECASE):
            return True
        return False

    def _is_clean_source_label(self, topic: str, source_fragment: str) -> bool:
        normalized_topic = self._normalize_topic(topic)
        normalized_source = self._normalize_text(source_fragment)
        if not normalized_topic or not normalized_source:
            return False
        source_lower = normalized_source.lower()
        if not source_lower.startswith(normalized_topic):
            return False
        tail = normalized_source[len(normalized_topic) :].lstrip()
        return tail.startswith(("-", "—", ":"))

    def _topic_is_weak(self, topic: str, source_fragment: str = "") -> bool:
        normalized = self._normalize_topic(topic)
        if not normalized:
            return True
        if self._looks_like_formula_topic(normalized):
            return True
        words = normalized.split()
        if len(normalized) > 60 or len(words) > 7:
            return True
        if normalized.startswith(("если ", "в этом ", "это ", "например ", "что ")):
            return True
        if re.search(r"\b\d+[.)]?$", normalized):
            return True
        if len(words) >= 4 and len(set(words)) < len(words):
            return True
        if len(words) >= 3 and words[-1] in {"признак", "определение", "таблица", "схема"}:
            return True
        if source_fragment:
            normalized_source = self._normalize_text(source_fragment)
            source_start = normalized_source.lower()[: max(len(normalized), 12)]
            if normalized == source_start and len(words) >= 4 and not self._is_clean_source_label(normalized, source_fragment):
                tail = normalized_source[len(normalized) :].lstrip(" ,;:-")
                first_tail = tail.split()[0] if tail.split() else ""
                if not first_tail or not first_tail[:1].islower():
                    return True
        return False

    def _definition_topic_from_text(self, text: str) -> str:
        normalized = self._normalize_text(text)
        patterns = (
            r"^([^:=]{2,80}?)\s*[—-]\s*это\s+.+$",
            r"^([^:=]{2,80}?)\s*[—-]\s+.+$",
            r"^(.{2,80}?)\s+это\s+.+$",
        )
        for pattern in patterns:
            match = re.match(pattern, normalized, flags=re.IGNORECASE)
            if match:
                return self._normalize_topic(match.group(1))
        return ""

    def _label_topic_from_text(self, text: str) -> str:
        normalized = self._normalize_text(text)
        match = re.match(r"^([^:]{2,80}?):\s+.+$", normalized)
        if not match:
            return ""
        label = self._normalize_topic(match.group(1))
        if label in {"признак", "свойство", "пример", "формула", "уравнение", "обозначение"} or label.startswith("по признаку "):
            return ""
        if label.endswith("сравнивают по признакам") or label.endswith("проводят по признакам"):
            return ""
        return label

    def _comparison_topic_from_text(self, text: str) -> str:
        normalized = self._normalize_text(text)
        patterns = (
            r"^(?:сравнение|различия|отличия)\s+(.+?)(?:\s+(?:по|признак|признаки|критерий|критерии)\b|[:.;]|$)",
            r"^(.+?)\s+отличаются\s+от\s+(.+?)(?:[.?!]|$)",
        )
        for pattern in patterns:
            match = re.match(pattern, normalized, flags=re.IGNORECASE)
            if not match:
                continue
            if len(match.groups()) == 2:
                return self._normalize_topic(f"{match.group(1)} и {match.group(2)}")
            return self._normalize_topic(match.group(1))
        return ""

    def _comparison_axis_from_text(self, text: str) -> str:
        normalized = self._normalize_text(text)
        match = re.match(r"^по признаку\s+[«\"]?([^:»\"]+)[»\"]?:", normalized, flags=re.IGNORECASE)
        if match:
            return self._normalize_topic(match.group(1))
        return ""

    def _topic_from_question_text(self, text: str) -> str:
        normalized = self._normalize_text(text)
        if normalized.endswith("?"):
            quoted = re.search(r"[«\"]([^»\"]+)[»\"]", normalized)
            if quoted:
                return self._normalize_topic(quoted.group(1))
        patterns = (
            r"^(?:что такое|что означает(?: термин)?)\s+(.+?)\?$",
            r"^(?:какой факт о|какая особенность|что входит в|что происходит на шаге)\s+[«\"]?(.+?)[»\"]?\?$",
            r"^(?:чем отличаются)\s+(.+?)\?$",
            r"^(?:чем характеризуется|с чем связано|от чего зависит|на что делится|что есть у|как(?:ова)? функция)\s+[«\"]?(.+?)[»\"]?\?$",
            r"^(?:чему равна|как вычислить|как записывается)\s+[«\"]?(.+?)[»\"]?\?$",
        )
        lowered = normalized.lower()
        for pattern in patterns:
            match = re.match(pattern, lowered, flags=re.IGNORECASE)
            if match:
                return self._normalize_topic(match.group(1))
        return ""

    def _resolve_topic(self, topic: str, *texts: str) -> str:
        candidates: list[str] = []
        reference_text = next((text for text in reversed(texts) if text), "")

        normalized_topic = self._normalize_topic(topic)
        if normalized_topic and not self._topic_is_weak(normalized_topic, reference_text):
            candidates.append(normalized_topic)

        for text in texts:
            if not text:
                continue
            candidates.extend(
                [
                    self._definition_topic_from_text(text),
                    self._comparison_topic_from_text(text),
                    self._label_topic_from_text(text),
                    self._topic_from_question_text(text),
                ]
            )
        if topic:
            candidates.extend(
                [
                    self._definition_topic_from_text(topic),
                    self._comparison_topic_from_text(topic),
                    self._label_topic_from_text(topic),
                    self._topic_from_question_text(topic),
                    self._normalize_topic(topic),
                ]
            )
        for text in texts:
            if text:
                candidates.append(self._normalize_topic(text))

        fallback = ""
        for candidate in candidates:
            if not candidate:
                continue
            if not fallback and not self._looks_like_formula_topic(candidate):
                fallback = candidate
            if not self._topic_is_weak(candidate, reference_text):
                return candidate
        return fallback

    def _question_is_wrapper(self, text: str) -> bool:
        lowered = self._normalize_text(text).lower()
        if not lowered:
            return True
        patterns = (
            r"^какой\s+факт\s+о\s+.+\s+указан\s+в\s+конспекте\?$",
            r"^какая\s+особенность\s+.+\s+указана\s+в\s+конспекте\?$",
        )
        return any(re.match(pattern, lowered, flags=re.IGNORECASE) for pattern in patterns)

    def _answer_has_wrapper(self, text: str) -> bool:
        lowered = self._normalize_text(text).lower()
        patterns = (r"^(?:верно следующее|факт о [^:]+|особенность в том, что|при сравнении видно, что)\s*[:,-]",)
        return any(re.match(pattern, lowered, flags=re.IGNORECASE) for pattern in patterns)

    def _truncate(self, text: str, limit: int) -> str:
        cleaned = self._normalize_text(text)
        if len(cleaned) <= limit:
            return cleaned
        truncated = cleaned[: limit - 1].rstrip(" ,;:")
        last_space = truncated.rfind(" ")
        if last_space > limit * 0.6:
            truncated = truncated[:last_space]
        return truncated.rstrip(" ,;:.") + "…"

    def _similarity(self, left: str, right: str) -> float:
        return SequenceMatcher(None, self._normalize_text(left).lower(), self._normalize_text(right).lower()).ratio()

    def _strip_prefixes(self, text: str) -> str:
        cleaned = self._normalize_text(text)
        cleaned = re.sub(r"^(?:вопрос)\s*:\s*", "", cleaned, flags=re.IGNORECASE)
        cleaned = re.sub(r"^(?:ответ)\s*:\s*", "", cleaned, flags=re.IGNORECASE)
        return cleaned

    def _is_definition_question(self, text: str) -> bool:
        lowered = self._normalize_text(text).lower()
        return lowered.startswith(("что такое ", "что означает термин "))

    def _definition_question_topic(self, text: str) -> str:
        normalized = self._normalize_text(text)
        quoted = re.search(r"[«\"]([^»\"]+)[»\"]", normalized)
        if quoted:
            return self._normalize_text(quoted.group(1))
        match = re.match(r"^(?:что такое)\s+(.+?)\?$", normalized, flags=re.IGNORECASE)
        if match:
            return self._normalize_text(match.group(1))
        return ""

    def _definition_answer_is_derived(self, front: str, back: str, source_fragment: str) -> bool:
        if not self._is_definition_question(front):
            return False
        topic = self._definition_question_topic(front)
        source = self._normalize_text(source_fragment)
        answer = self._normalize_text(back)
        if not topic or not source or not answer:
            return False
        patterns = (
            rf"^[«\"]?{re.escape(topic)}[»\"]?\s*[—-]\s*(?:это\s+)?(.+)$",
            rf"^[«\"]?{re.escape(topic)}[»\"]?\s+это\s+(.+)$",
        )
        for pattern in patterns:
            match = re.match(pattern, source, flags=re.IGNORECASE)
            if not match:
                continue
            tail = self._normalize_text(match.group(1))
            variants = {tail, f"Это {tail[0].lower()}{tail[1:]}" if tail else ""}
            if any(variant and self._similarity(answer, variant) >= 0.92 for variant in variants):
                return True
        return False

    def _back_is_too_close_to_source(self, front: str, back: str, source_fragment: str, threshold: float = 0.93) -> bool:
        source = self._normalize_text(source_fragment)
        answer = self._normalize_text(back)
        if not source or not answer:
            return False
        if self._similarity(answer, source) < threshold:
            return False
        if self._definition_answer_is_derived(front, back, source_fragment):
            return False
        return True

    def _extract_date_prefix(self, text: str) -> tuple[str, str] | None:
        normalized = self._normalize_text(text)
        match = re.match(
            r"^((?:\d{1,2}\s+[А-Яа-яё]+\s+\d{4}\s+года)|(?:[Вв]\s+\d{4}(?:-\d{4})?\s+год(?:у|ах)))\s+(.+)$",
            normalized,
            flags=re.IGNORECASE,
        )
        if not match:
            return None
        return match.group(1), match.group(2).strip()

    def _conditional_parts(self, text: str) -> tuple[str, str] | None:
        normalized = self._normalize_text(text)
        match = re.match(r"^Если\s+([^,]+),\s+(.+)$", normalized, flags=re.IGNORECASE)
        if not match:
            return None
        return match.group(1).strip(), match.group(2).strip()

    def _process_step_marker(self, text: str) -> tuple[str, str] | None:
        normalized = self._normalize_text(text)
        match = re.match(r"^(Сначала|Затем|После этого|В конце)\s+(.+)$", normalized, flags=re.IGNORECASE)
        if not match:
            return None
        return match.group(1), match.group(2).strip()

    def _action_question_from_topic(self, topic: str, content: str) -> str:
        normalized = self._normalize_text(content)
        topic_text = self._normalize_text(topic)
        if not topic_text:
            return ""
        match = re.match(rf"^[«\"]?{re.escape(topic_text)}[»\"]?\s+(.+)$", normalized, flags=re.IGNORECASE)
        if not match:
            return ""
        predicate = match.group(1).strip()
        first_word = predicate.split()[0].lower() if predicate.split() else ""
        action_verbs = {
            "отделяет",
            "отделяют",
            "регулирует",
            "регулируют",
            "синтезирует",
            "синтезируют",
            "обеспечивает",
            "обеспечивают",
            "осуществляет",
            "осуществляют",
            "хранит",
            "хранят",
            "управляет",
            "управляют",
            "определяет",
            "определяют",
            "вызывает",
            "вызывают",
            "подавляет",
            "подавляют",
            "проявляется",
            "проявляются",
            "противостоит",
            "противостоят",
            "помогает",
            "помогают",
        }
        if first_word not in action_verbs:
            return ""
        auxiliary = "делают" if first_word.endswith(("ют", "ут", "ят")) else "делает"
        return f"Что {auxiliary} «{topic}»?"

    def _extract_tail(self, content: str, topic: str) -> str:
        normalized = self._normalize_text(content).rstrip(".!?")
        topic_variants = {
            self._normalize_text(topic),
            self._normalize_text(topic).capitalize(),
        }
        date_match = self._extract_date_prefix(normalized)
        if date_match:
            return date_match[1]
        conditional = self._conditional_parts(normalized)
        if conditional:
            return conditional[1]
        process_step = self._process_step_marker(normalized)
        if process_step:
            return process_step[1]
        patterns = [
            rf"^(?:{re.escape(self._normalize_text(topic))}|{re.escape(self._normalize_text(topic).capitalize())})\s*[—-]\s*это\s+(.+)$",
            rf"^(?:{re.escape(self._normalize_text(topic))}|{re.escape(self._normalize_text(topic).capitalize())})\s*[—-]\s+(.+)$",
            rf"^(?:{re.escape(self._normalize_text(topic))}|{re.escape(self._normalize_text(topic).capitalize())})\s+это\s+(.+)$",
            rf"^(?:{re.escape(self._normalize_text(topic))}|{re.escape(self._normalize_text(topic).capitalize())})\s+имеет\s+вид\s+(.+)$",
            r"^главная идея(?: темы)?\s+[«\"]?.+?[»\"]?\s+состоит в том,\s+что\s+(.+)$",
            r"^[^:]{2,80}:\s+(.+)$",
            r"^.+?\b(?:включает|состоит из|содержит|отличается тем, что|связан[аоы]? с)\b\s+(.+)$",
            r"^пример(?: для [^—-]+)?\s*[—-]\s*(.+)$",
        ]
        for pattern in patterns:
            match = re.match(pattern, normalized, flags=re.IGNORECASE)
            if match:
                tail = match.group(1).strip(" ,;:.")
                if tail:
                    return tail
        if topic:
            topic_match = re.match(
                rf"^[«\"]?{re.escape(self._normalize_text(topic))}[»\"]?\s+(.+)$",
                normalized,
                flags=re.IGNORECASE,
            )
            if topic_match:
                tail = topic_match.group(1).strip(" ,;:.")
                if tail:
                    return tail
        words = normalized.split()
        if words and self._normalize_text(words[0]).lower() in {variant.lower() for variant in topic_variants if variant}:
            return " ".join(words[1:]).strip(" ,;:.")
        return normalized

    def _compact_definition_answer(self, answer: str) -> str:
        normalized = self._normalize_text(answer).rstrip(".")
        compact = re.sub(
            r"^Это\s+организмы,\s+клетки которых имеют\s+(.+)$",
            r"У их клеток есть \1",
            normalized,
            flags=re.IGNORECASE,
        )
        compact = re.sub(
            r"^Это\s+(.+?),\s+которые имеют\s+(.+)$",
            r"Для них характерно наличие \2",
            compact,
            flags=re.IGNORECASE,
        )
        return compact if compact != normalized else normalized

    def _strip_answer_wrappers(self, unit: KnowledgeUnit, answer: str) -> str:
        cleaned = self._normalize_text(answer).strip(" ,;:.")
        if unit.type not in {KnowledgeType.DEFINITION, KnowledgeType.TERM, KnowledgeType.EXAMPLE}:
            patterns = (r"^(?:верно следующее|факт о [^:]+|особенность в том, что|при сравнении видно, что|суть в том, что)\s*[:,-]?\s*",)
            for pattern in patterns:
                cleaned = re.sub(pattern, "", cleaned, flags=re.IGNORECASE)
        return cleaned.strip(" ,;:.")

    def _question_from_fact(self, topic: str, content: str, comparison_topic: str, comparison_axis: str) -> str:
        lowered = self._normalize_text(content).lower()
        dated = self._extract_date_prefix(content)
        conditional = self._conditional_parts(content)
        definition_topic = self._definition_topic_from_text(content)
        action_question = self._action_question_from_topic(topic, content)

        if comparison_topic and comparison_axis:
            return f"Чем отличаются «{comparison_topic}» по признаку «{comparison_axis}»?"
        if comparison_topic:
            return f"Чем отличаются «{comparison_topic}»?"
        if dated:
            question_verb = "происходило" if dated[0].lower().startswith("в ") else "произошло"
            date_text = dated[0].lower() if dated[0].startswith("В ") else dated[0]
            return f"Что {question_verb} {date_text}?"
        start_date = re.match(
            r"^(.+?)\s+начал(?:ся|ась|ось|ись)\s+в\s+(\d{4}\s+году)\.?$",
            self._normalize_text(content),
            flags=re.IGNORECASE,
        )
        if start_date:
            subject = self._normalize_topic(start_date.group(1))
            if subject:
                return f"Когда началась «{subject}»?"
        if definition_topic:
            return f"Что такое «{definition_topic}»?"
        if conditional:
            condition, result = conditional
            if "корн" in result.lower():
                return f"Сколько действительных корней имеет «{topic}», если {condition}?"
            return f"Что происходит с «{topic}», если {condition}?"

        # "Label: formula" facts are common in study notes. Prefer direct "value/formula" questions.
        label_formula = re.match(r"^([^:]{2,80}):\s*(.+?=\s*.+)$", self._normalize_text(content))
        if label_formula:
            label = self._normalize_topic(label_formula.group(1))
            base = label or topic
            if any(keyword in base for keyword in ("формула", "уравнение", "закон", "связь", "правило")):
                return f"Как записывается «{base}»?"
            return f"Как записывается формула для «{base}»?"
        formula_body = re.match(r"^(.+?)\s+имеет\s+вид\s+(.+?=\s*.+)$", self._normalize_text(content), flags=re.IGNORECASE)
        if formula_body:
            base = self._normalize_topic(formula_body.group(1)) or topic
            return f"Как записывается «{base}»?"

        if any(verb in lowered for verb in ("содержит", "включает", "состоит из")):
            return f"Что входит в «{topic}»?"
        if "имеет" in lowered:
            return f"Что есть у «{topic}»?"
        if "зависит от" in lowered:
            return f"От чего зависит «{topic}»?"
        if "связан" in lowered:
            return f"С чем связано «{topic}»?"
        if "делится на" in lowered:
            return f"На что делится «{topic}»?"
        if "функци" in lowered:
            return f"Какова функция «{topic}»?"
        if "происходит" in lowered or "приводит" in lowered:
            return f"Что происходит при «{topic}»?"
        if action_question:
            return action_question
        return f"Чем характеризуется «{topic}»?"

    def _question_from_relation(self, topic: str, content: str, comparison_topic: str, comparison_axis: str) -> str:
        lowered = self._normalize_text(content).lower()
        if comparison_topic and comparison_axis:
            return f"Чем отличаются «{comparison_topic}» по признаку «{comparison_axis}»?"
        if comparison_topic:
            return f"Чем отличаются «{comparison_topic}»?"
        if "зависит от" in lowered:
            return f"От чего зависит «{topic}»?"
        if "связан" in lowered:
            return f"С чем связано «{topic}»?"
        if "делится на" in lowered:
            return f"На что делится «{topic}»?"
        if "отличается" in lowered:
            return f"Чем отличается «{topic}»?"
        return f"Как связаны элементы темы «{topic}»?"

    def _paraphrase_answer(self, unit: KnowledgeUnit) -> str:
        topic = self._resolve_topic(unit.topic, unit.content, unit.source_fragment)
        tail = self._extract_tail(unit.content, topic)

        if unit.type in {KnowledgeType.DEFINITION, KnowledgeType.CONCEPT, KnowledgeType.TERM}:
            answer = tail
        elif unit.type == KnowledgeType.EXAMPLE:
            answer = re.sub(r"^(?:например|пример(?: для [^—-]+)?)\s*[—-]?\s*", "", tail, flags=re.IGNORECASE)
        elif unit.type == KnowledgeType.LIST_ITEM:
            answer = tail
            answer = re.sub(r"^(?:включает|состоит из|содержит)\s+", "", answer, flags=re.IGNORECASE)
        elif unit.type == KnowledgeType.PROCESS_STEP:
            answer = tail
        else:
            answer = tail

        answer = self._normalize_text(answer).strip(" ,;:")
        answer = self._make_answer_self_contained(unit, answer)
        answer = self._strip_answer_wrappers(unit, answer)
        if answer and answer[-1] not in ".!?":
            answer = f"{answer}."
        if answer and self._similarity(answer, unit.source_fragment) >= 0.92:
            answer = self._rephrase_close_answer(unit, answer)
            answer = self._normalize_text(answer).strip(" ,;:")
            answer = self._strip_answer_wrappers(unit, answer)
            if unit.type in {KnowledgeType.DEFINITION, KnowledgeType.TERM} and self._similarity(answer, unit.source_fragment) >= 0.9:
                answer = self._compact_definition_answer(answer)
                answer = self._normalize_text(answer).strip(" ,;:")
            if answer and answer[-1] not in ".!?":
                answer = f"{answer}."
        return self._truncate(answer, self.max_back_chars)

    def _capitalize(self, text: str) -> str:
        cleaned = self._normalize_text(text)
        if not cleaned:
            return cleaned
        return cleaned[0].upper() + cleaned[1:]

    def _looks_like_formula_answer(self, text: str) -> bool:
        normalized = self._normalize_text(text)
        if not normalized:
            return False
        if "=" in normalized or "≈" in normalized:
            return True
        if re.match(r"^[A-Za-zА-Яа-я][A-Za-zА-Яа-я0-9_()^]*\s*(?:/|\+|-|\*)", normalized):
            return True
        return False

    def _make_answer_self_contained(self, unit: KnowledgeUnit, answer: str) -> str:
        normalized = self._normalize_text(answer).strip(" ,;:.")
        if not normalized:
            return normalized

        lowered = normalized.lower()
        if unit.type in {KnowledgeType.DEFINITION, KnowledgeType.TERM} and not lowered.startswith("это "):
            normalized = f"Это {normalized[0].lower()}{normalized[1:]}"
        elif unit.type == KnowledgeType.CONCEPT:
            normalized = normalized
        elif unit.type == KnowledgeType.EXAMPLE and not lowered.startswith(("например", "один из примеров")):
            normalized = f"Например, {normalized[0].lower()}{normalized[1:]}"
        elif unit.type == KnowledgeType.LIST_ITEM and lowered.startswith(("включает ", "состоит из ", "содержит ")):
            normalized = re.sub(r"^(?:включает|состоит из|содержит)\s+", "", normalized, flags=re.IGNORECASE)
        elif unit.type == KnowledgeType.PROCESS_STEP:
            normalized = normalized

        if self._looks_like_formula_answer(normalized):
            return normalized
        return self._capitalize(normalized)

    def _rephrase_close_answer(self, unit: KnowledgeUnit, answer: str) -> str:
        normalized = self._normalize_text(answer).rstrip(".")
        lowered = normalized.lower()
        topic = self._resolve_topic(unit.topic, unit.content, unit.source_fragment)

        if unit.type in {KnowledgeType.DEFINITION, KnowledgeType.TERM} and not lowered.startswith("это "):
            return f"Это {normalized[0].lower()}{normalized[1:]}"
        if unit.type == KnowledgeType.CONCEPT:
            if lowered.startswith("вместо ") and "," in normalized:
                first_part, second_part = [part.strip(" ,;:.") for part in normalized.split(",", 1)]
                if second_part:
                    return f"{self._capitalize(second_part)} вместо {first_part}"
            return normalized
        if unit.type == KnowledgeType.EXAMPLE and not lowered.startswith("например"):
            return f"Например, {normalized[0].lower()}{normalized[1:]}"
        if unit.type in {KnowledgeType.FACT, KnowledgeType.RELATION, KnowledgeType.LIST_ITEM}:
            return normalized
        if unit.type == KnowledgeType.PROCESS_STEP:
            return normalized
        return normalized

    def _list_question(self, topic: str) -> str:
        return f"Что входит в «{topic}»?"

    def _question_from_unit(self, unit: KnowledgeUnit) -> str:
        topic = self._resolve_topic(unit.topic, unit.content, unit.source_fragment)
        comparison_topic = (
            self._comparison_topic_from_text(unit.content)
            or self._comparison_topic_from_text(unit.source_fragment)
            or (topic if " и " in topic else "")
        )
        comparison_axis = self._comparison_axis_from_text(unit.content) or self._comparison_axis_from_text(unit.source_fragment)
        lowered_content = self._normalize_text(unit.content).lower()
        if lowered_content.startswith("главная идея"):
            return f"Какова главная идея темы «{topic}»?"
        if unit.type in {KnowledgeType.DEFINITION, KnowledgeType.TERM}:
            if topic.startswith(("процесс ", "метод ", "этап ")):
                return f"Что означает термин «{topic}»?"
            return f"Что такое «{topic}»?"
        if unit.type == KnowledgeType.CONCEPT:
            return f"Какова суть «{topic}»?"
        if unit.type == KnowledgeType.FACT:
            return self._question_from_fact(topic, unit.content, comparison_topic, comparison_axis)
        if unit.type == KnowledgeType.RELATION:
            return self._question_from_relation(topic, unit.content, comparison_topic, comparison_axis)
        if unit.type == KnowledgeType.EXAMPLE:
            return f"Какой пример для «{topic}» приведён в конспекте?"
        if unit.type == KnowledgeType.LIST_ITEM:
            if comparison_topic and "по признакам" in lowered_content:
                return f"По каким признакам сравнивают «{comparison_topic}»?"
            return self._list_question(topic)
        if unit.type == KnowledgeType.PROCESS_STEP:
            marker = self._process_step_marker(unit.content)
            if marker:
                stage, _ = marker
                stage_lower = stage.lower()
                if stage_lower == "сначала":
                    return f"Что делают сначала при «{topic}»?"
                if stage_lower == "затем":
                    return f"Что делают затем при «{topic}»?"
                if stage_lower == "после этого":
                    return f"Что делают после этого при «{topic}»?"
                if stage_lower == "в конце":
                    return f"Что делают в конце при «{topic}»?"
            return f"Что происходит на шаге «{topic}»?"
        return f"Чем характеризуется «{topic}»?"

    def _normalize_flashcard(self, card: Flashcard) -> Flashcard | None:
        front = self._strip_prefixes(card.front)
        back = self._strip_prefixes(card.back)
        topic = self._resolve_topic(card.topic, card.front, card.back, card.source_fragment)
        source_fragment = self._normalize_text(card.source_fragment)

        if not front or not back or not topic:
            return None
        if not front.endswith("?"):
            front = front.rstrip(".!:;") + "?"
        front = self._truncate(front, self.max_front_chars)
        back = self._truncate(back, self.max_back_chars)

        if self._question_is_wrapper(front) or self._answer_has_wrapper(back):
            return None
        if self._back_is_too_close_to_source(front, back, source_fragment):
            return None
        return Flashcard(
            id=card.id,
            front=front,
            back=back,
            topic=topic,
            source_fragment=self._truncate(source_fragment, 220),
        )

    def _dedupe_flashcards(self, cards: List[Flashcard]) -> List[Flashcard]:
        unique: List[Flashcard] = []
        seen: list[tuple[str, str]] = []
        for card in cards:
            signature = (card.front.lower(), card.back.lower())
            if any(
                self._similarity(signature[0], existing_front) >= 0.9
                and self._similarity(signature[1], existing_back) >= 0.88
                for existing_front, existing_back in seen
            ):
                continue
            unique.append(card)
            seen.append(signature)
        return unique

    def _normalize_question(self, question: Question) -> Question | None:
        prompt = self._strip_prefixes(question.question)
        expected = self._strip_prefixes(question.expected_answer)
        topic = self._resolve_topic(question.topic, question.question, question.expected_answer, question.source_fragment)
        source_fragment = self._normalize_text(question.source_fragment)
        if not prompt or not expected or not topic:
            return None
        if not prompt.endswith("?"):
            prompt = prompt.rstrip(".!:;") + "?"
        prompt = self._truncate(prompt, self.max_front_chars)
        expected = self._truncate(expected, self.max_back_chars)
        if self._question_is_wrapper(prompt) or self._answer_has_wrapper(expected):
            return None
        return Question(
            id=question.id,
            question=prompt,
            expected_answer=expected,
            topic=topic,
            source_fragment=self._truncate(source_fragment, 220),
        )

    def _dedupe_questions(self, questions: List[Question]) -> List[Question]:
        unique: List[Question] = []
        seen: list[str] = []
        for question in questions:
            normalized = question.question.lower()
            if any(self._similarity(normalized, existing) >= 0.92 for existing in seen):
                continue
            unique.append(question)
            seen.append(normalized)
        return unique


class FlashcardGeneratorTool(_GenerationToolMixin):
    def __init__(self, llm_client: LLMClient | None = None, prompt_path: str | None = None) -> None:
        self.llm = llm_client
        self._prompt_template = Path(prompt_path).read_text(encoding="utf-8") if prompt_path else None

    def _build_flashcard(self, unit: KnowledgeUnit, idx: int) -> Flashcard | None:
        resolved_topic = self._resolve_topic(unit.topic, unit.content, unit.source_fragment)
        card = Flashcard(
            id=f"fc_{idx}",
            front=self._question_from_unit(unit),
            back=self._paraphrase_answer(unit),
            topic=resolved_topic,
            source_fragment=unit.source_fragment,
        )
        return self._normalize_flashcard(card)

    def _llm_cards(self, ctx: GenerationContext) -> List[Flashcard]:
        if not self.llm or not self._prompt_template:
            return []
        try:
            knowledge_units_json = json.dumps(
                [
                    {
                        "type": unit.type.value,
                        "topic": unit.topic,
                        "content": unit.content,
                        "source_fragment": unit.source_fragment,
                        "importance": unit.importance,
                        "confidence": unit.confidence,
                    }
                    for unit in ctx.knowledge_units
                ],
                ensure_ascii=False,
            )
            prompt = self._prompt_template.format(
                count=ctx.target_count,
                note_text=ctx.note_text,
                knowledge_units_json=knowledge_units_json,
                generation_rules_json=json.dumps(ctx.generation_rules, ensure_ascii=False),
            )
            raw = self.llm.generate(prompt, max_tokens=1400)
            items = self._extract_json_array(raw or "")
        except Exception:
            return []

        cards: List[Flashcard] = []
        for item in items:
            normalized = self._normalize_flashcard(
                Flashcard(
                    id=f"fc_{len(cards) + 1}",
                    front=str(item.get("front") or ""),
                    back=str(item.get("back") or ""),
                    topic=str(item.get("topic") or ""),
                    source_fragment=str(item.get("source_fragment") or ""),
                )
            )
            if normalized:
                cards.append(normalized)
            if len(cards) >= ctx.target_count:
                break
        return self._dedupe_flashcards(cards)

    def run(self, ctx: GenerationContext) -> List[Flashcard]:
        units = ctx.knowledge_units or []
        if not units:
            return []
        cards = self._llm_cards(ctx)
        used_topics = {card.topic for card in cards}

        for unit in units:
            if len(cards) >= ctx.target_count:
                break
            candidate = self._build_flashcard(unit, len(cards) + 1)
            if not candidate:
                continue
            if candidate.topic in used_topics and any(self._similarity(candidate.front, existing.front) >= 0.88 for existing in cards):
                continue
            cards.append(candidate)
            used_topics.add(candidate.topic)

        return self._dedupe_flashcards(cards)[: ctx.target_count]


class RandomQuestionGeneratorTool(_GenerationToolMixin):
    def __init__(self) -> None:
        pass

    def _build_question(self, unit: KnowledgeUnit, idx: int) -> Question | None:
        resolved_topic = self._resolve_topic(unit.topic, unit.content, unit.source_fragment)
        question = Question(
            id=f"q_{idx}",
            question=self._question_from_unit(unit),
            expected_answer=self._paraphrase_answer(unit),
            topic=resolved_topic,
            source_fragment=unit.source_fragment,
        )
        return self._normalize_question(question)

    def run(self, ctx: GenerationContext) -> List[Question]:
        units = ctx.knowledge_units or []
        if not units:
            return []
        questions: List[Question] = []
        for unit in units:
            if len(questions) >= ctx.target_count:
                break
            candidate = self._build_question(unit, len(questions) + 1)
            if not candidate:
                continue
            questions.append(candidate)
        return self._dedupe_questions(questions)[: ctx.target_count]
