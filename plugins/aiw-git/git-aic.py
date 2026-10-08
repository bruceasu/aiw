#!/usr/bin/env python3
"""Create a commit from all current changes using a CZ-configured AI provider."""

import argparse
import importlib.util
import os
import sys

META = {
    "name": "aic",
    "short": "Stage all changes and create a Conventional Commit with AI.",
    "long": "Stages tracked, untracked, and deleted files, generates a commit message from the staged diff, then commits it.",
    "usage": "aiw git aic",
    "examples": ["aiw git aic"],
}


def _core():
    path = os.path.join(os.path.dirname(__file__), "aiw-git-ai.py")
    spec = importlib.util.spec_from_file_location("aiw_git_ai", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def main(argv):
    parser = argparse.ArgumentParser(prog="aiw git aic", description=META["short"])
    parser.parse_args(argv)
    core = _core()
    try:
        core.require_git(core.git("add", "-A"), "add -A")
        diff = core.require_git(core.git("diff", "--cached", "--no-ext-diff", "--"), "diff --cached")
        if not diff.strip():
            return core.print_error("no staged changes to commit")
        message = core.generate(
            "Write one concise commit message for the staged Git diff. "
            "Follow Conventional Commits format. Return only the commit message, "
            "with an optional body when needed.\n\nStaged diff:\n" + diff
        )
        if not message:
            return core.print_error("AI returned an empty commit message")
        result = core.git("commit", "-F", "-", input_text=message + "\n")
        if result.stdout:
            print(result.stdout, end="")
        if result.stderr:
            print(result.stderr, end="", file=sys.stderr)
        return result.returncode
    except (OSError, RuntimeError) as exc:
        return core.print_error(str(exc))


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
