"""FD-031 recover-worker contract through the public FD CLI in temporary projects.

Run only with an authorization bound to the current FD event, revision, digest,
and Tester session. The fixture copies the CLI and writes only to a temporary
Git repository.
"""

import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


SOURCE = Path(__file__).resolve().parents[1]


class RecoverWorkerBlackBox(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="fd031-recover-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / "plugins").mkdir()
        (self.root / "docs" / "features").mkdir(parents=True)
        shutil.copy2(SOURCE / "plugins" / "aiw-fd.py", self.root / "plugins" / "aiw-fd.py")
        shutil.copy2(
            SOURCE / "docs" / "features" / "TEMPLATE.md",
            self.root / "docs" / "features" / "TEMPLATE.md",
        )
        self.env = os.environ.copy()
        self.env.pop("AIW_FD_ROLE_RUNNER", None)
        self.env["PYTHONDONTWRITEBYTECODE"] = "1"
        self.env["PYTHONIOENCODING"] = "utf-8"
        self.env["GIT_CONFIG_NOSYSTEM"] = "1"
        self.env["GIT_CONFIG_GLOBAL"] = os.devnull
        result = subprocess.run(
            ["git", "init", "--quiet"], cwd=self.root, env=self.env,
            capture_output=True, text=True, timeout=15,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def cli(self, *args, success=True):
        result = subprocess.run(
            [sys.executable, str(self.root / "plugins" / "aiw-fd.py"), *args],
            cwd=self.root, env=self.env, capture_output=True, text=True,
            timeout=15,
        )
        if success:
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        else:
            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        return result

    def receipts(self, fd_id):
        folder = self.root / ".ai" / "fd" / fd_id / "events"
        return [json.loads(path.read_text(encoding="utf-8")) for path in sorted(folder.glob("*.json"))]

    def fixture(self):
        self.cli("new", "Recovery fixture")
        (fd_path,) = (self.root / "docs" / "features").glob("FD-*_*.md")
        fd_id = fd_path.name.split("_", 1)[0]
        planner = self.receipts(fd_id)[-1]
        self.cli("claim", fd_id, planner["event_id"], "--session", "fixture-planner")
        fd_path.write_text(
            f"# {fd_id}: Recovery fixture\n\n"
            f"**Status:** Design\n**Revision:** {planner['fd_revision']}\n"
            "**Priority:** Medium\n**Test policy:** Independent\n"
            "**Evidence policy:** Dual\n\n"
            "## Problem\n\nA stopped Worker owns an event.\n\n"
            "## Options and decision\n\nUse the public recovery command.\n\n"
            "## Solution\n\nReplace only the named handoff.\n\n"
            "## Scope\n\nFD handoff only.\n\n"
            "## Work items\n\n- [ ] 1.1 Keep this item unchanged.\n\n"
            "## Acceptance\n\nThe old handoff is auditable.\n\n"
            "## Verification\n\n- Prepared fixture.\n\n"
            "## Sources\n\n- FD-031.\n",
            encoding="utf-8",
        )
        self.cli(
            "emit", fd_id, "design-ready", "--producer", "planner",
            "--artifact", fd_path.relative_to(self.root).as_posix(),
            "--source-event", planner["event_id"],
        )
        worker = self.receipts(fd_id)[-1]
        self.assertEqual((worker["target_role"], worker["dispatch_state"]), ("worker", "pending"))
        self.cli("claim", fd_id, worker["event_id"], "--session", "stopped-worker")
        worker = self.receipts(fd_id)[-1]
        self.assertEqual(worker["dispatch_state"], "dispatched")
        return fd_id, fd_path, worker

    def recover(self, fd_id, worker, *, event=None, session="stopped-worker", reason="Session stopped", success=True):
        return self.cli(
            "recover-worker", fd_id,
            "--expected-event", event or worker["event_id"],
            "--expected-session", session, "--reason", reason,
            success=success,
        )

    def test_recovery_creates_auditable_claimable_worker_event(self):
        fd_id, fd_path, worker = self.fixture()
        before = fd_path.read_text(encoding="utf-8")
        self.recover(fd_id, worker)
        old, successor = self.receipts(fd_id)[-2:]
        after = fd_path.read_text(encoding="utf-8")
        self.assertEqual(old["event_id"], worker["event_id"])
        self.assertEqual(old["dispatch_state"], "cancelled")
        self.assertEqual(old["superseded_by"], successor["event_id"])
        self.assertEqual(old["recovery_reason"], "Session stopped")
        self.assertEqual(successor["event_type"], "work-requested")
        self.assertEqual((successor["target_role"], successor["dispatch_state"]), ("worker", "pending"))
        self.assertEqual(successor["producer"], "pm")
        self.assertEqual(successor["supersedes"], worker["event_id"])
        self.assertEqual(successor["abandoned_session_ref"], "stopped-worker")
        self.assertIn("**Status:** Open", after)
        self.assertIn("- [ ] 1.1 Keep this item unchanged.", after)
        self.assertEqual(successor["fd_revision"], worker["fd_revision"] + 1)
        normalized = after.replace("\r\n", "\n").encode("utf-8")
        self.assertEqual(successor["fd_sha256"], hashlib.sha256(normalized).hexdigest())
        self.assertNotEqual(after, before)
        self.cli("claim", fd_id, successor["event_id"], "--session", "replacement-worker")
        self.assertEqual(self.receipts(fd_id)[-1]["session_ref"], "replacement-worker")

    def test_wrong_event_or_session_preserves_state(self):
        fd_id, fd_path, worker = self.fixture()
        before_fd = fd_path.read_bytes()
        before_receipts = self.receipts(fd_id)
        self.recover(fd_id, worker, event="FD-999-000001-work-requested", success=False)
        self.recover(fd_id, worker, session="another-worker", success=False)
        self.assertEqual(fd_path.read_bytes(), before_fd)
        self.assertEqual(self.receipts(fd_id), before_receipts)

    def test_invalid_reason_preserves_state(self):
        fd_id, fd_path, worker = self.fixture()
        before_fd = fd_path.read_bytes()
        before_receipts = self.receipts(fd_id)
        for reason in ("", "line one\nline two", "x" * 501):
            with self.subTest(reason=repr(reason)[:30]):
                self.recover(fd_id, worker, reason=reason, success=False)
                self.assertEqual(fd_path.read_bytes(), before_fd)
                self.assertEqual(self.receipts(fd_id), before_receipts)

    def test_non_dispatched_or_non_worker_handoff_cannot_be_recovered(self):
        self.cli("new", "Pending planner fixture")
        (fd_path,) = (self.root / "docs" / "features").glob("FD-*_*.md")
        fd_id = fd_path.name.split("_", 1)[0]
        planner = self.receipts(fd_id)[-1]
        before_fd = fd_path.read_bytes()
        before_receipts = self.receipts(fd_id)
        self.recover(fd_id, planner, session="fixture-planner", success=False)
        self.assertEqual(fd_path.read_bytes(), before_fd)
        self.assertEqual(self.receipts(fd_id), before_receipts)

    def test_index_decode_failure_rolls_back_all_recovery_writes(self):
        fd_id, fd_path, worker = self.fixture()
        index = self.root / "docs" / "features" / "FEATURE_INDEX.md"
        self.assertTrue(index.is_file())
        events = self.root / ".ai" / "fd" / fd_id / "events"
        before_fd = fd_path.read_bytes()
        before_index = index.read_bytes()
        before_events = {path.name: path.read_bytes() for path in events.glob("*.json")}

        # A separate numbered FD breaks index regeneration after recovery writes.
        # It is deliberately outside the target FD, and only inside this fixture.
        broken = self.root / "docs" / "features" / "FD-002_BROKEN.md"
        broken.write_bytes(b"\xff")
        result = self.recover(fd_id, worker, success=False)
        self.assertTrue(result.stdout or result.stderr)

        self.assertEqual(fd_path.read_bytes(), before_fd)
        self.assertEqual(index.read_bytes(), before_index)
        self.assertEqual(
            {path.name: path.read_bytes() for path in events.glob("*.json")},
            before_events,
        )
        self.assertEqual(broken.read_bytes(), b"\xff")


if __name__ == "__main__":
    unittest.main()
