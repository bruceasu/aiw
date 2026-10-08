#!/usr/bin/env python3
"""Summarize the current branch relative to a base ref with AI."""

import argparse
import importlib.util
import os
import sys

META = {
    "name": "aib",
    "short": "Summarize the current branch from its commit history with AI.",
    "long": "Summarizes one-line commit history in BASE..HEAD. The default base is main; no repository state is changed.",
    "usage": "aiw git aib [--base REF]",
    "args": [{"flag": "--base REF", "description": "Base ref to compare with HEAD (default: main)."}],
    "examples": ["aiw git aib", "aiw git aib --base develop"],
}


def _core():
    path = os.path.join(os.path.dirname(__file__), "aiw-git-ai.py")
    spec = importlib.util.spec_from_file_location("aiw_git_ai", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def main(argv):
    parser = argparse.ArgumentParser(prog="aiw git aib", description=META["short"])
    parser.add_argument("--base", default="main", metavar="REF")
    args = parser.parse_args(argv)
    core = _core()
    try:
        base = args.base + "^{commit}"
        base_commit = core.require_git(
            core.git("rev-parse", "--verify", "--quiet", "--end-of-options", base), "rev-parse base"
        ).strip()
        history = core.require_git(
            core.git("log", "--oneline", "--no-decorate", f"{base_commit}..HEAD"),
            "log branch history",
        )
        if not history.strip():
            return core.print_error(f"no commits found in {args.base}..HEAD")
        branch = core.require_git(core.git("branch", "--show-current"), "branch --show-current").strip()
        summary = core.generate(
            f"Summarize what branch {branch or 'HEAD'} does based on its commit history. "
            "Describe the main purpose and notable changes concisely.\n\nCommits:\n" + history
        )
        print(summary)
        return 0
    except (OSError, RuntimeError) as exc:
        return core.print_error(str(exc))


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
