"""Configuration and locale loading for the Python cz plugin."""

from __future__ import annotations

import locale
import os
import re
import sys
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
    provider_explicit: bool = False
    global_provider: str = ""
    global_generic: ProviderConfig = field(default_factory=ProviderConfig)
    candidates: int = 3
    editor: str = ""
    max_subject_length: int = 72
    language: str = "en"
    messages: dict[str, str] = field(default_factory=dict)
    types: list[Choice] = field(default_factory=list)
    scopes: list[Choice] = field(default_factory=list)
    providers: dict[str, ProviderConfig] = field(
        default_factory=lambda: {name: ProviderConfig() for name in ("codex", "copilot", "openai", "deepseek")}
    )
    global_providers: dict[str, ProviderConfig] = field(
        default_factory=lambda: {name: ProviderConfig() for name in ("codex", "copilot", "openai", "deepseek")}
    )
    provider_fields: dict[str, set[str]] = field(
        default_factory=lambda: {name: set() for name in ("codex", "copilot", "openai", "deepseek")}
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
    if not raw and sys.platform == "win32":
        import ctypes

        buffer = ctypes.create_unicode_buffer(85)
        if ctypes.windll.kernel32.GetUserDefaultLocaleName(buffer, len(buffer)):
            raw = buffer.value
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
    ai = data.get("ai", {})
    if isinstance(ai, dict):
        provider = ai.get("provider", ai.get("llm_provider"))
        if isinstance(provider, str):
            config.global_provider = "" if provider.lower() == "auto" else provider
        for key in ("model", "base_url", "api_key"):
            value = ai.get(key)
            if isinstance(value, str):
                setattr(config.global_generic, key, value)
        prefixes = {"codex": "openai_codex", "copilot": "copilot", "openai": "openai", "deepseek": "deepseek"}
        for name, prefix in prefixes.items():
            target = config.global_providers[name]
            for key in ("model", "base_url", "api_key"):
                value = ai.get(f"{prefix}_{key}")
                if isinstance(value, str):
                    setattr(target, key, value)

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
        config.provider_explicit = True
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
                config.provider_fields[name].add(key)


def load_config(plugin_dir: Path, requested_language: str = "") -> Config:
    config = Config()
    root = Path.cwd()
    candidates = [_first_existing(plugin_dir / "cz.toml", plugin_dir / ".cz.toml")]
    if os.environ.get("AIW_ROOT"):
        aiw_root = Path(os.environ["AIW_ROOT"])
        candidates.append(_first_existing(aiw_root / "aiw.toml", aiw_root / ".aiw.toml"))
    candidates.append(_first_existing(root / "aiw.toml", root / ".aiw.toml"))
    for path in candidates:
        if path:
            _merge(config, _toml(path))

    provider_env = os.environ.get("AIW_LLM_PROVIDER", "").strip()
    global_provider = provider_env or config.global_provider
    use_global_generic = not provider_env or not config.global_provider or provider_env.lower() == config.global_provider.lower()
    if not config.provider_explicit:
        config.provider = "" if global_provider.lower() == "auto" else global_provider
    if "CZ_LLM_PROVIDER" in os.environ:
        value = os.environ["CZ_LLM_PROVIDER"].strip()
        config.provider = "" if value.lower() == "auto" else value
        config.provider_explicit = True
    aiw_overrides = {
        "model": os.environ.get("AIW_LLM_MODEL", "").strip(),
        "base_url": os.environ.get("AIW_LLM_BASE_URL", "").strip(),
        "api_key": os.environ.get("AIW_LLM_API_KEY", "").strip(),
    }
    prefixes = {"codex": "openai_codex", "copilot": "copilot", "openai": "openai", "deepseek": "deepseek"}
    env_prefixes = {"codex": "AIW_CODEX", "copilot": "COPILOT", "openai": "OPENAI", "deepseek": "DEEPSEEK"}
    for name, prefix in prefixes.items():
        target = config.providers[name]
        global_values = config.global_providers[name]
        for key in ("model", "base_url", "api_key"):
            if key in config.provider_fields[name]:
                continue
            value = getattr(global_values, key)
            if name == global_provider and use_global_generic and not value:
                value = getattr(config.global_generic, key)
            provider_env = os.environ.get(f"{env_prefixes[name]}_{key.upper()}", "").strip()
            if provider_env:
                value = provider_env
            if name == global_provider and aiw_overrides[key]:
                value = aiw_overrides[key]
            if value:
                setattr(target, key, value)

    explicit_language = requested_language or os.environ.get("CZ_LANGUAGE", "")
    if not explicit_language:
        for path in candidates:
            if path:
                data = _toml(path)
                i18n = data.get("i18n", {})
                if isinstance(i18n, dict) and isinstance(i18n.get("default_language"), str):
                    explicit_language = i18n["default_language"]
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
    openai = config.providers["openai"]
    if not openai.model and "CZ_OPENAI_MODEL" not in os.environ:
        openai.model = "gpt-4o-mini"
    if not openai.base_url and "CZ_OPENAI_BASE_URL" not in os.environ:
        openai.base_url = "https://api.openai.com/v1"
    deepseek = config.providers["deepseek"]
    if not deepseek.model and "CZ_DEEPSEEK_MODEL" not in os.environ:
        deepseek.model = "deepseek-flash"
    if not deepseek.base_url and "CZ_DEEPSEEK_BASE_URL" not in os.environ:
        deepseek.base_url = "https://api.deepseek.com"
    if not config.types:
        raise ValueError("cz requires at least one commit type")
    return config
