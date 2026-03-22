FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    STUDY_SERVICE_PORT=8085

WORKDIR /srv

COPY study-service/requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

COPY app ./app

RUN useradd --create-home --shell /bin/bash appuser && \
    chown -R appuser:appuser /srv

USER appuser

EXPOSE 8085

CMD ["sh", "-c", "uvicorn app.main:app --host 0.0.0.0 --port ${STUDY_SERVICE_PORT}"]
