from __future__ import annotations

import os
from pathlib import Path
from typing import Dict

from codex_flow.models import AppConfig

try:
    import tomllib  # type: ignore[attr-defined]
except ModuleNotFoundError:  # pragma: no cover
    tomllib = None


def default_data_dir() -> Path:
    if os.name == "nt":
        app_data = os.environ.get("APPDATA")
        if app_data:
            return Path(app_data) / "aiw-flow"
    xdg = os.environ.get("XDG_CONFIG_HOME")
    if xdg:
        return Path(xdg) / "aiw-flow"
    return Path.home() / ".config" / "aiw-flow"


def default_session_root() -> Path:
    return Path.cwd() / ".ai"


def config_file_path() -> Path:
    return default_data_dir() / "config.toml"


def _parse_value(raw: str):
    lowered = raw.lower()
    if lowered in {"true", "false"}:
        return lowered == "true"
    if raw.isdigit():
        return int(raw)
    if raw.startswith(("\"", "'")) and raw.endswith(("\"", "'")):
        return raw[1:-1]
    if raw.startswith("[") and raw.endswith("]"):
        inner = raw[1:-1].strip()
        if not inner:
            return []
        return [item.strip().strip("\"'") for item in inner.split(",")]
    return raw


def _fallback_parse_toml(text: str) -> Dict[str, object]:
    result: Dict[str, object] = {}
    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        result[key.strip()] = _parse_value(value.strip())
    return result


def load_global_config() -> AppConfig:
    path = config_file_path()
    if not path.exists():
        return AppConfig()
    text = path.read_text(encoding="utf-8")
    if tomllib is not None:
        data = tomllib.loads(text)
    else:
        data = _fallback_parse_toml(text)
    return AppConfig(
        model=data.get("model"),
        profile=data.get("profile"),
        sandbox=data.get("sandbox"),
        approval_policy=data.get("approval_policy"),
        codex_home=data.get("codex_home"),
        timeout=data.get("timeout"),
        additional_codex_args=list(data.get("additional_codex_args", [])),
    )


def merge_config(base: AppConfig, override: AppConfig) -> AppConfig:
    return AppConfig(
        model=override.model or base.model,
        profile=override.profile or base.profile,
        sandbox=override.sandbox or base.sandbox,
        approval_policy=override.approval_policy or base.approval_policy,
        codex_home=override.codex_home or base.codex_home,
        timeout=override.timeout if override.timeout is not None else base.timeout,
        additional_codex_args=override.additional_codex_args or base.additional_codex_args,
    )
