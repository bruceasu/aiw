#!/usr/bin/env python3
"""FD-native worktree commands for AIW."""

from __future__ import annotations

import json
import os
import re
import stat
import subprocess
import sys
import uuid
from pathlib import Path


META = {
    "name": "wt",
    "short": "manage FD worktrees and local delivery",
    "long": "Create, inspect, sync, commit, cherry-pick, delete, and squash-deliver numbered FD worktrees.",
    "usage": "aiw git wt <subcommand> [args...]",
    "examples": ["aiw git wt add FD-001", "aiw git wt sync FD-001", "aiw git wt local-merge FD-001"],
    "commands": ["add", "status", "commit", "sync", "cherry-pick", "delete", "local-merge", "list"],
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
        raise WorktreeError(f"FD worktree is not registered: {fd_id}; run `aiw git wt add {fd_id}`")
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
    unignored: list[str] = []
    for relative, rule in ((f".wt/{fd_id}", ".wt/"),
                           (f".ai/fd/{fd_id}/workspace.json", ".ai/")):
        ignored = git(root, "check-ignore", "-q", "--", relative)
        if ignored.returncode == 1:
            unignored.append(rule)
        elif ignored.returncode:
            raise WorktreeError(ignored.stderr.strip() or
                                f"cannot check Git ignore rules for {relative}")
    if unignored:
        raise WorktreeError("FD management paths must be ignored before add; "
                            "add " + ", ".join(f"`{rule}`" for rule in unignored) +
                            " to .gitignore, commit it, and retry")
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


def worktree_sync(raw_id: str, source_branch: str | None = None) -> int:
    root = repository_root()
    fd_id, data, worktree_path, _ = workspace(root, raw_id)
    parent_branch = source_branch
    if parent_branch is None:
        current = git(root, "branch", "--show-current")
        if current.returncode or not current.stdout.strip():
            raise WorktreeError("cannot sync from a detached primary worktree; specify a local branch")
        parent_branch = current.stdout.strip()
    checked = git(root, "check-ref-format", "--branch", parent_branch)
    if checked.returncode:
        raise WorktreeError(f"invalid source branch: {parent_branch}")
    source = git(root, "rev-parse", "--verify", "--quiet", "--end-of-options",
                 f"refs/heads/{parent_branch}^{{commit}}")
    if source.returncode:
        raise WorktreeError(f"source branch does not exist: {parent_branch}")
    fd_branch = git(worktree_path, "branch", "--show-current")
    if fd_branch.returncode or fd_branch.stdout.strip() != data["branch"]:
        raise WorktreeError(f"FD worktree is not on recorded branch {data['branch']}")
    clean_worktree(worktree_path, "FD worktree")
    print(f"Merging {parent_branch} into {data['branch']}.")
    result = git(worktree_path, "merge", "--no-edit", "--", f"refs/heads/{parent_branch}")
    command_output(result)
    if result.returncode:
        if content_conflict(worktree_path):
            print(f"Resolve conflicts in {worktree_path}, then commit the merge or abort it with "
                  f"`git -C \"{worktree_path}\" merge --abort`.", file=sys.stderr)
        return result.returncode
    return 0


def worktree_cherry_pick(raw_id: str, commit_id: str) -> int:
    root = repository_root()
    fd_id, data, worktree_path, _ = workspace(root, raw_id)
    target = git(root, "rev-parse", "--verify", "--quiet", "--end-of-options",
                 f"{commit_id}^{{commit}}")
    if target.returncode or not target.stdout.strip():
        raise WorktreeError(f"commit does not resolve to a commit: {commit_id}")
    fd_branch = git(worktree_path, "branch", "--show-current")
    if fd_branch.returncode or fd_branch.stdout.strip() != data["branch"]:
        raise WorktreeError(f"FD worktree is not on recorded branch {data['branch']}")
    clean_worktree(worktree_path, "FD worktree")
    picked_oid = target.stdout.strip()
    print(f"Cherry-picking {picked_oid} into {data['branch']}.")
    result = git(worktree_path, "cherry-pick", picked_oid)
    command_output(result)
    if result.returncode and git(worktree_path, "rev-parse", "--verify", "-q", "CHERRY_PICK_HEAD").returncode == 0:
        print(f"Resolve conflicts in {worktree_path}, then continue with `git -C \"{worktree_path}\" cherry-pick --continue` "
              "or abort with `git cherry-pick --abort` from that worktree.", file=sys.stderr)
    return result.returncode


def worktree_delete(raw_id: str) -> int:
    root = repository_root()
    fd_id = normalize_fd_id(raw_id)
    worktree_dir = root / ".wt"
    worktree_path = worktree_dir / fd_id
    branch = f"feature/{fd_id}"
    if worktree_dir.is_symlink() or not worktree_dir.resolve().is_relative_to(root):
        raise WorktreeError(".wt must be a real directory inside the repository")

    metadata_path = root / ".ai" / "fd" / fd_id / "workspace.json"
    for path in (root / ".ai", root / ".ai" / "fd", metadata_path.parent, metadata_path):
        if path.is_symlink():
            raise WorktreeError(f"FD workspace metadata cannot use symlinks: {path}")
    if metadata_path.exists():
        if not metadata_path.resolve().is_relative_to(root):
            raise WorktreeError("FD workspace metadata is outside the repository")
        try:
            record = json.loads(metadata_path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as exc:
            raise WorktreeError(f"cannot read FD workspace metadata: {exc}") from exc
        if (not isinstance(record, dict) or not isinstance(record.get("fd_id"), str)
                or not isinstance(record.get("branch"), str)
                or not isinstance(record.get("worktree"), str)):
            raise WorktreeError(f"FD workspace metadata is incomplete: {metadata_path}")
        recorded_path = Path(record["worktree"])
        if not recorded_path.is_absolute():
            recorded_path = root / recorded_path
        if (record["fd_id"].upper() != fd_id or record["branch"] != branch
                or recorded_path.resolve() != worktree_path.resolve()):
            raise WorktreeError(f"FD workspace metadata does not match {fd_id}")

    if worktree_path.exists() or worktree_path.is_symlink():
        if worktree_path.is_symlink() or not worktree_path.resolve().is_relative_to(root):
            raise WorktreeError(f"FD worktree path is unsafe: {worktree_path}")
        entries = registered_worktrees(root)
        matches = [entry for entry in entries
                   if Path(entry.get("path", "")).resolve() == worktree_path.resolve()]
        if len(matches) != 1 or matches[0].get("branch") != branch:
            raise WorktreeError(f"Git does not register {worktree_path} on {branch}")
        clean_worktree(worktree_path, "FD worktree")
        os.chdir(root)
        remove_shared_ai_link(root, worktree_path)
        removed = git(root, "worktree", "remove", "--", str(worktree_path))
        command_output(removed)
        if removed.returncode:
            return removed.returncode
        print(f"removed worktree {worktree_path}")
    else:
        print(f"worktree does not exist: {worktree_path}")

    existing = git(root, "show-ref", "--verify", "--quiet", f"refs/heads/{branch}")
    if existing.returncode == 0:
        deleted = git(root, "branch", "-D", "--", branch)
        command_output(deleted)
        if deleted.returncode:
            return deleted.returncode
        print(f"removed branch {branch}")
    elif existing.returncode == 1:
        print(f"branch does not exist: {branch}")
    else:
        raise WorktreeError(existing.stderr.strip() or f"cannot inspect branch {branch}")

    if metadata_path.exists():
        metadata_path.unlink()
        print(f"removed workspace record {metadata_path}; FD receipts were retained")
    return 0


def clean_worktree(path: Path, label: str) -> None:
    result = git(path, "status", "--porcelain")
    if result.returncode:
        raise WorktreeError(result.stderr.strip() or f"cannot inspect {label}")
    if result.stdout.strip():
        raise WorktreeError(f"{label} has uncommitted changes; no operation was started")
    for marker, operation in (("MERGE_HEAD", "merge"), ("CHERRY_PICK_HEAD", "cherry-pick"),
                              ("REVERT_HEAD", "revert"), ("REBASE_HEAD", "rebase")):
        if git(path, "rev-parse", "--verify", "-q", marker).returncode == 0:
            raise WorktreeError(f"{label} already has a {operation} in progress")


def content_conflict(path: Path) -> bool:
    merge_head = git(path, "rev-parse", "--verify", "-q", "MERGE_HEAD")
    unmerged = git(path, "ls-files", "--unmerged")
    return merge_head.returncode == 0 and unmerged.returncode == 0 and bool(unmerged.stdout.strip())


def remove_shared_ai_link(root: Path, worktree_path: Path) -> None:
    """Detach only a verified .ai link to the primary workspace before Git removes the worktree."""
    ai_path = worktree_path / ".ai"
    try:
        info = ai_path.lstat()
    except FileNotFoundError:
        return
    except OSError as exc:
        raise WorktreeError(f"cannot inspect worktree .ai before cleanup: {exc}") from exc

    attributes = getattr(info, "st_file_attributes", 0)
    reparse_tag = getattr(info, "st_reparse_tag", None)
    is_posix_symlink = stat.S_ISLNK(info.st_mode)
    if not (is_posix_symlink or attributes & 0x400):  # FILE_ATTRIBUTE_REPARSE_POINT
        return

    expected_target = (root / ".ai").resolve()
    try:
        actual_target = ai_path.resolve(strict=True)
    except OSError as exc:
        raise WorktreeError(f"worktree .ai link cannot be resolved safely: {exc}") from exc
    if actual_target != expected_target or not actual_target.is_dir():
        raise WorktreeError("worktree .ai link does not target the primary workspace .ai; cleanup stopped")

    if os.name == "nt":
        if reparse_tag == 0xA0000003:  # IO_REPARSE_TAG_MOUNT_POINT (junction)
            os.rmdir(ai_path)
        elif reparse_tag == 0xA000000C:  # IO_REPARSE_TAG_SYMLINK
            ai_path.unlink()
        else:
            raise WorktreeError(f"worktree .ai uses an unknown reparse tag {reparse_tag!r}; cleanup stopped")
    elif is_posix_symlink:
        ai_path.unlink()
    else:
        raise WorktreeError("worktree .ai uses an unsupported reparse point; cleanup stopped")

    if not expected_target.is_dir():
        raise WorktreeError("primary workspace .ai disappeared while detaching the worktree link")


def cleanup_delivered_worktree(
        root: Path, fd_id: str, data: dict[str, str], worktree_path: Path,
        parent_path: Path, source_oid: str, delivered_oid: str,
        expected_parent_before_oid: str | None) -> int:
    """Remove the exact FD worktree and branch only after validating its squash delivery."""
    try:
        clean_worktree(parent_path, "parent worktree after squash delivery")
        clean_worktree(worktree_path, "FD worktree before cleanup")
        current_source = git(worktree_path, "rev-parse", "--verify", "HEAD")
        if current_source.returncode or current_source.stdout.strip() != source_oid:
            raise WorktreeError("FD HEAD changed after squash delivery; worktree and branch were preserved")

        parent_branch = git(parent_path, "branch", "--show-current")
        if parent_branch.returncode or parent_branch.stdout.strip() != data["parent_branch"]:
            raise WorktreeError("parent worktree left its recorded branch; worktree and branch were preserved")
        parent_head = git(parent_path, "rev-parse", "--verify", "HEAD")
        if parent_head.returncode:
            raise WorktreeError("cannot inspect the recorded parent branch; worktree and branch were preserved")
        delivered_ancestor = git(parent_path, "merge-base", "--is-ancestor",
                                 delivered_oid, parent_head.stdout.strip())
        if delivered_ancestor.returncode:
            raise WorktreeError("verified squash is not on the recorded parent history; worktree and branch were preserved")

        parents = git(parent_path, "rev-list", "--parents", "-n", "1", delivered_oid)
        if parents.returncode:
            raise WorktreeError("cannot inspect squash commit parents; worktree and branch were preserved")
        parent_fields = parents.stdout.split()
        if (len(parent_fields) != 2 or
                (expected_parent_before_oid is not None and
                 parent_fields[1] != expected_parent_before_oid)):
            raise WorktreeError("squash delivery is not the expected single-parent commit; worktree and branch were preserved")
        message = git(parent_path, "show", "-s", "--format=%B", delivered_oid)
        if message.returncode or f"FD-Source: {source_oid}" not in message.stdout.splitlines():
            raise WorktreeError("squash commit FD-Source does not match the current FD HEAD; worktree and branch were preserved")

        # Windows cannot remove a worktree while this process is using it as cwd.
        os.chdir(parent_path)
        remove_shared_ai_link(root, worktree_path)
        removed = git(parent_path, "worktree", "remove", "--", str(worktree_path))
        command_output(removed)
        if removed.returncode:
            print("Delivery succeeded, but worktree cleanup failed; the branch was preserved.", file=sys.stderr)
            return removed.returncode

        deleted = git(parent_path, "branch", "-D", "--", data["branch"])
        command_output(deleted)
        if deleted.returncode:
            print(f"Delivery succeeded and the worktree was removed, but branch cleanup failed; "
                  f"remove it after inspection with `git branch -D -- {data['branch']}`.", file=sys.stderr)
            return deleted.returncode

        metadata_path = root / ".ai" / "fd" / fd_id / "workspace.json"
        if metadata_path.is_symlink() or not metadata_path.resolve().is_relative_to(root):
            print("Delivery and Git cleanup succeeded, but workspace metadata is unsafe and was retained.",
                  file=sys.stderr)
            return 2
        try:
            metadata_path.unlink()
        except OSError as exc:
            print(f"Delivery and Git cleanup succeeded, but stale workspace metadata remains: {exc}",
                  file=sys.stderr)
            return 2
    except WorktreeError as exc:
        print(f"Delivery succeeded, but automatic cleanup stopped: {exc}", file=sys.stderr)
        return 2
    except OSError as exc:
        print(f"Delivery succeeded, but automatic cleanup failed safely: {exc}; "
              "inspect the worktree and branch before retrying.", file=sys.stderr)
        return 2

    print(f"cleanup: removed {worktree_path} and {data['branch']}; FD receipts were retained")
    return 0


def local_merge(raw_id: str) -> int:
    root = repository_root()
    fd_id, data, worktree_path, parent_path = workspace(root, raw_id)
    expected_parent = data["parent_branch"]
    parent_branch = git(parent_path, "branch", "--show-current")
    fd_branch = git(worktree_path, "branch", "--show-current")
    if parent_branch.returncode or parent_branch.stdout.strip() != expected_parent:
        raise WorktreeError(f"parent worktree is not on recorded branch {expected_parent}")
    if fd_branch.returncode or fd_branch.stdout.strip() != data["branch"]:
        raise WorktreeError(f"FD worktree is not on recorded branch {data['branch']}")
    clean_worktree(parent_path, "parent worktree")
    clean_worktree(worktree_path, "FD worktree")

    source = git(worktree_path, "rev-parse", "--verify", "HEAD")
    if source.returncode or not source.stdout.strip():
        raise WorktreeError("cannot identify the FD branch head for squash delivery")
    source_oid = source.stdout.strip()
    parent_before = git(parent_path, "rev-parse", "--verify", "HEAD")
    if parent_before.returncode or not parent_before.stdout.strip():
        raise WorktreeError("cannot identify the parent head before squash delivery")
    delivered = git(parent_path, "log", "--format=%H", "--extended-regexp",
                    f"--grep=^FD-Source: {source_oid}$")
    if delivered.returncode:
        raise WorktreeError("cannot inspect prior FD squash deliveries")
    if delivered.stdout.strip():
        prior_delivery_oid = delivered.stdout.splitlines()[0].strip()
        print(f"FD source {source_oid} is already delivered; retrying verified cleanup.")
        return cleanup_delivered_worktree(root, fd_id, data, worktree_path,
                                          parent_path, source_oid,
                                          prior_delivery_oid, None)

    print(f"Squashing {data['branch']} into {expected_parent}.")
    delivery = git(parent_path, "merge", "--squash", data["branch"])
    command_output(delivery)
    if delivery.returncode == 0:
        committed = git(parent_path, "commit", "-m",
                        f"Squash {fd_id} from {data['branch']}",
                        "-m", f"FD-Source: {source_oid}")
        command_output(committed)
        if committed.returncode:
            print("Squash commit failed; inspect the parent worktree before retrying.",
                  file=sys.stderr)
            return committed.returncode
        print(f"delivery: squashed {data['branch']} into {expected_parent}; "
              f"source {source_oid}")
        delivered_oid = git(parent_path, "rev-parse", "--verify", "HEAD")
        if delivered_oid.returncode:
            print("Squash commit exists, but its ID could not be verified; "
                  "the worktree and branch were preserved.", file=sys.stderr)
            return 2
        return cleanup_delivered_worktree(root, fd_id, data, worktree_path,
                                          parent_path, source_oid,
                                          delivered_oid.stdout.strip(),
                                          parent_before.stdout.strip())
    unmerged = git(parent_path, "ls-files", "--unmerged")
    if unmerged.returncode or not unmerged.stdout.strip():
        print("Squash delivery failed without a detected content conflict; "
              "parent state is preserved for manual inspection.", file=sys.stderr)
        return delivery.returncode

    current_parent = git(parent_path, "rev-parse", "--verify", "HEAD")
    if current_parent.returncode or current_parent.stdout.strip() != parent_before.stdout.strip():
        print("Parent HEAD changed during squash; inspect it before recovery.",
              file=sys.stderr)
        return 2
    print("Squash conflicts detected in the parent; resetting that attempt before recovery.",
          file=sys.stderr)
    aborted = git(parent_path, "reset", "--merge")
    command_output(aborted)
    if aborted.returncode:
        print("Could not reset the parent squash. Both worktrees are preserved; "
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
                  f"`aiw git wt commit {fd_id} \"<message>\"`, then rerun "
                  f"`aiw git wt local-merge {fd_id}`.", file=sys.stderr)
        else:
            print("Parent-to-FD merge failed without a detected content conflict; "
                  "inspect the FD worktree before continuing.", file=sys.stderr)
        return recovery.returncode
    print("Parent was merged into the FD worktree. Review the result, then rerun "
          f"`aiw git wt local-merge {fd_id}` to deliver explicitly.")
    return 2


def list_worktrees() -> int:
    result = git(repository_root(), "worktree", "list")
    command_output(result)
    return result.returncode


def usage() -> None:
    print("Usage: aiw git wt <command> [args...]")
    print("Commands:")
    print("  add <fd-id>                         Create the managed FD worktree.")
    print("  status <fd-id>                      Show parent and FD worktree status.")
    print('  commit <fd-id> "message"             Commit changes in the FD worktree.')
    print("  sync <fd-id> [branch]               Merge a primary-worktree branch into the FD worktree.")
    print("  cherry-pick <fd-id> <commit-id>     Pick a commit into the FD worktree.")
    print("  delete <fd-id>                      Remove the FD worktree and its feature branch.")
    print("  local-merge <fd-id>                 Squash-deliver to parent with conflict recovery.")
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
        if command == "sync" and len(rest) in {1, 2}:
            return worktree_sync(rest[0], rest[1] if len(rest) == 2 else None)
        if command == "cherry-pick" and len(rest) == 2:
            return worktree_cherry_pick(rest[0], rest[1])
        if command == "delete" and len(rest) == 1:
            return worktree_delete(rest[0])
        if command == "local-merge" and len(rest) == 1:
            return local_merge(rest[0])
        if command == "list" and not rest:
            return list_worktrees()
        if command in {"add", "status", "commit", "sync", "cherry-pick", "delete", "local-merge"}:
            shapes = {
                "sync": "<fd-id> [branch]",
                "cherry-pick": "<fd-id> <commit-id>",
                "commit": '<fd-id> "message"',
            }
            print(f"usage: aiw git wt {command} {shapes.get(command, '<fd-id>')}", file=sys.stderr)
            return 2
        print(f"unknown wt command: {command}; run `aiw git wt help`", file=sys.stderr)
        return 2
    except (OSError, WorktreeError) as exc:
        print(f"wt: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
