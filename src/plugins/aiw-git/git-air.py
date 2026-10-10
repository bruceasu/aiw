#!/usr/bin/env python3
"""Review staged Git changes with a CZ-configured AI provider."""

import argparse
import importlib.util
import os
import sys

META = {
    "name": "air",
    "short": "Review staged changes for issues and improvements with AI.",
    "long": "Sends only the staged diff to the configured AI provider and prints its review; it does not modify files or commits.",
    "usage": "aiw git air",
    "examples": ["aiw git air"],
}


def _core():
    path = os.path.join(os.path.dirname(__file__), "aiw-git-ai.py")
    spec = importlib.util.spec_from_file_location("aiw_git_ai", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def main(argv):
    parser = argparse.ArgumentParser(prog="aiw git air", description=META["short"])
    parser.parse_args(argv)
    core = _core()
    try:
        diff = core.require_git(core.git("diff", "--cached", "--no-ext-diff", "--"), "diff --cached")
        if not diff.strip():
            return core.print_error("no staged changes to review")
        review = core.generate(
            "Review this staged Git diff for bugs, correctness issues, or useful improvements. "
            "Explain findings with file and line references when possible. If there are no findings, say so.\n\n"
            "Staged diff:\n" + diff
        )
        print(review)
        return 0
    except (OSError, RuntimeError) as exc:
        return core.print_error(str(exc))


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
