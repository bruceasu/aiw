#!/usr/bin/env python3
"""FD-native worktree commands for AIW."""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import uuid
from pathlib import Path


META = {
    "name": "aiw-wt",
    "short": "manage FD worktrees and local delivery",
    "description": "Create, inspect, commit, and locally merge numbered FD worktrees.",
    "commands": ["add", "status", "commit", "local-merge", "list"],
    "readOnly": False,
    "mutatesFiles": True,
    "requiresConfirmation": False,
    "outputFormat": "text",
}

FD_ID_RE = re.compile(r"^FD-(\d{3,})$", re.IGNORECASE)


class WorktreeError(Exception):
    pass


def git(cwd: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(["git", *args], cwd=cwd, text=True,
                          capture_output=True, check=False)


def repository_root() -> Path:
    result = subprocess.run(["git", "rev-parse", "--show-toplevel"],
                            text=True, capture_output=True, check=False)
    if result.returncode:
        raise WorktreeError("run this command inside an FD repository")
    current = Path(result.stdout.strip()).resolve()
    listing = git(current, "worktree", "list", "--porcelain")
    if listing.returncode:
        raise WorktreeError(listing.stderr.strip() or "cannot list Git worktrees")
    paths = [Path(line.removeprefix("worktree ")).resolve()
             for line in listing.stdout.splitlines() if line.startswith("worktree ")]
    if not paths:
        raise WorktreeError("primary Git worktree is not registered")
    return paths[0]


def normalize_fd_id(raw: str) -> str:
    match = FD_ID_RE.fullmatch(raw)
    if not match or int(match.group(1)) == 0:
        raise WorktreeError(f"{raw!r} is not an FD ID; use FD-001 format")
    return "FD-" + match.group(1)


def registered_worktrees(root: Path) -> list[dict[str, str]]:
    result = git(root, "worktree", "list", "--porcelain")
    if result.returncode:
        raise WorktreeError(result.stderr.strip() or "cannot list Git worktrees")
    entries: list[dict[str, str]] = []
    current: dict[str, str] = {}
    for line in result.stdout.splitlines() + [""]:
        if not line:
            if current:
                entries.append(current)
                current = {}
        elif line.startswith("worktree "):
            current["path"] = line.removeprefix("worktree ")
        elif line.startswith("branch "):
            current["branch"] = line.removeprefix("branch ").removeprefix("refs/heads/")
        elif line == "detached":
            current["branch"] = ""
    return entries


def workspace(root: Path, raw_id: str) -> tuple[str, dict[str, str], Path, Path]:
    fd_id = normalize_fd_id(raw_id)
    fd_files = list((root / "docs" / "features").glob(fd_id + "_*.md"))
    if len(fd_files) != 1 or fd_files[0].is_symlink():
        raise WorktreeError(f"expected one active FD design for {fd_id}")
    metadata_path = root / ".ai" / "fd" / fd_id / "workspace.json"
    for path in (root / ".ai", root / ".ai" / "fd", metadata_path.parent, metadata_path):
        if path.is_symlink():
            raise WorktreeError(f"FD workspace metadata cannot use symlinks: {path}")
    if not metadata_path.is_file():
        raise WorktreeError(f"FD worktree is not registered: {fd_id}; run `aiw wt add {fd_id}`")
    if not metadata_path.resolve().is_relative_to(root):
        raise WorktreeError("FD workspace metadata is outside the repository")
    try:
        data = json.loads(metadata_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise WorktreeError(f"cannot read FD workspace metadata: {exc}") from exc
    if not isinstance(data, dict) or any(
            not isinstance(data.get(key), str) or not data[key].strip()
            for key in ("fd_id", "parent_branch", "branch", "worktree")):
        raise WorktreeError(f"FD workspace metadata is incomplete: {metadata_path}")
    expected_branch = f"feature/{fd_id}"
    expected_path = (root / ".wt" / fd_id).resolve()
    actual_path = Path(data["worktree"])
    if not actual_path.is_absolute():
        actual_path = root / actual_path
    if (data["fd_id"].upper() != fd_id or data["branch"] != expected_branch
            or actual_path.resolve() != expected_path):
        raise WorktreeError(f"FD workspace metadata does not match {fd_id}")
    parent_ref = git(root, "check-ref-format", "--branch", data["parent_branch"])
    if parent_ref.returncode:
        raise WorktreeError("FD workspace metadata has an invalid parent branch")
    if ((root / ".wt").is_symlink() or not expected_path.is_relative_to(root)
            or expected_path.is_symlink() or not expected_path.is_dir()):
        raise WorktreeError(f"FD worktree path is missing or unsafe: {expected_path}")
    entries = registered_worktrees(root)
    fd_entries = [entry for entry in entries
                  if Path(entry.get("path", "")).resolve() == expected_path]
    parent_entries = [entry for entry in entries
                      if entry.get("branch") == data["parent_branch"]]
    if len(fd_entries) != 1 or fd_entries[0].get("branch") != expected_branch:
        raise WorktreeError(f"Git does not register {expected_path} on {expected_branch}")
    if len(parent_entries) != 1:
        raise WorktreeError(f"recorded parent branch is not checked out exactly once: {data['parent_branch']}")
    return fd_id, data, expected_path, Path(parent_entries[0]["path"]).resolve()


def command_output(result: subprocess.CompletedProcess[str]) -> None:
    if result.stdout:
        print(result.stdout, end="")
    if result.stderr:
        print(result.stderr, end="", file=sys.stderr)


def worktree_add(raw_id: str) -> int:
    fd_id = normalize_fd_id(raw_id)
    root = repository_root()
    fd_files = list((root / "docs" / "features").glob(fd_id + "_*.md"))
    if len(fd_files) != 1 or fd_files[0].is_symlink():
        raise WorktreeError(f"expected one active FD design for {fd_id}")
    fd_file = fd_files[0].relative_to(root).as_posix()
    tracked = git(root, "ls-files", "--error-unmatch", "--", fd_file)
    committed = git(root, "diff", "--quiet", "HEAD", "--", fd_file)
    if tracked.returncode or committed.returncode:
        raise WorktreeError("commit the FD plan before creating its worktree")
    status = git(root, "status", "--porcelain")
    if status.returncode or status.stdout.strip():
        raise WorktreeError("parent worktree must be clean before creating an FD worktree")
    parent = git(root, "symbolic-ref", "--quiet", "--short", "HEAD")
    if parent.returncode or not parent.stdout.strip():
        raise WorktreeError("cannot create an FD worktree from a detached HEAD")

    worktree_dir = root / ".wt"
    target = worktree_dir / fd_id
    if worktree_dir.is_symlink() or not worktree_dir.resolve().is_relative_to(root):
        raise WorktreeError(".wt must be a real directory inside the repository")
    if target.exists() or target.is_symlink():
        raise WorktreeError(f"FD worktree target already exists: {target}")
    branch = f"feature/{fd_id}"
    existing = git(root, "show-ref", "--verify", "--quiet", f"refs/heads/{branch}")
    if existing.returncode == 0:
        raise WorktreeError(f"branch already exists; inspect it before reuse: {branch}")
    if existing.returncode > 1:
        raise WorktreeError(existing.stderr.strip() or f"cannot inspect branch {branch}")

    metadata_path = root / ".ai" / "fd" / fd_id / "workspace.json"
    for directory in (root / ".ai", root / ".ai" / "fd", metadata_path.parent):
        if directory.is_symlink() or not directory.resolve().is_relative_to(root):
            raise WorktreeError("FD metadata directories must stay inside the repository")
    if metadata_path.exists() or metadata_path.is_symlink():
        raise WorktreeError(f"FD workspace record already exists: {metadata_path}")
    created = git(root, "worktree", "add", "-b", branch, str(target), "HEAD")
    command_output(created)
    if created.returncode:
        return created.returncode

    metadata = {"fd_id": fd_id, "parent_branch": parent.stdout.strip(),
                "branch": branch, "worktree": str(target)}
    metadata_path.parent.mkdir(parents=True, exist_ok=True)
    temporary = metadata_path.with_name(metadata_path.name + f".tmp-{uuid.uuid4().hex}")
    try:
        with temporary.open("x", encoding="utf-8", newline="\n") as stream:
            json.dump(metadata, stream, ensure_ascii=False, indent=2)
            stream.write("\n")
        os.replace(temporary, metadata_path)
    except OSError as exc:
        temporary.unlink(missing_ok=True)
        raise WorktreeError(f"worktree created at {target}, but workspace record failed: {exc}") from exc
    print(f"created {target} on {branch}; parent {parent.stdout.strip()}")
    return 0


def worktree_status(raw_id: str) -> int:
    fd_id, data, worktree_path, parent_path = workspace(repository_root(), raw_id)
    print(f"FD: {fd_id}")
    print(f"Parent: {data['parent_branch']} ({parent_path})")
    print(f"Branch: {data['branch']}")
    print(f"Worktree: {worktree_path}")
    for label, path in (("Parent status", parent_path), ("FD status", worktree_path)):
        result = git(path, "status", "--short", "--branch")
        if result.returncode:
            print(result.stderr.strip(), file=sys.stderr)
            return result.returncode
        print(f"{label}:")
        print(result.stdout, end="" if result.stdout.endswith("\n") else "\n")
    return 0


def worktree_commit(raw_id: str, message: str) -> int:
    fd_id, data, worktree_path, _ = workspace(repository_root(), raw_id)
    if not message.strip() or "\n" in message or "\r" in message:
        raise WorktreeError("commit message must be a non-empty single line")
    current = git(worktree_path, "branch", "--show-current")
    if current.returncode or current.stdout.strip() != data["branch"]:
        raise WorktreeError(f"FD worktree is not on recorded branch {data['branch']}")
    changed = git(worktree_path, "status", "--porcelain")
    if changed.returncode:
        raise WorktreeError(changed.stderr.strip() or "cannot inspect FD worktree")
    if not changed.stdout.strip():
        raise WorktreeError(f"FD worktree has no changes to commit: {fd_id}")
    staged = git(worktree_path, "add", "-A")
    command_output(staged)
    if staged.returncode:
        return staged.returncode
    committed = git(worktree_path, "commit", "-m", message)
    command_output(committed)
    return committed.returncode


def clean_worktree(path: Path, label: str) -> None:
    result = git(path, "status", "--porcelain")
    if result.returncode:
        raise WorktreeError(result.stderr.strip() or f"cannot inspect {label}")
    if result.stdout.strip():
        raise WorktreeError(f"{label} has uncommitted changes; no merge was started")
    merge_head = git(path, "rev-parse", "--verify", "-q", "MERGE_HEAD")
    if merge_head.returncode == 0:
        raise WorktreeError(f"{label} already has a merge in progress")


def content_conflict(path: Path) -> bool:
    merge_head = git(path, "rev-parse", "--verify", "-q", "MERGE_HEAD")
    unmerged = git(path, "ls-files", "--unmerged")
    return merge_head.returncode == 0 and unmerged.returncode == 0 and bool(unmerged.stdout.strip())


def local_merge(raw_id: str) -> int:
    fd_id, data, worktree_path, parent_path = workspace(repository_root(), raw_id)
    expected_parent = data["parent_branch"]
    parent_branch = git(parent_path, "branch", "--show-current")
    fd_branch = git(worktree_path, "branch", "--show-current")
    if parent_branch.returncode or parent_branch.stdout.strip() != expected_parent:
        raise WorktreeError(f"parent worktree is not on recorded branch {expected_parent}")
    if fd_branch.returncode or fd_branch.stdout.strip() != data["branch"]:
        raise WorktreeError(f"FD worktree is not on recorded branch {data['branch']}")
    clean_worktree(parent_path, "parent worktree")
    clean_worktree(worktree_path, "FD worktree")

    print(f"Merging {data['branch']} into {expected_parent}.")
    delivery = git(parent_path, "merge", "--no-edit", data["branch"])
    command_output(delivery)
    if delivery.returncode == 0:
        print(f"delivery: merged {data['branch']} into {expected_parent}")
        return 0
    if not content_conflict(parent_path):
        print("Delivery merge failed without a detected content conflict; "
              "parent state is preserved for manual inspection.", file=sys.stderr)
        return delivery.returncode

    print("Content conflicts detected in the parent; aborting that merge before recovery.",
          file=sys.stderr)
    aborted = git(parent_path, "merge", "--abort")
    command_output(aborted)
    if aborted.returncode:
        print("Could not abort the parent merge. Both worktrees are preserved; "
              "inspect the parent before continuing.", file=sys.stderr)
        return aborted.returncode
    try:
        clean_worktree(parent_path, "parent worktree after abort")
    except WorktreeError as exc:
        print(f"{exc}; parent-to-FD recovery was not started.", file=sys.stderr)
        return 2

    print(f"Merging {expected_parent} into {data['branch']} for conflict resolution.")
    recovery = git(worktree_path, "merge", "--no-edit", expected_parent)
    command_output(recovery)
    if recovery.returncode:
        if content_conflict(worktree_path):
            print(f"Resolve conflicts in {worktree_path}, commit with "
                  f"`aiw wt commit {fd_id} \"<message>\"`, then rerun "
                  f"`aiw wt local-merge {fd_id}`.", file=sys.stderr)
        else:
            print("Parent-to-FD merge failed without a detected content conflict; "
                  "inspect the FD worktree before continuing.", file=sys.stderr)
        return recovery.returncode
    print("Parent was merged into the FD worktree. Review the result, then rerun "
          f"`aiw wt local-merge {fd_id}` to deliver explicitly.")
    return 2


def list_worktrees() -> int:
    result = git(repository_root(), "worktree", "list")
    command_output(result)
    return result.returncode


def usage() -> None:
    print("Usage: aiw wt <command> [args...]")
    print("Commands:")
    print("  add <fd-id>                         Create the managed FD worktree.")
    print("  status <fd-id>                      Show parent and FD worktree status.")
    print('  commit <fd-id> "message"             Commit changes in the FD worktree.')
    print("  local-merge <fd-id>                 Deliver to parent with conflict recovery.")
    print("  list                                List registered Git worktrees.")


def main(args: list[str] | None = None) -> int:
    values = list(sys.argv[1:] if args is None else args)
    if not values or values[0] in {"help", "-h", "--help"}:
        usage()
        return 0
    command, rest = values[0], values[1:]
    try:
        if command == "add" and len(rest) == 1:
            return worktree_add(rest[0])
        if command == "status" and len(rest) == 1:
            return worktree_status(rest[0])
        if command == "commit" and len(rest) == 2:
            return worktree_commit(rest[0], rest[1])
        if command == "local-merge" and len(rest) == 1:
            return local_merge(rest[0])
        if command in {"list", "ls"} and not rest:
            return list_worktrees()
        if command in {"add", "status", "commit", "local-merge"}:
            print(f"usage: aiw wt {command} <fd-id>", file=sys.stderr)
            return 2
        print(f"unknown wt command: {command}; run `aiw wt help`", file=sys.stderr)
        return 2
    except (OSError, WorktreeError) as exc:
        print(f"wt: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
