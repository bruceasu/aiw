"""Shared CZ text generation and Git context helpers for AI Git commands."""

from __future__ import annotations

import importlib
import subprocess
import sys
from pathlib import Path


def load_cz():
    cz_dir = Path(__file__).resolve().parent.parent / "aiw-cz"
    if not (cz_dir / "cz_llm.py").is_file():
        raise RuntimeError("aiw-cz provider files are unavailable beside aiw-git")
    path = str(cz_dir)
    if path not in sys.path:
        sys.path.insert(0, path)
    config_module = importlib.import_module("cz_config")
    llm_module = importlib.import_module("cz_llm")
    return config_module.load_config(cz_dir), llm_module.generate_text


def git(*args: str, input_text: str | None = None) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args], input=input_text, capture_output=True, text=True,
        encoding="utf-8", errors="replace", check=False,
    )


def require_git(result: subprocess.CompletedProcess[str], action: str) -> str:
    if result.returncode:
        detail = (result.stderr or result.stdout).strip()
        raise RuntimeError(detail or f"git {action} failed with exit code {result.returncode}")
    return result.stdout


def generate(prompt: str) -> str:
    try:
        config, generate_text = load_cz()
        text = generate_text(prompt, config, Path.cwd(), config.provider)
    except (OSError, RuntimeError, ValueError) as exc:
        raise RuntimeError(f"cannot initialize CZ AI provider: {exc}") from exc
    if not text:
        raise RuntimeError("all configured CZ AI providers failed")
    return text.strip()


def print_error(message: str) -> int:
    print(f"aiw git: {message}", file=sys.stderr)
    return 1
