"""Provider fallback orchestration for aiw-cz."""

from __future__ import annotations

import sys
from pathlib import Path

from cz_config import Config
from cz_core import Draft, build_prompt, parse_candidates
from cz_openai import OpenAIUnavailable, generate as openai_generate, generate_text as openai_generate_text
from cz_providers import ProviderUnavailable, run_cli, run_cli_text


def generate_text(prompt: str, config: Config, root: Path, selected: str = "") -> str | None:
    providers = [selected] if selected else [config.provider] if config.provider else ["codex", "copilot", "openai"]
    for provider in providers:
        try:
            if provider in ("codex", "copilot"):
                raw, _ = run_cli_text(provider, prompt, config.providers[provider].model, root)
            elif provider == "openai":
                raw = openai_generate_text(prompt, config.providers[provider])
            else:
                raise ProviderUnavailable(f"unsupported provider: {provider}")
            if raw.strip():
                return raw.strip()
            raise ProviderUnavailable("provider returned empty text")
        except (OpenAIUnavailable, ProviderUnavailable, ValueError, KeyError) as exc:
            print(f"cz: {provider} unavailable: {exc}", file=sys.stderr, flush=True)
    return None


def candidates(config: Config, root: Path, selected: str = "") -> list[Draft] | None:
    prompt, context = build_prompt(config, root)
    providers = [selected] if selected else ["codex", "copilot", "openai"]
    for provider in providers:
        try:
            if provider in ("codex", "copilot"):
                raw, _ = run_cli(provider, prompt, config.providers[provider].model, root)
            elif provider == "openai":
                raw = openai_generate(prompt, config.providers[provider])
            else:
                raise ProviderUnavailable(f"unsupported provider: {provider}")
            return parse_candidates(raw, config, context)
        except (OpenAIUnavailable, ProviderUnavailable, ValueError, KeyError) as exc:
            print(f"cz: {provider} unavailable: {exc}", flush=True)
    print("cz: all LLM providers unavailable; continuing with interactive wizard")
    return None
