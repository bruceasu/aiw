"""Configuration and locale loading for the Python cz plugin."""

from __future__ import annotations

import locale
import os
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


SUPPORTED_LOCALES = ("en", "zh", "ja")


@dataclass
class ProviderConfig:
    model: str = ""
    api_key: str = ""
    base_url: str = ""


@dataclass
class Choice:
    value: str
    name: str


@dataclass
class Config:
    llm: bool = False
    provider: str = ""
    candidates: int = 3
    editor: str = ""
    max_subject_length: int = 72
    language: str = "en"
    messages: dict[str, str] = field(default_factory=dict)
    types: list[Choice] = field(default_factory=list)
    scopes: list[Choice] = field(default_factory=list)
    providers: dict[str, ProviderConfig] = field(
        default_factory=lambda: {name: ProviderConfig() for name in ("codex", "copilot", "openai")}
    )


def _scalar(value: str) -> Any:
    value = value.strip()
    if value.lower() in ("true", "false"):
        return value.lower() == "true"
    if value.startswith('"') and value.endswith('"'):
        return value[1:-1].replace('\\"', '"').replace("\\n", "\n")
    if value.startswith("'") and value.endswith("'"):
        return value[1:-1]
    try:
        return int(value)
    except ValueError:
        return value


def _strip_comment(raw: str) -> str:
    quoted = False
    quote = ""
    escaped = False
    for index, char in enumerate(raw):
        if escaped:
            escaped = False
            continue
        if quoted and char == "\\" and quote == '"':
            escaped = True
            continue
        if char in ('"', "'"):
            if not quoted:
                quoted, quote = True, char
            elif quote == char:
                quoted = False
            continue
        if char == "#" and not quoted:
            return raw[:index]
    return raw


def _toml(path: Path) -> dict[str, Any]:
    """Read the subset needed by cz without adding a TOML dependency."""
    result: dict[str, Any] = {}
    section: dict[str, Any] = result
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = _strip_comment(raw).strip()
        if not line:
            continue
        if line.startswith("[[") and line.endswith("]]" ):
            name = line[2:-2].strip()
            rows = result.setdefault(name, [])
            if not isinstance(rows, list):
                raise ValueError(f"invalid TOML table: {name}")
            section = {}
            rows.append(section)
            continue
        if line.startswith("[") and line.endswith("]"):
            section = result
            for part in line[1:-1].split("."):
                section = section.setdefault(part.strip(), {})
            continue
        if "=" in line:
            key, value = line.split("=", 1)
            section[key.strip()] = _scalar(value)
    return result


def _first_existing(*paths: Path | None) -> Path | None:
    return next((path for path in paths if path and path.is_file()), None)


def _locale_name(raw: str) -> str:
    return (re.split(r"[.@_-]", raw.lower(), maxsplit=1)[0] or "en")


def _system_locale() -> str:
    raw = os.environ.get("LC_ALL") or os.environ.get("LANG") or ""
    if not raw:
        try:
            raw = locale.getlocale()[0] or ""
        except ValueError:
            raw = ""
    return _locale_name(raw)


def _choices(rows: Any) -> list[Choice]:
    if not isinstance(rows, list):
        return []
    return [Choice(str(row["value"]), str(row.get("name", row["value"])))
            for row in rows if isinstance(row, dict) and row.get("value")]


def _merge(config: Config, data: dict[str, Any]) -> None:
    cz = data.get("cz", {})
    if not isinstance(cz, dict):
        return
    if isinstance(cz.get("llm"), bool):
        config.llm = cz["llm"]
    if isinstance(cz.get("use_llm"), bool):
        config.llm = cz["use_llm"]
    provider = cz.get("provider", cz.get("llm_provider"))
    if isinstance(provider, str):
        config.provider = "" if provider == "auto" else provider
    if isinstance(cz.get("candidates"), int) and cz["candidates"] > 0:
        config.candidates = cz["candidates"]
    if isinstance(cz.get("editor"), str):
        config.editor = cz["editor"]
    if isinstance(cz.get("max_subject_length"), int) and cz["max_subject_length"] > 0:
        config.max_subject_length = cz["max_subject_length"]
    config.types = _choices(cz.get("types")) or config.types
    config.scopes = _choices(cz.get("scopes")) or config.scopes
    messages = cz.get("messages")
    if isinstance(messages, dict):
        config.messages.update({str(k): str(v) for k, v in messages.items()})
    for name in config.providers:
        section = cz.get(name, {})
        if not isinstance(section, dict):
            continue
        target = config.providers[name]
        for key in ("model", "api_key", "base_url"):
            if isinstance(section.get(key), str):
                setattr(target, key, section[key])


def load_config(plugin_dir: Path, requested_language: str = "") -> Config:
    config = Config()
    root = Path.cwd()
    candidates = [plugin_dir / "cz.toml", plugin_dir / ".cz.toml"]
    if os.environ.get("AIW_ROOT"):
        aiw_root = Path(os.environ["AIW_ROOT"])
        candidates.extend((aiw_root / "aiw.toml", aiw_root / ".aiw.toml"))
    candidates.extend((root / "aiw.toml", root / ".aiw.toml"))
    for path in candidates:
        if path.is_file():
            _merge(config, _toml(path))

    explicit_language = requested_language or os.environ.get("CZ_LANGUAGE", "")
    if not explicit_language:
        for path in candidates:
            if path.is_file():
                data = _toml(path)
                i18n = data.get("i18n", {})
                if isinstance(i18n, dict) and isinstance(i18n.get("default_language"), str):
                    explicit_language = i18n["default_language"]
                    break
    language = explicit_language or _system_locale()
    language = _locale_name(language)
    if language not in SUPPORTED_LOCALES or not (plugin_dir / "locales" / f"{language}.toml").is_file():
        language = "en"
    locale_data = _toml(plugin_dir / "locales" / f"{language}.toml")
    messages = locale_data.get("messages", {})
    if isinstance(messages, dict):
        merged = {str(k): str(v) for k, v in messages.items()}
        merged.update(config.messages)
        config.messages = merged
    config.types = config.types or _choices(locale_data.get("types"))
    config.scopes = config.scopes or _choices(locale_data.get("scopes"))
    config.language = language
    for name in config.providers:
        prefix = f"CZ_{name.upper()}_"
        target = config.providers[name]
        target.model = os.environ.get(prefix + "MODEL", target.model)
        target.api_key = os.environ.get(prefix + "API_KEY", target.api_key)
        target.base_url = os.environ.get(prefix + "BASE_URL", target.base_url)
    config.provider = os.environ.get("CZ_LLM_PROVIDER", config.provider)
    if not config.types:
        raise ValueError("cz requires at least one commit type")
    return config
