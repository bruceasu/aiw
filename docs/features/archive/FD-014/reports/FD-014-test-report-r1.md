# FD-014 independent Tester report, round 1

**Implementation event:** FD-014-000010-implementation-ready
**Tested FD revision:** 10
**Tested FD digest:** 534e15b840f3c69d1b426a9db9d5132461a8734c41fab83141581117f8d17c8c
**Tester session:** fd014-tester-20261002-f7f15d14
**Applicable scenarios:** 9
**Covered scenarios:** 0
**Executed behavior tests:** 0
**Passed behavior tests:** 0
**Failed behavior tests:** 0
**Unrun behavior tests:** 6
**Requirements coverage:** 0%
**Branch coverage:** not measured
**Raw coverage evidence:** not measured
**Coverage unavailable reason:** No test or branch-coverage execution was authorized in this handoff. The newly requested Planner authorization rule is pending a revised FD and implementation.
**Test files:** plugins/tests/test_fd014_blackbox.py
**Commands:** none
**Recommendation:** blocked
**Residual risk:** The public CLI transitions, role isolation, legacy compatibility, report validation, and branch coverage remain unverified at runtime. The new Planner approval policy is outside the tested revision and has no execution evidence.

## Scenario mapping

The denominator uses the nine numbered FD-014 acceptance items. Some items
also require static or prior-lifecycle evidence; they are retained in the
inventory and cannot be credited as executed-test coverage. No scenario is
covered until an applicable behavior test executes; passing is reported
separately and is required for acceptance. Six cases are
prepared and unrun; a passing test would still cover only the part of each
acceptance item that its assertions actually check.

| Acceptance item | Prepared public-interface case | Current result |
| --- | --- | --- |
| 1. Role ownership and independence | `test_independent_policy_routes_through_tester_and_pm` checks the Tester and PM route plus different session claims. PM creation, merge/archive, Planner, Worker compile, and independent Reviewer result still need evidence. | Uncovered; unrun |
| 2. Scenario inventory, measures, threshold and pass semantics | `test_pm_cannot_accept_failed_executed_case` checks a failed-case acceptance gate; the report and PM exception fixture exercise labelled coverage fields. The 70% boundary and uncovered denominator need more cases. | Uncovered; unrun |
| 3. Black-box authoring and separate code coverage | All six prepared tests use CLI subprocesses and public receipts; independent coverage collection is not prepared. | Uncovered; unrun |
| 4. Business branch denominator and exclusions | A project coverage command and raw branch report are not available in this round. | Uncovered; not measured |
| 5. Exact report commands and prepared/executed separation | This report and the test case fixture distinguish preparation from execution; `test_report_requires_labelled_tester_session` checks a required report field. | Uncovered; unrun |
| 6. Version-linked PM decision and Reviewer behavior | `test_independent_policy_routes_through_tester_and_pm` prepares a revision-linked PM decision and Reviewer route; `test_pm_rejection_returns_to_worker_and_next_implementation_retests` checks rejection and new Tester round; `test_reviewer_cannot_bypass_pending_tester` checks early review rejection. Reviewer respect for the PM decision requires independent review evidence. | Uncovered; unrun |
| 7. Runtime authorization guard | No test or coverage tool has been run. No automated permission gate is claimed. | Uncovered; static policy only |
| 8. Source and installed Skill, spec and usage consistency | Worker reports source/installed Skill SHA-256 equality; this is static evidence, not an executed behavior test. | Uncovered; static evidence only |
| 9. Stale Worker refresh | FD-015's archived review supplies prior evidence; no FD-014 Tester test has been prepared or executed for this item. | Uncovered; prior FD evidence only |

`test_legacy_fd_routes_directly_to_reviewer` also prepares the stable spec's
legacy compatibility scenario. That is a regression contract outside the nine
numbered FD-014 acceptance items; it remains unrun.

## Observations

- Tester claimed `FD-014-000010-implementation-ready` using the independent
  session above. The Worker session in the receipt is different.
- Tester wrote only this report and `plugins/tests/test_fd014_blackbox.py`.
  The test harness copies top-level CLI Python files and the FD template into
  a temporary project and invokes the public CLI through subprocesses. It does not
  import or read the production module as source. The fixture has not been
  exercised, so its ability to run in that temporary project is unverified.
- No test, coverage, external service, network, or credential command has run.
  Worker-authored white-box tests: none identified from public evidence; no
  implementation source or Worker test changes were inspected by Tester.
- After this implementation handoff, the user requested that Planner review
  Tester test commands, automatically approve safe commands, and refer risky
  commands to a human. FD revision 10 and its implementation have not yet
  incorporated that policy, so it confers no authorization in this round.
  The PM must reconcile the new requirement before treating this report as
  acceptance evidence for that policy.
- Proposed first focused command for future Planner review or other valid
  authorization:
  `python -B -m unittest plugins.tests.test_fd014_blackbox -v`. Expected
  duration: under 60 seconds for six isolated CLI contract cases. It writes
  only to temporary directories, does not require network access or elevated
  permissions, and produces no final build artifact. Branch coverage would
  need a separate authorized tool/command and a defined business-code boundary.
