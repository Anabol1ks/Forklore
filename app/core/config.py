import os
from pydantic import BaseModel
from dotenv import load_dotenv

# Load local `.env` in dev runs (does not override real env vars).
load_dotenv()

class Settings(BaseModel):
    # Which backend to use for LLM calls. Supported: "gigachat", "yandex", "openai".
    llm_provider: str = os.getenv("LLM_PROVIDER", "gigachat")

    # Shared
    llm_model: str = os.getenv("LLM_MODEL", "GigaChat")

    # OpenAI
    llm_api_key: str | None = os.getenv("OPENAI_API_KEY")

    # GigaChat (expects a ready-to-use access token)
    gigachat_access_token: str | None = os.getenv("GIGACHAT_ACCESS_TOKEN")
    gigachat_base_url: str = os.getenv("GIGACHAT_BASE_URL", "https://gigachat.devices.sberbank.ru")
    gigachat_verify_ssl: bool = os.getenv("GIGACHAT_VERIFY_SSL", "true").lower() in ("1", "true", "yes", "on")
    gigachat_timeout_s: float = float(os.getenv("GIGACHAT_TIMEOUT_S", "30"))

    # YandexGPT / Yandex Cloud Foundation Models
    # Auth: prefer API key for a service account, otherwise an IAM token.
    yandex_api_key: str | None = os.getenv("YANDEX_API_KEY")
    yandex_iam_token: str | None = os.getenv("YANDEX_IAM_TOKEN")
    yandex_folder_id: str | None = os.getenv("YANDEX_FOLDER_ID")
    yandex_base_url: str = os.getenv("YANDEX_BASE_URL", "https://llm.api.cloud.yandex.net")
    yandex_verify_ssl: bool = os.getenv("YANDEX_VERIFY_SSL", "true").lower() in ("1", "true", "yes", "on")
    yandex_timeout_s: float = float(os.getenv("YANDEX_TIMEOUT_S", "30"))

    min_note_length: int = 30

settings = Settings()
