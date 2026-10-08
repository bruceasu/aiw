"""FD-032 public FD CLI risk-decision tests.

Run only after the Planner authorizes the exact revision-bound command. The
existing FD-014 fixture copies the CLI into a temporary Git repository; all
receipts and reports produced below stay in that temporary repository.
"""

import json
import os
from pathlib import Path
import unittest

import tests.test_fd014_blackbox as fixture


class RiskDecisionBlackBox(unittest.TestCase):
    def setUp(self):
        fixture.FD014BlackBox.setUp(self)
        coverage_config = os.environ.get("FD035_COVERAGE_CONFIG")
        if coverage_config:
            bootstrap = os.environ["FD035_COVERAGE_BOOTSTRAP"]
            self.env["COVERAGE_PROCESS_START"] = coverage_config
            self.env["COVERAGE_FILE"] = os.environ["FD035_COVERAGE_FILE"]
            self.env["PYTHONPATH"] = os.pathsep.join(
                item for item in (bootstrap, self.env.get("PYTHONPATH", "")) if item
            )

    cli = fixture.FD014BlackBox.cli
    latest = fixture.FD014BlackBox.latest
    write = fixture.FD014BlackBox.write
    write_dual = fixture.FD014BlackBox.write_dual
    ready_worker = fixture.FD014BlackBox.ready_worker
    make_dual_tester_report = fixture.FD014BlackBox.make_dual_tester_report

    def authorization_record(self, fd_id, implementation):
        return self.write_dual(
            f"docs/features/reports/{fd_id}-test-authorization-fixture.md",
            "planner-authorization", implementation["event_id"],
            {
                "decision": "approved",
                "basis": "planner-low-risk",
                "implementation_event": implementation["event_id"],
                "fd_revision": implementation["fd_revision"],
                "fd_digest": implementation["fd_sha256"],
                "tester_session": "tester-1",
                "command": "python -B -m unittest fixture",
                "working_directory": "temporary project root",
                "scope": "Synthetic authorization parser fixture only.",
                "expected_duration": "Under one second.",
                "side_effects": "Temporary fixture files only.",
                "risk_review": "No real test command is executed by this record.",
                "human_approval": "not required",
                "planner_identity": "planner-fixture",
                "decision_time": "2026-10-08T04:00:00+00:00",
            },
        )

    def begin_pm(self, *, failed=True, low_coverage=False):
        fd_id, implementation = self.ready_worker(dual=True)
        self.cli("claim", fd_id, implementation["event_id"], "--session", "tester-1")
        report = self.make_dual_tester_report(fd_id, implementation)
        sidecar = self.root / Path(report).with_suffix(".json")
        payload = json.loads(sidecar.read_text(encoding="utf-8"))
        authorization = self.authorization_record(fd_id, implementation)
        payload["data"].update(
            {
                "covered_scenarios": 0 if failed else 1,
                "executed_behavior_tests": 1,
                "passed_behavior_tests": 0 if failed else 1,
                "failed_behavior_tests": 1 if failed else 0,
                "unrun_behavior_tests": 0,
                "requirements_coverage": "0%" if failed else "100%",
                "authorization_records": [authorization],
                "commands": ["python -B -m unittest fixture"],
                "recommendation": "fail" if failed else "pass",
                "residual_risk": "One behavior failed." if failed else "Branch coverage was not measured.",
                "scenarios": [{"id": "S01", "behavior": "Synthetic behavior",
                               "status": "uncovered" if failed else "passed",
                               **({} if failed else {"test_cases": ["synthetic_case"]})}],
            }
        )
        if low_coverage:
            payload["data"].update(
                {
                    "applicable_scenarios": 25,
                    "covered_scenarios": 19,
                    "executed_behavior_tests": 8,
                    "passed_behavior_tests": 8,
                    "failed_behavior_tests": 0,
                    "unrun_behavior_tests": 6,
                    "requirements_coverage": "76%",
                    "scenarios": [
                        {
                            "id": f"S{index:02d}",
                            "behavior": f"Synthetic observable behavior {index:02d}",
                            "status": "passed" if index <= 19 else "uncovered",
                            **({"test_cases": [f"synthetic_case_{index:02d}"]}
                               if index <= 19 else {"reason": "Not executed in this fixture."}),
                        }
                        for index in range(1, 26)
                    ],
                }
            )
            report_path = self.root / report
            report_path.write_text(
                f"# {fd_id} synthetic low-coverage report\n\n"
                f"<!-- aiw-data: {sidecar.name} -->\n\n"
                f"**Implementation event:** {implementation['event_id']}\n"
                f"**Tested FD revision:** {implementation['fd_revision']}\n"
                f"**Tested FD digest:** {implementation['fd_sha256']}\n"
                "**Tester session:** tester-1\n"
                "**Applicable scenarios:** 25\n"
                "**Covered scenarios:** 19\n"
                "**Executed behavior tests:** 8\n"
                "**Passed behavior tests:** 8\n"
                "**Failed behavior tests:** 0\n"
                "**Unrun behavior tests:** 6\n"
                "**Requirements coverage:** 76%\n"
                "**Branch coverage:** not measured\n"
                "**Raw coverage evidence:** Synthetic fixture; branch measurement is separate.\n"
                "**Coverage unavailable reason:** Not part of this synthetic fixture.\n"
                "**Test files:** tests/test_fd032_risk_decision_blackbox.py\n"
                f"**Authorization records:** {authorization}\n"
                "**Commands:** python -B -m unittest fixture\n"
                "**Recommendation:** pass\n"
                "**Residual risk:** Six synthetic scenarios remain uncovered.\n",
                encoding="utf-8",
            )
        sidecar.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")
        self.cli(
            "emit", fd_id, "test-report-ready", "--producer", "tester",
            "--artifact", report, "--source-event", implementation["event_id"],
        )
        pm_event = self.latest(fd_id)
        self.assertEqual(pm_event["target_role"], "pm")
        # PM is a human handoff; the fixture emits its decision without a claim.
        return fd_id, report, pm_event

    def assessment(self, fd_id, report, pm_event, letter, vote, **changes):
        data = {
            "tester_report": report,
            "fd_revision": pm_event["fd_revision"],
            "fd_digest": pm_event["fd_sha256"],
            "assessor_session": f"assessor-{letter}",
            "vote": vote,
            "severity": "Medium: synthetic behavior failed.",
            "impact_scope": "The single synthetic behavior in the temporary FD.",
            "estimated_repair_time": "One hour in this fixture.",
            "delivery_impact": "One hour delay if repaired.",
            "rationale": "The Tester report preserves the failed case and 0% coverage.",
            "residual_risk": "The failed behavior remains if delivered.",
            "uncertainty": "No production impact measured.",
            "assessment_time": "2026-10-08T04:00:00+00:00",
        }
        data.update(changes)
        return self.write_dual(
            f"docs/features/reports/{fd_id}-test-risk-assessment-r1-{letter}.md",
            "test-risk-assessment", pm_event["event_id"], data,
        )

    def decision(self, fd_id, report, pm_event, assessments, votes, disposition, **changes):
        data = {
            "disposition": disposition,
            "tester_report": report,
            "fd_revision": pm_event["fd_revision"],
            "fd_digest": pm_event["fd_sha256"],
            "requirements_coverage": pm_event["test_summary"]["requirements_coverage"],
            "branch_coverage": "not measured",
            "failed_behavior_tests": pm_event["test_summary"]["failed_behavior_tests"],
            "assessments": assessments,
            "accept_votes": votes,
            "repair_votes": len(assessments) - votes,
            "rationale": "Follow the independent vote result.",
            "exceptions": "The test facts remain unchanged.",
            "residual_risk": "Any uncovered behavior remains unverified.",
            "pm_identity": "pm-1",
            "decision_time": "2026-10-08T04:05:00+00:00",
        }
        data.update(changes)
        return self.write_dual(
            f"docs/features/reports/{fd_id}-test-decision-r1.md",
            "pm-decision", pm_event["event_id"], data,
        )

    def adaptive_decision(self, fd_id, report, pm_event, assessments, votes,
                          disposition, *, mode, reason="none", gap="bounded"):
        return self.decision(
            fd_id, report, pm_event, assessments, votes, disposition,
            assessment_policy="adaptive-v1", assessment_mode=mode,
            escalation_reason=reason, escalation_detail="Synthetic PM reason.",
            coverage_gap_disposition=gap,
            coverage_gap_reason="PM inspected the available and missing evidence.",
        )

    def emit_decision(self, fd_id, pm_event, decision, accepted=True, ok=True):
        return self.cli(
            "emit", fd_id, "test-accepted" if accepted else "test-rejected",
            "--producer", "pm", "--artifact", decision,
            "--source-event", pm_event["event_id"], ok=ok,
        )

    def test_clean_single_vote_routes_and_keeps_reviewer_independent(self):
        fd_id, report, pm_event = self.begin_pm(failed=False)
        assessor = self.assessment(
            fd_id, report, pm_event, "a", "accept-with-risk",
            assessment_focus="acceptance-impact",
        )
        decision = self.adaptive_decision(
            fd_id, report, pm_event, [assessor], 1, "accepted", mode="single",
        )
        self.emit_decision(fd_id, pm_event, decision)
        reviewer = self.latest(fd_id)
        self.assertEqual(reviewer["assessor_session_refs"], ["assessor-a"])
        self.cli("claim", fd_id, reviewer["event_id"], "--session", "assessor-a", ok=False)
        self.cli("claim", fd_id, reviewer["event_id"], "--session", "reviewer-1")

    def test_clean_single_repair_vote_cannot_be_accepted(self):
        fd_id, report, pm_event = self.begin_pm(failed=False)
        assessor = self.assessment(
            fd_id, report, pm_event, "a", "repair",
            assessment_focus="acceptance-impact",
        )
        decision = self.adaptive_decision(
            fd_id, report, pm_event, [assessor], 0, "accepted", mode="single",
        )
        self.emit_decision(fd_id, pm_event, decision, ok=False)
        self.assertEqual(self.latest(fd_id)["event_id"], pm_event["event_id"])
        decision = self.adaptive_decision(
            fd_id, report, pm_event, [assessor], 0, "rejected", mode="single",
        )
        self.emit_decision(fd_id, pm_event, decision, accepted=False)
        self.assertEqual(self.latest(fd_id)["target_role"], "worker")

    def test_failed_behavior_requires_three_distinct_focuses(self):
        fd_id, report, pm_event = self.begin_pm()
        a = self.assessment(fd_id, report, pm_event, "a", "accept-with-risk",
                            assessment_focus="acceptance-impact")
        decision = self.adaptive_decision(
            fd_id, report, pm_event, [a], 1, "accepted", mode="single",
        )
        self.emit_decision(fd_id, pm_event, decision, ok=False)
        b = self.assessment(fd_id, report, pm_event, "b", "repair",
                            assessment_focus="acceptance-impact")
        c = self.assessment(fd_id, report, pm_event, "c", "accept-with-risk",
                            assessment_focus="delivery-operations")
        decision = self.adaptive_decision(
            fd_id, report, pm_event, [a, b, c], 2, "accepted",
            mode="escalated", reason="failed-tests",
        )
        self.emit_decision(fd_id, pm_event, decision, ok=False)
        b = self.assessment(fd_id, report, pm_event, "b", "repair",
                            assessment_focus="technical-repair")
        decision = self.adaptive_decision(
            fd_id, report, pm_event, [a, b, c], 2, "accepted",
            mode="escalated", reason="failed-tests",
        )
        self.emit_decision(fd_id, pm_event, decision)
        self.assertEqual(self.latest(fd_id)["assessor_session_refs"],
                         ["assessor-a", "assessor-b", "assessor-c"])

    def assert_escalated_three_votes(self, reason, gap):
        fd_id, report, pm_event = self.begin_pm(failed=False)
        a = self.assessment(fd_id, report, pm_event, "a", "repair",
                            assessment_focus="acceptance-impact")
        b = self.assessment(fd_id, report, pm_event, "b", "accept-with-risk",
                            assessment_focus="technical-repair")
        c = self.assessment(fd_id, report, pm_event, "c", "accept-with-risk",
                            assessment_focus="delivery-operations")
        decision = self.adaptive_decision(
            fd_id, report, pm_event, [a, b, c], 2, "accepted",
            mode="escalated", reason=reason, gap=gap,
        )
        self.emit_decision(fd_id, pm_event, decision)
        self.assertEqual(self.latest(fd_id)["target_role"], "reviewer")

    def test_material_gap_uses_three_votes(self):
        self.assert_escalated_three_votes("material-evidence-gap", "material")

    def test_pm_disagreement_uses_three_votes(self):
        self.assert_escalated_three_votes("pm-disagreement", "bounded")

    def test_two_accept_votes_route_failed_low_coverage_report_to_reviewer(self):
        fd_id, report, pm_event = self.begin_pm()
        before_report = (self.root / report).read_bytes()
        before_data = (self.root / Path(report).with_suffix(".json")).read_bytes()
        assessments = [
            self.assessment(fd_id, report, pm_event, letter, vote)
            for letter, vote in zip("abc", ("accept-with-risk", "repair", "accept-with-risk"))
        ]
        decision = self.decision(fd_id, report, pm_event, assessments, 2, "accepted")
        self.emit_decision(fd_id, pm_event, decision)
        reviewer = self.latest(fd_id)
        self.assertEqual((reviewer["event_type"], reviewer["target_role"]), ("test-accepted", "reviewer"))
        self.assertEqual((self.root / report).read_bytes(), before_report)
        self.assertEqual((self.root / Path(report).with_suffix(".json")).read_bytes(), before_data)
        for session in ("worker-1", "tester-1", "assessor-a", "assessor-b", "assessor-c"):
            with self.subTest(session=session):
                self.cli("claim", fd_id, reviewer["event_id"], "--session", session, ok=False)
        self.cli("claim", fd_id, reviewer["event_id"], "--session", "reviewer-1")

    def test_one_accept_vote_returns_to_worker_and_mismatched_pm_vote_fails(self):
        fd_id, report, pm_event = self.begin_pm()
        assessments = [
            self.assessment(fd_id, report, pm_event, letter, vote)
            for letter, vote in zip("abc", ("repair", "accept-with-risk", "repair"))
        ]
        decision = self.decision(fd_id, report, pm_event, assessments, 1, "accepted")
        self.emit_decision(fd_id, pm_event, decision, accepted=True, ok=False)
        self.assertEqual(self.latest(fd_id)["event_id"], pm_event["event_id"])
        decision = self.decision(fd_id, report, pm_event, assessments, 1, "rejected")
        self.emit_decision(fd_id, pm_event, decision, accepted=False)
        self.assertEqual((self.latest(fd_id)["event_type"], self.latest(fd_id)["target_role"]),
                         ("test-rejected", "worker"))

    def test_missing_duplicate_stale_forged_and_conflicting_assessments_fail(self):
        fd_id, report, pm_event = self.begin_pm()
        a = self.assessment(fd_id, report, pm_event, "a", "accept-with-risk")
        b = self.assessment(fd_id, report, pm_event, "b", "accept-with-risk")
        c = self.assessment(fd_id, report, pm_event, "c", "repair")
        decision = self.decision(fd_id, report, pm_event, [a, b, c], 2, "accepted")
        c_data_path = self.root / Path(c).with_suffix(".json")
        original_c = c_data_path.read_text(encoding="utf-8")
        decision_path = self.root / Path(decision).with_suffix(".json")
        original_decision = decision_path.read_text(encoding="utf-8")

        def reject_with_assessments(paths):
            payload = json.loads(original_decision)
            payload["data"]["assessments"] = paths
            decision_path.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
            self.emit_decision(fd_id, pm_event, decision, ok=False)
            self.assertEqual(self.latest(fd_id)["event_id"], pm_event["event_id"])

        with self.subTest(case="missing"):
            reject_with_assessments([a, b, f"docs/features/reports/{fd_id}-missing.md"])
        with self.subTest(case="duplicate-path"):
            reject_with_assessments([a, a, c])
        for case, update in (
            ("stale-digest", {"fd_digest": "0" * 64}),
            ("duplicate-session", {"assessor_session": "assessor-a"}),
            ("worker-session", {"assessor_session": "worker-1"}),
            ("tester-session", {"assessor_session": "tester-1"}),
            ("pm-session", {"assessor_session": "pm-1"}),
        ):
            with self.subTest(case=case):
                payload = json.loads(original_c)
                payload["data"].update(update)
                c_data_path.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
                reject_with_assessments([a, b, c])
                c_data_path.write_text(original_c, encoding="utf-8")
        with self.subTest(case="forged-source-event"):
            payload = json.loads(original_c)
            payload["source_event"] = "FD-999-000001-test-report-ready"
            c_data_path.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
            reject_with_assessments([a, b, c])

    def test_invalid_adaptive_policy_mode_or_reason_is_rejected(self):
        fd_id, report, pm_event = self.begin_pm(failed=False)
        assessor = self.assessment(
            fd_id, report, pm_event, "a", "accept-with-risk",
            assessment_focus="acceptance-impact",
        )
        decision = self.adaptive_decision(
            fd_id, report, pm_event, [assessor], 1, "accepted", mode="single",
        )
        decision_data = self.root / Path(decision).with_suffix(".json")
        original_decision = json.loads(decision_data.read_text(encoding="utf-8"))

        invalid_decisions = (
            ("policy", {"assessment_policy": "adaptive-v2"}),
            ("mode", {"assessment_mode": "majority"}),
            ("reason", {"assessment_mode": "escalated", "escalation_reason": "coverage-threshold"}),
        )
        for case, changes in invalid_decisions:
            with self.subTest(case=case):
                payload = json.loads(json.dumps(original_decision))
                payload["data"].update(changes)
                decision_data.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")

                result = self.emit_decision(fd_id, pm_event, decision, ok=False)
                self.assertTrue((result.stdout + result.stderr).strip())
                self.assertEqual(self.latest(fd_id)["event_id"], pm_event["event_id"])

    def test_assessment_revision_mismatch_is_rejected(self):
        fd_id, report, pm_event = self.begin_pm(failed=False)
        assessor = self.assessment(
            fd_id, report, pm_event, "a", "accept-with-risk",
            assessment_focus="acceptance-impact",
        )
        assessment_data = self.root / Path(assessor).with_suffix(".json")
        payload = json.loads(assessment_data.read_text(encoding="utf-8"))
        payload["data"]["fd_revision"] = pm_event["fd_revision"] - 1
        assessment_data.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
        decision = self.adaptive_decision(
            fd_id, report, pm_event, [assessor], 1, "accepted", mode="single",
        )

        result = self.emit_decision(fd_id, pm_event, decision, ok=False)
        self.assertTrue((result.stdout + result.stderr).strip())
        self.assertEqual(self.latest(fd_id)["event_id"], pm_event["event_id"])

    def test_assessment_missing_required_risk_field_is_rejected(self):
        fd_id, report, pm_event = self.begin_pm(failed=False)
        assessor = self.assessment(
            fd_id, report, pm_event, "a", "accept-with-risk",
            assessment_focus="acceptance-impact",
        )
        assessment_data = self.root / Path(assessor).with_suffix(".json")
        payload = json.loads(assessment_data.read_text(encoding="utf-8"))
        del payload["data"]["severity"]
        assessment_data.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
        decision = self.adaptive_decision(
            fd_id, report, pm_event, [assessor], 1, "accepted", mode="single",
        )

        result = self.emit_decision(fd_id, pm_event, decision, ok=False)
        self.assertTrue((result.stdout + result.stderr).strip())
        self.assertEqual(self.latest(fd_id)["event_id"], pm_event["event_id"])

    def test_material_gap_cannot_use_single_assessment(self):
        fd_id, report, pm_event = self.begin_pm(failed=False)
        assessor = self.assessment(
            fd_id, report, pm_event, "a", "accept-with-risk",
            assessment_focus="acceptance-impact",
        )
        decision = self.adaptive_decision(
            fd_id, report, pm_event, [assessor], 1, "accepted", mode="single", gap="material",
        )

        result = self.emit_decision(fd_id, pm_event, decision, ok=False)
        self.assertTrue((result.stdout + result.stderr).strip())
        self.assertEqual(self.latest(fd_id)["event_id"], pm_event["event_id"])

    def test_escalated_decision_requires_all_assessment_focuses(self):
        fd_id, report, pm_event = self.begin_pm(failed=False)
        assessments = [
            self.assessment(fd_id, report, pm_event, "a", "accept-with-risk",
                            assessment_focus="acceptance-impact"),
            self.assessment(fd_id, report, pm_event, "b", "accept-with-risk",
                            assessment_focus="technical-repair"),
            self.assessment(fd_id, report, pm_event, "c", "accept-with-risk",
                            assessment_focus="delivery-operations"),
        ]
        third_data = self.root / Path(assessments[2]).with_suffix(".json")
        payload = json.loads(third_data.read_text(encoding="utf-8"))
        del payload["data"]["assessment_focus"]
        third_data.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
        decision = self.adaptive_decision(
            fd_id, report, pm_event, assessments, 3, "accepted",
            mode="escalated", reason="material-evidence-gap", gap="material",
        )

        result = self.emit_decision(fd_id, pm_event, decision, ok=False)
        self.assertTrue((result.stdout + result.stderr).strip())
        self.assertEqual(self.latest(fd_id)["event_id"], pm_event["event_id"])

    def test_low_scenario_coverage_does_not_force_escalation(self):
        fd_id, report, pm_event = self.begin_pm(failed=False, low_coverage=True)
        self.assertEqual(pm_event["test_summary"]["requirements_coverage"], "76%")
        self.assertEqual(pm_event["test_summary"]["failed_behavior_tests"], 0)
        assessor = self.assessment(
            fd_id, report, pm_event, "a", "accept-with-risk",
            assessment_focus="acceptance-impact",
        )
        decision = self.adaptive_decision(
            fd_id, report, pm_event, [assessor], 1, "accepted", mode="single",
        )

        self.emit_decision(fd_id, pm_event, decision)
        accepted = self.latest(fd_id)
        self.assertEqual((accepted["event_type"], accepted["target_role"]),
                         ("test-accepted", "reviewer"))
        self.assertEqual(accepted["test_summary"]["requirements_coverage"], "76%")
        self.assertEqual(accepted["assessor_session_refs"], ["assessor-a"])


if __name__ == "__main__":
    unittest.main()
