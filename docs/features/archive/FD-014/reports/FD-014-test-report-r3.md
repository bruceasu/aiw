# FD-014 independent Tester report, round 3 (blocked)

**Implementation event:** FD-014-000017-implementation-ready
**Tested FD revision:** 17
**Tested FD digest:** a9eeedd2db7cc524e66f7772ad5a56c85eff60c7bedd219e2fef7d02632b0227
**Tester session:** fd014-tester-20261002-f7f15d14
**Applicable scenarios:** 30
**Covered scenarios:** 0
**Executed behavior tests:** 0
**Passed behavior tests:** 0
**Failed behavior tests:** 0
**Unrun behavior tests:** 11
**Requirements coverage:** 0%
**Branch coverage:** not measured
**Raw coverage evidence:** not measured
**Coverage unavailable reason:** No test or branch-coverage command was authorized or executed for revision 17. The prepared CLI suite does not measure business-code branches.
**Test files:** tests/test_fd014_blackbox.py
**Authorization records:** none
**Commands:** none
**Recommendation:** blocked
**Residual risk:** Eleven prepared CLI cases are unrun, nine additional executable scenarios have no case in this round, and business-code branch coverage is unavailable. The newly requested Chinese human-readable Markdown plus structured JSON/YAML machine evidence is absent from the revision-17 implementation; this legacy-format report does not satisfy that new policy.

## Scenario mapping

The denominator is the 30 independently observable behaviors below, split
from the FD's broad numbered acceptance items and stable workflow spec. A
row is covered only if its precise behavior is exercised by an executed case.
One case may cover several rows; an untested part of the same acceptance item
remains a separate uncovered row. The 11 prepared test methods have not run.

| ID | Acceptance | Distinct observable behavior | Prepared public case or gap | Current result |
| --- | --- | --- | --- | --- |
| E01 | 1 | Creating a numbered FD yields a Planner handoff. | Shared CLI fixture `ready_worker` in all cases | Uncovered; unrun |
| E02 | 1 | Planner `design-ready` yields a Worker handoff. | Shared CLI fixture `ready_worker` | Uncovered; unrun |
| E03 | 1 | Worker completion of independent-policy FD yields Tester. | `test_independent_policy_routes_through_tester_and_pm` | Uncovered; unrun |
| E04 | 1 | Worker session cannot claim Tester handoff. | `test_independent_policy_routes_through_tester_and_pm` | Uncovered; unrun |
| E05 | 1, 6 | Tester `test-report-ready` yields PM. | `test_independent_policy_routes_through_tester_and_pm` | Uncovered; unrun |
| E06 | 1, 6 | PM acceptance yields independent Reviewer handoff. | `test_independent_policy_routes_through_tester_and_pm` | Uncovered; unrun |
| E07 | 1, 6 | Tester session cannot claim Reviewer handoff. | `test_independent_policy_routes_through_tester_and_pm` | Uncovered; unrun |
| E08 | 6 | PM rejection returns the FD to Worker. | `test_pm_rejection_returns_to_worker_and_next_implementation_retests` | Uncovered; unrun |
| E09 | 6, 10 | Worker reimplementation after rejection creates a fresh Tester round. | `test_pm_rejection_returns_to_worker_and_next_implementation_retests` | Uncovered; unrun |
| E10 | 1 | A legacy FD without the policy marker routes directly to Reviewer. | `test_legacy_fd_routes_directly_to_reviewer` | Uncovered; unrun |
| E11 | 1, 6 | `request-review` cannot bypass Pending Test. | `test_reviewer_cannot_bypass_pending_tester` | Uncovered; unrun |
| E12 | 1, 6 | Reviewer cannot emit `verification-passed` while Pending Test. | `test_reviewer_cannot_bypass_pending_tester` | Uncovered; unrun |
| E13 | 2, 5 | Tester report missing required session metadata is rejected. | `test_report_requires_labelled_tester_session` | Uncovered; unrun |
| E14 | 7 | Executed-test evidence without authorization is rejected. | `test_executed_evidence_requires_authorization` | Uncovered; unrun |
| E15 | 7 | Authorization for another implementation event is rejected. | `test_authorization_for_other_implementation_is_rejected` | Uncovered; unrun |
| E16 | 7, 11 | Human-approved record saying `denied` is rejected. | `test_human_approval_denied_is_rejected` | Uncovered; unrun |
| E17 | 7, 11 | Human-approved record saying `pending` is rejected. | `test_human_approval_pending_is_rejected` | Uncovered; unrun |
| E18 | 7, 11 | Affirmative auditable human reference passes the authorization syntax gate. | `test_affirmative_human_approval_reference_is_accepted` (synthetic temporary reference only) | Uncovered; unrun |
| E19 | 2, 6 | PM cannot accept a report with failed executed behavior case. | `test_pm_cannot_accept_failed_executed_case` | Uncovered; unrun |
| E20 | 2, 6 | Explicit PM exception allows a below-threshold or unavailable measure to proceed to Reviewer. | `test_independent_policy_routes_through_tester_and_pm` | Uncovered; unrun |
| E21 | 2, 6 | PM acceptance without a required coverage exception is rejected. | No case prepared | Uncovered |
| E22 | 6 | PM decision with stale FD revision or digest is rejected. | No case prepared | Uncovered |
| E23 | 2, 5, 11 | Tester report with inconsistent scenario count and percentage is rejected. | No case prepared | Uncovered |
| E24 | 7, 10 | Authorization with wrong FD revision or digest is rejected. | No case prepared; E15 checks a different event only | Uncovered |
| E25 | 7 | Authorization for a different exact command is rejected. | No case prepared | Uncovered |
| E26 | 4, 5 | Report with unavailable branch coverage includes a reason while keeping it distinct from scenario coverage. | `test_independent_policy_routes_through_tester_and_pm` | Uncovered; unrun |
| E27 | 4 | Business-code branch coverage is measured with generated/test code excluded and unreachable branches documented. | No independent coverage tool or command authorized | Uncovered; not measured |
| E28 | 9 | Stale pending Worker handoff can be superseded by `refresh-worker` with a digest-bound successor. | Prior FD-015 review, no current Tester case | Uncovered in this round |
| E29 | 9 | Claimed or in-flight Worker handoff cannot be refreshed. | Prior FD-015 review, no current Tester case | Uncovered in this round |
| E30 | 1, 6 | A current Reviewer pass permits Complete archive with FD evidence. | No Tester case; requires later Reviewer result | Uncovered in this round |

No case ran in this round. The denominator is not reduced merely because a
scenario needs a future role, past FD, or separate tool. No credit is taken
for an earlier revision's passing suite or a synthetic fixture's stated test
result.

### Static-only evidence excluded from executable denominator

- Acceptance 2 and 11 require the authored scenario inventory and accurate
  arithmetic. This report's table and fields are reviewed as documents;
  E23 separately covers the CLI's executable count validation.
- Acceptance 3's root `tests/` location and black-box test authorship are
  verified from the test path and case structure. Separate branch measurement
  remains E27, uncovered.
- Acceptance 5's exact command/result transcription and preparation versus
  execution distinction are report evidence, not another product behavior.
- Acceptance 7's Planner judgment about whether an actual command is low risk
  or needs human escalation is a human workflow decision. E14-E18 and E24-E25
  separately cover the executable authorization gate; the actual Planner
  record for this round will document the risk judgment.
- Acceptance 8's source/installed Skill, stable spec, and usage consistency
  is static comparison evidence for Reviewer, not a runtime CLI scenario.
- Acceptance 10's r1 report preservation is historical file evidence. E09
  separately checks a fresh Tester handoff after PM rejection.

## Observations

- Tester claimed the exact `FD-014-000017-implementation-ready` handoff in
  session `fd014-tester-20261002-f7f15d14`, distinct from Worker and Reviewer.
- This round writes only the root black-box test file and this new r3 report.
  Historical r1/r2 reports and their results are preserved.
- After the revision-17 handoff, the user required human-facing reports in
  Chinese Markdown and AI/CLI evidence in structured JSON or YAML. That
  requirement is not implemented in this revision. This English Markdown
  report follows the then-current CLI template only to preserve a truthful
  blocked handoff; PM should reject it and route the change through a new
  Worker implementation and Tester round. Historical r1/r2 evidence was
  not edited.
- Proposed focused command, not executed: `python -B -m unittest
  tests.test_fd014_blackbox -v` from the repository root. The suite now has
  eleven intended public CLI cases. It reads `plugins/aiw-fd.py` and the FD
  template, copies them into one temporary Git project per case, writes and
  cleans temporary fixture files, and invokes the copied CLI in subprocesses.
  The test file has no network, elevated permission, dependency download,
  final artifact, or real repository write operation. Planner must inspect
  the copied CLI and invoked command before classifying full effects.
  Expected normal duration is under 90 seconds; each subprocess has a
  15-second timeout. `AIW_FD_ROLE_RUNNER` is removed and bytecode writes are
  disabled. No branch coverage command is proposed in this round.
