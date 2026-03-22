from typing import List
from uuid import UUID
from elasticsearch import Elasticsearch, AsyncElasticsearch
from elasticsearch_dsl import Search
from models import SearchHit, SearchRequest, TutorIndexItem
from fastapi import Depends
from extractor import ParamExtractor

class TutorSearchService:
    def __init__(self, es_client: AsyncElasticsearch, index_name: str, ex: ParamExtractor):
        self.es = es_client
        self.index = index_name
        self.ex = ex

    async def index_tutor(self, tutor: TutorIndexItem) -> str:
        doc = tutor.dict()
        # Используем tutor_id как _id
        resp = await self.es.index(index=self.index, id=str(tutor.tutor_id), document=doc)
        return resp["result"]

    async def delete_tutor(self, tutor_id: UUID) -> str:
        try:
            resp = await self.es.delete(index=self.index, id=str(tutor_id), ignore=[404])
        except Exception as e:
            raise
        return resp.get("result", "not_found")

    async def search(self, req: SearchRequest) -> tuple[List[SearchHit], int]:
        # Извлекаем параметры как словарь
        extracted = self.ex.extract(req.raw_query)

        must_clauses = []
        if req.raw_query:
            must_clauses.append({
                "multi_match": {
                    "query": req.raw_query,
                    "fields": ["full_name^3", "bio", "subjects", "topics"],
                    "type": "best_fields",
                    "fuzziness": "AUTO"
                }
            })

        filters = []
        if extracted.get("university"):
            filters.append({"term": {"university": extracted["university"]}})
        if extracted.get("course"):
            filters.append({"term": {"course": extracted["course"]}})
        if extracted.get("subjects"):
            filters.append({"terms": {"subjects": extracted["subjects"]}})
        if extracted.get("topics"):
            filters.append({"terms": {"topics": extracted["topics"]}})

        body = {
            "query": {
                "bool": {
                    "must": must_clauses,
                    "filter": filters
                }
            },
            "from": req.offset,
            "size": req.limit,
            "sort": [{"_score": {"order": "desc"}}]
        }

        try:
            resp = await self.es.search(index=self.index, body=body)
        except Exception as e:
            # Логируйте ошибку
            raise

        hits = []
        for hit in resp["hits"]["hits"]:
            src = hit["_source"]
            snippet = src.get("bio")[:200] if src.get("bio") else None
            hits.append(SearchHit(
                tutor_id=UUID(src["tutor_id"]),
                full_name=src["full_name"],
                university=src.get("university"),
                course=src.get("course"),
                subjects=src.get("subjects", []),
                topics=src.get("topics", []),
                snippet=snippet,
                rank=hit["_score"] if hit["_score"] is not None else 0.0
            ))
        total = resp["hits"]["total"]["value"]
        return hits, total