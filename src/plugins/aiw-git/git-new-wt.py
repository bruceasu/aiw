#!/usr/bin/env python3
"""Create a new worktree and optionally move current changes into it."""

from __future__ import annotations

import argparse
import subprocess
import sys
import uuid
from pathlib import Path


META = {
    "name": "new-wt",
    "short": "Create a worktree and optionally move current changes into it.",
    "long": (
        "Creates a new worktree on a new or existing local branch. "
        "Use --move-changes to move tracked and untracked changes from the "
        "current worktree. The safety stash is kept after applying. "
        "For a new branch in the current directory, use `git switch -c`."
    ),
    "usage": "aiw git new-wt <branch> <path> [--base REV] [--existing] [--move-changes]",
    "args": [
        {"flag": "<branch>", "description": "New branch name, or an existing local branch with --existing."},
        {"flag": "<path>", "description": "New worktree directory; relative paths use the current directory."},
        {"flag": "--base REV", "description": "Starting commit for a new branch (default: HEAD)."},
        {"flag": "--existing", "description": "Use an existing local branch instead of creating one."},
        {"flag": "--move-changes", "description": "Move tracked and untracked changes from the current worktree."},
    ],
    "examples": [
        "aiw git new-wt feature/report ..\\report",
        "aiw git new-wt feature/report ..\\report --base develop --move-changes",
        "aiw git new-wt feature/report ..\\report --existing --move-changes",
    ],
}


class WorktreeError(Exception):
    pass


def git(cwd: Path, *args: str) -> subprocess.CompletedProcess[str]:
    try:
        return subprocess.run(["git", *args], cwd=cwd, text=True,
                              capture_output=True, check=False)
    except OSError as error:
        raise WorktreeError(f"cannot run Git: {error}") from None


def git_error(result: subprocess.CompletedProcess[str]) -> str:
    return (result.stderr or result.stdout).strip() or "Git command failed"


def commit_id(root: Path, revision: str) -> str:
    result = git(root, "rev-parse", "--verify", "--quiet", "--end-of-options",
                 revision + "^{commit}")
    if result.returncode:
        raise WorktreeError(f"invalid commit or branch {revision!r}: {git_error(result)}")
    return result.stdout.strip()


def stash_entry(root: Path, marker: str) -> tuple[str, str] | None:
    result = git(root, "stash", "list", "--format=%gd%x09%H%x09%gs")
    if result.returncode:
        raise WorktreeError(git_error(result))
    for line in result.stdout.splitlines():
        fields = line.split("\t", 2)
        if len(fields) == 3 and marker in fields[2]:
            return fields[0], fields[1]
    return None


def add_worktree(args: argparse.Namespace, invocation_dir: Path) -> int:
    root_result = git(invocation_dir, "rev-parse", "--show-toplevel")
    if root_result.returncode:
        raise WorktreeError("run this command inside a Git worktree")
    root = Path(root_result.stdout.strip()).resolve()

    if args.branch.startswith("-") or git(root, "check-ref-format", "--branch", args.branch).returncode:
        raise WorktreeError(f"invalid branch name: {args.branch}")
    if args.existing:
        if args.base is not None:
            raise WorktreeError("--base applies only when creating a new branch")
        base = commit_id(root, f"refs/heads/{args.branch}")
    else:
        branch_ref = git(root, "show-ref", "--verify", "--quiet", f"refs/heads/{args.branch}")
        if branch_ref.returncode == 0:
            raise WorktreeError(f"branch already exists; use --existing to attach it: {args.branch}")
        if branch_ref.returncode != 1:
            raise WorktreeError(git_error(branch_ref))
        base = commit_id(root, args.base or "HEAD")

    target = Path(args.path).expanduser()
    if not target.is_absolute():
        target = invocation_dir / target
    target = target.resolve()
    if target == root or target.exists() or target.is_symlink():
        raise WorktreeError(f"worktree path already exists: {target}")
    if not target.parent.is_dir():
        raise WorktreeError(f"parent directory does not exist: {target.parent}")
    if target.is_relative_to(root):
        ignored = git(root, "check-ignore", "-q", "--", target.relative_to(root).as_posix())
        if ignored.returncode != 0:
            raise WorktreeError("a worktree path inside the repository must be ignored by Git")

    command = ["worktree", "add"]
    if not args.existing:
        command.extend(["-b", args.branch])
        command.extend([str(target), base])
    else:
        command.extend([str(target), args.branch])
    created = git(root, *command)
    if created.stdout:
        print(created.stdout, end="")
    if created.stderr:
        print(created.stderr, end="", file=sys.stderr)
    if created.returncode:
        return created.returncode

    if not args.move_changes:
        print(f"Created worktree at {target}; the current worktree was not changed.")
        return 0

    status = git(root, "status", "--porcelain", "--untracked-files=all")
    if status.returncode:
        raise WorktreeError(git_error(status))
    if not status.stdout.strip():
        print("Created worktree; there were no tracked or untracked changes to move.")
        return 0

    marker = "aiw-new-wt-" + uuid.uuid4().hex
    saved = git(root, "stash", "push", "--include-untracked", "--message", marker)
    if saved.stdout:
        print(saved.stdout, end="")
    if saved.stderr:
        print(saved.stderr, end="", file=sys.stderr)
    if saved.returncode:
        raise WorktreeError(f"worktree created at {target}, but changes were not moved: {git_error(saved)}")

    entry = stash_entry(root, marker)
    remaining = git(root, "status", "--porcelain", "--untracked-files=all")
    if remaining.returncode or remaining.stdout.strip():
        stash_note = f" Safety stash: {entry[0]}." if entry else " Inspect `git stash list`."
        raise WorktreeError(
            f"worktree created at {target}; some source changes remain unstashed. "
            f"Inspect both worktrees before continuing.{stash_note}"
        )

    if not entry:
        raise WorktreeError(
            f"source changes were stashed but the stash reference could not be found. "
            f"Inspect `git stash list`; worktree: {target}"
        )
    stash_ref, stash_commit = entry
    applied = git(target, "stash", "apply", "--index", stash_commit)
    if applied.stdout:
        print(applied.stdout, end="")
    if applied.stderr:
        print(applied.stderr, end="", file=sys.stderr)
    if applied.returncode:
        raise WorktreeError(
            f"worktree created at {target}, but stash application needs attention. "
            f"The source is clean and {stash_ref} was kept; inspect the new worktree and stash."
        )

    print(f"Moved tracked and untracked changes to {target}.")
    print(f"Safety stash kept as {stash_ref}; after checking the new worktree, run `git stash drop {stash_ref}` if no longer needed.")
    return 0


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(
        prog="aiw git new-wt",
        description="Create a worktree; optionally move current tracked and untracked changes.",
    )
    parser.add_argument("branch", help="new branch name, or an existing local branch with --existing")
    parser.add_argument("path", help="new worktree directory")
    parser.add_argument("--base", help="starting commit for a new branch (default: HEAD)")
    parser.add_argument("--existing", action="store_true", help="use an existing local branch")
    parser.add_argument("--move-changes", action="store_true",
                        help="move current tracked and untracked changes; keep a safety stash")
    try:
        args = parser.parse_args(argv)
        return add_worktree(args, Path.cwd().resolve())
    except SystemExit as error:
        return int(error.code or 0)
    except WorktreeError as error:
        print(f"error: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
