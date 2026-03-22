import os
import logging
from uuid import UUID
from contextlib import asynccontextmanager
from fastapi import FastAPI, HTTPException
from elasticsearch import AsyncElasticsearch
from dotenv import load_dotenv

from dictionary_loader import DictionaryLoader
from extractor import ParamExtractor
from service import TutorSearchService
from models import SearchRequest, SearchResponse, TutorIndexItem

load_dotenv()
ELASTICSEARCH_URL = os.getenv("ELASTICSEARCH_URL", "http://localhost:9200")
INDEX_NAME = os.getenv("INDEX_NAME", "tutors")

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


es_client = AsyncElasticsearch(ELASTICSEARCH_URL)

@asynccontextmanager
async def lifespan(app: FastAPI):
    try:
        await es_client.info()
        logger.info("Connected to Elasticsearch")
        await create_index_if_not_exists(es_client, INDEX_NAME)
    except Exception as e:
        logger.error(f"Failed to connect to Elasticsearch: {e}")
        raise

    # Загружаем словари из Go (или статику)
    dict_loader = DictionaryLoader()
    await dict_loader.load()
    extractor = ParamExtractor(dict_loader)
    search_service = TutorSearchService(es_client, INDEX_NAME, extractor)
    app.state.search_service = search_service
    yield
    # Закрываем клиент при завершении
    await es_client.close()

app = FastAPI(lifespan=lifespan)

async def create_index_if_not_exists(es: AsyncElasticsearch, index_name: str):
    if await es.indices.exists(index=index_name):
        logger.info(f"Index {index_name} already exists")
        return

    mapping = {
        "mappings": {
            "properties": {
                "tutor_id": {"type": "keyword"},
                "full_name": {"type": "text", "analyzer": "russian"},
                "bio": {"type": "text", "analyzer": "russian"},
                "university": {"type": "keyword"},
                "course": {"type": "keyword"},
                "subjects": {"type": "keyword"},
                "topics": {"type": "keyword"},
                "is_active": {"type": "boolean"},
                "updated_at": {"type": "date"}
            }
        }
    }
    await es.indices.create(index=index_name, body=mapping)
    logger.info(f"Index {index_name} created")

@app.post("/tutors", response_model=dict)
async def index_tutor(tutor: TutorIndexItem):
    search_service = app.state.search_service
    try:
        result = await search_service.index_tutor(tutor)
    except Exception as e:
        logger.error(f"Index error: {e}")
        raise HTTPException(status_code=500, detail="Indexing failed")
    return {"result": result, "id": str(tutor.tutor_id)}

@app.delete("/tutors/{tutor_id}", response_model=dict)
async def delete_tutor(tutor_id: UUID):
    search_service = app.state.search_service
    try:
        result = await search_service.delete_tutor(tutor_id)
    except Exception as e:
        logger.error(f"Delete error: {e}")
        raise HTTPException(status_code=500, detail="Deletion failed")
    return {"result": result, "id": str(tutor_id)}

@app.post("/search", response_model=SearchResponse)
async def search(req: SearchRequest):
    search_service = app.state.search_service
    try:
        hits, total = await search_service.search(req)
    except Exception as e:
        import traceback
        traceback.print_exc()
        raise HTTPException(status_code=500, detail=str(e))
    return SearchResponse(total=total, results=hits)