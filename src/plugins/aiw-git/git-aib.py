#!/usr/bin/env python3
"""Summarize the current branch relative to a base ref with AI."""

import argparse
import importlib.util
import os
import sys

META = {
    "name": "aib",
    "short": "Summarize the current branch changes with AI.",
    "long": "Summarizes commit history and bounded file/diff context in BASE..HEAD. The default base is main; no repository state is changed.",
    "usage": "aiw git aib [--base REF]",
    "args": [{"flag": "--base REF", "description": "Base ref to compare with HEAD (default: main)."}],
    "examples": ["aiw git aib", "aiw git aib --base develop"],
}

MAX_FILE_CONTEXT_CHARS = 8_000
MAX_DIFF_CONTEXT_CHARS = 12_000


def _bounded_file_list(file_list):
    lines = file_list.splitlines()
    if len(file_list) <= MAX_FILE_CONTEXT_CHARS:
        return file_list or "(no changed files)"

    kept = []
    for line in lines:
        omitted = len(lines) - len(kept) - 1
        marker = f"[{omitted} more file entries omitted by the {MAX_FILE_CONTEXT_CHARS}-character limit]"
        candidate = "\n".join([*kept, line])
        if len(candidate) + 1 + len(marker) > MAX_FILE_CONTEXT_CHARS:
            break
        kept.append(line)

    omitted = len(lines) - len(kept)
    marker = f"[{omitted} more file entries omitted by the {MAX_FILE_CONTEXT_CHARS}-character limit]"
    return "\n".join([*kept, marker])


def _bounded_diff(diff):
    if len(diff) <= MAX_DIFF_CONTEXT_CHARS:
        return diff or "(no file diff)"

    marker = "\n[diff truncated; remaining content omitted]"
    prefix = diff[:MAX_DIFF_CONTEXT_CHARS - len(marker)]
    return prefix + marker


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
        branch_diff_range = f"{base_commit}...HEAD"
        file_list = core.require_git(
            core.git("diff", "--no-ext-diff", "--no-renames", "--name-status", branch_diff_range),
            "list branch changes",
        )
        change_stats = core.require_git(
            core.git("diff", "--no-ext-diff", "--no-renames", "--shortstat", branch_diff_range),
            "summarize branch changes",
        ).strip() or "no file changes"
        diff = core.require_git(
            core.git("diff", "--no-ext-diff", "--no-renames", "--unified=3", branch_diff_range),
            "diff branch changes",
        )
        branch = core.require_git(core.git("branch", "--show-current"), "branch --show-current").strip()
        bounded_files = _bounded_file_list(file_list)
        bounded_diff = _bounded_diff(diff)
        omission_guidance = (
            "Some file entries or diff content are omitted where marked. Do not infer details "
            "about changes that are not visible in the supplied context."
        )
        summary = core.generate(
            f"Summarize what branch {branch or 'HEAD'} does relative to {args.base}. "
            "Use the commit history and visible file changes as evidence. "
            "Describe the main purpose and notable changes concisely. "
            + omission_guidance
            + "\n\nCommits:\n"
            + history
            + "\n\nChanged files (status and path):\n"
            + bounded_files
            + "\n\nChange summary:\n"
            + change_stats
            + "\n\nDiff (limited to 12,000 characters):\n"
            + bounded_diff
        )
        print(summary)
        return 0
    except (OSError, RuntimeError) as exc:
        return core.print_error(str(exc))


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
