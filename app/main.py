from fastapi import FastAPI
from app.api.routes.generate import router as generate_router

app = FastAPI(title="Forklore Study Service", version="0.1.0")
app.include_router(generate_router)


@app.get("/")
def health() -> dict:
    return {"status": "ok"}


@app.get("/healthz")
def healthz() -> dict:
    return {"status": "ok"}
