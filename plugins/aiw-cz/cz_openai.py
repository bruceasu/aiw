"""Small standard-library OpenAI Responses API client for aiw-cz."""

from __future__ import annotations

import json
import os
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

from cz_config import ProviderConfig


class OpenAIUnavailable(RuntimeError):
    pass


SCHEMA = {
    "type": "object", "additionalProperties": False,
    "properties": {"candidates": {"type": "array", "items": {
        "type": "object", "additionalProperties": False,
        "properties": {name: {"type": "string"} for name in ("type", "scope", "subject", "body", "breaking", "footer")},
        "required": ["type", "scope", "subject", "body", "breaking", "footer"],
    }}},
    "required": ["candidates"],
}


def _text(response: dict) -> str:
    if isinstance(response.get("output_text"), str):
        return response["output_text"]
    for item in response.get("output", []):
        for content in item.get("content", []) if isinstance(item, dict) else []:
            if isinstance(content, dict) and isinstance(content.get("text"), str):
                return content["text"]
    raise OpenAIUnavailable("OpenAI response contains no text")


def generate(prompt: str, config: ProviderConfig, timeout: float = 60.0) -> str:
    api_key = config.api_key or os.environ.get("OPENAI_API_KEY", "")
    if not api_key:
        raise OpenAIUnavailable("OpenAI API key is not configured")
    if not config.model:
        raise OpenAIUnavailable("OpenAI model is not configured")
    base_url = (config.base_url or os.environ.get("CZ_OPENAI_BASE_URL") or "https://api.openai.com/v1").rstrip("/")
    payload = {"model": config.model, "input": prompt, "text": {"format": {
        "type": "json_schema", "name": "commit_candidates", "strict": True, "schema": SCHEMA,
    }}}
    request = Request(
        f"{base_url}/responses", data=json.dumps(payload).encode("utf-8"),
        headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"}, method="POST",
    )
    try:
        with urlopen(request, timeout=timeout) as response:
            body = json.loads(response.read().decode("utf-8"))
    except HTTPError as exc:
        raise OpenAIUnavailable(f"OpenAI HTTP {exc.code}") from exc
    except (URLError, TimeoutError, json.JSONDecodeError) as exc:
        raise OpenAIUnavailable(f"OpenAI request failed: {exc}") from exc
    return _text(body)
