#!/usr/bin/env python3
"""Squash-merge a local branch into the current branch as one commit."""

import importlib.util
import os
import subprocess
import sys


HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, "aiw-git-core.py")
spec = importlib.util.spec_from_file_location("aiw_git_core", CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

HELP_FLAGS = {"-h", "--help", "-help", "-?"}

META = {
    "name": "fold",
    "short": "Squash-merge a local branch as one commit.",
    "long": (
        "Runs git merge --squash followed by git commit. With --prune, "
        "deletes only the local source branch after a successful commit and "
        "confirmation. Remote branches are never deleted."
    ),
    "usage": "aiw git fold <branch> [-m <message>] [--prune] [--force]",
    "args": [
        {"flag": "<branch>", "description": "Local branch to squash-merge into the current branch."},
        {"flag": "-m <message>", "description": "Commit message; otherwise Git opens its configured editor."},
        {"flag": "--prune", "description": "Delete the local source branch after commit confirmation."},
        {"flag": "--force", "description": "Skip the --prune confirmation; requires --prune."},
    ],
    "examples": [
        "aiw git fold feature/login -m \"feat: add login\"",
        "aiw git fold feature/login -m \"feat: add login\" --prune",
    ],
}


def run(command):
    print(">", " ".join(command), file=sys.stderr)
    return subprocess.run(command).returncode


def confirm_prune(branch, force):
    if force:
        return True
    if not sys.stdin.isatty():
        print("error: --prune requires an interactive confirmation or --force", file=sys.stderr)
        return False
    answer = input(f'Delete local branch "{branch}"? [y/N]: ').strip().lower()
    return answer in {"y", "yes"}


def parse_args(argv):
    if not argv or any(flag in argv for flag in HELP_FLAGS):
        return None
    branch = argv[0]
    message = None
    prune = False
    force = False
    index = 1
    while index < len(argv):
        arg = argv[index]
        if arg == "-m":
            if index + 1 >= len(argv) or not argv[index + 1]:
                raise ValueError("-m requires a commit message")
            message = argv[index + 1]
            index += 2
        elif arg == "--prune":
            prune = True
            index += 1
        elif arg == "--force":
            force = True
            index += 1
        else:
            raise ValueError(f"unknown option: {arg}")
    if force and not prune:
        raise ValueError("--force requires --prune")
    return branch, message, prune, force


def main(argv):
    try:
        parsed = parse_args(argv)
    except ValueError as err:
        print(f"error: {err}", file=sys.stderr)
        core.print_help_meta(META)
        return 2
    if parsed is None:
        core.print_help_meta(META)
        return 0 if argv else 2

    branch, message, prune, force = parsed
    if run(["git", "show-ref", "--verify", "--quiet", f"refs/heads/{branch}"]) != 0:
        print(f"error: local branch not found: {branch}", file=sys.stderr)
        return 2
    if run(["git", "merge", "--squash", branch]) != 0:
        print("error: squash merge failed; resolve conflicts or abort the merge before retrying", file=sys.stderr)
        return 1

    commit_command = ["git", "commit"]
    if message is not None:
        commit_command.extend(["-m", message])
    if run(commit_command) != 0:
        print("error: commit failed; local source branch was not deleted", file=sys.stderr)
        return 1

    if not prune:
        return 0
    if not confirm_prune(branch, force):
        print("local source branch retained", file=sys.stderr)
        return 0
    if run(["git", "branch", "-D", branch]) != 0:
        return 1
    print(f"deleted local branch: {branch}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
