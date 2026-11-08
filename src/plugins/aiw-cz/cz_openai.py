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


def _text(response: object, label: str = "OpenAI") -> str:
    if not isinstance(response, dict):
        raise OpenAIUnavailable(f"{label} response is not a JSON object")
    if isinstance(response.get("output_text"), str) and response["output_text"].strip():
        return response["output_text"]
    output = response.get("output", [])
    if not isinstance(output, list):
        raise OpenAIUnavailable(f"{label} response has invalid output")
    texts: list[str] = []
    for item in output:
        if not isinstance(item, dict) or item.get("type") != "message":
            continue
        content_items = item.get("content", [])
        if not isinstance(content_items, list):
            continue
        for content in content_items:
            if (isinstance(content, dict)
                    and content.get("type") == "output_text"
                    and isinstance(content.get("text"), str)):
                texts.append(content["text"])
    if texts:
        return "".join(texts)
    raise OpenAIUnavailable(f"{label} response contains no final message text")


def _generate(prompt: str, config: ProviderConfig, timeout: float, structured: bool, provider: str = "openai") -> str:
    deepseek = provider == "deepseek"
    label = "DeepSeek" if deepseek else "OpenAI"
    api_key = config.api_key if deepseek else config.api_key or os.environ.get("OPENAI_API_KEY", "")
    if not api_key:
        raise OpenAIUnavailable(f"{label} API key is not configured")
    if not config.model:
        raise OpenAIUnavailable(f"{label} model is not configured")
    if deepseek:
        default_base, env_base = "https://api.deepseek.com", ""
    else:
        default_base, env_base = "https://api.openai.com/v1", "OPENAI_BASE_URL"
    base_url = (config.base_url or (os.environ.get(env_base) if env_base else "") or default_base).rstrip("/")
    payload = {"model": config.model, "input": prompt}
    if structured:
        payload["text"] = {"format": {
            "type": "json_schema", "name": "commit_candidates", "strict": True, "schema": SCHEMA,
        }}
    request = Request(
        f"{base_url}/responses", data=json.dumps(payload).encode("utf-8"),
        headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"}, method="POST",
    )
    try:
        with urlopen(request, timeout=timeout) as response:
            body = json.loads(response.read().decode("utf-8"))
    except HTTPError as exc:
        raise OpenAIUnavailable(f"{label} HTTP {exc.code}") from exc
    except (URLError, TimeoutError, json.JSONDecodeError) as exc:
        raise OpenAIUnavailable(f"{label} request failed: {exc}") from exc
    return _text(body, label)


def generate(prompt: str, config: ProviderConfig, timeout: float = 60.0) -> str:
    return _generate(prompt, config, timeout, structured=True)


def generate_text(prompt: str, config: ProviderConfig, timeout: float = 60.0) -> str:
    return _generate(prompt, config, timeout, structured=False)


def generate_deepseek(prompt: str, config: ProviderConfig, timeout: float = 60.0) -> str:
    return _generate(prompt, config, timeout, structured=True, provider="deepseek")


def generate_text_deepseek(prompt: str, config: ProviderConfig, timeout: float = 60.0) -> str:
    return _generate(prompt, config, timeout, structured=False, provider="deepseek")
