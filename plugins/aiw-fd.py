#!/usr/bin/env python3
"""FD-first work management. Markdown owns progress; receipts own handoffs."""

from __future__ import annotations

import argparse
from contextlib import contextmanager
import hashlib
import getpass
import json
import os
import re
import shutil
import subprocess
import sys
import unicodedata
import uuid
from datetime import datetime, timezone
from pathlib import Path


META = {
    "name": "aiw-fd",
    "short": "manage numbered Feature Designs and role handoffs",
    "description": "Create, inspect, and advance FD-first work.",
    "commands": ["new", "list", "show", "show-report", "show-review", "emit", "claim", "resume", "request-review", "refresh-worker", "recover-worker", "refresh-tester", "reopen", "close", "cancel-event", "set-status", "force-emit"],
    "readOnly": False,
    "mutatesFiles": True,
    "requiresConfirmation": False,
    "outputFormat": "text",
}

FD_RE = re.compile(r"^FD-(\d{3,})$", re.IGNORECASE)
FD_FILE_RE = re.compile(r"^FD-(\d{3,})_[A-Z0-9_]+\.md$", re.IGNORECASE)
STATUS_RE = re.compile(r"(?m)^\*\*Status:\*\*[ \t]*(.+?)[ \t]*$")
REVISION_RE = re.compile(r"(?m)^\*\*Revision:\*\*[ \t]*(\d+)[ \t\r]*$")
PRIORITY_RE = re.compile(r"(?m)^\*\*Priority:\*\*[ \t]*(.+?)[ \t]*$")
TEST_POLICY_RE = re.compile(r"(?m)^\*\*Test policy:\*\*[ \t]*(.+?)[ \t]*$")
EVIDENCE_POLICY_RE = re.compile(r"(?m)^\*\*Evidence policy:\*\*[ \t]*(.+?)[ \t]*$")
DATA_REF_RE = re.compile(r"(?m)^<!-- aiw-data: ([^\r\n]+\.json) -->[ \t]*\r?$")
ITEM_RE = re.compile(r"(?m)^\s*- \[([ xX-])\] (\d+(?:\.\d+)*)\s+(.+)$")
ALLOWED = {
    "design-requested": ({"Planned"}, "Design", "planner"),
    "design-ready": ({"Design"}, "Open", "worker"),
    "implementation-ready": ({"Open", "In Progress"}, "Pending Verification", "reviewer"),
    "test-report-ready": ({"Pending Test"}, "Pending Test Acceptance", "pm"),
    "test-accepted": ({"Pending Test Acceptance"}, "Pending Verification", "reviewer"),
    "test-rejected": ({"Pending Test Acceptance"}, "In Progress", "worker"),
    "changes-requested": ({"Pending Verification"}, "In Progress", "worker"),
    "verification-passed": ({"Pending Verification"}, "Complete", "pm"),
    "needs-decision": ({"Design", "Open", "In Progress", "Pending Verification"}, None, "human"),
    "decision-recorded": ({"Design", "Open", "In Progress", "Pending Verification"}, None, "resume"),
}
ROLE_PRODUCERS = {
    "design-ready": "planner",
    "implementation-ready": "worker",
    "test-report-ready": "tester",
    "test-accepted": "pm",
    "test-rejected": "pm",
    "changes-requested": "reviewer",
    "verification-passed": "reviewer",
}
DEFINED_STATUSES = {"Planned", "Design", "Open", "In Progress", "Pending Test",
                    "Pending Test Acceptance", "Pending Verification", "Complete",
                    "Deferred", "Closed"}
DEFAULT_TEMPLATE = """# {{FD_ID}}: {{TITLE}}

**Status:** Planned
**Revision:** 1
**Priority:** Medium
**Evidence policy:** Dual

## Problem

%% NEEDS_INPUT: Describe the problem and affected people.

## Options and decision

%% NEEDS_INPUT: Compare options and record the chosen approach.

## Solution

%% NEEDS_INPUT: Describe the chosen design and compatibility effects.

## Scope

%% NEEDS_INPUT: State scope and exclusions.

## Work items

- [ ] 1.1 Define the first independently reviewable outcome and its acceptance evidence.

## Acceptance

%% NEEDS_INPUT: List observable success, failure and recovery behavior.

## Verification

- Compile-only check and static review by default. Optional tests use `$fd-test`
  when requested and do not gate FD acceptance.

## Sources

{{ISSUE}}
"""


class FDError(Exception):
    pass


def root() -> Path:
    result = subprocess.run(["git", "rev-parse", "--show-toplevel"], text=True,
                            capture_output=True, check=False)
    if result.returncode:
        raise FDError("run this command inside a Git worktree")
    return Path(result.stdout.strip()).resolve()


def feature_dir(base: Path) -> Path:
    return base / "docs" / "features"


def fd_id(raw: str) -> str:
    match = FD_RE.fullmatch(raw.upper())
    if not match or int(match.group(1)) == 0:
        raise FDError("FD ID must look like FD-001")
    return "FD-" + match.group(1)


def slug(title: str) -> str:
    result = re.sub(r"[^A-Z0-9]+", "_", title.upper()).strip("_")
    return result[:48] or "FEATURE"


def active_files(base: Path) -> list[Path]:
    directory = feature_dir(base)
    return sorted(path for path in directory.glob("FD-*.md") if FD_FILE_RE.fullmatch(path.name))


def archived_files(base: Path) -> list[Path]:
    archive = feature_dir(base) / "archive"
    return sorted(path for path in archive.rglob("FD-*.md")
                  if FD_FILE_RE.fullmatch(path.name)
                  and (path.parent == archive
                       or (path.parent.parent == archive
                           and path.parent.name == path.name.split("_", 1)[0])))


def is_archived_fd(base: Path, path: Path) -> bool:
    return path.is_relative_to(feature_dir(base) / "archive")


def all_files(base: Path) -> list[Path]:
    return active_files(base) + archived_files(base)


def resolve_fd(base: Path, raw_id: str) -> Path:
    name = fd_id(raw_id)
    matches = [path for path in all_files(base) if path.name.upper().startswith(name + "_")]
    if len(matches) != 1:
        raise FDError(f"expected one FD file for {name}, found {len(matches)}")
    if not matches[0].resolve().is_relative_to(base) or matches[0].is_symlink():
        raise FDError("FD path must be a regular file inside the repository")
    return matches[0]


def status(content: str) -> str:
    match = STATUS_RE.search(content)
    if not match:
        raise FDError("FD is missing **Status:**")
    return match.group(1).strip()


def revision(content: str) -> int:
    match = REVISION_RE.search(content)
    if not match:
        raise FDError("FD is missing **Revision:**; migrate the FD before emitting events")
    return int(match.group(1))


def fd_digest(content: bytes | str) -> str:
    data = content.encode("utf-8") if isinstance(content, str) else content
    return hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()


def title(content: str) -> str:
    heading = content.splitlines()[0].lstrip("# ").strip()
    return re.sub(r"^FD-\d+\s*[:：]\s*", "", heading, flags=re.IGNORECASE)


def terminal_text(value: str) -> str:
    visible = "".join(char for char in value
                      if unicodedata.category(char) not in {"Cc", "Cf"})
    encoding = getattr(sys.stdout, "encoding", None)
    if not encoding:
        return visible
    return visible.encode(encoding, errors="backslashreplace").decode(encoding)


def fd_list_rows(base: Path) -> dict[str, list[tuple[str, str, str, str]]]:
    groups: dict[str, list[tuple[str, str, str, str]]] = {}
    for path in all_files(base):
        content = path.read_text(encoding="utf-8")
        fd = path.name.split("_", 1)[0]
        state = status(content)
        priority_match = PRIORITY_RE.search(content)
        priority = priority_match.group(1).strip() if priority_match else "—"
        row = tuple(terminal_text(value) for value in
                    (fd, state, priority, title(content)))
        groups.setdefault(state, []).append(row)
    return groups


def supports_ansi_color() -> bool:
    is_tty = getattr(sys.stdout, "isatty", None)
    if (not is_tty or not is_tty() or "NO_COLOR" in os.environ
            or os.environ.get("TERM", "").strip().casefold() == "dumb"):
        return False
    term = os.environ.get("TERM", "").strip().casefold()
    term_program = os.environ.get("TERM_PROGRAM", "").casefold()
    color_term_prefixes = (
        "alacritty", "ansi", "cygwin", "eterm", "foot", "iterm", "kitty",
        "konsole", "linux", "msys", "putty", "rxvt", "screen", "st-",
        "tmux", "wezterm", "xterm",
    )
    color_programs = {
        "alacritty", "apple_terminal", "hyper", "iterm.app", "kitty",
        "tabby", "vscode", "wezterm",
    }
    windows_ansi = bool(os.environ.get("WT_SESSION") or os.environ.get("ANSICON")
                        or os.environ.get("ConEmuANSI", "").casefold() == "on")
    return bool(term.startswith(color_term_prefixes) or term_program in color_programs
                or windows_ansi)


def ansi_color(value: str, code: str, enabled: bool) -> str:
    if not enabled:
        return value
    return f"\x1b[{code}m{value}\x1b[0m"


def render_fd_list(base: Path, color: bool = False) -> str:
    groups = fd_list_rows(base)
    if not groups:
        return "No feature designs found."
    status_order = ("Planned", "Design", "Open", "In Progress", "Pending Test",
                    "Pending Test Acceptance", "Pending Verification", "Complete",
                    "Deferred", "Closed")
    order = {value: index for index, value in enumerate(status_order)}
    status_colors = {"Planned": "36", "Design": "36", "Open": "32",
                     "In Progress": "33", "Pending Test": "35",
                     "Pending Test Acceptance": "35", "Pending Verification": "33",
                     "Complete": "32", "Deferred": "31", "Closed": "90"}
    priority_colors = {"High": "31", "Medium": "33", "Low": "36"}
    lines: list[str] = []
    for state in sorted(groups, key=lambda value: (order.get(value, len(order)), value.casefold())):
        rows = groups[state]
        state_color = status_colors.get(state, "")
        widths = (
            max(len("FD"), *(len(row[0]) for row in rows)),
            max(len("STATUS"), *(len(row[1]) for row in rows)),
            max(len("PRIORITY"), *(len(row[2]) for row in rows)),
        )
        lines.extend((f"{ansi_color(terminal_text(state), state_color, color)} ({len(rows)})",
                      f"{'FD':<{widths[0]}}  {'STATUS':<{widths[1]}}  "
                      f"{'PRIORITY':<{widths[2]}}  TITLE"))
        for fd, row_state, priority, row_title in rows:
            priority_color = priority_colors.get(priority, "")
            colored_state = ansi_color(row_state, state_color, color)
            colored_priority = ansi_color(priority, priority_color, color)
            state_width = widths[1] + (
                len(f"\x1b[{state_color}m\x1b[0m") if color and state_color else 0)
            priority_width = widths[2] + (
                len(f"\x1b[{priority_color}m\x1b[0m")
                if color and priority_color else 0)
            lines.append(f"{fd:<{widths[0]}}  {colored_state:<{state_width}}  "
                         f"{colored_priority:<{priority_width}}  {row_title}")
        lines.append("")
    return "\n".join(lines).rstrip()


def independent_testing(content: str) -> bool:
    match = TEST_POLICY_RE.search(content)
    return bool(match and match.group(1).strip() == "Independent")


def dual_evidence(content: str) -> bool:
    match = EVIDENCE_POLICY_RE.search(content)
    return bool(match and match.group(1).strip() == "Dual")


def atomic_text(path: Path, content: str) -> None:
    atomic_bytes(path, content.encode("utf-8"))


def atomic_bytes(path: Path, content: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(path.name + f".tmp-{os.getpid()}-{uuid.uuid4().hex}")
    with temporary.open("xb") as stream:
        stream.write(content)
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temporary, path)


def atomic_json(path: Path, value: dict) -> None:
    atomic_text(path, json.dumps(value, ensure_ascii=False, indent=2) + "\n")


def runtime_dir(base: Path, name: str) -> Path:
    result = subprocess.run(["git", "-C", str(base), "worktree", "list", "--porcelain"],
                            text=True, capture_output=True, check=False)
    if result.returncode:
        raise FDError("could not locate the shared .ai directory for this Git worktree")
    primary = next((line.removeprefix("worktree ") for line in result.stdout.splitlines()
                    if line.startswith("worktree ")), None)
    if not primary:
        raise FDError("Git did not report a primary worktree for the shared .ai directory")
    return Path(primary).resolve() / ".ai" / "fd" / name


@contextmanager
def fd_lock(base: Path, name: str):
    directory = runtime_dir(base, name)
    directory.mkdir(parents=True, exist_ok=True)
    lock = directory / ".mutation-lock"
    try:
        lock.mkdir()
    except FileExistsError as exc:
        raise FDError(f"FD {name} is being updated; inspect {lock} before retrying") from exc
    try:
        yield
    finally:
        lock.rmdir()


def event_paths(base: Path, name: str) -> list[Path]:
    return sorted((runtime_dir(base, name) / "events").glob("*.json"))


def latest_event(base: Path, name: str) -> tuple[Path, dict] | None:
    paths = event_paths(base, name)
    if not paths:
        return None
    path = paths[-1]
    return path, json.loads(path.read_text(encoding="utf-8"))


def git_result(base: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(["git", "-C", str(base), *args], text=True,
                          encoding="utf-8", errors="replace",
                          capture_output=True, check=False)


def registered_worktrees(base: Path) -> list[dict[str, str]]:
    result = git_result(base, "worktree", "list", "--porcelain")
    if result.returncode:
        raise FDError(result.stderr.strip() or "could not list Git worktrees")
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


def valid_local_branch(base: Path, branch: str) -> bool:
    if not branch.strip() or git_result(base, "check-ref-format", "--branch", branch).returncode:
        return False
    return git_result(base, "show-ref", "--verify", "--quiet",
                      f"refs/heads/{branch}").returncode == 0


def workspace_info(base: Path, name: str) -> tuple[dict | None, list[str]]:
    warnings: list[str] = []
    try:
        runtime = runtime_dir(base, name)
        primary = runtime.parents[2]
        metadata_path = runtime / "workspace.json"
        if not metadata_path.exists() and not metadata_path.is_symlink():
            return None, warnings
        if (metadata_path.is_symlink() or (primary / ".ai").is_symlink()
                or (primary / ".ai" / "fd").is_symlink()
                or metadata_path.parent.is_symlink()
                or not metadata_path.is_file()
                or not metadata_path.resolve().is_relative_to(primary)):
            raise FDError("workspace metadata path is missing or unsafe")
        data = json.loads(metadata_path.read_text(encoding="utf-8"))
        required = ("fd_id", "parent_branch", "branch", "worktree")
        if not isinstance(data, dict) or any(
                not isinstance(data.get(key), str) or not data[key].strip()
                for key in required):
            raise FDError("workspace metadata is incomplete")
        expected_branch = f"feature/{name}"
        worktree_path = primary / ".wt" / name
        expected_worktree = worktree_path.resolve()
        actual_worktree = Path(data["worktree"])
        if not actual_worktree.is_absolute():
            actual_worktree = primary / actual_worktree
        if (
            data["fd_id"].upper() != name
            or data["branch"] != expected_branch
            or not expected_worktree.is_relative_to(primary.resolve())
            or actual_worktree.resolve() != expected_worktree
            or (primary / ".wt").is_symlink()
            or worktree_path.is_symlink()
            or not expected_worktree.is_dir()
        ):
            raise FDError("workspace metadata does not match the expected FD worktree")
        matching = [entry for entry in registered_worktrees(base)
                    if Path(entry.get("path", "")).resolve() == expected_worktree]
        if len(matching) != 1 or matching[0].get("branch") != expected_branch:
            raise FDError("Git does not register the recorded FD worktree and branch")
        if not valid_local_branch(base, data["branch"]):
            raise FDError("recorded FD branch is not a valid local ref")
        if not valid_local_branch(base, data["parent_branch"]):
            raise FDError("recorded parent branch is not a valid local ref")
        data["worktree"] = str(expected_worktree)
        return data, warnings
    except (FDError, OSError, RuntimeError, json.JSONDecodeError) as exc:
        warnings.append(f"cannot use FD workspace metadata: {exc}")
        return None, warnings


def evidence_source_locations(base: Path, name: str) -> tuple[list[dict[str, str]], list[str]]:
    branch_result = git_result(base, "symbolic-ref", "--quiet", "--short", "HEAD")
    current_branch = branch_result.stdout.strip() if branch_result.returncode == 0 else ""
    current_label = current_branch or "detached HEAD"
    sources = [
        {"kind": "worktree", "path": str(base.resolve()),
         "label": f"worktree {current_label}"},
        {"kind": "git", "ref": "HEAD", "label": f"HEAD ({current_label})"},
    ]
    data, warnings = workspace_info(base, name)
    if data:
        worktree = Path(data["worktree"])
        if worktree.resolve() != base.resolve():
            sources.append({"kind": "worktree", "path": str(worktree),
                            "label": f"worktree {data['branch']}"})
        known_refs = {current_branch} if current_branch else set()
        for branch in (data["branch"], data["parent_branch"]):
            if branch not in known_refs:
                sources.append({"kind": "git", "ref": branch, "label": f"branch {branch}"})
                known_refs.add(branch)
    return sources, warnings


def evidence_directories(name: str, kind: str) -> tuple[str, str]:
    folder = {"report": "reports", "review": "reviews"}.get(kind)
    if not folder:
        raise FDError(f"unsupported evidence kind: {kind}")
    return (f"docs/features/{folder}",
            f"docs/features/archive/{name}/{folder}")


def is_fd_markdown(path: str, name: str) -> bool:
    filename = path.rsplit("/", 1)[-1]
    return filename.startswith(name + "-") and filename.lower().endswith(".md")


def add_evidence_record(inventory: dict, kind: str, relative: str, content: bytes,
                        modified_at: float, source_label: str, sidecar: str,
                        warnings: list[str]) -> None:
    normalized = content.replace(b"\r\n", b"\n")
    digest = hashlib.sha256(normalized).hexdigest()
    key = (kind, digest)
    item = inventory.get(key)
    if item is None:
        try:
            rendered = normalized.decode("utf-8")
        except UnicodeDecodeError:
            warnings.append(f"skip non-UTF-8 evidence file: {relative} on {source_label}")
            return
        item = {"kind": kind, "digest": digest, "content": rendered,
                "modified_at": modified_at, "sources": set(), "sidecars": set()}
        inventory[key] = item
    item["modified_at"] = max(item["modified_at"], modified_at)
    item["sources"].add(f"{source_label}:{relative}")
    if sidecar:
        item["sidecars"].add(f"{source_label}:{sidecar}")


def scan_worktree_evidence(base: Path, repo_root: Path, name: str, kind: str,
                           source: dict[str, str], inventory: dict,
                           warnings: list[str]) -> None:
    source_root = Path(source["path"])
    source_label = source["label"]
    if source_root.is_symlink():
        warnings.append(f"skip linked evidence worktree: {source_root}")
        return
    try:
        source_root = source_root.resolve(strict=True)
        if not source_root.is_relative_to(repo_root) or not source_root.is_dir():
            warnings.append(f"skip unsafe evidence worktree: {source_root}")
            return
    except (OSError, RuntimeError) as exc:
        warnings.append(f"cannot inspect evidence worktree {source_label}: {exc}")
        return
    for directory in evidence_directories(name, kind):
        folder = source_root / Path(directory)
        if not folder.exists() and not folder.is_symlink():
            continue
        if folder.is_symlink() or not folder.is_dir():
            warnings.append(f"skip unsafe evidence directory: {directory} in {source_label}")
            continue
        try:
            if not folder.resolve(strict=True).is_relative_to(repo_root):
                warnings.append(f"skip evidence directory outside repository: {directory}")
                continue
            candidates = sorted(folder.iterdir())
        except (OSError, RuntimeError) as exc:
            warnings.append(f"cannot list evidence directory {directory}: {exc}")
            continue
        for candidate in candidates:
            if not is_fd_markdown(candidate.name, name):
                continue
            relative = candidate.relative_to(source_root).as_posix()
            try:
                resolved = candidate.resolve(strict=True)
                if (candidate.is_symlink() or not resolved.is_relative_to(repo_root)
                        or not resolved.is_file()):
                    warnings.append(f"skip unsafe evidence file: {relative}")
                    continue
                content = resolved.read_bytes()
                modified_at = resolved.stat().st_mtime
            except (OSError, RuntimeError) as exc:
                warnings.append(f"cannot read evidence file {relative}: {exc}")
                continue
            sidecar_path = candidate.with_suffix(".json")
            sidecar = ""
            if not sidecar_path.is_symlink() and sidecar_path.is_file():
                try:
                    sidecar_resolved = sidecar_path.resolve(strict=True)
                    if sidecar_resolved.is_relative_to(repo_root) and sidecar_resolved.is_file():
                        sidecar = sidecar_path.relative_to(source_root).as_posix()
                except (OSError, RuntimeError):
                    pass
            add_evidence_record(inventory, kind, relative, content, modified_at,
                                source_label, sidecar, warnings)


def git_tree_evidence(base: Path, name: str, kind: str, ref: str,
                      source_label: str, inventory: dict,
                      warnings: list[str]) -> None:
    directories = evidence_directories(name, kind)
    result = git_result(base, "ls-tree", "-r", "-z", ref, "--", *directories)
    if result.returncode:
        warnings.append(f"cannot list evidence on {source_label}: {result.stderr.strip()}")
        return
    records = {}
    for raw in result.stdout.split("\0"):
        if not raw or "\t" not in raw:
            continue
        metadata, relative = raw.split("\t", 1)
        parts = metadata.split()
        if len(parts) != 3 or parts[1] != "blob" or parts[0] == "120000":
            continue
        records[relative] = parts[0]
    paths = set(records)
    for relative, mode in records.items():
        if mode not in {"100644", "100755"} or not is_fd_markdown(relative, name):
            continue
        blob = subprocess.run(["git", "-C", str(base), "show", f"{ref}:{relative}"],
                              capture_output=True, check=False)
        if blob.returncode:
            warnings.append(f"cannot read evidence {relative} from {source_label}")
            continue
        timestamp = git_result(base, "log", "-1", "--format=%ct", ref, "--", relative)
        try:
            modified_at = float(timestamp.stdout.strip())
        except ValueError:
            warnings.append(f"cannot determine evidence time for {relative} on {source_label}")
            continue
        sidecar = relative[:-3] + ".json"
        if sidecar not in paths or records.get(sidecar) == "120000":
            sidecar = ""
        add_evidence_record(inventory, kind, relative, blob.stdout, modified_at,
                            source_label, sidecar, warnings)


def evidence_inventory(base: Path, name: str, kind: str) -> tuple[list[dict], list[str]]:
    repo_root = runtime_dir(base, name).parents[2]
    sources, warnings = evidence_source_locations(base, name)
    inventory: dict[tuple[str, str], dict] = {}
    for source in sources:
        if source["kind"] == "worktree":
            scan_worktree_evidence(base, repo_root, name, kind, source, inventory, warnings)
        else:
            git_tree_evidence(base, name, kind, source["ref"], source["label"],
                              inventory, warnings)
    items = sorted(inventory.values(),
                   key=lambda item: (-item["modified_at"], sorted(item["sources"])))
    return items, warnings


def evidence_time(timestamp: float) -> str:
    return datetime.fromtimestamp(timestamp, timezone.utc).isoformat(timespec="seconds").replace(
        "+00:00", "Z")


def print_evidence_list(items: list[dict]) -> None:
    for index, item in enumerate(items, start=1):
        print(f"{index}. {evidence_time(item['modified_at'])}")
        for source in sorted(item["sources"]):
            print(f"   Markdown: {source}")
        sidecars = sorted(item["sidecars"])
        if sidecars:
            for sidecar in sidecars:
                print(f"   JSON: {sidecar}")
        else:
            print("   JSON: 无")


def show_evidence(base: Path, name: str, kind: str, last: bool) -> None:
    items, warnings = evidence_inventory(base, name, kind)
    items = items[:20]
    for warning in warnings:
        print(f"fd: warning: {warning}", file=sys.stderr)
    if not items:
        print(f"{name} 没有可用的 {kind} 证据。")
        return
    if last:
        print(items[0]["content"].rstrip())
        return

    print_evidence_list(items)
    if not (sys.stdin.isatty() and sys.stdout.isatty()):
        print(f"请在交互式终端运行 aiw fd show-{kind} {name} 后选择编号，或使用 --last。")
        return

    while True:
        try:
            choice = input("选择编号（回车或 q 取消）：").strip()
        except EOFError:
            return
        if not choice or choice.lower() == "q":
            return
        if choice.isdecimal() and 1 <= int(choice) <= len(items):
            print(items[int(choice) - 1]["content"].rstrip())
            return
        print(f"请输入 1 到 {len(items)} 之间的编号，或输入 q 取消。")


def show_status_summary(base: Path, name: str, content: str,
                        latest: tuple[Path, dict] | None) -> None:
    print("\nStatus summary:")
    print("Status:", status(content))
    workspace, warnings = workspace_info(base, name)
    if workspace:
        print("Verified worktree:", workspace["worktree"])
        print("Verified branch:", workspace["branch"])
        print("Verified parent branch:", workspace["parent_branch"])
    elif warnings:
        print("Verified workspace: unavailable")
        for warning in warnings:
            print(f"Workspace warning: {warning}")
    else:
        print("Verified workspace: not configured")
        print("Verified branch: not configured")

    if not latest:
        print("Current handoff: none")
        print("Latest event: none")
        return

    event = latest[1]
    if event.get("dispatch_state") in {"pending", "launching", "dispatched"}:
        print(f"Current handoff ({event.get('created_at', '未记录')}):")
        print(json.dumps(event, ensure_ascii=False, indent=2))
    else:
        print("Current handoff: none")
    print(f"Latest event ({event.get('created_at', '未记录')}):")
    print(json.dumps(event, ensure_ascii=False, indent=2))


def current_test_acceptance(base: Path, name: str) -> dict | None:
    for path in reversed(event_paths(base, name)):
        event = json.loads(path.read_text(encoding="utf-8"))
        if event.get("event_type") == "test-accepted":
            return event
        if event.get("event_type") in {"implementation-ready", "test-requested"}:
            return None
    return None


def safe_artifact(base: Path, raw: str) -> str:
    candidate = Path(raw)
    normalized = raw.replace("\\", "/")
    if (candidate.is_absolute() or ".." in candidate.parts or not raw.strip()
            or normalized.startswith(".git/") or normalized.startswith(".ai/credentials/")):
        raise FDError("artifact must be a project-relative path")
    resolved = (base / candidate).resolve()
    if not resolved.is_relative_to(base) or not resolved.is_file():
        raise FDError(f"artifact does not exist inside the repository: {raw}")
    return resolved.relative_to(base).as_posix()


def structured_evidence(base: Path, artifact_ref: str, expected_kind: str = "",
                        fd_name: str = "", source_event: str = "") -> dict:
    if not artifact_ref.endswith(".md"):
        raise FDError("human report must be a Markdown file")
    content = (base / artifact_ref).read_text(encoding="utf-8")
    matches = DATA_REF_RE.findall(content)
    if len(matches) != 1:
        raise FDError("Dual evidence report requires one aiw-data JSON reference")
    raw_ref = matches[0]
    if Path(raw_ref).name == raw_ref:
        raw_ref = (Path(artifact_ref).parent / raw_ref).as_posix()
    data_ref = safe_artifact(base, raw_ref)
    if data_ref != Path(artifact_ref).with_suffix(".json").as_posix():
        raise FDError("machine JSON must share the human report basename")
    def unique_pairs(pairs: list[tuple[str, object]]) -> dict:
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError(f"duplicate JSON key: {key}")
            result[key] = value
        return result
    try:
        payload = json.loads((base / data_ref).read_text(encoding="utf-8"),
                             object_pairs_hook=unique_pairs)
    except (ValueError, UnicodeError) as exc:
        raise FDError("machine evidence JSON is invalid") from exc
    if not isinstance(payload, dict) or payload.get("schema") != "aiw.fd.evidence.v1":
        raise FDError("machine evidence requires schema aiw.fd.evidence.v1")
    if payload.get("human_report") != Path(artifact_ref).name or not isinstance(payload.get("data"), dict):
        raise FDError("machine evidence must cross-reference its human report and data")
    if expected_kind and payload.get("kind") != expected_kind:
        raise FDError("machine evidence kind differs from the handoff")
    if fd_name and payload.get("fd_id") != fd_name:
        raise FDError("machine evidence FD differs from the handoff")
    if source_event and payload.get("source_event") != source_event:
        raise FDError("machine evidence source event differs from the handoff")
    return payload


def labelled_fields(base: Path, artifact_ref: str) -> dict[str, str]:
    content = (base / artifact_ref).read_text(encoding="utf-8")
    if DATA_REF_RE.search(content):
        payload = structured_evidence(base, artifact_ref)
        fields = {}
        for key, value in payload["data"].items():
            if not isinstance(key, str) or not re.fullmatch(r"[a-z][a-z0-9_]*", key):
                raise FDError("machine evidence data keys must be snake_case")
            label = " ".join(
                {"fd": "FD", "pm": "PM"}.get(
                    part, part.capitalize() if index == 0 else part)
                for index, part in enumerate(key.split("_")))
            if isinstance(value, (str, int, float)) and not isinstance(value, bool):
                fields[label] = str(value)
            elif isinstance(value, (list, dict)):
                fields[label] = json.dumps(value, ensure_ascii=False)
            else:
                raise FDError(f"machine evidence field {key} has an unsupported value")
        return fields
    pairs = re.findall(r"(?m)^\*\*([^*]+):\*\*[ \t]*(.+?)[ \t]*$", content)
    if len(pairs) != len({name for name, _ in pairs}):
        raise FDError("test evidence has duplicate labelled fields")
    return {name: value.strip() for name, value in pairs}


def required_field(fields: dict[str, str], name: str) -> str:
    value = fields.get(name, "").strip()
    if not value:
        raise FDError(f"test evidence requires **{name}:**")
    return value


def coverage_value(raw: str) -> float | None:
    if raw in {"not measured", "not applicable"}:
        return None
    if not raw.endswith("%"):
        raise FDError("coverage must be a percentage, not measured, or not applicable")
    try:
        value = float(raw[:-1])
    except ValueError as exc:
        raise FDError("coverage percentage is invalid") from exc
    if not 0 <= value <= 100:
        raise FDError("coverage percentage must be between 0 and 100")
    return value


def validate_test_authorization(base: Path, raw_ref: str, command: str,
                                previous: dict) -> None:
    ref = safe_artifact(base, raw_ref)
    if dual_evidence((base / previous["fd_path"]).read_text(encoding="utf-8")):
        structured_evidence(base, ref, "planner-authorization",
                            previous["fd_id"], previous["event_id"])
    fields = labelled_fields(base, ref)
    for name, expected in (("Decision", "approved"),
                           ("Implementation event", previous["event_id"]),
                           ("FD revision", str(previous["fd_revision"])),
                           ("FD digest", previous["fd_sha256"]),
                           ("Tester session", previous.get("session_ref")
                            or str(previous.get("pid", "")))):
        if required_field(fields, name) != expected:
            raise FDError(f"test authorization {name} differs from the Tester handoff")
    if required_field(fields, "Command") != command:
        raise FDError("test report command differs from Planner authorization")
    basis = required_field(fields, "Basis")
    human = required_field(fields, "Human approval")
    if basis == "planner-low-risk":
        if human != "not required":
            raise FDError("low-risk Planner approval must state human approval is not required")
    elif basis == "human-approved":
        match = re.fullmatch(
            r"approved:(?:conversation|user-message|ticket|approval-record):"
            r"([A-Za-z0-9][A-Za-z0-9._:/-]{0,199})", human)
        if not match or match.group(1).casefold() in {
                "none", "pending", "denied", "rejected", "not-required"}:
            raise FDError("human-approved test authorization requires an affirmative approval reference")
    else:
        raise FDError("test authorization basis must be planner-low-risk or human-approved")
    for name in ("Working directory", "Scope", "Expected duration", "Side effects",
                 "Risk review", "Planner identity", "Decision time"):
        required_field(fields, name)
    try:
        decision_time = datetime.fromisoformat(required_field(fields, "Decision time"))
    except ValueError as exc:
        raise FDError("test authorization time must be ISO 8601") from exc
    if decision_time.tzinfo is None:
        raise FDError("test authorization time needs a timezone")


def validate_test_report(base: Path, artifact_ref: str, previous: dict) -> dict:
    fields = labelled_fields(base, artifact_ref)
    for name, expected in (("Implementation event", previous["event_id"]),
                           ("Tested FD revision", str(previous["fd_revision"])),
                           ("Tested FD digest", previous["fd_sha256"]),
                           ("Tester session", previous.get("session_ref")
                            or str(previous.get("pid", "")))):
        if required_field(fields, name) != expected:
            raise FDError(f"test report {name} differs from its claimed handoff")
    counts = {}
    for name in ("Applicable scenarios", "Covered scenarios",
                 "Executed behavior tests", "Passed behavior tests",
                 "Failed behavior tests", "Unrun behavior tests"):
        raw = required_field(fields, name)
        if not raw.isdecimal():
            raise FDError(f"test report {name} must be a nonnegative count")
        counts[name] = int(raw)
    if (counts["Covered scenarios"] > counts["Applicable scenarios"]
            or counts["Passed behavior tests"] + counts["Failed behavior tests"]
            != counts["Executed behavior tests"]):
        raise FDError("test report scenario or executed-test counts are inconsistent")
    requirements = required_field(fields, "Requirements coverage")
    branches = required_field(fields, "Branch coverage")
    requirement_percent = coverage_value(requirements)
    coverage_value(branches)
    if branches in {"not measured", "not applicable"}:
        required_field(fields, "Coverage unavailable reason")
    applicable = counts["Applicable scenarios"]
    if applicable:
        expected_percent = counts["Covered scenarios"] * 100 / applicable
        if requirement_percent is None or abs(requirement_percent - expected_percent) > 0.5:
            raise FDError("requirements coverage differs from scenario counts")
    elif requirements != "not applicable" or not required_field(fields, "Not applicable reason"):
        raise FDError("zero applicable scenarios require a not applicable reason")
    if dual_evidence((base / previous["fd_path"]).read_text(encoding="utf-8")):
        data = structured_evidence(base, artifact_ref, "tester-report",
                                   previous["fd_id"], previous["event_id"])["data"]
        scenarios = data.get("scenarios")
        if not isinstance(scenarios, list) or len(scenarios) != applicable:
            raise FDError("machine scenario inventory differs from applicable count")
        ids = set()
        passed_scenarios = 0
        for scenario in scenarios:
            if not isinstance(scenario, dict):
                raise FDError("machine scenario inventory entries must be objects")
            ident = scenario.get("id")
            behavior = scenario.get("behavior")
            state = scenario.get("status")
            if (not isinstance(ident, str) or not ident.strip() or ident in ids
                    or not isinstance(behavior, str) or not behavior.strip()
                    or state not in {"passed", "uncovered", "blocked"}):
                raise FDError("machine scenario requires unique id, behavior, and status")
            ids.add(ident)
            if state == "passed":
                cases = scenario.get("test_cases")
                if not isinstance(cases, list) or not cases or not all(
                        isinstance(case, str) and case.strip() for case in cases):
                    raise FDError("passed machine scenario requires executed test cases")
                passed_scenarios += 1
        if passed_scenarios != counts["Covered scenarios"]:
            raise FDError("machine scenario statuses differ from covered count")
    for name in ("Raw coverage evidence", "Test files", "Commands",
                 "Recommendation", "Residual risk"):
        required_field(fields, name)
    if counts["Executed behavior tests"] or branches not in {"not measured", "not applicable"}:
        raw_commands = required_field(fields, "Commands")
        raw_records = required_field(fields, "Authorization records")
        try:
            commands = json.loads(raw_commands) if raw_commands.startswith("[") else [raw_commands]
            records = json.loads(raw_records) if raw_records.startswith("[") else [raw_records]
        except json.JSONDecodeError as exc:
            raise FDError("test commands and authorization records must be valid JSON arrays") from exc
        if (not isinstance(commands, list) or not isinstance(records, list)
                or not commands or len(commands) != len(records)
                or not all(isinstance(item, str) and item.strip() for item in commands + records)):
            raise FDError("each executed command requires one Planner authorization record")
        if len(set(records)) != len(records) or "none" in commands:
            raise FDError("each executed command requires its own authorization record")
        for command, record in zip(commands, records):
            validate_test_authorization(base, record, command, previous)
    recommendation = required_field(fields, "Recommendation")
    if recommendation not in {"pass", "fail", "blocked"}:
        raise FDError("test recommendation must be pass, fail, or blocked")
    if recommendation == "pass" and counts["Failed behavior tests"]:
        raise FDError("failed behavior tests cannot have a pass recommendation")
    return {"requirements_coverage": requirements, "branch_coverage": branches,
            "failed_behavior_tests": counts["Failed behavior tests"]}


def validate_test_risk_assessment(base: Path, raw_ref: str, previous: dict,
                                  excluded_sessions: set[str],
                                  expected_focus: str = "") -> tuple[str, str]:
    ref = safe_artifact(base, raw_ref)
    path = Path(ref)
    if (path.parent.as_posix() != "docs/features/reports"
            or not path.name.startswith(previous["fd_id"] + "-")):
        raise FDError("risk assessment must be an FD-prefixed report")
    structured_evidence(base, ref, "test-risk-assessment",
                        previous["fd_id"], previous["event_id"])
    fields = labelled_fields(base, ref)
    for name, expected in (("Tester report", previous["artifact_ref"]),
                           ("FD revision", str(previous["fd_revision"])),
                           ("FD digest", previous["fd_sha256"])):
        if required_field(fields, name) != expected:
            raise FDError(f"risk assessment {name} differs from Tester evidence")
    session = required_field(fields, "Assessor session")
    if (not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._:/-]{0,199}", session)
            or session in excluded_sessions):
        raise FDError("risk assessors need distinct sessions from PM, Worker, and Tester")
    vote = required_field(fields, "Vote")
    if vote not in {"accept-with-risk", "repair"}:
        raise FDError("risk assessment vote must be accept-with-risk or repair")
    if expected_focus and required_field(fields, "Assessment focus") != expected_focus:
        raise FDError(f"risk assessment focus must be {expected_focus}")
    for name in ("Severity", "Impact scope", "Estimated repair time",
                 "Delivery impact", "Rationale", "Residual risk", "Uncertainty"):
        required_field(fields, name)
    try:
        assessed_at = datetime.fromisoformat(required_field(fields, "Assessment time"))
    except ValueError as exc:
        raise FDError("risk assessment time must be ISO 8601") from exc
    if assessed_at.tzinfo is None:
        raise FDError("risk assessment time needs a timezone")
    return session, vote


def validate_test_decision(base: Path, artifact_ref: str, previous: dict,
                           disposition: str) -> list[str]:
    fields = labelled_fields(base, artifact_ref)
    for name, expected in (("Disposition", disposition),
                           ("Tester report", previous["artifact_ref"]),
                           ("FD revision", str(previous["fd_revision"])),
                           ("FD digest", previous["fd_sha256"]),
                           ("Requirements coverage", previous["test_summary"]["requirements_coverage"]),
                           ("Branch coverage", previous["test_summary"]["branch_coverage"]),
                           ("Failed behavior tests", str(previous["test_summary"]["failed_behavior_tests"]))):
        if required_field(fields, name) != expected:
            raise FDError(f"PM decision {name} differs from Tester evidence")
    for name in ("Rationale", "Exceptions", "Residual risk", "PM identity", "Decision time"):
        required_field(fields, name)
    try:
        decision_time = datetime.fromisoformat(required_field(fields, "Decision time"))
    except ValueError as exc:
        raise FDError("PM decision time must be ISO 8601") from exc
    if decision_time.tzinfo is None:
        raise FDError("PM decision time needs a timezone")
    try:
        assessments = json.loads(required_field(fields, "Assessments"))
    except json.JSONDecodeError as exc:
        raise FDError("PM assessments must be a JSON array") from exc
    policy = fields.get("Assessment policy", "").strip()
    failed_tests = previous["test_summary"]["failed_behavior_tests"]
    if policy == "":
        # Existing in-flight three-assessor decisions keep their original format.
        mode = "legacy-three"
        expected_count = 3
    elif policy == "adaptive-v1":
        mode = required_field(fields, "Assessment mode")
        if mode not in {"single", "escalated"}:
            raise FDError("PM assessment mode must be single or escalated")
        reason = required_field(fields, "Escalation reason")
        required_field(fields, "Escalation detail")
        gap = required_field(fields, "Coverage gap disposition")
        required_field(fields, "Coverage gap reason")
        if gap not in {"bounded", "material"}:
            raise FDError("PM coverage gap disposition must be bounded or material")
        if mode == "single":
            if failed_tests or gap != "bounded" or reason != "none":
                raise FDError("single assessment requires no failed tests, a bounded gap, and no escalation")
            expected_count = 1
        else:
            if reason not in {"failed-tests", "material-evidence-gap", "pm-disagreement"}:
                raise FDError("escalated assessment needs a supported reason")
            if reason == "failed-tests" and not failed_tests:
                raise FDError("failed-tests escalation requires a failed behavior test")
            if reason == "material-evidence-gap" and gap != "material":
                raise FDError("material-evidence-gap escalation requires a material gap")
            expected_count = 3
    else:
        raise FDError("unsupported PM assessment policy")
    if (not isinstance(assessments, list) or len(assessments) != expected_count
            or not all(isinstance(ref, str) and ref.strip() for ref in assessments)
            or len({ref.casefold() for ref in assessments}) != expected_count):
        raise FDError(f"PM decision requires {expected_count} distinct risk assessment reports")
    excluded = {previous["worker_session_ref"], previous["tester_session_ref"],
                required_field(fields, "PM identity")}
    sessions = []
    votes = []
    focuses = ("acceptance-impact", "technical-repair", "delivery-operations")
    for index, ref in enumerate(assessments):
        focus = focuses[index] if policy == "adaptive-v1" else ""
        session, vote = validate_test_risk_assessment(base, ref, previous, excluded, focus)
        excluded.add(session)
        sessions.append(session)
        votes.append(vote)
    accept_votes = votes.count("accept-with-risk")
    repair_votes = votes.count("repair")
    for name, expected in (("Accept votes", accept_votes), ("Repair votes", repair_votes)):
        raw = required_field(fields, name)
        if not raw.isdecimal() or int(raw) != expected:
            raise FDError(f"PM decision {name} differs from risk assessments")
    required_accepts = 1 if mode == "single" else 2
    result = "accepted" if accept_votes >= required_accepts else "rejected"
    if disposition != result:
        raise FDError("PM disposition differs from the assessor vote result")
    return sessions


def update_index(base: Path) -> None:
    with fd_lock(base, "_index"):
        write_index(base)


def write_index(base: Path) -> None:
    active, completed, parked = [], [], []
    for path in all_files(base):
        content = path.read_text(encoding="utf-8")
        state = status(content)
        priority = PRIORITY_RE.search(content)
        rank = priority.group(1).strip() if priority else "-"
        link = path.relative_to(feature_dir(base)).as_posix()
        display_title = title(content).replace("|", "\\|")
        row = f"| [{path.name.split('_', 1)[0]}]({link}) | {display_title} | {state} | {rank} |"
        if state == "Complete":
            completed.append(row)
        elif state in {"Deferred", "Closed"}:
            parked.append(row)
        else:
            active.append(row)
    def section(name: str, rows: list[str]) -> str:
        return f"## {name}\n\n| FD | Title | Status | Priority |\n| --- | --- | --- | --- |\n" + "\n".join(rows or ["| - | - | - | - |"])
    content = ("# Feature Design Index\n\nFD 文件是设计与进度的主记录；本索引可由 `aiw fd` 重建。\n\n"
               + section("Active Features", active) + "\n\n"
               + section("Completed", completed) + "\n\n"
               + section("Deferred / Closed", parked) + "\n")
    atomic_text(feature_dir(base) / "FEATURE_INDEX.md", content)


def ensure_ready(content: str, event_type: str) -> None:
    if event_type == "design-ready":
        for heading in ("## Problem", "## Solution", "## Work items", "## Acceptance", "## Verification"):
            if heading not in content:
                raise FDError(f"design-ready requires {heading}")
        if "%% NEEDS_INPUT" in content:
            raise FDError("design-ready cannot leave unresolved %% NEEDS_INPUT annotations")
        if not ITEM_RE.search(content) or "Define the first independently reviewable outcome" in content:
            raise FDError("design-ready requires numbered Work Items")
    if event_type in {"implementation-ready", "verification-passed"}:
        items = ITEM_RE.findall(content)
        if not items or any(mark == " " for mark, _, _ in items):
            raise FDError("all FD Work Items must be completed or explicitly cancelled")
        if "%% NEEDS_INPUT" in content:
            raise FDError("FD has unresolved %% NEEDS_INPUT annotations")


def dispatch(base: Path, path: Path, event: dict) -> None:
    role = event["target_role"]
    if role in {"human", "pm"}:
        print(f"handoff {event['event_id']} awaits {role}; no agent dispatched")
        return
    runner = os.environ.get("AIW_FD_ROLE_RUNNER", "").strip()
    if not runner:
        print(f"handoff {event['event_id']} pending {role}; "
              f"claim with: aiw fd claim {event['fd_id']} {event['event_id']} --session <host-session-id>")
        return
    executable = Path(runner).expanduser().resolve()
    if not executable.is_file():
        raise FDError(f"role runner does not exist: {runner}")
    name = event["fd_id"]
    with fd_lock(base, name):
        current = json.loads(path.read_text(encoding="utf-8"))
        latest = latest_event(base, name)
        if current["dispatch_state"] != "pending" or not latest or latest[0] != path:
            raise FDError("handoff changed before dispatch; inspect the latest event")
        fd_path = resolve_fd(base, name)
        fd_content = fd_path.read_bytes()
        if revision(fd_content.decode("utf-8")) != current["fd_revision"] or fd_digest(fd_content) != current["fd_sha256"]:
            raise FDError("FD changed since the handoff; reconcile before dispatch")
        current["dispatch_state"] = "launching"
        atomic_json(path, current)
    log_path = path.with_suffix(".log")
    try:
        with log_path.open("ab") as log:
            process = subprocess.Popen([str(executable), role, event["fd_path"], str(path)],
                                       cwd=base, stdin=subprocess.DEVNULL, stdout=log,
                                       stderr=subprocess.STDOUT, shell=False)
    except OSError as exc:
        with fd_lock(base, name):
            current = json.loads(path.read_text(encoding="utf-8"))
            if current["dispatch_state"] == "launching":
                current["dispatch_state"] = "pending"
                current["error"] = str(exc)
                atomic_json(path, current)
        raise FDError(f"could not start role runner: {exc}") from exc
    with fd_lock(base, name):
        current = json.loads(path.read_text(encoding="utf-8"))
        if current["dispatch_state"] == "launching":
            current["dispatch_state"] = "dispatched"
            current["pid"] = process.pid
            atomic_json(path, current)
    print(f"dispatched {role} for {event['event_id']} (pid {process.pid}); log: {log_path}")


def emit(base: Path, name: str, kind: str, producer: str, artifact: str,
         source_event: str = "") -> None:
    with fd_lock(base, name):
        target, event = prepare_event(base, name, kind, producer, artifact, source_event)
    dispatch(base, target, event)


def request_review(base: Path, name: str, reason: str) -> None:
    if not reason.strip() or len(reason) > 500 or any(char in reason for char in "\r\n"):
        raise FDError("request-review requires a one-line --reason of at most 500 characters")
    with fd_lock(base, name):
        fd_path = resolve_fd(base, name)
        content = fd_path.read_text(encoding="utf-8")
        if not is_archived_fd(base, fd_path):
            if status(content) != "Pending Verification":
                raise FDError("active request-review requires a Pending Verification FD")
            latest = latest_event(base, name)
            if latest and latest[1]["dispatch_state"] in {"launching", "dispatched"}:
                raise FDError(f"{latest[1]['event_id']} is in flight; reconcile its session before requesting review")
            digest = fd_digest(content)
            if (latest and latest[1]["dispatch_state"] == "pending"
                    and latest[1]["target_role"] == "reviewer"
                    and latest[1]["fd_revision"] == revision(content)
                    and latest[1]["fd_sha256"] == digest):
                raise FDError(f"Reviewer handoff {latest[1]['event_id']} is already pending; claim or resume it")
            new_revision = revision(content) + 1
            updated = REVISION_RE.sub(f"**Revision:** {new_revision}", content, count=1)
            active_ref = fd_path.relative_to(base).as_posix()
            event_id = f"{name}-{new_revision:06d}-review-requested"
            target = runtime_dir(base, name) / "events" / f"{new_revision:06d}-review-requested.json"
            if target.exists():
                raise FDError(f"event already exists: {event_id}")
            event = {"event_id": event_id, "fd_id": name, "event_type": "review-requested",
                     "fd_revision": new_revision, "producer": "pm",
                     "fd_sha256": fd_digest(updated),
                     "artifact_ref": active_ref, "fd_path": active_ref,
                     "target_role": "reviewer", "dispatch_state": "pending",
                     "reason": reason.strip(), "created_at": datetime.now(timezone.utc).isoformat()}
            if latest:
                event["supersedes"] = latest[1]["event_id"]
            atomic_json(target, event)
            atomic_text(fd_path, updated)
            if latest and latest[1]["dispatch_state"] == "pending":
                previous_path, previous_event = latest
                previous_event["dispatch_state"] = "cancelled"
                previous_event["superseded_by"] = event_id
                atomic_json(previous_path, previous_event)
            update_index(base)
        else:
            if status(content) != "Complete":
                raise FDError("request-review requires an archived Complete FD")
            latest = latest_event(base, name)
            if (not latest or latest[1].get("event_type") != "verification-passed"
                    or latest[1].get("producer") != "reviewer"
                    or latest[1].get("forced")
                    or latest[1].get("dispatch_state") != "acknowledged"):
                raise FDError("request-review requires an acknowledged Reviewer verification-passed event")

            active_path = feature_dir(base) / fd_path.name
            if active_path.exists() or active_path.is_symlink():
                raise FDError(f"active FD path already exists: {active_path}")
            new_revision = revision(content) + 1
            updated = STATUS_RE.sub("**Status:** Pending Verification", content, count=1)
            updated = REVISION_RE.sub(f"**Revision:** {new_revision}", updated, count=1)
            updated = re.sub(r"(?m)^\*\*Completed:\*\*", "**Previously completed:**", updated, count=1)
            active_ref = active_path.relative_to(base).as_posix()
            event_id = f"{name}-{new_revision:06d}-review-requested"
            target = runtime_dir(base, name) / "events" / f"{new_revision:06d}-review-requested.json"
            if target.exists():
                raise FDError(f"event already exists: {event_id}")
            event = {"event_id": event_id, "fd_id": name, "event_type": "review-requested",
                     "fd_revision": new_revision, "producer": "pm",
                     "fd_sha256": fd_digest(updated),
                     "artifact_ref": active_ref, "fd_path": active_ref,
                     "target_role": "reviewer", "dispatch_state": "pending",
                     "reason": reason.strip(), "created_at": datetime.now(timezone.utc).isoformat()}
            # Move first so a failed receipt write leaves one unambiguous Complete FD.
            fd_path.replace(active_path)
            atomic_json(target, event)
            atomic_text(active_path, updated)
            latest_path, previous_event = latest
            previous_event["review_requested_by"] = event_id
            atomic_json(latest_path, previous_event)
            update_index(base)
    dispatch(base, target, event)


def refresh_worker(base: Path, name: str, reason: str) -> None:
    if not reason.strip() or len(reason) > 500 or any(char in reason for char in "\r\n"):
        raise FDError("refresh-worker requires a one-line --reason of at most 500 characters")
    with fd_lock(base, name):
        fd_path = resolve_fd(base, name)
        if fd_path.parent != feature_dir(base):
            raise FDError("refresh-worker requires an active FD")
        original = fd_path.read_text(encoding="utf-8")
        if status(original) not in {"Open", "In Progress"}:
            raise FDError("refresh-worker requires an Open or In Progress FD")
        item = latest_event(base, name)
        if not item:
            raise FDError("refresh-worker requires a pending Worker handoff")
        old_path, old_event = item
        fd_ref = fd_path.relative_to(base).as_posix()
        if (old_event.get("fd_id") != name or old_event.get("fd_path") != fd_ref
                or old_event.get("target_role") != "worker"
                or old_event.get("dispatch_state") != "pending"):
            raise FDError("refresh-worker requires the latest pending Worker handoff")
        if (old_event["fd_revision"] == revision(original)
                and old_event["fd_sha256"] == fd_digest(original)):
            raise FDError("Worker handoff already matches the current FD")
        new_revision = revision(original) + 1
        if new_revision <= old_event["fd_revision"]:
            raise FDError("FD revision is behind the Worker handoff; reconcile manually")
        updated = REVISION_RE.sub(f"**Revision:** {new_revision}", original, count=1)
        event_id = f"{name}-{new_revision:06d}-work-requested"
        target = runtime_dir(base, name) / "events" / f"{new_revision:06d}-work-requested.json"
        if target.exists() or target.is_symlink():
            raise FDError(f"event already exists: {event_id}")
        event = {"event_id": event_id, "fd_id": name,
                 "event_type": "work-requested", "fd_revision": new_revision,
                 "producer": "pm", "fd_sha256": fd_digest(updated),
                 "artifact_ref": fd_ref, "fd_path": fd_ref,
                 "target_role": "worker", "dispatch_state": "pending",
                 "reason": reason.strip(), "supersedes": old_event["event_id"],
                 "created_at": datetime.now(timezone.utc).isoformat()}
        cancelled = dict(old_event)
        cancelled["dispatch_state"] = "cancelled"
        cancelled["superseded_by"] = event_id
        try:
            # Keep the new receipt unclaimable until all other writes succeed.
            atomic_json(target, {**event, "dispatch_state": "preparing"})
            atomic_text(fd_path, updated)
            atomic_json(old_path, cancelled)
            update_index(base)
            atomic_json(target, event)
        except (OSError, FDError) as exc:
            rollback_errors = []
            for restore in (lambda: target.unlink(missing_ok=True),
                            lambda: atomic_text(fd_path, original),
                            lambda: atomic_json(old_path, old_event)):
                try:
                    restore()
                except OSError as rollback_error:
                    rollback_errors.append(str(rollback_error))
            if rollback_errors:
                raise FDError("refresh-worker rollback incomplete: "
                              + "; ".join(rollback_errors)) from exc
            raise
    dispatch(base, target, event)


def recover_worker(base: Path, name: str, expected_event: str,
                   expected_session: str, reason: str) -> None:
    if not reason.strip() or len(reason) > 500 or any(char in reason for char in "\r\n"):
        raise FDError("recover-worker requires a one-line --reason of at most 500 characters")
    with fd_lock(base, name):
        fd_path = resolve_fd(base, name)
        if fd_path.parent != feature_dir(base):
            raise FDError("recover-worker requires an active FD")
        original = fd_path.read_text(encoding="utf-8")
        if status(original) not in {"Open", "In Progress"}:
            raise FDError("recover-worker requires an Open or In Progress FD")
        item = latest_event(base, name)
        if not item:
            raise FDError("recover-worker requires a dispatched Worker handoff")
        old_path, old_event = item
        fd_ref = fd_path.relative_to(base).as_posix()
        if (old_event.get("fd_id") != name or old_event.get("fd_path") != fd_ref
                or old_event.get("event_id") != expected_event
                or old_event.get("target_role") != "worker"
                or old_event.get("dispatch_state") != "dispatched"
                or old_event.get("session_ref") != expected_session):
            raise FDError("latest dispatched Worker event and session must match expectations")
        current_revision = revision(original)
        if current_revision < old_event["fd_revision"]:
            raise FDError("FD revision is behind the Worker handoff; reconcile manually")
        if (current_revision == old_event["fd_revision"]
                and fd_digest(original) != old_event["fd_sha256"]):
            raise FDError("FD changed without a revision; reconcile manually")
        new_revision = current_revision + 1
        updated = REVISION_RE.sub(f"**Revision:** {new_revision}", original, count=1)
        event_id = f"{name}-{new_revision:06d}-work-requested"
        target = runtime_dir(base, name) / "events" / f"{new_revision:06d}-work-requested.json"
        if target.exists() or target.is_symlink():
            raise FDError(f"event already exists: {event_id}")
        index_path = feature_dir(base) / "FEATURE_INDEX.md"
        original_index = index_path.read_bytes() if index_path.exists() else None
        now = datetime.now(timezone.utc).isoformat()
        event = {"event_id": event_id, "fd_id": name,
                 "event_type": "work-requested", "fd_revision": new_revision,
                 "producer": "pm", "fd_sha256": fd_digest(updated),
                 "artifact_ref": fd_ref, "fd_path": fd_ref,
                 "target_role": "worker", "dispatch_state": "pending",
                 "reason": reason.strip(), "supersedes": old_event["event_id"],
                 "abandoned_session_ref": expected_session, "created_at": now}
        cancelled = {**old_event, "dispatch_state": "cancelled",
                     "superseded_by": event_id, "recovery_reason": reason.strip(),
                     "recovered_at": now}
        try:
            atomic_json(target, {**event, "dispatch_state": "preparing"})
            atomic_text(fd_path, updated)
            atomic_json(old_path, cancelled)
            update_index(base)
            atomic_json(target, event)
        except Exception as exc:
            rollback_errors = []
            for label, restore in (("new event", lambda: target.unlink(missing_ok=True)),
                                   ("FD", lambda: atomic_text(fd_path, original)),
                                   ("old receipt", lambda: atomic_json(old_path, old_event)),
                                   ("index", lambda: atomic_bytes(index_path, original_index)
                                    if original_index is not None else index_path.unlink(missing_ok=True))):
                try:
                    restore()
                except Exception as rollback_error:
                    rollback_errors.append(f"{label}: {type(rollback_error).__name__}: {rollback_error}")
            if rollback_errors:
                raise FDError("recover-worker rollback incomplete: "
                              + "; ".join(rollback_errors)) from exc
            raise
    dispatch(base, target, event)


def force_identity(reason: str, operator: str) -> None:
    for label, value, limit in (("reason", reason, 500), ("operator", operator, 200)):
        if (not value.strip() or len(value) > limit
                or any(ord(char) < 32 or ord(char) == 127 for char in value)):
            raise FDError(f"--{label} requires non-empty single-line text of at most {limit} characters")


def force_context(base: Path, name: str, action: str, reason: str,
                  operator: str) -> tuple[Path, str, tuple[Path, dict] | None, Path, dict]:
    force_identity(reason, operator)
    fd_path = resolve_fd(base, name)
    if is_archived_fd(base, fd_path):
        raise FDError("force operations require an active FD; use reopen or request-review for archives")
    content = fd_path.read_text(encoding="utf-8")
    revision(content)
    if status(content) not in DEFINED_STATUSES:
        raise FDError("force operations require a defined current FD status")
    previous = latest_event(base, name)
    now = datetime.now(timezone.utc).isoformat()
    audit_path = runtime_dir(base, name) / "operations" / f"{uuid.uuid4().hex}.json"
    audit = {"schema": "aiw.fd.operation.v1", "fd_id": name, "action": action,
             "forced": True, "operator": operator.strip(), "local_user": getpass.getuser(),
             "reason": reason.strip(), "created_at": now,
             "fd_path": fd_path.relative_to(base).as_posix(),
             "previous_status": status(content), "previous_revision": revision(content),
             "previous_event": previous[1] if previous else None}
    return fd_path, content, previous, audit_path, audit


def force_transaction(base: Path, changes: dict[Path, bytes],
                      refresh_index: bool = False) -> None:
    """Commit one locked recovery operation; restore every attempted write on failure."""
    index_path = feature_dir(base) / "FEATURE_INDEX.md"
    paths = list(changes)
    if refresh_index:
        paths.append(index_path)
    originals = {}
    for path in paths:
        if path.is_symlink() or (path.exists() and not path.is_file()):
            raise FDError(f"unsafe recovery destination: {path}")
        originals[path] = path.read_bytes() if path.exists() else None
    attempted = []
    try:
        for path, value in changes.items():
            attempted.append(path)
            atomic_bytes(path, value)
        if refresh_index:
            attempted.append(index_path)
            update_index(base)
    except Exception as exc:
        errors = []
        for path in reversed(attempted):
            try:
                if originals[path] is None:
                    path.unlink(missing_ok=True)
                else:
                    atomic_bytes(path, originals[path])
            except Exception as rollback_error:
                errors.append(f"{path}: {type(rollback_error).__name__}: {rollback_error}")
        if errors:
            raise FDError("force operation rollback incomplete: " + "; ".join(errors)) from exc
        raise


def json_bytes(value: dict) -> bytes:
    return (json.dumps(value, ensure_ascii=False, indent=2) + "\n").encode("utf-8")


def cancel_event(base: Path, name: str, expected_event: str, reason: str,
                 operator: str) -> None:
    with fd_lock(base, name):
        _, _, previous, audit_path, audit = force_context(
            base, name, "cancel-event", reason, operator)
        if (not previous or previous[1].get("event_id") != expected_event
                or previous[1].get("fd_id") != name
                or previous[1].get("dispatch_state") != "dispatched"):
            raise FDError("cancel-event requires the exact latest dispatched event ID")
        old_path, old_event = previous
        audit_ref = audit_path.relative_to(runtime_dir(base, name)).as_posix()
        cancelled = {**old_event, "dispatch_state": "cancelled",
                     "cancelled_at": audit["created_at"], "cancellation_reason": reason.strip(),
                     "cancelled_by": operator.strip(), "cancellation_operation": audit_ref}
        audit.update({"result_status": audit["previous_status"],
                      "result_event": cancelled, "skipped_checks": ["agent-session-stopped"],
                      "agent_stopped": False})
        force_transaction(base, {audit_path: json_bytes(audit), old_path: json_bytes(cancelled)})
    print(f"cancelled {expected_event}; receipt only, Agent has NOT been stopped; audit: {audit_path}")


def set_status(base: Path, name: str, outcome: str, reason: str, operator: str) -> None:
    if outcome not in DEFINED_STATUSES:
        raise FDError(f"undefined FD status: {outcome}")
    with fd_lock(base, name):
        fd_path, content, _, audit_path, audit = force_context(
            base, name, "set-status", reason, operator)
        updated = STATUS_RE.sub(f"**Status:** {outcome}", content, count=1)
        new_revision = revision(content) + 1
        updated = REVISION_RE.sub(f"**Revision:** {new_revision}", updated, count=1)
        audit.update({"result_status": outcome, "result_revision": new_revision,
                      "result_fd_sha256": fd_digest(updated),
                      "skipped_checks": ["status-transition", "terminal-evidence"],
                      "review_verified": False})
        force_transaction(base, {audit_path: json_bytes(audit), fd_path: updated.encode("utf-8")},
                          refresh_index=True)
    print(f"forced status {audit['previous_status']} -> {outcome}; transition/evidence checks skipped; "
          f"no review evidence created, receipts unchanged and may be stale; audit: {audit_path}")


def force_emit(base: Path, name: str, kind: str, producer: str, artifact: str,
               reason: str, operator: str) -> None:
    if kind not in ALLOWED:
        raise FDError(f"undefined event: {kind}")
    expected = {**ROLE_PRODUCERS, "design-requested": "pm",
                "decision-recorded": "human"}.get(kind)
    roles = {"pm", "planner", "worker", "tester", "reviewer", "human"}
    if producer not in roles or (expected and producer != expected):
        raise FDError(f"{kind} requires {expected or 'a defined producer role'}")
    with fd_lock(base, name):
        fd_path, content, previous, audit_path, audit = force_context(
            base, name, "force-emit", reason, operator)
        artifact_ref = safe_artifact(base, artifact)
        _, new_status, target_role = ALLOWED[kind]
        if kind == "implementation-ready" and independent_testing(content):
            new_status, target_role = "Pending Test", "tester"
        if target_role == "resume":
            target_role = {"Planned": "planner", "Design": "planner", "Open": "worker",
                           "In Progress": "worker", "Pending Test": "tester",
                           "Pending Test Acceptance": "pm", "Pending Verification": "reviewer",
                           "Complete": "pm", "Deferred": "pm", "Closed": "pm"}[status(content)]
        # FD revisions can lag behind older receipts after manual recovery.
        recorded_revisions = [json.loads(path.read_text(encoding="utf-8"))["fd_revision"]
                              for path in event_paths(base, name)]
        new_revision = max([revision(content), *recorded_revisions]) + 1
        event_id = f"{name}-{new_revision:06d}-{kind}"
        target = runtime_dir(base, name) / "events" / f"{new_revision:06d}-{kind}.json"
        if target.exists() or target.is_symlink():
            raise FDError(f"event already exists: {event_id}")
        updated = REVISION_RE.sub(f"**Revision:** {new_revision}", content, count=1)
        if new_status:
            updated = STATUS_RE.sub(f"**Status:** {new_status}", updated, count=1)
        audit_ref = audit_path.relative_to(runtime_dir(base, name)).as_posix()
        event = {"event_id": event_id, "fd_id": name, "event_type": kind,
                 "fd_revision": new_revision, "producer": producer,
                 "fd_sha256": fd_digest(updated), "artifact_ref": artifact_ref,
                 "fd_path": fd_path.relative_to(base).as_posix(), "target_role": target_role,
                 "dispatch_state": "pending", "created_at": audit["created_at"],
                 "forced": True, "reason": reason.strip(), "operator": operator.strip(),
                 "force_operation": audit_ref}
        if previous:
            event["supersedes"] = previous[1]["event_id"]
            event["previous_session_ref"] = previous[1].get("session_ref")
        audit.update({"result_status": status(updated), "result_revision": new_revision,
                      "result_event": event, "review_verified": False,
                      "skipped_checks": ["status-transition", "work-items", "needs-input",
                                         "evidence", "previous-handoff", "claim", "session-independence"],
                      "agent_stopped": False})
        # The audit and FD/index are committed before making the event claimable.
        changes = {audit_path: json_bytes(audit), fd_path: updated.encode("utf-8")}
        if previous and previous[1].get("dispatch_state") in {"pending", "launching", "dispatched", "preparing"}:
            old_path, old_event = previous
            cancelled = {**old_event, "dispatch_state": "cancelled", "superseded_by": event_id,
                         "cancelled_at": audit["created_at"], "cancellation_reason": reason.strip(),
                         "cancelled_by": operator.strip(), "cancellation_operation": audit_ref}
            changes[old_path] = json_bytes(cancelled)
        # A preparing receipt is never eligible for claim/dispatch.
        changes[target] = json_bytes({**event, "dispatch_state": "preparing"})
        index_path = feature_dir(base) / "FEATURE_INDEX.md"
        originals = {path: path.read_bytes() if path.exists() else None
                     for path in [*changes, index_path]}
        force_transaction(base, changes, refresh_index=True)
        try:
            atomic_json(target, event)
        except Exception as exc:
            errors = []
            # Remove/restore the receipt first so failed recovery cannot be claimed.
            for path in [target, *reversed([path for path in originals if path != target])]:
                try:
                    if originals[path] is None:
                        path.unlink(missing_ok=True)
                    else:
                        atomic_bytes(path, originals[path])
                except Exception as rollback_error:
                    errors.append(f"{path}: {type(rollback_error).__name__}: {rollback_error}")
            if errors:
                raise FDError("force-emit rollback incomplete: " + "; ".join(errors)) from exc
            raise
    print(f"forced {event_id} pending {target_role}; workflow checks skipped, not verification evidence; "
          f"original Agent has NOT been stopped; no runner started; audit: {audit_path}")


def refresh_tester(base: Path, name: str, reason: str, artifact: str) -> None:
    if not reason.strip() or len(reason) > 500 or any(char in reason for char in "\r\n"):
        raise FDError("refresh-tester requires a one-line --reason of at most 500 characters")
    with fd_lock(base, name):
        fd_path = resolve_fd(base, name)
        if fd_path.parent != feature_dir(base):
            raise FDError("refresh-tester requires an active FD")
        original = fd_path.read_text(encoding="utf-8")
        if status(original) != "Pending Test" or not independent_testing(original):
            raise FDError("refresh-tester requires an independent Pending Test FD")
        item = latest_event(base, name)
        if not item:
            raise FDError("refresh-tester requires a pending Tester handoff")
        old_path, old_event = item
        fd_ref = fd_path.relative_to(base).as_posix()
        if (old_event.get("fd_id") != name or old_event.get("fd_path") != fd_ref
                or old_event.get("target_role") != "tester"
                or old_event.get("dispatch_state") != "pending"
                or old_event.get("event_type") not in {"implementation-ready", "test-requested"}
                or old_event.get("session_ref") or old_event.get("pid")):
            raise FDError("refresh-tester requires the latest unclaimed pending Tester handoff")
        if (old_event["fd_revision"] == revision(original)
                and old_event["fd_sha256"] == fd_digest(original)):
            raise FDError("Tester handoff already matches the current FD")
        worker_session = old_event.get("worker_session_ref")
        if not isinstance(worker_session, str) or not worker_session.strip():
            raise FDError("refresh-tester requires a known Worker session")
        artifact_ref = safe_artifact(base, artifact)
        if dual_evidence(original):
            evidence = structured_evidence(base, artifact_ref, "worker-report", name)
            report_revision = evidence["data"].get("fd_revision")
            if report_revision is not None and report_revision != revision(original):
                raise FDError("implementation report revision differs from the current FD")
        new_revision = revision(original) + 1
        if new_revision <= old_event["fd_revision"]:
            raise FDError("FD revision is behind the Tester handoff; reconcile manually")
        updated = REVISION_RE.sub(f"**Revision:** {new_revision}", original, count=1)
        event_id = f"{name}-{new_revision:06d}-test-requested"
        target = runtime_dir(base, name) / "events" / f"{new_revision:06d}-test-requested.json"
        if target.exists() or target.is_symlink():
            raise FDError(f"event already exists: {event_id}")
        event = {"event_id": event_id, "fd_id": name,
                 "event_type": "test-requested", "fd_revision": new_revision,
                 "producer": "pm", "fd_sha256": fd_digest(updated),
                 "artifact_ref": artifact_ref, "fd_path": fd_ref,
                 "target_role": "tester", "dispatch_state": "pending",
                 "reason": reason.strip(), "supersedes": old_event["event_id"],
                 "implementation_event": old_event.get("implementation_event", old_event["event_id"]),
                 "worker_session_ref": worker_session,
                 "created_at": datetime.now(timezone.utc).isoformat()}
        cancelled = {**old_event, "dispatch_state": "cancelled", "superseded_by": event_id}
        try:
            atomic_json(target, {**event, "dispatch_state": "preparing"})
            atomic_text(fd_path, updated)
            atomic_json(old_path, cancelled)
            update_index(base)
            atomic_json(target, event)
        except (OSError, FDError) as exc:
            rollback_errors = []
            restores = [lambda: target.unlink(missing_ok=True),
                        lambda: atomic_text(fd_path, original),
                        lambda: atomic_json(old_path, old_event),
                        lambda: update_index(base)]
            for restore in restores:
                try:
                    restore()
                except (OSError, FDError) as rollback_error:
                    rollback_errors.append(str(rollback_error))
            if rollback_errors:
                raise FDError("refresh-tester rollback incomplete: "
                              + "; ".join(rollback_errors)) from exc
            raise
    dispatch(base, target, event)


def reopen(base: Path, name: str, reason: str) -> None:
    if not reason.strip() or len(reason) > 500 or any(char in reason for char in "\r\n"):
        raise FDError("reopen requires a one-line --reason of at most 500 characters")
    with fd_lock(base, name):
        fd_path = resolve_fd(base, name)
        if not is_archived_fd(base, fd_path):
            raise FDError("reopen requires an archived FD")
        content = fd_path.read_text(encoding="utf-8")
        old_status = status(content)
        if old_status not in {"Closed", "Deferred"}:
            raise FDError("reopen requires Closed or Deferred; use request-review for Complete")
        latest = latest_event(base, name)
        if latest and latest[1]["dispatch_state"] in {"launching", "dispatched"}:
            raise FDError("cannot reopen while a role execution has an unknown result")
        if latest and latest[1]["dispatch_state"] == "pending":
            raise FDError("cannot reopen while a prior handoff is pending")
        active_path = feature_dir(base) / fd_path.name
        if active_path.exists() or active_path.is_symlink():
            raise FDError(f"active FD path already exists: {active_path}")
        new_revision = revision(content) + 1
        event_id = f"{name}-{new_revision:06d}-reopen-requested"
        target = runtime_dir(base, name) / "events" / f"{new_revision:06d}-reopen-requested.json"
        if target.exists() or target.is_symlink():
            raise FDError(f"event already exists: {event_id}")
        updated = STATUS_RE.sub("**Status:** In Progress", content, count=1)
        updated = REVISION_RE.sub(f"**Revision:** {new_revision}", updated, count=1)
        updated = (updated.rstrip() + f"\n\n**Reopened:** {datetime.now(timezone.utc).date()}\n"
                   f"**Reopen reason:** {reason.strip()}\n")
        active_ref = active_path.relative_to(base).as_posix()
        event = {"event_id": event_id, "fd_id": name, "event_type": "reopen-requested",
                 "fd_revision": new_revision, "producer": "pm",
                 "fd_sha256": fd_digest(updated),
                 "artifact_ref": active_ref, "fd_path": active_ref,
                 "target_role": "worker", "dispatch_state": "pending",
                 "previous_status": old_status, "reason": reason.strip(),
                 "created_at": datetime.now(timezone.utc).isoformat()}
        if latest:
            event["previous_event"] = latest[1]["event_id"]
        fd_path.replace(active_path)
        try:
            atomic_json(target, event)
            atomic_text(active_path, updated)
            update_index(base)
        except (OSError, FDError):
            atomic_text(active_path, content)
            active_path.replace(fd_path)
            if target.exists():
                target.unlink()
            raise
    dispatch(base, target, event)


def correct_reopen_reason(base: Path, name: str, reason: str) -> None:
    if not reason.strip() or len(reason) > 500 or any(char in reason for char in "\r\n"):
        raise FDError("reopen requires a one-line --reason of at most 500 characters")
    with fd_lock(base, name):
        fd_path = resolve_fd(base, name)
        if fd_path.parent != feature_dir(base):
            raise FDError("reason correction requires an active FD")
        item = latest_event(base, name)
        if not item:
            raise FDError("reason correction requires a pending reopen handoff")
        event_path, event = item
        if (event["event_type"] != "reopen-requested" or event["target_role"] != "worker"
                or event["dispatch_state"] != "pending"):
            raise FDError("reason correction requires an unclaimed reopen handoff")
        content = fd_path.read_text(encoding="utf-8")
        if (status(content) != "In Progress" or revision(content) != event["fd_revision"]
                or fd_digest(content) != event["fd_sha256"]):
            raise FDError("FD changed since the reopen handoff; reconcile before correcting")
        matches = list(re.finditer(r"(?m)^\*\*Reopen reason:\*\*[ \t]*(.*)$", content))
        if not matches or matches[-1].group(1) != event["reason"]:
            raise FDError("FD reopen reason differs from the handoff receipt")
        if reason.strip() == event["reason"]:
            print(f"reopen reason already matches for {name}")
            return
        match = matches[-1]
        updated = content[:match.start(1)] + reason.strip() + content[match.end(1):]
        corrected = dict(event)
        corrected["reason"] = reason.strip()
        corrected["fd_sha256"] = fd_digest(updated)
        corrected["reason_corrections"] = list(event.get("reason_corrections", [])) + [{
            "previous_reason": event["reason"],
            "corrected_at": datetime.now(timezone.utc).isoformat(),
        }]
        atomic_json(event_path, corrected)
        try:
            atomic_text(fd_path, updated)
        except OSError:
            atomic_json(event_path, event)
            raise
    print(f"corrected reopen reason for {name}; handoff {event['event_id']} remains pending")


def prepare_event(base: Path, name: str, kind: str, producer: str, artifact: str,
                  source_event: str) -> tuple[Path, dict]:
    if kind not in ALLOWED:
        raise FDError(f"unknown event: {kind}")
    fd_path = resolve_fd(base, name)
    if is_archived_fd(base, fd_path):
        raise FDError("archived FD cannot emit a work event")
    content = fd_path.read_text(encoding="utf-8")
    old_status = status(content)
    allowed, new_status, target_role = ALLOWED[kind]
    if kind == "implementation-ready" and independent_testing(content):
        new_status, target_role = "Pending Test", "tester"
    if old_status not in allowed:
        raise FDError(f"{kind} is not valid from {old_status}")
    expected_producer = ROLE_PRODUCERS.get(kind)
    if expected_producer and producer != expected_producer:
        raise FDError(f"{kind} must come from {expected_producer}")
    if kind == "decision-recorded" and producer != "human":
        raise FDError("decision-recorded must come from human")
    if (kind in {"changes-requested", "verification-passed"}
            and independent_testing(content)
            and not current_test_acceptance(base, name)):
        raise FDError("independent review requires a current PM test acceptance")
    ensure_ready(content, kind)
    artifact_ref = safe_artifact(base, artifact)
    previous = latest_event(base, name)
    evidence_kinds = {
        "implementation-ready": "worker-report",
        "test-report-ready": "tester-report",
        "test-accepted": "pm-decision", "test-rejected": "pm-decision",
        "verification-passed": "reviewer-report",
        "changes-requested": "reviewer-report",
    }
    if dual_evidence(content) and kind in evidence_kinds:
        structured_evidence(base, artifact_ref, evidence_kinds[kind], name,
                            source_event)
    required_previous = {
        "design-ready": "planner",
        "implementation-ready": "worker",
        "test-report-ready": "tester",
        "changes-requested": "reviewer",
        "verification-passed": "reviewer",
    }.get(kind)
    if previous and required_previous and previous[1]["target_role"] != required_previous:
        raise FDError(f"{kind} requires a preceding {required_previous} handoff")
    if kind == "decision-recorded" and (not previous or previous[1]["event_type"] != "needs-decision"):
        raise FDError("decision-recorded requires a preceding needs-decision handoff")
    if kind in {"test-accepted", "test-rejected"} and (
            not previous or previous[1]["event_type"] != "test-report-ready"
            or source_event != previous[1]["event_id"]):
        raise FDError("PM test decision requires the latest test-report-ready source event")
    if previous and previous[1]["target_role"] in {"planner", "worker", "tester", "reviewer"}:
        if previous[1]["dispatch_state"] not in {"dispatched", "launching"}:
            raise FDError(f"claim or dispatch {previous[1]['event_id']} before completing its handoff")
        if source_event != previous[1]["event_id"]:
            raise FDError(f"role handoff {previous[1]['event_id']} requires --source-event")
        if producer != previous[1]["target_role"]:
            raise FDError("only the dispatched role may complete its handoff")
    elif source_event and (not previous or source_event != previous[1]["event_id"]):
        raise FDError("--source-event does not match the latest handoff")
    if kind in {"test-report-ready", "test-accepted", "test-rejected"}:
        if not previous or (revision(content) != previous[1]["fd_revision"]
                            or fd_digest(content) != previous[1]["fd_sha256"]):
            raise FDError("FD changed since test handoff; reconcile before continuing")
    assessor_sessions = []
    if kind == "test-report-ready":
        if previous[1]["event_type"] not in {"implementation-ready", "test-requested"}:
            raise FDError("test report requires an implementation-ready or test-requested handoff")
        test_summary = validate_test_report(base, artifact_ref, previous[1])
    elif kind in {"test-accepted", "test-rejected"}:
        assessor_sessions = validate_test_decision(
            base, artifact_ref, previous[1],
            "accepted" if kind == "test-accepted" else "rejected")
    new_revision = revision(content) + 1
    if target_role == "resume":
        target_role = {"Design": "planner", "Open": "worker", "In Progress": "worker",
                       "Pending Verification": "reviewer"}[old_status]
    event_id = f"{name}-{new_revision:06d}-{kind}"
    target = runtime_dir(base, name) / "events" / f"{new_revision:06d}-{kind}.json"
    if target.exists():
        raise FDError(f"event already exists: {event_id}")
    updated = REVISION_RE.sub(f"**Revision:** {new_revision}", content, count=1)
    if new_status:
        updated = STATUS_RE.sub(f"**Status:** {new_status}", updated, count=1)
    event = {"event_id": event_id, "fd_id": name, "event_type": kind,
             "fd_revision": new_revision, "producer": producer,
             "fd_sha256": fd_digest(updated),
             "artifact_ref": artifact_ref, "fd_path": fd_path.relative_to(base).as_posix(),
             "target_role": target_role, "dispatch_state": "pending",
             "created_at": datetime.now(timezone.utc).isoformat()}
    if previous and source_event:
        event["source_event"] = source_event
    if kind == "implementation-ready" and target_role == "tester":
        worker_ref = previous[1].get("session_ref") or str(previous[1].get("pid", ""))
        if not worker_ref:
            raise FDError("independent testing requires a known Worker session")
        event["worker_session_ref"] = worker_ref
    elif kind == "test-report-ready":
        event["worker_session_ref"] = previous[1]["worker_session_ref"]
        event["tester_session_ref"] = previous[1].get("session_ref") or str(previous[1].get("pid", ""))
        event["test_summary"] = test_summary
    elif kind in {"test-accepted", "test-rejected"}:
        event["worker_session_ref"] = previous[1]["worker_session_ref"]
        event["tester_session_ref"] = previous[1]["tester_session_ref"]
        event["assessor_session_refs"] = assessor_sessions
        event["test_summary"] = previous[1]["test_summary"]
    # A receipt is written first. A failed FD write leaves a diagnosable pending
    # event; resume refuses to dispatch it until the FD revision matches.
    atomic_json(target, event)
    atomic_text(fd_path, updated)
    update_index(base)
    if previous:
        old_path, old_event = previous
        old_event["dispatch_state"] = "acknowledged"
        old_event["result_event"] = event_id
        atomic_json(old_path, old_event)
    return target, event


def approved_issue_id(base: Path, requested: str) -> str:
    """Use the Issue CLI's canonical resolver and captured-source checks."""
    executable = "aiw.exe" if os.name == "nt" else "aiw"
    configured = os.environ.get("AIW_ROOT", "")
    candidate = Path(configured) / executable if configured else None
    cli = str(candidate) if candidate and candidate.is_file() else shutil.which("aiw")
    if not cli:
        raise FDError("cannot find AIW CLI to resolve the source Issue")
    result = subprocess.run([cli, "issue", "show", requested, "--json"],
                            cwd=base, capture_output=True, text=True, encoding="utf-8")
    if result.returncode:
        raise FDError(f"cannot resolve Issue {requested}: {result.stderr.strip()}")
    try:
        if len(result.stdout) > 4096:
            raise ValueError("Issue response exceeds 4096 characters")
        record = json.loads(result.stdout)
        if not isinstance(record, dict):
            raise ValueError("expected an Issue object")
        canonical = record.get("id")
        if not isinstance(canonical, str) or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,79}", canonical):
            raise ValueError("invalid canonical Issue ID")
    except (ValueError, TypeError) as error:
        raise FDError(f"invalid Issue CLI response: {error}") from error
    if record.get("status") not in {"APPROVED", "PROMOTED"} or record.get("approval_status") != "APPROVED":
        raise FDError(f"Issue is not approved: {canonical}")
    return canonical


def new_fd(base: Path, name: str, issue: str) -> None:
    if not name.strip() or any(char in name for char in "\r\n") or len(name) > 120:
        raise FDError("title must be one non-empty line of at most 120 characters")
    if issue and not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,79}", issue):
        raise FDError("issue ID contains unsupported characters")
    if issue:
        issue = approved_issue_id(base, issue)
    with fd_lock(base, "_allocation"):
        if issue:
            for linked in all_files(base):
                if f"- Issue: {issue}\n" in linked.read_text(encoding="utf-8"):
                    raise FDError(f"Issue {issue} already links to {linked.name}")
        numbers = [int(FD_FILE_RE.fullmatch(path.name).group(1)) for path in all_files(base)]
        number = max(numbers, default=0) + 1
        ident = f"FD-{number:03d}"
        target = feature_dir(base) / f"{ident}_{slug(name)}.md"
        if target.exists():
            raise FDError(f"FD already exists: {target}")
        template = base / "docs" / "templates" / "TEMPLATE.md"
        if not template.is_file():
            template.parent.mkdir(parents=True, exist_ok=True)
            atomic_text(template, DEFAULT_TEMPLATE)
        source = f"- Issue: {issue}" if issue else "- Issue: none"
        content = (template.read_text(encoding="utf-8")
                   .replace("{{FD_ID}}", ident).replace("{{TITLE}}", name)
                   .replace("{{ISSUE}}", source))
        atomic_text(target, content)
        update_index(base)
    emit(base, ident, "design-requested", "pm", target.relative_to(base).as_posix())
    print(f"created {ident}: {target}")


def resume(base: Path, name: str) -> None:
    fd_path = resolve_fd(base, name)
    if is_archived_fd(base, fd_path):
        raise FDError("archived FD cannot resume dispatch")
    item = latest_event(base, name)
    if not item:
        raise FDError("FD has no event; use emit for a deliberate handoff")
    event_path, event = item
    current_revision = revision(fd_path.read_text(encoding="utf-8"))
    if current_revision != event["fd_revision"]:
        raise FDError(f"FD revision {current_revision} differs from event {event['fd_revision']}; reconcile manually")
    state = event["dispatch_state"]
    if state == "pending":
        dispatch(base, event_path, event)
    elif state in {"launching", "dispatched"}:
        reference = event.get("session_ref") or event.get("pid") or "the event log"
        print(f"{event['event_id']} is {state}; inspect {reference}, do not dispatch again")
    else:
        print(f"{event['event_id']} is {state}; no pending dispatch")


def claim(base: Path, name: str, event_id: str, session_ref: str) -> None:
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._:/-]{0,199}", session_ref):
        raise FDError("session reference must be a non-empty ID of at most 200 safe characters")
    with fd_lock(base, name):
        fd_path = resolve_fd(base, name)
        if is_archived_fd(base, fd_path):
            raise FDError("archived FD cannot claim a handoff")
        item = latest_event(base, name)
        if not item or item[1]["event_id"] != event_id:
            raise FDError("claim requires the latest event ID")
        event_path, event = item
        if event["target_role"] in {"human", "pm"}:
            raise FDError("human and PM handoffs cannot be claimed by an agent session")
        if event["target_role"] == "tester" and session_ref == event.get("worker_session_ref"):
            raise FDError("Tester must use a separate session from Worker")
        if event["target_role"] == "reviewer":
            refs = {event.get("worker_session_ref"), event.get("tester_session_ref")}
            refs.update(event.get("assessor_session_refs", []))
            if independent_testing(fd_path.read_text(encoding="utf-8")):
                accepted = current_test_acceptance(base, name)
                if not accepted:
                    raise FDError("independent Reviewer requires current PM test acceptance")
                refs.update({accepted.get("worker_session_ref"),
                             accepted.get("tester_session_ref")})
                refs.update(accepted.get("assessor_session_refs", []))
            if session_ref in refs:
                raise FDError("Reviewer must use a separate session from Worker, Tester, and assessors")
        if event["dispatch_state"] == "dispatched" and event.get("session_ref") == session_ref:
            print(f"already claimed {event_id} by {session_ref}")
            return
        if event["dispatch_state"] != "pending":
            raise FDError(f"{event_id} is {event['dispatch_state']}; inspect its original session or log")
        content = fd_path.read_bytes()
        if (revision(content.decode("utf-8")) != event["fd_revision"]
                or fd_digest(content) != event["fd_sha256"]):
            raise FDError("FD changed since the handoff; reconcile before claiming")
        event["dispatch_state"] = "dispatched"
        event["session_ref"] = session_ref
        event["claimed_at"] = datetime.now(timezone.utc).isoformat()
        atomic_json(event_path, event)
    print(f"claimed {event_id} by {session_ref}")


def close(base: Path, name: str, outcome: str, reason: str, force: bool = False) -> None:
    with fd_lock(base, name):
        close_locked(base, name, outcome, reason, force)


def evidence_moves(base: Path, name: str) -> list[tuple[Path, Path]]:
    features = feature_dir(base)
    moves = []
    for folder in ("reports", "reviews"):
        source_dir = features / folder
        for source in sorted(source_dir.glob(f"{name}*")):
            if source.suffix not in {".md", ".json"}:
                continue
            if source.name != f"{name}.md" and not source.name.startswith(f"{name}-"):
                continue
            if not source.is_file() or source.is_symlink():
                raise FDError(f"FD evidence must be a regular file: {source}")
            target = features / "archive" / name / folder / source.name
            if target.exists() or target.is_symlink():
                raise FDError(f"archived FD evidence already exists: {target}")
            parent = target.parent
            archive_root = features / "archive"
            while parent != archive_root.parent:
                if parent.is_symlink() or (parent.exists() and not parent.is_dir()):
                    raise FDError(f"archive destination parent is unsafe: {parent}")
                parent = parent.parent
            moves.append((source, target))
    return moves


def close_locked(base: Path, name: str, outcome: str, reason: str,
                 force: bool = False) -> None:
    if force and outcome != "Complete":
        raise FDError("--force is only supported for Complete archive")
    force_audit_path = None
    force_audit = None
    if force:
        operator = getpass.getuser()
        fd_path, content, latest, force_audit_path, force_audit = force_context(
            base, name, "close", reason, operator)
        audit_dir = force_audit_path.parent
        if (force_audit_path.exists() or force_audit_path.is_symlink()
                or audit_dir.is_symlink()
                or (audit_dir.exists() and not audit_dir.is_dir())):
            raise FDError(f"unsafe force-close audit destination: {force_audit_path}")
    else:
        fd_path = resolve_fd(base, name)
        if is_archived_fd(base, fd_path):
            raise FDError("FD is already archived")
        content = fd_path.read_text(encoding="utf-8")
        latest = latest_event(base, name)
    original_content = content
    current = status(content)
    if outcome == "Complete" and not force and current != "Complete":
        raise FDError("Complete requires a verification-passed event")
    if not force and outcome != "Complete" and current in {"Closed", "Deferred"}:
        raise FDError("FD is already closed or deferred")
    if not force and outcome != "Complete" and (not reason.strip() or any(char in reason for char in "\r\n")):
        raise FDError("Deferred and Closed require a one-line --reason")
    if outcome == "Complete" and not force and (not latest or latest[1]["event_type"] != "verification-passed"
                                  or latest[1]["producer"] != "reviewer"
                                  or latest[1].get("forced")
                                  or latest[1]["fd_revision"] != revision(content)
                                  or latest[1]["fd_sha256"] != fd_digest(content)):
        raise FDError("Complete requires the current Reviewer's verification-passed event")
    if latest and latest[1]["dispatch_state"] in {"launching", "dispatched"}:
        raise FDError("cannot archive while a role execution has an unknown result")
    archive = feature_dir(base) / "archive" / name / fd_path.name
    if archive.exists() or archive.is_symlink():
        raise FDError(f"archived FD already exists: {archive}")
    parent = archive.parent
    archive_root = feature_dir(base) / "archive"
    while parent != archive_root.parent:
        if parent.is_symlink() or (parent.exists() and not parent.is_dir()):
            raise FDError(f"archive destination parent is unsafe: {parent}")
        parent = parent.parent
    moves = evidence_moves(base, name)
    if outcome != "Complete" or force:
        content = STATUS_RE.sub(f"**Status:** {outcome}", content, count=1)
        content = REVISION_RE.sub(f"**Revision:** {revision(content) + 1}", content, count=1)
    date_label = "Completed" if outcome == "Complete" else "Closed"
    content = content.rstrip() + f"\n\n**{date_label}:** {datetime.now(timezone.utc).date()}\n"
    if reason.strip():
        content += f"**Disposition reason:** {reason.strip()}\n"
    if force:
        pending_receipt = bool(
            latest
            and latest[1].get("dispatch_state") == "pending"
            and latest[1].get("event_type") != "verification-passed")
        force_audit.update({
            "result_status": "Complete",
            "result_revision": revision(content),
            "result_fd_sha256": fd_digest(content),
            "archive_path": (feature_dir(base) / "archive" / name / fd_path.name)
                .relative_to(base).as_posix(),
            "skipped_checks": ["status-must-be-complete", "current-reviewer-verification"],
            "review_verified": False,
            "receipt_changed": pending_receipt,
            "receipt_disposition": ("cancelled" if pending_receipt else
                                    "preserved-reviewer-verification" if latest and
                                    latest[1].get("event_type") == "verification-passed" else
                                    "no-pending-receipt"),
        })
    moved = []
    event_path = latest[0] if latest else None
    original_event = dict(latest[1]) if latest else None
    index_path = feature_dir(base) / "FEATURE_INDEX.md"
    original_index = index_path.read_text(encoding="utf-8") if index_path.exists() else None
    receipt_attempted = False
    try:
        for source, target in moves:
            target.parent.mkdir(parents=True, exist_ok=True)
            source.replace(target)
            moved.append((source, target))
        atomic_text(fd_path, content)
        archive.parent.mkdir(parents=True, exist_ok=True)
        fd_path.replace(archive)
        update_receipt = bool(
            latest
            and latest[1]["dispatch_state"] == "pending"
            and (not force or latest[1].get("event_type") != "verification-passed"))
        if update_receipt:
            event = dict(latest[1])
            if force:
                event.update({
                    "dispatch_state": "cancelled",
                    "cancelled_at": force_audit["created_at"],
                    "cancellation_reason": reason.strip(),
                    "cancelled_by": force_audit["local_user"],
                })
                force_audit["result_event"] = event
            else:
                event["dispatch_state"] = "cancelled" if outcome != "Complete" else "acknowledged"
            receipt_attempted = True
            atomic_json(event_path, event)
        update_index(base)
        if force:
            atomic_json(force_audit_path, force_audit)
    except (OSError, FDError) as exc:
        rollback_errors = []
        if force and force_audit_path.exists():
            try:
                force_audit_path.unlink()
            except OSError as rollback_error:
                rollback_errors.append(str(rollback_error))
        if archive.exists():
            try:
                archive.replace(fd_path)
            except OSError as rollback_error:
                rollback_errors.append(str(rollback_error))
        if fd_path.exists():
            try:
                atomic_text(fd_path, original_content)
            except OSError as rollback_error:
                rollback_errors.append(str(rollback_error))
        for source, target in reversed(moved):
            if target.exists():
                try:
                    source.parent.mkdir(parents=True, exist_ok=True)
                    target.replace(source)
                except (OSError, FDError) as rollback_error:
                    rollback_errors.append(str(rollback_error))
        if receipt_attempted and event_path and original_event:
            try:
                atomic_json(event_path, original_event)
            except OSError as rollback_error:
                rollback_errors.append(str(rollback_error))
        try:
            if original_index is None:
                index_path.unlink(missing_ok=True)
            else:
                atomic_text(index_path, original_index)
        except OSError as rollback_error:
            rollback_errors.append(str(rollback_error))
        if rollback_errors:
            raise FDError("close rollback incomplete: " + "; ".join(rollback_errors)) from exc
        raise
    print(f"archived {name}: {archive}")
    if force:
        print("forced Complete archive; Reviewer verification was skipped and not recorded; "
              f"audit: {force_audit_path}")


def main() -> int:
    parser = argparse.ArgumentParser(prog="aiw fd", description="FD-first role handoffs")
    commands = parser.add_subparsers(dest="command", required=True)
    created = commands.add_parser("new", help="create a numbered FD and request Planner")
    created.add_argument("title")
    created.add_argument("--issue", default="")
    commands.add_parser("list", help="show the terminal FD table")
    shown = commands.add_parser("show", help="show an FD and its last handoff")
    shown.add_argument("fd_id")
    for command, kind in (("show-report", "report"), ("show-review", "review")):
        evidence = commands.add_parser(command, help=f"list or display FD {kind} evidence")
        evidence.add_argument("fd_id")
        evidence.add_argument("--last", action="store_true",
                              help="display the most recent verified evidence without prompting")
    emitted = commands.add_parser("emit", help="record a stage result and route the next role")
    emitted.add_argument("fd_id")
    emitted.add_argument("event_type", choices=sorted(ALLOWED))
    emitted.add_argument("--producer", required=True)
    emitted.add_argument("--artifact", required=True)
    emitted.add_argument("--source-event", default="")
    claimed = commands.add_parser("claim", help="bind a pending role handoff to one host session")
    claimed.add_argument("fd_id")
    claimed.add_argument("event_id")
    claimed.add_argument("--session", required=True)
    resumed = commands.add_parser("resume", help="resume a pending handoff safely")
    resumed.add_argument("fd_id")
    review = commands.add_parser("request-review", help="request Reviewer for Pending Verification or archived Complete FD")
    review.add_argument("fd_id")
    review.add_argument("--reason", required=True)
    refreshed = commands.add_parser("refresh-worker", help="replace a stale pending Worker handoff")
    refreshed.add_argument("fd_id")
    refreshed.add_argument("--reason", required=True)
    recovered = commands.add_parser("recover-worker", help="replace an abandoned dispatched Worker handoff")
    recovered.add_argument("fd_id")
    recovered.add_argument("--expected-event", required=True)
    recovered.add_argument("--expected-session", required=True)
    recovered.add_argument("--reason", required=True)
    cancelled = commands.add_parser("cancel-event", help="cancel the latest dispatched receipt without stopping its Agent")
    cancelled.add_argument("fd_id")
    cancelled.add_argument("--expected-event", required=True)
    cancelled.add_argument("--reason", required=True)
    cancelled.add_argument("--operator", required=True, help="declared operator identity, not authentication")
    overridden = commands.add_parser("set-status", help="force an active FD status without transition or terminal evidence checks")
    overridden.add_argument("fd_id")
    overridden.add_argument("status", choices=sorted(DEFINED_STATUSES))
    overridden.add_argument("--reason", required=True)
    overridden.add_argument("--operator", required=True, help="declared operator identity, not authentication")
    forced = commands.add_parser("force-emit", help="emit a defined event while skipping workflow gates; leave pending without starting a runner")
    forced.add_argument("fd_id")
    forced.add_argument("event_type", choices=sorted(ALLOWED))
    forced.add_argument("--producer", required=True)
    forced.add_argument("--artifact", required=True)
    forced.add_argument("--reason", required=True)
    forced.add_argument("--operator", required=True, help="declared operator identity, not authentication")
    tester_refresh = commands.add_parser("refresh-tester", help="replace a stale unclaimed Tester handoff")
    tester_refresh.add_argument("fd_id")
    tester_refresh.add_argument("--reason", required=True)
    tester_refresh.add_argument("--artifact", required=True,
                                help="current implementation report inside the repository")
    reopened = commands.add_parser("reopen", help="resume an archived Closed or Deferred FD")
    reopened.add_argument("fd_id")
    reopened.add_argument("--reason", required=True)
    reopened.add_argument("--correct-reason", action="store_true",
                          help="correct the reason on an unclaimed reopen handoff")
    closed = commands.add_parser("close", help="archive an FD; --force can set Complete without Reviewer verification")
    closed.add_argument("fd_id")
    closed.add_argument("outcome", choices=["Complete", "Deferred", "Closed"])
    closed.add_argument("--reason", default="")
    closed.add_argument("-f", "--force", action="store_true",
                        help="force Complete archive, set status Complete, and record skipped checks")
    args = parser.parse_args()
    try:
        base = root()
        if args.command == "new":
            new_fd(base, args.title, args.issue)
        elif args.command == "list":
            print(render_fd_list(base, color=supports_ansi_color()))
        elif args.command == "show":
            path = resolve_fd(base, args.fd_id)
            name = fd_id(args.fd_id)
            content = path.read_text(encoding="utf-8")
            print(content)
            item = latest_event(base, name)
            if item:
                print("\nLast handoff:", json.dumps(item[1], ensure_ascii=False))
            show_status_summary(base, name, content, item)
        elif args.command in {"show-report", "show-review"}:
            name = fd_id(args.fd_id)
            resolve_fd(base, name)
            show_evidence(base, name, "report" if args.command == "show-report" else "review",
                          args.last)
        elif args.command == "emit":
            emit(base, fd_id(args.fd_id), args.event_type, args.producer,
                 args.artifact, args.source_event)
        elif args.command == "claim":
            claim(base, fd_id(args.fd_id), args.event_id, args.session)
        elif args.command == "resume":
            resume(base, fd_id(args.fd_id))
        elif args.command == "request-review":
            request_review(base, fd_id(args.fd_id), args.reason)
        elif args.command == "refresh-worker":
            refresh_worker(base, fd_id(args.fd_id), args.reason)
        elif args.command == "recover-worker":
            recover_worker(base, fd_id(args.fd_id), args.expected_event,
                           args.expected_session, args.reason)
        elif args.command == "cancel-event":
            cancel_event(base, fd_id(args.fd_id), args.expected_event, args.reason, args.operator)
        elif args.command == "set-status":
            set_status(base, fd_id(args.fd_id), args.status, args.reason, args.operator)
        elif args.command == "force-emit":
            force_emit(base, fd_id(args.fd_id), args.event_type, args.producer,
                       args.artifact, args.reason, args.operator)
        elif args.command == "refresh-tester":
            refresh_tester(base, fd_id(args.fd_id), args.reason, args.artifact)
        elif args.command == "reopen":
            if args.correct_reason:
                correct_reopen_reason(base, fd_id(args.fd_id), args.reason)
            else:
                reopen(base, fd_id(args.fd_id), args.reason)
        elif args.command == "close":
            close(base, fd_id(args.fd_id), args.outcome, args.reason, args.force)
    except (FDError, OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"fd: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
