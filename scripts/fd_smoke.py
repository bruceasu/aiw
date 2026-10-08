#!/usr/bin/env python3
"""Focused local FD lifecycle check in a disposable Git repository."""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
from pathlib import Path


SOURCE = Path(__file__).resolve().parents[1]
PLUGIN = SOURCE / "plugins" / "aiw-fd.py"


def command(root: Path, *args: str, success: bool = True, env: dict | None = None):
    process = subprocess.run(args, cwd=root, env=env, text=True, capture_output=True,
                             check=False)
    if success and process.returncode:
        raise AssertionError(f"command failed: {args}\n{process.stdout}\n{process.stderr}")
    if not success and process.returncode == 0:
        raise AssertionError(f"command unexpectedly passed: {args}")
    return process


def fd(root: Path, *args: str, success: bool = True, env: dict | None = None):
    return command(root, sys.executable, str(PLUGIN), *args, success=success, env=env)


def main() -> None:
    with tempfile.TemporaryDirectory(prefix="aiw-fd-smoke-") as directory:
        root = Path(directory)
        command(root, "git", "init", "-q")
        command(root, "git", "config", "user.name", "FD Smoke")
        command(root, "git", "config", "user.email", "fd-smoke@example.invalid")
        features = root / "docs" / "features"
        features.mkdir(parents=True)
        templates = root / "docs" / "templates"
        templates.mkdir(parents=True)
        shutil.copy2(SOURCE / "docs" / "templates" / "TEMPLATE.md", templates / "TEMPLATE.md")

        fd(root, "new", "Smoke flow")
        design = features / "FD-001_SMOKE_FLOW.md"
        assert design.is_file()
        first_event = root / ".ai" / "fd" / "FD-001" / "events" / "000002-design-requested.json"
        assert json.loads(first_event.read_text(encoding="utf-8"))["dispatch_state"] == "pending"
        relative = "docs/features/FD-001_SMOKE_FLOW.md"
        fd(root, "emit", "FD-001", "design-ready", "--producer", "planner",
           "--artifact", relative, success=False)
        marker = root / "runner-role.txt"
        if os.name == "nt":
            runner = root / "role-runner.cmd"
            runner.write_text("@echo off\necho %1 > runner-role.txt\n", encoding="utf-8")
        else:
            runner = root / "role-runner.sh"
            runner.write_text("#!/bin/sh\nprintf '%s' \"$1\" > runner-role.txt\n", encoding="utf-8")
            runner.chmod(0o755)
        runner_env = os.environ.copy()
        runner_env["AIW_FD_ROLE_RUNNER"] = str(runner)
        fd(root, "resume", "FD-001", env=runner_env)
        for _ in range(20):
            if marker.is_file():
                break
            time.sleep(0.1)
        assert marker.read_text(encoding="utf-8").strip() == "planner"
        fd(root, "resume", "FD-001", env=runner_env)
        assert json.loads(first_event.read_text(encoding="utf-8"))["dispatch_state"] == "dispatched"
        assert len(list(first_event.parent.glob("*.json"))) == 1

        content = design.read_text(encoding="utf-8")
        content = re.sub(r"(?m)^%% NEEDS_INPUT:.*$", "Confirmed for smoke flow.", content)
        content = content.replace("Define the first independently reviewable outcome and its acceptance evidence.",
                                  "Implement the smoke flow with a reviewable report.")
        design.write_text(content, encoding="utf-8")
        fd(root, "emit", "FD-001", "design-ready", "--producer", "planner", "--artifact", relative,
           "--source-event", "FD-001-000002-design-requested")
        fd(root, "emit", "FD-001", "design-ready", "--producer", "planner", "--artifact", relative,
           success=False)
        assert "**Status:** Open" in design.read_text(encoding="utf-8")
        worker_event = "FD-001-000003-design-ready"
        fd(root, "claim", "FD-001", worker_event, "--session", "smoke-worker")
        fd(root, "claim", "FD-001", worker_event, "--session", "smoke-worker")
        fd(root, "claim", "FD-001", worker_event, "--session", "second-writer", success=False)
        resumed = fd(root, "resume", "FD-001", env=runner_env)
        assert "smoke-worker" in resumed.stdout

        content = design.read_text(encoding="utf-8").replace("- [ ] 1.1", "- [x] 1.1")
        design.write_text(content, encoding="utf-8")
        report_dir = features / "reports"
        report_dir.mkdir()
        implementation = report_dir / "FD-001.md"
        implementation.write_text("Changed only the smoke fixture. Compile: not run.\n", encoding="utf-8")
        fd(root, "emit", "FD-001", "implementation-ready", "--producer", "worker",
           "--artifact", "docs/features/reports/FD-001.md", "--source-event", worker_event)

        review_dir = features / "reviews"
        review_dir.mkdir()
        review = review_dir / "FD-001.md"
        review.write_text("Changes requested: include the final evidence note.\n", encoding="utf-8")
        review_event = "FD-001-000004-implementation-ready"
        fd(root, "emit", "FD-001", "changes-requested", "--producer", "reviewer",
           "--artifact", "docs/features/reviews/FD-001.md", success=False)
        fd(root, "claim", "FD-001", review_event, "--session", "smoke-reviewer")
        fd(root, "emit", "FD-001", "changes-requested", "--producer", "reviewer",
           "--artifact", "docs/features/reviews/FD-001.md", "--source-event", review_event)
        assert "**Status:** In Progress" in design.read_text(encoding="utf-8")
        returned_event = "FD-001-000005-changes-requested"
        fd(root, "claim", "FD-001", returned_event, "--session", "smoke-worker")
        implementation.write_text("Final evidence note recorded. Compile: not run.\n", encoding="utf-8")
        fd(root, "emit", "FD-001", "implementation-ready", "--producer", "worker",
           "--artifact", "docs/features/reports/FD-001.md", "--source-event", returned_event)
        final_review_event = "FD-001-000006-implementation-ready"
        fd(root, "claim", "FD-001", final_review_event, "--session", "smoke-reviewer")
        review.write_text("Passed: scoped files and reports reviewed. Runtime checks not run.\n", encoding="utf-8")
        fd(root, "emit", "FD-001", "verification-passed", "--producer", "reviewer",
           "--artifact", "docs/features/reviews/FD-001.md", "--source-event", final_review_event)

        reviewed_content = design.read_text(encoding="utf-8")
        design.write_text(reviewed_content + "\nUnreviewed change.\n", encoding="utf-8")
        fd(root, "close", "FD-001", "Complete", success=False)
        design.write_text(reviewed_content, encoding="utf-8")

        command(root, "git", "add", "--", "docs/features")
        command(root, "git", "commit", "-q", "-m", "FD-001: smoke flow")
        fd(root, "worktree", "add", "FD-001")
        assert (root / ".wt" / "FD-001" / relative).is_file()
        fd(root, "close", "FD-001", "Complete")
        assert (features / "archive" / design.name).is_file()
        assert "FD-001" in (features / "FEATURE_INDEX.md").read_text(encoding="utf-8")
        fd(root, "resume", "FD-001", success=False)

        fd(root, "new", "Stale review")
        stalled = features / "FD-002_STALE_REVIEW.md"
        old_event = root / ".ai" / "fd" / "FD-002" / "events" / "000002-design-requested.json"
        stalled_content = stalled.read_text(encoding="utf-8")
        stalled_content = stalled_content.replace("**Status:** Design", "**Status:** Pending Verification")
        stalled_content = stalled_content.replace("**Revision:** 2", "**Revision:** 3")
        stalled.write_text(stalled_content, encoding="utf-8")
        fd(root, "request-review", "FD-002", "--reason", "recover stale handoff")
        new_event = old_event.parent / "000004-review-requested.json"
        receipt = json.loads(new_event.read_text(encoding="utf-8"))
        assert receipt["target_role"] == "reviewer"
        assert receipt["supersedes"] == "FD-002-000002-design-requested"
        assert json.loads(old_event.read_text(encoding="utf-8"))["dispatch_state"] == "cancelled"
        fd(root, "request-review", "FD-002", "--reason", "duplicate", success=False)
        fd(root, "claim", "FD-002", receipt["event_id"], "--session", "smoke-reviewer")
        fd(root, "request-review", "FD-002", "--reason", "in flight", success=False)
        review.write_text("Changes requested: resolve open evidence gates.\n", encoding="utf-8")
        fd(root, "emit", "FD-002", "changes-requested", "--producer", "reviewer",
           "--artifact", "docs/features/reviews/FD-001.md", "--source-event", receipt["event_id"])
        assert "**Status:** In Progress" in stalled.read_text(encoding="utf-8")
        print("FD smoke flow passed in a disposable local Git repository")


if __name__ == "__main__":
    main()
