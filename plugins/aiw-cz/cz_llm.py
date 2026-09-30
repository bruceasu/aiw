"""Provider fallback orchestration for aiw-cz."""

from __future__ import annotations

from pathlib import Path

from cz_config import Config
from cz_core import Draft, build_prompt, parse_candidates
from cz_openai import OpenAIUnavailable, generate as openai_generate
from cz_providers import ProviderUnavailable, run_cli


def candidates(config: Config, root: Path, selected: str = "") -> list[Draft] | None:
    prompt, context = build_prompt(config, root)
    providers = [selected] if selected else ["codex", "copilot", "openai"]
    for provider in providers:
        try:
            if provider in ("codex", "copilot"):
                raw, _ = run_cli(provider, prompt)
            elif provider == "openai":
                raw = openai_generate(prompt, config.providers[provider])
            else:
                raise ProviderUnavailable(f"unsupported provider: {provider}")
            return parse_candidates(raw, config, context)
        except (OpenAIUnavailable, ProviderUnavailable, ValueError, KeyError) as exc:
            print(f"cz: {provider} unavailable: {exc}", flush=True)
    print("cz: all LLM providers unavailable; continuing with interactive wizard")
    return None
