import re
import difflib
import pymorphy3
from typing import List, Optional
from models import SearchParamsData
from dictionary_loader import DictionaryLoader

class ParamExtractor:
    def __init__(self, dict_loader: DictionaryLoader):
        self.dict_loader = dict_loader
        self.morph = pymorphy3.MorphAnalyzer()
        self.university_names = dict_loader.get_all_university_names()
        self.subject_names = dict_loader.get_all_subject_names()
        self.topic_names = dict_loader.get_all_topic_names()
        # Слова-числительные для курса
        self.course_words = {
            "первый": "1",
            "второй": "2",
            "третий": "3",
            "четвертый": "4",
            "пятый": "5",
            "шестой": "6",
        }

    def _normalize(self, word: str) -> str:
        """Лемматизация слова."""
        return self.morph.parse(word)[0].normal_form

    def _find_closest(self, word: str, candidates: List[str], cutoff: float = 0.85) -> Optional[str]:
        """
        Находит ближайшее совпадение в списке candidates.
        Использует difflib.get_close_matches.
        """
        matches = difflib.get_close_matches(word, candidates, n=1, cutoff=cutoff)
        return matches[0] if matches else None

    def extract(self, raw_query: str) -> dict:
        words = re.findall(r'\b[а-яё]+\b', raw_query.lower())
        lemmas = [self._normalize(w) for w in words]

        # 1. Университет
        university = None
        for lemma in lemmas:
            matched = self._find_closest(lemma, self.university_names)
            if matched:
                # Найти каноническое название (университет может быть найден по синониму)
                for uni in self.dict_loader.universities:
                    if matched == uni.name or matched in uni.synonyms:
                        university = uni.name
                        break
                if university:
                    break

        # 2. Курс
        course = None
        # Поиск по числам
        for pattern in [r'\b([1-6])\b', r'\b([1-6])\s*(?:-й\s*)?(?:курс[ае]?|год[ау]?)\b']:
            match = re.search(pattern, raw_query.lower())
            if match:
                course = match.group(1)
                break
        # Если не нашли по цифрам, ищем по словам-числительным
        if not course:
            for lemma in lemmas:
                if lemma in self.course_words:
                    course = self.course_words[lemma]
                    break

        # 3. Предметы
        subjects = []
        used_lemmas = set()
        for lemma in lemmas:
            if lemma in used_lemmas:
                continue
            matched = self._find_closest(lemma, self.subject_names)
            if matched:
                # Найти каноническое название предмета
                for subj in self.dict_loader.subjects:
                    if matched == subj.name or matched in subj.synonyms:
                        subjects.append(subj.name)
                        used_lemmas.add(lemma)
                        break

        # 4. Темы
        topics = []
        for lemma in lemmas:
            if lemma in used_lemmas:
                continue
            matched = self._find_closest(lemma, self.topic_names)
            if matched:
                # Найти каноническое название темы (можно просто добавить matched)
                topics.append(matched)
                used_lemmas.add(lemma)

        return {
            "university": university,
            "course": course,
            "subjects": subjects,
            "topics": topics,
        }

    def extract_search_params(self, raw_query: str) -> SearchParamsData:
        extracted = self.extract(raw_query)
        return SearchParamsData(
            query=raw_query,
            university=extracted["university"],
            course=extracted["course"],
            subjects=extracted["subjects"],
            topics=extracted["topics"],
            limit=20,
            offset=0
        )