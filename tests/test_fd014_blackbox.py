"""FD-014 public CLI contract tests. Run only when test execution is authorized.

Suggested command: python -B -m unittest tests.test_fd014_blackbox -v

The CLI is copied into a temporary project so these cases cannot change the
working repository's FD files or receipts. Tests use commands and receipts,
without importing or inspecting the implementation module.
"""

import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


SOURCE_ROOT = Path(__file__).resolve().parents[1]


class FD014BlackBox(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="fd014-blackbox-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / "plugins").mkdir()
        (self.root / "docs" / "features").mkdir(parents=True)
        (self.root / "docs" / "templates").mkdir(parents=True)
        shutil.copyfile(SOURCE_ROOT / "plugins" / "aiw-fd.py", self.root / "plugins" / "aiw-fd.py")
        shutil.copyfile(
            SOURCE_ROOT / "docs" / "templates" / "TEMPLATE.md",
            self.root / "docs" / "templates" / "TEMPLATE.md",
        )
        self.env = os.environ.copy()
        self.env.pop("AIW_FD_ROLE_RUNNER", None)
        self.env["PYTHONDONTWRITEBYTECODE"] = "1"
        git_init = subprocess.run(
            ["git", "init", "--quiet"],
            cwd=self.root,
            env=self.env,
            capture_output=True,
            text=True,
            timeout=15,
        )
        self.assertEqual(git_init.returncode, 0, git_init.stdout + git_init.stderr)

    def cli(self, *args, ok=True):
        result = subprocess.run(
            [sys.executable, str(self.root / "plugins" / "aiw-fd.py"), *args],
            cwd=self.root,
            env=self.env,
            capture_output=True,
            text=True,
            timeout=15,
        )
        if ok:
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        else:
            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        return result

    def latest(self, fd_id):
        paths = sorted((self.root / ".ai" / "fd" / fd_id / "events").glob("*.json"))
        self.assertTrue(paths)
        return json.loads(paths[-1].read_text(encoding="utf-8"))

    def write(self, relative, content):
        path = self.root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
        return relative

    def write_dual(self, relative, kind, source_event, data, *, omit_json=False, json_event=None):
        md_path = Path(relative)
        json_name = md_path.with_suffix(".json").name
        self.write(
            relative,
            f"# {md_path.stem} 测试证据\n\n"
            f"<!-- aiw-data: {json_name} -->\n\n"
            "## 结论\n\n这是一份临时项目中的公开 CLI 契约夹具。\n",
        )
        if not omit_json:
            payload = {
                "schema": "aiw.fd.evidence.v1",
                "kind": kind,
                "fd_id": md_path.name.split("-", 2)[0] + "-" + md_path.name.split("-", 2)[1],
                "source_event": json_event or source_event,
                "human_report": md_path.name,
                "data": data,
            }
            self.write(str(md_path.with_suffix(".json")).replace("\\", "/"), json.dumps(payload, ensure_ascii=False, indent=2))
        return relative

    def ready_worker(self, independent=True, dual=False):
        self.cli("new", "Independent test contract")
        fd_path = next((self.root / "docs" / "features").glob("FD-*_*.md"))
        fd_id = fd_path.name.split("_", 1)[0]
        planner = self.latest(fd_id)
        self.cli("claim", fd_id, planner["event_id"], "--session", "planner-1")
        policy = "**Test policy:** Independent\n" if independent else ""
        evidence_policy = "**Evidence policy:** Dual\n" if dual else ""
        fd_path.write_text(
            f"# {fd_id}: Contract fixture\n\n"
            f"**Status:** Design\n**Revision:** {planner['fd_revision']}\n**Priority:** Medium\n"
            f"{policy}{evidence_policy}\n"
            "## Problem\n\nNeed a public workflow.\n\n"
            "## Options and decision\n\nUse an explicit event sequence.\n\n"
            "## Solution\n\nExpose role handoffs.\n\n"
            "## Scope\n\nCLI handoffs only.\n\n"
            "## Work items\n\n- [ ] 1.1 Implement handoff.\n\n"
            "## Acceptance\n\n1. Tester and PM precede Reviewer.\n\n"
            "## Verification\n\n- Prepared for black-box testing.\n\n"
            "## Sources\n\n- FD-014.\n",
            encoding="utf-8",
        )
        design_result = self.cli(
            "emit", fd_id, "design-ready", "--producer", "planner",
            "--artifact", fd_path.relative_to(self.root).as_posix(),
            "--source-event", planner["event_id"],
        )
        worker = self.latest(fd_id)
        if worker["target_role"] != "worker":
            receipts = [
                (path.name, receipt["event_type"], receipt["target_role"])
                for path in sorted((self.root / ".ai" / "fd" / fd_id / "events").glob("*.json"))
                for receipt in [json.loads(path.read_text(encoding="utf-8"))]
            ]
            self.fail(
                f"Expected Worker after design-ready; latest={worker!r}; "
                f"CLI stdout={design_result.stdout!r}, stderr={design_result.stderr!r}; "
                f"receipts={receipts!r}"
            )
        self.cli("claim", fd_id, worker["event_id"], "--session", "worker-1")
        fd_path.write_text(fd_path.read_text(encoding="utf-8").replace("- [ ] 1.1", "- [x] 1.1"), encoding="utf-8")
        report_path = f"docs/features/reports/{fd_id}-implementation.md"
        report = (
            self.write_dual(
                report_path, "worker-report", worker["event_id"],
                {
                    "summary": "临时夹具已实现交接。",
                    "changed_paths": [], "commands": [], "results": [],
                    "residual_risk": "仅为测试夹具。",
                },
            ) if dual else self.write(report_path, "# Worker report\n\nWork item complete.\n")
        )
        self.cli(
            "emit", fd_id, "implementation-ready", "--producer", "worker",
            "--artifact", report, "--source-event", worker["event_id"],
        )
        return fd_id, self.latest(fd_id)

    def authorization_record(
        self, fd_id, implementation, *, wrong_event=False,
        basis="planner-low-risk", human_approval="not required",
    ):
        event_id = "FD-999-000001-implementation-ready" if wrong_event else implementation["event_id"]
        return self.write(
            f"docs/features/reports/{fd_id}-test-authorization-fixture.md",
            f"# {fd_id} synthetic Planner authorization fixture\n\n"
            "**Decision:** approved\n"
            f"**Basis:** {basis}\n"
            f"**Implementation event:** {event_id}\n"
            f"**FD revision:** {implementation['fd_revision']}\n"
            f"**FD digest:** {implementation['fd_sha256']}\n"
            "**Tester session:** tester-1\n"
            "**Command:** python -B -m unittest fixture\n"
            "**Working directory:** temporary project root\n"
            "**Scope:** Synthetic authorization parser fixture only.\n"
            "**Expected duration:** under 1 second\n"
            "**Side effects:** Temporary fixture files only.\n"
            "**Risk review:** No real command is executed by this fixture record.\n"
            f"**Human approval:** {human_approval}\n"
            "**Planner identity:** planner-fixture\n"
            "**Decision time:** 2026-10-02T00:00:00+00:00\n",
        )

    def make_tester_report(
        self, fd_id, implementation, *, failed=0, omit_session=False,
        omit_authorization=False, wrong_authorization_event=False,
        approval_basis="planner-low-risk", human_approval="not required",
    ):
        executed = failed
        covered = 1 if executed else 0
        session = "" if omit_session else "**Tester session:** tester-1\n"
        authorization = (
            self.authorization_record(
                fd_id, implementation, wrong_event=wrong_authorization_event,
                basis=approval_basis, human_approval=human_approval,
            )
            if executed and not omit_authorization else "none"
        )
        command = "python -B -m unittest fixture" if executed else "none"
        report = self.write(
            f"docs/features/reports/{fd_id}-test-report-r1.md",
            f"# {fd_id} test report\n\n"
            f"**Implementation event:** {implementation['event_id']}\n"
            f"**Tested FD revision:** {implementation['fd_revision']}\n"
            f"**Tested FD digest:** {implementation['fd_sha256']}\n"
            f"{session}"
            "**Applicable scenarios:** 1\n"
            f"**Covered scenarios:** {covered}\n"
            f"**Executed behavior tests:** {executed}\n"
            "**Passed behavior tests:** 0\n"
            f"**Failed behavior tests:** {failed}\n"
            f"**Unrun behavior tests:** {0 if executed else 1}\n"
            f"**Requirements coverage:** {'100%' if covered else '0%'}\n"
            "**Branch coverage:** not measured\n"
            "**Raw coverage evidence:** not measured\n"
            "**Coverage unavailable reason:** Fixture has no coverage tool.\n"
            "**Test files:** tests/test_fd014_blackbox.py\n"
            f"**Authorization records:** {authorization}\n"
            f"**Commands:** {command}\n"
            f"**Recommendation:** {'fail' if failed else 'blocked'}\n"
            "**Residual risk:** Fixture is synthetic and cannot establish production behavior.\n\n"
            "## Scenario mapping\n\nOne synthetic fixture scenario.\n",
        )
        return report

    def make_dual_tester_report(self, fd_id, implementation, *, omit_json=False, json_event=None):
        return self.write_dual(
            f"docs/features/reports/{fd_id}-test-report-r1.md",
            "tester-report", implementation["event_id"],
            {
                "implementation_event": implementation["event_id"],
                "tested_fd_revision": implementation["fd_revision"],
                "tested_fd_digest": implementation["fd_sha256"],
                "tester_session": "tester-1",
                "applicable_scenarios": 1,
                "covered_scenarios": 0,
                "executed_behavior_tests": 0,
                "passed_behavior_tests": 0,
                "failed_behavior_tests": 0,
                "unrun_behavior_tests": 1,
                "requirements_coverage": "0%",
                "branch_coverage": "not measured",
                "raw_coverage_evidence": "not measured",
                "coverage_unavailable_reason": "临时夹具没有覆盖率命令。",
                "test_files": ["tests/test_fd014_blackbox.py"],
                "authorization_records": [],
                "commands": [],
                "recommendation": "blocked",
                "not_applicable_reason": "none",
                "residual_risk": "该场景未执行。",
                "scenarios": [{"id": "E01", "status": "uncovered", "behavior": "临时场景"}],
                "static_exclusions": [],
            },
            omit_json=omit_json, json_event=json_event,
        )

    def pm_decision(self, fd_id, pm_event, report, *, disposition="accepted", exception="PM accepts fixture gap", requirements_coverage="0%"):
        return self.write(
            f"docs/features/reports/{fd_id}-test-decision-r1.md",
            f"# {fd_id} PM decision\n\n"
            f"**Disposition:** {disposition}\n"
            f"**Tester report:** {report}\n"
            f"**FD revision:** {pm_event['fd_revision']}\n"
            f"**FD digest:** {pm_event['fd_sha256']}\n"
            f"**Requirements coverage:** {requirements_coverage}\n"
            "**Branch coverage:** not measured\n"
            "**Rationale:** Check the documented decision gate.\n"
            f"**Exceptions:** {exception}\n"
            "**Residual risk:** No real coverage measurement exists in the fixture.\n"
            "**PM identity:** pm-1\n"
            "**Decision time:** 2026-10-02T00:00:00+00:00\n",
        )

    def test_independent_policy_routes_through_tester_and_pm(self):
        fd_id, implementation = self.ready_worker()
        self.assertEqual(implementation["target_role"], "tester")
        self.assertEqual(implementation["event_type"], "implementation-ready")
        self.cli("claim", fd_id, implementation["event_id"], "--session", "worker-1", ok=False)
        self.cli("claim", fd_id, implementation["event_id"], "--session", "tester-1")
        report = self.write(
            f"docs/features/reports/{fd_id}-test-report-r1.md",
            f"# {fd_id} test report\n\n"
            f"**Implementation event:** {implementation['event_id']}\n"
            f"**Tested FD revision:** {implementation['fd_revision']}\n"
            f"**Tested FD digest:** {implementation['fd_sha256']}\n"
            "**Tester session:** tester-1\n"
            "**Applicable scenarios:** 1\n**Covered scenarios:** 0\n"
            "**Executed behavior tests:** 0\n**Passed behavior tests:** 0\n"
            "**Failed behavior tests:** 0\n**Unrun behavior tests:** 1\n"
            "**Requirements coverage:** 0%\n**Branch coverage:** not measured\n"
            "**Raw coverage evidence:** not measured\n"
            "**Coverage unavailable reason:** test execution not authorized\n"
            "**Test files:** tests/test_fd014_blackbox.py\n"
            "**Authorization records:** none\n"
            "**Commands:** none\n**Recommendation:** blocked\n"
            "**Residual risk:** No behavior or branch measurement has run.\n\n"
            "## Scenario mapping\n\n1. Prepared; unrun because authorization is absent.\n",
        )
        self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", implementation["event_id"],
        )
        pm_event = self.latest(fd_id)
        self.assertEqual(pm_event["target_role"], "pm")
        self.assertEqual(pm_event["event_type"], "test-report-ready")
        decision = self.write(
            f"docs/features/reports/{fd_id}-test-decision-r1.md",
            f"# {fd_id} PM decision\n\n**Disposition:** accepted\n"
            f"**Tester report:** {report}\n"
            f"**FD revision:** {pm_event['fd_revision']}\n"
            f"**FD digest:** {pm_event['fd_sha256']}\n"
            "**Requirements coverage:** 0%\n**Branch coverage:** not measured\n"
            "**Rationale:** Exercise explicit PM exception handling.\n"
            "**Exceptions:** Accept unexecuted fixture coverage for this CLI case.\n"
            "**Residual risk:** No behavior test has run in this fixture.\n"
            "**PM identity:** pm-1\n**Decision time:** 2026-10-02T00:00:00+00:00\n",
        )
        self.cli(
            "emit", fd_id, "test-accepted", "--producer", "pm",
            "--artifact", decision, "--source-event", pm_event["event_id"],
        )
        reviewer = self.latest(fd_id)
        self.assertEqual(reviewer["target_role"], "reviewer")
        self.assertEqual(reviewer["event_type"], "test-accepted")
        self.cli("claim", fd_id, reviewer["event_id"], "--session", "tester-1", ok=False)

    def test_legacy_fd_routes_directly_to_reviewer(self):
        _, implementation = self.ready_worker(independent=False)
        self.assertEqual(implementation["target_role"], "reviewer")

    def test_report_requires_labelled_tester_session(self):
        fd_id, implementation = self.ready_worker()
        self.cli("claim", fd_id, implementation["event_id"], "--session", "tester-1")
        report = self.make_tester_report(fd_id, implementation, omit_session=True)
        self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", implementation["event_id"],
            ok=False,
        )
        self.assertEqual(self.latest(fd_id)["event_id"], implementation["event_id"])

    def test_executed_evidence_requires_authorization(self):
        fd_id, implementation = self.ready_worker()
        self.cli("claim", fd_id, implementation["event_id"], "--session", "tester-1")
        report = self.make_tester_report(fd_id, implementation, failed=1, omit_authorization=True)
        self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", implementation["event_id"],
            ok=False,
        )
        self.assertEqual(self.latest(fd_id)["event_id"], implementation["event_id"])

    def test_authorization_for_other_implementation_is_rejected(self):
        fd_id, implementation = self.ready_worker()
        self.cli("claim", fd_id, implementation["event_id"], "--session", "tester-1")
        report = self.make_tester_report(fd_id, implementation, failed=1, wrong_authorization_event=True)
        self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", implementation["event_id"],
            ok=False,
        )
        self.assertEqual(self.latest(fd_id)["event_id"], implementation["event_id"])

    def check_human_approval_value(self, value, *, accepted):
        fd_id, implementation = self.ready_worker()
        self.cli("claim", fd_id, implementation["event_id"], "--session", "tester-1")
        report = self.make_tester_report(
            fd_id, implementation, failed=1,
            approval_basis="human-approved", human_approval=value,
        )
        self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", implementation["event_id"],
            ok=accepted,
        )
        outcome = self.latest(fd_id)
        self.assertEqual(outcome["event_type"], "test-report-ready" if accepted else "implementation-ready")

    def test_human_approval_denied_is_rejected(self):
        self.check_human_approval_value("denied", accepted=False)

    def test_human_approval_pending_is_rejected(self):
        self.check_human_approval_value("pending", accepted=False)

    def test_affirmative_human_approval_reference_is_accepted(self):
        self.check_human_approval_value("approved:ticket:FD014-fixture-1", accepted=True)

    def test_dual_report_handoff_with_chinese_markdown_and_json(self):
        fd_id, implementation = self.ready_worker(dual=True)
        self.cli("claim", fd_id, implementation["event_id"], "--session", "tester-1")
        report = self.make_dual_tester_report(fd_id, implementation)
        self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", implementation["event_id"],
        )
        self.assertEqual(self.latest(fd_id)["target_role"], "pm")

    def test_dual_report_missing_json_blocks_handoff(self):
        fd_id, implementation = self.ready_worker(dual=True)
        self.cli("claim", fd_id, implementation["event_id"], "--session", "tester-1")
        report = self.make_dual_tester_report(fd_id, implementation, omit_json=True)
        result = self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", implementation["event_id"],
            ok=False,
        )
        self.assertIn("artifact does not exist inside the repository:", result.stderr)
        self.assertEqual(self.latest(fd_id)["event_id"], implementation["event_id"])

    def test_dual_report_mismatched_json_event_blocks_handoff(self):
        fd_id, implementation = self.ready_worker(dual=True)
        self.cli("claim", fd_id, implementation["event_id"], "--session", "tester-1")
        report = self.make_dual_tester_report(
            fd_id, implementation, json_event="FD-999-000001-implementation-ready"
        )
        result = self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", implementation["event_id"],
            ok=False,
        )
        self.assertIn("machine evidence source event differs from the handoff", result.stderr)
        self.assertEqual(self.latest(fd_id)["event_id"], implementation["event_id"])

    def test_dual_complete_archives_markdown_and_json(self):
        fd_id, implementation = self.ready_worker(dual=True)
        self.cli("claim", fd_id, implementation["event_id"], "--session", "tester-1")
        report = self.make_dual_tester_report(fd_id, implementation)
        self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", implementation["event_id"],
        )
        pm_event = self.latest(fd_id)
        decision = self.write_dual(
            f"docs/features/reports/{fd_id}-test-decision-r1.md",
            "pm-decision", pm_event["event_id"],
            {
                "disposition": "accepted", "tester_report": report,
                "fd_revision": pm_event["fd_revision"],
                "fd_digest": pm_event["fd_sha256"],
                "requirements_coverage": "0%", "branch_coverage": "not measured",
                "rationale": "验证双份证据归档。",
                "exceptions": "临时夹具允许未测的覆盖率。",
                "residual_risk": "仅验证归档路径。", "pm_identity": "pm-1",
                "decision_time": "2026-10-02T00:00:00+00:00",
            },
        )
        self.cli(
            "emit", fd_id, "test-accepted", "--producer", "pm",
            "--artifact", decision, "--source-event", pm_event["event_id"],
        )
        reviewer = self.latest(fd_id)
        self.cli("claim", fd_id, reviewer["event_id"], "--session", "reviewer-1")
        review = self.write_dual(
            f"docs/features/reviews/{fd_id}-review-r1.md",
            "reviewer-report", reviewer["event_id"],
            {"outcome": "verification-passed", "findings": [], "commands": [], "residual_risk": "归档夹具。"},
        )
        self.cli(
            "emit", fd_id, "verification-passed", "--producer", "reviewer",
            "--artifact", review, "--source-event", reviewer["event_id"],
        )
        self.cli("close", fd_id, "Complete")
        archive = self.root / "docs" / "features" / "archive" / fd_id
        for stem in (f"{fd_id}-implementation", f"{fd_id}-test-report-r1", f"{fd_id}-test-decision-r1"):
            self.assertTrue((archive / "reports" / f"{stem}.md").is_file())
            self.assertTrue((archive / "reports" / f"{stem}.json").is_file())
        self.assertTrue((archive / "reviews" / f"{fd_id}-review-r1.md").is_file())
        self.assertTrue((archive / "reviews" / f"{fd_id}-review-r1.json").is_file())

    def test_reviewer_cannot_bypass_pending_tester(self):
        fd_id, implementation = self.ready_worker()
        self.cli("request-review", fd_id, "--reason", "skip tester", ok=False)
        review = self.write(f"docs/features/reviews/{fd_id}-review.md", "# Premature review\n")
        self.cli(
            "emit", fd_id, "verification-passed", "--producer", "reviewer",
            "--artifact", review, "--source-event", implementation["event_id"],
            ok=False,
        )
        self.assertEqual(self.latest(fd_id)["event_id"], implementation["event_id"])

    def test_pm_cannot_accept_failed_executed_case(self):
        fd_id, implementation = self.ready_worker()
        self.cli("claim", fd_id, implementation["event_id"], "--session", "tester-1")
        report = self.make_tester_report(fd_id, implementation, failed=1)
        self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", implementation["event_id"],
        )
        pm_event = self.latest(fd_id)
        decision = self.pm_decision(fd_id, pm_event, report, requirements_coverage="100%")
        self.cli(
            "emit", fd_id, "test-accepted", "--producer", "pm",
            "--artifact", decision, "--source-event", pm_event["event_id"],
            ok=False,
        )
        self.assertEqual(self.latest(fd_id)["event_id"], pm_event["event_id"])

    def test_pm_rejection_returns_to_worker_and_next_implementation_retests(self):
        fd_id, implementation = self.ready_worker()
        self.cli("claim", fd_id, implementation["event_id"], "--session", "tester-1")
        report = self.make_tester_report(fd_id, implementation)
        self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", implementation["event_id"],
        )
        pm_event = self.latest(fd_id)
        decision = self.pm_decision(fd_id, pm_event, report, disposition="rejected", exception="none")
        self.cli(
            "emit", fd_id, "test-rejected", "--producer", "pm",
            "--artifact", decision, "--source-event", pm_event["event_id"],
        )
        worker = self.latest(fd_id)
        self.assertEqual(worker["target_role"], "worker")
        self.cli("claim", fd_id, worker["event_id"], "--session", "worker-2")
        second_report = self.write(
            f"docs/features/reports/{fd_id}-implementation-r2.md", "# Revised worker report\n"
        )
        self.cli(
            "emit", fd_id, "implementation-ready", "--producer", "worker",
            "--artifact", second_report, "--source-event", worker["event_id"],
        )
        next_tester = self.latest(fd_id)
        self.assertEqual(next_tester["target_role"], "tester")
        self.assertNotEqual(next_tester["event_id"], implementation["event_id"])


if __name__ == "__main__":
    unittest.main()
