"""FD-024 public CLI tests. Execute only with revision-bound authorization.

The shared fixture copies the public CLI into a temporary Git repository.
These cases inspect commands and receipts, without importing implementation code.
"""

import json
from pathlib import Path
import unittest

import tests.test_fd014_blackbox as fixture


class RefreshTesterBlackBox(unittest.TestCase):
    setUp = fixture.FD014BlackBox.setUp
    cli = fixture.FD014BlackBox.cli
    latest = fixture.FD014BlackBox.latest
    write = fixture.FD014BlackBox.write
    write_dual = fixture.FD014BlackBox.write_dual
    ready_worker = fixture.FD014BlackBox.ready_worker
    make_dual_tester_report = fixture.FD014BlackBox.make_dual_tester_report

    def stale_worker(self):
        fd_id, old = self.ready_worker(dual=True)
        fd_path = next((self.root / "docs" / "features").glob(f"{fd_id}_*.md"))
        fd_path.write_text(fd_path.read_text(encoding="utf-8") + "\nAcceptance clarification.\n", encoding="utf-8")
        return fd_id, old, fd_path

    def receipt(self, fd_id, event_id):
        for path in (self.root / ".ai" / "fd" / fd_id / "events").glob("*.json"):
            value = json.loads(path.read_text(encoding="utf-8"))
            if value["event_id"] == event_id:
                return value
        self.fail(f"Missing receipt: {event_id}")

    def test_stale_refresh_preserves_provenance_and_tester_can_report(self):
        fd_id, old, fd_path = self.stale_worker()
        original_items = [line for line in fd_path.read_text(encoding="utf-8").splitlines() if line.startswith("- [")]
        self.cli(
            "refresh-tester", fd_id, "--reason", "Acceptance clarified",
            "--artifact", old["artifact_ref"],
        )
        new = self.latest(fd_id)
        self.assertEqual(new["event_type"], "test-requested")
        self.assertEqual(new["producer"], "pm")
        self.assertEqual(new["target_role"], "tester")
        self.assertEqual(new["fd_revision"], old["fd_revision"] + 1)
        self.assertEqual(new["artifact_ref"], old["artifact_ref"])
        self.assertEqual(new["worker_session_ref"], old["worker_session_ref"])
        self.assertEqual(new["supersedes"], old["event_id"])
        self.assertEqual(new["implementation_event"], old["event_id"])
        self.assertEqual(self.receipt(fd_id, old["event_id"])["superseded_by"], new["event_id"])
        self.assertEqual(self.receipt(fd_id, old["event_id"])["dispatch_state"], "cancelled")
        self.assertIn("**Status:** Pending Test", fd_path.read_text(encoding="utf-8"))
        self.assertEqual(original_items, [line for line in fd_path.read_text(encoding="utf-8").splitlines() if line.startswith("- [")])
        self.cli("claim", fd_id, new["event_id"], "--session", "tester-1")
        report = self.make_dual_tester_report(fd_id, new)
        self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", new["event_id"],
        )
        self.assertEqual(self.latest(fd_id)["target_role"], "pm")

    def test_current_handoff_is_rejected_without_change(self):
        fd_id, current = self.ready_worker(dual=True)
        result = self.cli(
            "refresh-tester", fd_id, "--reason", "No stale revision",
            "--artifact", current["artifact_ref"], ok=False,
        )
        self.assertTrue(result.stderr)
        self.assertEqual(self.latest(fd_id), current)

    def test_claimed_handoff_is_rejected_without_change(self):
        fd_id, old = self.ready_worker(dual=True)
        self.cli("claim", fd_id, old["event_id"], "--session", "tester-1")
        fd_path = next((self.root / "docs" / "features").glob(f"{fd_id}_*.md"))
        fd_path.write_text(fd_path.read_text(encoding="utf-8") + "\nAcceptance clarification.\n", encoding="utf-8")
        before = self.latest(fd_id)
        before_text = fd_path.read_text(encoding="utf-8")
        self.cli(
            "refresh-tester", fd_id, "--reason", "Claimed event",
            "--artifact", old["artifact_ref"], ok=False,
        )
        self.assertEqual(self.latest(fd_id), before)
        self.assertEqual(fd_path.read_text(encoding="utf-8"), before_text)

    def test_missing_report_is_rejected_without_change(self):
        fd_id, old, fd_path = self.stale_worker()
        before_text = fd_path.read_text(encoding="utf-8")
        self.cli(
            "refresh-tester", fd_id, "--reason", "Missing report",
            "--artifact", f"docs/features/reports/{fd_id}-missing.md", ok=False,
        )
        self.assertEqual(self.latest(fd_id), old)
        self.assertEqual(fd_path.read_text(encoding="utf-8"), before_text)
