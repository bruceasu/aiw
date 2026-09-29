#!/usr/bin/env python3
"""FD-first work management. Markdown owns progress; receipts own handoffs."""

from __future__ import annotations

import argparse
from contextlib import contextmanager
import hashlib
import json
import os
import re
import subprocess
import sys
import uuid
from datetime import datetime, timezone
from pathlib import Path


META = {
    "name": "aiw-fd",
    "short": "manage numbered Feature Designs and role handoffs",
    "description": "Create, inspect, and advance FD-first work without Workflow Core.",
    "commands": ["new", "list", "show", "emit", "claim", "resume", "close", "worktree"],
    "readOnly": False,
    "mutatesFiles": True,
    "requiresConfirmation": False,
    "outputFormat": "text",
}

FD_RE = re.compile(r"^FD-(\d{3,})$", re.IGNORECASE)
FD_FILE_RE = re.compile(r"^FD-(\d{3,})_[A-Z0-9_]+\.md$", re.IGNORECASE)
STATUS_RE = re.compile(r"(?m)^\*\*Status:\*\*[ \t]*(.+?)[ \t]*$")
REVISION_RE = re.compile(r"(?m)^\*\*Revision:\*\*[ \t]*(\d+)[ \t]*$")
PRIORITY_RE = re.compile(r"(?m)^\*\*Priority:\*\*[ \t]*(.+?)[ \t]*$")
ITEM_RE = re.compile(r"(?m)^\s*- \[([ xX-])\] (\d+(?:\.\d+)*)\s+(.+)$")
ALLOWED = {
    "design-requested": ({"Planned"}, "Design", "planner"),
    "design-ready": ({"Design"}, "Open", "worker"),
    "implementation-ready": ({"Open", "In Progress"}, "Pending Verification", "reviewer"),
    "changes-requested": ({"Pending Verification"}, "In Progress", "worker"),
    "verification-passed": ({"Pending Verification"}, "Complete", "pm"),
    "needs-decision": ({"Design", "Open", "In Progress", "Pending Verification"}, None, "human"),
    "decision-recorded": ({"Design", "Open", "In Progress", "Pending Verification"}, None, "resume"),
}
ROLE_PRODUCERS = {
    "design-ready": "planner",
    "implementation-ready": "worker",
    "changes-requested": "reviewer",
    "verification-passed": "reviewer",
}
DEFAULT_TEMPLATE = """# {{FD_ID}}: {{TITLE}}

**Status:** Planned  
**Revision:** 1  
**Priority:** Medium

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

- Not run.

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


def all_files(base: Path) -> list[Path]:
    directory = feature_dir(base)
    return active_files(base) + sorted(path for path in (directory / "archive").glob("FD-*.md")
                                      if FD_FILE_RE.fullmatch(path.name))


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


def title(content: str) -> str:
    heading = content.splitlines()[0].lstrip("# ").strip()
    return re.sub(r"^FD-\d+\s*[:：]\s*", "", heading, flags=re.IGNORECASE)


def atomic_text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(path.name + f".tmp-{os.getpid()}-{uuid.uuid4().hex}")
    with temporary.open("x", encoding="utf-8", newline="\n") as stream:
        stream.write(content)
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temporary, path)


def atomic_json(path: Path, value: dict) -> None:
    atomic_text(path, json.dumps(value, ensure_ascii=False, indent=2) + "\n")


def runtime_dir(base: Path, name: str) -> Path:
    return base / ".ai" / "fd" / name


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
        if revision(fd_content.decode("utf-8")) != current["fd_revision"] or hashlib.sha256(fd_content).hexdigest() != current["fd_sha256"]:
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


def prepare_event(base: Path, name: str, kind: str, producer: str, artifact: str,
                  source_event: str) -> tuple[Path, dict]:
    if kind not in ALLOWED:
        raise FDError(f"unknown event: {kind}")
    fd_path = resolve_fd(base, name)
    if fd_path.parent.name == "archive":
        raise FDError("archived FD cannot emit a work event")
    content = fd_path.read_text(encoding="utf-8")
    old_status = status(content)
    allowed, new_status, target_role = ALLOWED[kind]
    if old_status not in allowed:
        raise FDError(f"{kind} is not valid from {old_status}")
    expected_producer = ROLE_PRODUCERS.get(kind)
    if expected_producer and producer != expected_producer:
        raise FDError(f"{kind} must come from {expected_producer}")
    if kind == "decision-recorded" and producer != "human":
        raise FDError("decision-recorded must come from human")
    ensure_ready(content, kind)
    artifact_ref = safe_artifact(base, artifact)
    previous = latest_event(base, name)
    required_previous = {
        "design-ready": "planner",
        "implementation-ready": "worker",
        "changes-requested": "reviewer",
        "verification-passed": "reviewer",
    }.get(kind)
    if previous and required_previous and previous[1]["target_role"] != required_previous:
        raise FDError(f"{kind} requires a preceding {required_previous} handoff")
    if kind == "decision-recorded" and (not previous or previous[1]["event_type"] != "needs-decision"):
        raise FDError("decision-recorded requires a preceding needs-decision handoff")
    if previous and previous[1]["target_role"] in {"planner", "worker", "reviewer"}:
        if previous[1]["dispatch_state"] not in {"dispatched", "launching"}:
            raise FDError(f"claim or dispatch {previous[1]['event_id']} before completing its handoff")
        if source_event != previous[1]["event_id"]:
            raise FDError(f"role handoff {previous[1]['event_id']} requires --source-event")
        if producer != previous[1]["target_role"]:
            raise FDError("only the dispatched role may complete its handoff")
    elif source_event and (not previous or source_event != previous[1]["event_id"]):
        raise FDError("--source-event does not match the latest handoff")
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
             "fd_sha256": hashlib.sha256(updated.encode("utf-8")).hexdigest(),
             "artifact_ref": artifact_ref, "fd_path": fd_path.relative_to(base).as_posix(),
             "target_role": target_role, "dispatch_state": "pending",
             "created_at": datetime.now(timezone.utc).isoformat()}
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


def new_fd(base: Path, name: str, issue: str) -> None:
    if not name.strip() or any(char in name for char in "\r\n") or len(name) > 120:
        raise FDError("title must be one non-empty line of at most 120 characters")
    if issue and not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,79}", issue):
        raise FDError("issue ID contains unsupported characters")
    if issue:
        record = base / "docs" / "requirements" / issue / "requirement.toml"
        if (not record.is_file() or record.is_symlink()
                or not record.resolve().is_relative_to(base)):
            raise FDError(f"Issue record not found: {issue}")
        source = record.read_text(encoding="utf-8")
        top = source.split("[approval]", 1)[0]
        approval = source.split("[approval]", 1)[1].split("\n[", 1)[0] if "[approval]" in source else ""
        if not re.search(r'(?m)^status = "(?:APPROVED|PROMOTED)"$', top) or not re.search(r'(?m)^status = "APPROVED"$', approval):
            raise FDError(f"Issue is not approved: {issue}")
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
        template = feature_dir(base) / "TEMPLATE.md"
        if not template.is_file():
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
    if fd_path.parent.name == "archive":
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
        if fd_path.parent.name == "archive":
            raise FDError("archived FD cannot claim a handoff")
        item = latest_event(base, name)
        if not item or item[1]["event_id"] != event_id:
            raise FDError("claim requires the latest event ID")
        event_path, event = item
        if event["target_role"] in {"human", "pm"}:
            raise FDError("human and PM handoffs cannot be claimed by an agent session")
        if event["dispatch_state"] == "dispatched" and event.get("session_ref") == session_ref:
            print(f"already claimed {event_id} by {session_ref}")
            return
        if event["dispatch_state"] != "pending":
            raise FDError(f"{event_id} is {event['dispatch_state']}; inspect its original session or log")
        content = fd_path.read_bytes()
        if (revision(content.decode("utf-8")) != event["fd_revision"]
                or hashlib.sha256(content).hexdigest() != event["fd_sha256"]):
            raise FDError("FD changed since the handoff; reconcile before claiming")
        event["dispatch_state"] = "dispatched"
        event["session_ref"] = session_ref
        event["claimed_at"] = datetime.now(timezone.utc).isoformat()
        atomic_json(event_path, event)
    print(f"claimed {event_id} by {session_ref}")


def close(base: Path, name: str, outcome: str, reason: str) -> None:
    with fd_lock(base, name):
        close_locked(base, name, outcome, reason)


def close_locked(base: Path, name: str, outcome: str, reason: str) -> None:
    fd_path = resolve_fd(base, name)
    if fd_path.parent.name == "archive":
        raise FDError("FD is already archived")
    content = fd_path.read_text(encoding="utf-8")
    current = status(content)
    if outcome == "Complete" and current != "Complete":
        raise FDError("Complete requires a verification-passed event")
    if outcome != "Complete" and current in {"Closed", "Deferred"}:
        raise FDError("FD is already closed or deferred")
    if outcome != "Complete" and (not reason.strip() or any(char in reason for char in "\r\n")):
        raise FDError("Deferred and Closed require a one-line --reason")
    latest = latest_event(base, name)
    if outcome == "Complete" and (not latest or latest[1]["event_type"] != "verification-passed"
                                  or latest[1]["producer"] != "reviewer"
                                  or latest[1]["fd_revision"] != revision(content)
                                  or latest[1]["fd_sha256"] != hashlib.sha256(content.encode("utf-8")).hexdigest()):
        raise FDError("Complete requires the current Reviewer's verification-passed event")
    if latest and latest[1]["dispatch_state"] in {"launching", "dispatched"}:
        raise FDError("cannot archive while a role execution has an unknown result")
    if outcome != "Complete":
        content = STATUS_RE.sub(f"**Status:** {outcome}", content, count=1)
        content = REVISION_RE.sub(f"**Revision:** {revision(content) + 1}", content, count=1)
    date_label = "Completed" if outcome == "Complete" else "Closed"
    content = content.rstrip() + f"\n\n**{date_label}:** {datetime.now(timezone.utc).date()}\n"
    if reason.strip():
        content += f"**Disposition reason:** {reason.strip()}\n"
    atomic_text(fd_path, content)
    if latest and latest[1]["dispatch_state"] == "pending":
        event_path, event = latest
        event["dispatch_state"] = "cancelled" if outcome != "Complete" else "acknowledged"
        atomic_json(event_path, event)
    archive = feature_dir(base) / "archive" / fd_path.name
    archive.parent.mkdir(parents=True, exist_ok=True)
    fd_path.replace(archive)
    update_index(base)
    print(f"archived {name}: {archive}")


def worktree(base: Path, name: str, operation: str) -> None:
    fd_path = resolve_fd(base, name)
    target = base / ".wt" / name
    branch = "feature/" + name
    if operation == "status":
        result = subprocess.run(["git", "worktree", "list", "--porcelain"], cwd=base,
                                text=True, capture_output=True, check=False)
        if result.returncode:
            raise FDError(result.stderr.strip() or "cannot inspect worktrees")
        print(result.stdout)
        return
    if fd_path.parent.name == "archive":
        raise FDError("cannot create a worktree for an archived FD")
    relative = fd_path.relative_to(base).as_posix()
    tracked = subprocess.run(["git", "ls-files", "--error-unmatch", "--", relative],
                             cwd=base, capture_output=True, check=False)
    committed = subprocess.run(["git", "diff", "--quiet", "HEAD", "--", relative],
                               cwd=base, check=False)
    if tracked.returncode or committed.returncode:
        raise FDError("commit the FD plan before creating its isolated worktree")
    if target.exists() or target.is_symlink():
        raise FDError(f"worktree target already exists: {target}")
    exists = subprocess.run(["git", "show-ref", "--verify", "--quiet",
                             "refs/heads/" + branch], cwd=base, check=False)
    if exists.returncode == 0:
        raise FDError(f"branch already exists: {branch}; inspect it before reuse")
    parent = subprocess.run(["git", "symbolic-ref", "--quiet", "--short", "HEAD"],
                            cwd=base, text=True, capture_output=True, check=False)
    if parent.returncode:
        raise FDError("cannot create an FD worktree from a detached HEAD")
    result = subprocess.run(["git", "worktree", "add", "-b", branch, str(target), "HEAD"],
                            cwd=base, text=True, capture_output=True, check=False)
    if result.returncode:
        raise FDError(result.stderr.strip() or "worktree add failed")
    atomic_json(runtime_dir(base, name) / "workspace.json",
                {"fd_id": name, "parent_branch": parent.stdout.strip(),
                 "branch": branch, "worktree": str(target)})
    print(f"created {target} on {branch}")


def main() -> int:
    parser = argparse.ArgumentParser(prog="aiw fd", description="FD-first role handoffs")
    commands = parser.add_subparsers(dest="command", required=True)
    created = commands.add_parser("new", help="create a numbered FD and request Planner")
    created.add_argument("title")
    created.add_argument("--issue", default="")
    commands.add_parser("list", help="show the FD index")
    shown = commands.add_parser("show", help="show an FD and its last handoff")
    shown.add_argument("fd_id")
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
    closed = commands.add_parser("close", help="archive a completed, deferred, or closed FD")
    closed.add_argument("fd_id")
    closed.add_argument("outcome", choices=["Complete", "Deferred", "Closed"])
    closed.add_argument("--reason", default="")
    isolated = commands.add_parser("worktree", help="inspect or add an FD worktree")
    isolated.add_argument("operation", choices=["add", "status"])
    isolated.add_argument("fd_id")
    args = parser.parse_args()
    try:
        base = root()
        if args.command == "new":
            new_fd(base, args.title, args.issue)
        elif args.command == "list":
            print((feature_dir(base) / "FEATURE_INDEX.md").read_text(encoding="utf-8"))
        elif args.command == "show":
            path = resolve_fd(base, args.fd_id)
            print(path.read_text(encoding="utf-8"))
            item = latest_event(base, fd_id(args.fd_id))
            if item:
                print("\nLast handoff:", json.dumps(item[1], ensure_ascii=False))
        elif args.command == "emit":
            emit(base, fd_id(args.fd_id), args.event_type, args.producer,
                 args.artifact, args.source_event)
        elif args.command == "claim":
            claim(base, fd_id(args.fd_id), args.event_id, args.session)
        elif args.command == "resume":
            resume(base, fd_id(args.fd_id))
        elif args.command == "close":
            close(base, fd_id(args.fd_id), args.outcome, args.reason)
        elif args.command == "worktree":
            worktree(base, fd_id(args.fd_id), args.operation)
    except (FDError, OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"fd: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
