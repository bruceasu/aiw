#!/usr/bin/env python3
"""Python entry point for the aiw-cz plugin."""

from __future__ import annotations

import argparse
from pathlib import Path
import subprocess
import sys

from cz_config import load_config
from cz_core import build_prompt, has_staged_changes, previous_draft
from cz_llm import candidates
from cz_ui import choose_candidate, review_and_commit, wizard


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="aiw cz", description="Conventional Commit wizard")
    parser.add_argument("--llm", action="store_true", help="generate candidates with an LLM")
    parser.add_argument("--no-llm", action="store_true", help="use the interactive wizard")
    parser.add_argument("-N", "--candidates", type=int, default=None)
    parser.add_argument("--provider", choices=("codex", "copilot", "openai"))
    parser.add_argument("--model")
    parser.add_argument("--lang")
    parser.add_argument("-r", "--retry", action="store_true")
    args = parser.parse_args(argv)
    config = load_config(Path(__file__).resolve().parent, args.lang or "")
    if args.llm:
        config.llm = True
    if args.no_llm:
        config.llm = False
    if args.candidates:
        config.candidates = args.candidates
    if args.provider:
        config.provider = args.provider
    if args.model:
        config.providers[config.provider or "codex"].model = args.model
    try:
        if not has_staged_changes(Path.cwd()):
            print("aiw-cz: no staged changes; run git add first", file=sys.stderr)
            return 1
        build_prompt(config, Path.cwd())
    except subprocess.CalledProcessError as exc:
        print(f"aiw-cz: git context unavailable: {exc}", file=sys.stderr)
        return 1
    if args.retry:
        review_and_commit(Path.cwd(), config, previous_draft(Path.cwd()))
    elif config.llm:
        generated = candidates(config, Path.cwd(), config.provider)
        if not generated:
            review_and_commit(Path.cwd(), config, wizard(config))
        else:
            review_and_commit(Path.cwd(), config, choose_candidate(config, generated))
    else:
        review_and_commit(Path.cwd(), config, wizard(config))
    print(f"aiw-cz Python entry ready (language={config.language}, llm={config.llm})", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
