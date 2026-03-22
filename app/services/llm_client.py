import hashlib
import json
import random
import ssl
import urllib.error
import urllib.request
from typing import List

from app.core.config import settings

try:
    from openai import OpenAI  # type: ignore
except ImportError:  # pragma: no cover - openai optional
    OpenAI = None


class _GigaChatHTTPClient:
    """
    Minimal HTTP client for GigaChat OpenAI-like chat/completions endpoint.

    This expects a ready access token in `GIGACHAT_ACCESS_TOKEN`.
    """

    def __init__(self, access_token: str, base_url: str, verify_ssl: bool, timeout_s: float) -> None:
        self._access_token = access_token
        self._base_url = base_url.rstrip("/")
        self._timeout_s = timeout_s
        self._ssl_context = ssl.create_default_context() if verify_ssl else ssl._create_unverified_context()  # noqa: S501

    def chat_completions(self, *, model: str, prompt: str, max_tokens: int) -> str:
        url = f"{self._base_url}/api/v1/chat/completions"
        body = {
            "model": model,
            "messages": [{"role": "user", "content": prompt}],
            "max_tokens": max_tokens,
        }
        req = urllib.request.Request(
            url=url,
            method="POST",
            data=json.dumps(body).encode("utf-8"),
            headers={
                "Content-Type": "application/json",
                "Accept": "application/json",
                "Authorization": f"Bearer {self._access_token}",
            },
        )

        try:
            with urllib.request.urlopen(req, timeout=self._timeout_s, context=self._ssl_context) as resp:
                raw = resp.read().decode("utf-8")
        except urllib.error.HTTPError as e:
            detail = e.read().decode("utf-8", errors="replace")
            raise RuntimeError(f"GigaChat HTTP {e.code}: {detail}") from e
        except Exception as e:  # pragma: no cover
            raise RuntimeError(f"GigaChat request failed: {e}") from e

        data = json.loads(raw or "{}")
        # Try to follow OpenAI-style response first.
        try:
            return data["choices"][0]["message"]["content"]
        except Exception:
            # Fallback: tolerate variants
            content = (
                (data.get("choices") or [{}])[0].get("message", {}).get("content")
                or (data.get("choices") or [{}])[0].get("text")
            )
            if not content:
                raise RuntimeError(f"Unexpected GigaChat response: {data}")
            return content


class _YandexGPTHTTPClient:
    """
    Minimal HTTP client for Yandex Cloud Foundation Models (YandexGPT).

    Supports either:
    - API key auth: Authorization: Api-Key <YANDEX_API_KEY>
    - IAM token auth: Authorization: Bearer <YANDEX_IAM_TOKEN>
    """

    def __init__(
        self,
        *,
        api_key: str | None,
        iam_token: str | None,
        folder_id: str,
        base_url: str,
        verify_ssl: bool,
        timeout_s: float,
        model: str,
    ) -> None:
        self._api_key = (api_key or "").strip()
        self._iam_token = (iam_token or "").strip()
        self._folder_id = folder_id.strip()
        self._base_url = base_url.rstrip("/")
        self._timeout_s = timeout_s
        self._model = (model or "").strip()
        self._ssl_context = ssl.create_default_context() if verify_ssl else ssl._create_unverified_context()  # noqa: S501

    def chat_completions(self, *, prompt: str, max_tokens: int) -> str:
        # Docs: https://llm.api.cloud.yandex.net/foundationModels/v1/completion
        url = f"{self._base_url}/foundationModels/v1/completion"

        model_uri = self._model
        if not model_uri.startswith(("gpt://", "ds://")):
            # Common pattern: gpt://<folder_id>/<model_name>
            model_uri = f"gpt://{self._folder_id}/{model_uri}"

        body = {
            "modelUri": model_uri,
            "completionOptions": {
                "stream": False,
                # Lower temp helps JSON-only prompts.
                "temperature": 0.2,
                # API expects a string in many examples; accept int here.
                "maxTokens": str(max_tokens),
            },
            "messages": [{"role": "user", "text": prompt}],
        }

        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json",
        }
        if self._api_key:
            headers["Authorization"] = f"Api-Key {self._api_key}"
        elif self._iam_token:
            headers["Authorization"] = f"Bearer {self._iam_token}"
            # For IAM token auth, Yandex examples commonly include the folder header.
            headers["x-folder-id"] = self._folder_id
        else:
            raise RuntimeError("YandexGPT auth missing: set YANDEX_API_KEY or YANDEX_IAM_TOKEN")

        req = urllib.request.Request(
            url=url,
            method="POST",
            data=json.dumps(body).encode("utf-8"),
            headers=headers,
        )

        try:
            with urllib.request.urlopen(req, timeout=self._timeout_s, context=self._ssl_context) as resp:
                raw = resp.read().decode("utf-8")
        except urllib.error.HTTPError as e:
            detail = e.read().decode("utf-8", errors="replace")
            raise RuntimeError(f"YandexGPT HTTP {e.code}: {detail}") from e
        except Exception as e:  # pragma: no cover
            raise RuntimeError(f"YandexGPT request failed: {e}") from e

        data = json.loads(raw or "{}")
        # Expected: result.alternatives[0].message.text
        try:
            return data["result"]["alternatives"][0]["message"]["text"]
        except Exception:
            # Fallbacks: tolerate minor shape differences
            try:
                return data["result"]["alternatives"][0]["text"]
            except Exception:
                raise RuntimeError(f"Unexpected YandexGPT response: {data}") from None


class LLMClient:
    """Minimal LLM client with deterministic fallback for tests."""

    def __init__(self) -> None:
        self.provider = (settings.llm_provider or "").strip().lower()
        self.model = settings.llm_model

        self.client = None
        if self.provider == "gigachat":
            token = settings.gigachat_access_token
            if token:
                self.client = _GigaChatHTTPClient(
                    access_token=token,
                    base_url=settings.gigachat_base_url,
                    verify_ssl=settings.gigachat_verify_ssl,
                    timeout_s=settings.gigachat_timeout_s,
                )
        elif self.provider in {"yandex", "yandexgpt"}:
            folder_id = (settings.yandex_folder_id or "").strip()
            if folder_id and (settings.yandex_api_key or settings.yandex_iam_token):
                self.client = _YandexGPTHTTPClient(
                    api_key=settings.yandex_api_key,
                    iam_token=settings.yandex_iam_token,
                    folder_id=folder_id,
                    base_url=settings.yandex_base_url,
                    verify_ssl=settings.yandex_verify_ssl,
                    timeout_s=settings.yandex_timeout_s,
                    model=self.model or "yandexgpt",
                )
        elif self.provider == "openai":
            api_key = settings.llm_api_key
            if api_key and OpenAI:
                self.client = OpenAI(api_key=api_key)

    def generate(self, prompt: str, max_tokens: int = 256) -> str:
        if self.client and self.provider == "openai":
            response = self.client.chat.completions.create(  # type: ignore[union-attr]
                model=self.model,
                messages=[{"role": "user", "content": prompt}],
                max_tokens=max_tokens,
            )
            return response.choices[0].message.content  # type: ignore[return-value]
        if self.client and self.provider == "gigachat":
            return self.client.chat_completions(model=self.model, prompt=prompt, max_tokens=max_tokens)  # type: ignore[union-attr]
        if self.client and self.provider in {"yandex", "yandexgpt"}:
            return self.client.chat_completions(prompt=prompt, max_tokens=max_tokens)  # type: ignore[union-attr]
        # deterministic fallback: sample sentences by hashing prompt
        # return self._fallback(prompt)

    # def _fallback(self, prompt: str) -> str:
    #     seed = int(hashlib.sha256(prompt.encode()).hexdigest(), 16) % (2**32)
    #     random.seed(seed)
    #     phrases = [
    #         "???????? ??????: ?????????",
    #         "???????????: ??????? ???????? ????",
    #         "????: ?????? ???????????",
    #         "?????: ???????? ??????? ???? ?? ?????",
    #         "??????: ?????? ??? ??????",
    #         "?????: ?????? ??? ?????? ?? ??????????",
    #     ]
    #     return "\n".join(random.sample(phrases, k=min(4, len(phrases))))
