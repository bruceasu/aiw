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
    "description": "Create, inspect, and advance FD-first work.",
    "commands": ["new", "list", "show", "emit", "claim", "resume", "request-review", "refresh-worker", "refresh-tester", "reopen", "close"],
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
DEFAULT_TEMPLATE = """# {{FD_ID}}: {{TITLE}}

**Status:** Planned
**Revision:** 1
**Priority:** Medium
**Test policy:** Independent
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


def independent_testing(content: str) -> bool:
    match = TEST_POLICY_RE.search(content)
    return bool(match and match.group(1).strip() == "Independent")


def dual_evidence(content: str) -> bool:
    match = EVIDENCE_POLICY_RE.search(content)
    return bool(match and match.group(1).strip() == "Dual")


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


def validate_test_decision(base: Path, artifact_ref: str, previous: dict,
                           disposition: str) -> None:
    fields = labelled_fields(base, artifact_ref)
    for name, expected in (("Disposition", disposition),
                           ("Tester report", previous["artifact_ref"]),
                           ("FD revision", str(previous["fd_revision"])),
                           ("FD digest", previous["fd_sha256"]),
                           ("Requirements coverage", previous["test_summary"]["requirements_coverage"]),
                           ("Branch coverage", previous["test_summary"]["branch_coverage"])):
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
    summary = previous["test_summary"]
    if disposition == "accepted":
        if summary["failed_behavior_tests"]:
            raise FDError("PM cannot accept failed executed behavior tests")
        low_or_missing = any(coverage_value(summary[key]) is None
                             or coverage_value(summary[key]) < 70
                             for key in ("requirements_coverage", "branch_coverage"))
        if low_or_missing and required_field(fields, "Exceptions").casefold() in {
                "none", "n/a", "not applicable"}:
            raise FDError("accepting a coverage gap requires an exception")


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
    if kind == "test-report-ready":
        if previous[1]["event_type"] not in {"implementation-ready", "test-requested"}:
            raise FDError("test report requires an implementation-ready or test-requested handoff")
        test_summary = validate_test_report(base, artifact_ref, previous[1])
    elif kind in {"test-accepted", "test-rejected"}:
        validate_test_decision(base, artifact_ref, previous[1],
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
            if independent_testing(fd_path.read_text(encoding="utf-8")):
                accepted = current_test_acceptance(base, name)
                if not accepted:
                    raise FDError("independent Reviewer requires current PM test acceptance")
                refs.update({accepted.get("worker_session_ref"),
                             accepted.get("tester_session_ref")})
            if session_ref in refs:
                raise FDError("Reviewer must use a separate session from Worker and Tester")
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


def close(base: Path, name: str, outcome: str, reason: str) -> None:
    with fd_lock(base, name):
        close_locked(base, name, outcome, reason)


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


def close_locked(base: Path, name: str, outcome: str, reason: str) -> None:
    fd_path = resolve_fd(base, name)
    if is_archived_fd(base, fd_path):
        raise FDError("FD is already archived")
    content = fd_path.read_text(encoding="utf-8")
    original_content = content
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
    if outcome != "Complete":
        content = STATUS_RE.sub(f"**Status:** {outcome}", content, count=1)
        content = REVISION_RE.sub(f"**Revision:** {revision(content) + 1}", content, count=1)
    date_label = "Completed" if outcome == "Complete" else "Closed"
    content = content.rstrip() + f"\n\n**{date_label}:** {datetime.now(timezone.utc).date()}\n"
    if reason.strip():
        content += f"**Disposition reason:** {reason.strip()}\n"
    moved = []
    event_path = latest[0] if latest else None
    original_event = dict(latest[1]) if latest else None
    index_path = feature_dir(base) / "FEATURE_INDEX.md"
    original_index = index_path.read_text(encoding="utf-8") if index_path.exists() else None
    try:
        for source, target in moves:
            target.parent.mkdir(parents=True, exist_ok=True)
            source.replace(target)
            moved.append((source, target))
        atomic_text(fd_path, content)
        archive.parent.mkdir(parents=True, exist_ok=True)
        fd_path.replace(archive)
        if latest and latest[1]["dispatch_state"] == "pending":
            event = dict(latest[1])
            event["dispatch_state"] = "cancelled" if outcome != "Complete" else "acknowledged"
            atomic_json(event_path, event)
        update_index(base)
    except (OSError, FDError) as exc:
        rollback_errors = []
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
        if event_path and original_event:
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
    review = commands.add_parser("request-review", help="request Reviewer for Pending Verification or archived Complete FD")
    review.add_argument("fd_id")
    review.add_argument("--reason", required=True)
    refreshed = commands.add_parser("refresh-worker", help="replace a stale pending Worker handoff")
    refreshed.add_argument("fd_id")
    refreshed.add_argument("--reason", required=True)
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
    closed = commands.add_parser("close", help="archive a completed, deferred, or closed FD")
    closed.add_argument("fd_id")
    closed.add_argument("outcome", choices=["Complete", "Deferred", "Closed"])
    closed.add_argument("--reason", default="")
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
        elif args.command == "request-review":
            request_review(base, fd_id(args.fd_id), args.reason)
        elif args.command == "refresh-worker":
            refresh_worker(base, fd_id(args.fd_id), args.reason)
        elif args.command == "refresh-tester":
            refresh_tester(base, fd_id(args.fd_id), args.reason, args.artifact)
        elif args.command == "reopen":
            if args.correct_reason:
                correct_reopen_reason(base, fd_id(args.fd_id), args.reason)
            else:
                reopen(base, fd_id(args.fd_id), args.reason)
        elif args.command == "close":
            close(base, fd_id(args.fd_id), args.outcome, args.reason)
    except (FDError, OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"fd: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
