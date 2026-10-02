# FD-014 independent Tester report, round 2

**Implementation event:** FD-014-000013-implementation-ready
**Tested FD revision:** 13
**Tested FD digest:** f1891ea45d5d35aa5c9145a84113e975c0fd35cd137671398525a722bc04fd4f
**Tester session:** fd014-tester-20261002-f7f15d14
**Applicable scenarios:** 10
**Covered scenarios:** 6
**Executed behavior tests:** 8
**Passed behavior tests:** 8
**Failed behavior tests:** 0
**Unrun behavior tests:** 0
**Requirements coverage:** 60%
**Branch coverage:** not measured
**Raw coverage evidence:** not measured
**Coverage unavailable reason:** No business-code branch tool has been selected or authorized. The focused black-box suite does not measure business-code branches.
**Test files:** tests/test_fd014_blackbox.py
**Authorization records:** ["docs/features/reports/FD-014-test-authorization-r2.md", "docs/features/reports/FD-014-test-authorization-r2-retry.md", "docs/features/reports/FD-014-test-authorization-r2-human-retry.md"]
**Commands:** ["python -B -m unittest tests.test_fd014_blackbox -v", "python -B -m unittest tests.test_fd014_blackbox -v", "python -B -m unittest tests.test_fd014_blackbox -v"]
**Recommendation:** blocked
**Residual risk:** The corrected eight-case public CLI suite passed, but requirements scenario coverage is 6/10 (60%), business-code branch coverage is unavailable, and four acceptance items need separate evidence. PM must explicitly decide whether to accept these gaps; the prior two fixture failures remain historical results.

## Scenario mapping

The denominator uses the ten numbered FD acceptance items. Pure
documentation or static review evidence is listed separately and is not
credited as an executed behavior case. The final authorized run executed all
eight prepared CLI cases and all passed. Six numbered items have at least one
applicable executed case; four remain uncovered. This is a conservative
item-level measure: a passed case may cover only part of a broad item, as the
unverified parts below state. The first two suite attempts stopped in shared
fixture setup and are recorded separately from the final behavior results.

| Acceptance item | Prepared public-interface or other evidence | Current result |
| --- | --- | --- |
| 1. Role sequence and independence | `test_independent_policy_routes_through_tester_and_pm` passed: Tester and PM routing and session separation. Independent Reviewer and PM merge/archive require later lifecycle evidence. | Covered in part; passed |
| 2. Scenario and coverage accounting | `test_pm_cannot_accept_failed_executed_case` passed: failed executed case cannot be PM-accepted. Report fixtures carry counts and coverage fields. The 70% boundary and denominator arithmetic still lack dedicated cases. | Covered in part; passed |
| 3. Black-box tests under root `tests/` | All eight cases use the public CLI from root `tests/` in temporary projects; separate branch measurement is not prepared. This is test authorship and static path evidence, not a separate executed requirement scenario. | Uncovered as a complete item |
| 4. Business branch coverage and exclusions | A project branch-coverage tool, denominator, exclusions, and raw report are not established. | Uncovered; not measured |
| 5. Exact results and prepared/executed split | `test_report_requires_labelled_tester_session` passed for a missing required report field; this report records the exact three command attempts and final results. | Covered in part; passed |
| 6. Version-linked PM decision and Reviewer interaction | `test_independent_policy_routes_through_tester_and_pm`, `test_pm_rejection_returns_to_worker_and_next_implementation_retests`, and `test_reviewer_cannot_bypass_pending_tester` passed for routing and replay. Independent Reviewer behavior remains later evidence. | Covered in part; passed |
| 7. Planner command authorization | `test_executed_evidence_requires_authorization` and `test_authorization_for_other_implementation_is_rejected` passed for CLI authorization gates using synthetic temporary records. Three actual Planner records cover the suite attempts. | Covered in part; passed |
| 8. Skill/spec/usage consistency | Worker report describes static synchronization; Tester does not count this as executed behavior. | Uncovered; static evidence only |
| 9. Stale Worker refresh | FD-015 prior evidence remains applicable historically; no new FD-014 Tester case targets it. | Uncovered; prior FD evidence only |
| 10. Earlier Tester round remains historical | The r1 report and PM rejection exist; `test_pm_rejection_returns_to_worker_and_next_implementation_retests` passed for a new Tester round in a temporary project. | Covered in part; passed |

`test_legacy_fd_routes_directly_to_reviewer` prepares the stable spec's legacy
compatibility scenario outside these ten FD-014 acceptance items. It passed.

## Observations

- Tester claimed `FD-014-000013-implementation-ready` using the independent
  session above; this handoff is bound to FD revision 13 and the digest shown.
- Tester updated only `tests/test_fd014_blackbox.py` and this r2 report in this
  round. The r1 report remains historical and was not edited.
- Eight black-box cases ran and passed in the third authorized suite attempt.
  The first two attempts failed in shared fixture setup, before target
  behavior assertions. No coverage, network, external service, or credential
  command ran. Worker-authored white-box test evidence was not inspected by
  Tester.
- First authorized command, from the repository root:
  `python -B -m unittest tests.test_fd014_blackbox -v`; record
  `FD-014-test-authorization-r2.md`. Exit code 1, 3.456 seconds. Raw summary:
  `Ran 9 tests in 3.366s; FAILED (failures=8, errors=1)`. Eight intended
  methods failed at `ready_worker()` line 101 with
  `AssertionError: 'planner' != 'worker'`. The ninth discovered method was an
  unintended helper `tester_report`; it errored with
  `TypeError: FD014BlackBox.tester_report() missing 2 required positional arguments: 'fd_id' and 'implementation'`.
  Tester renamed that helper to `make_tester_report` and added fixture receipt
  diagnostics before the one permitted corrected rerun.
- Second authorized command was the same exact command; record
  `FD-014-test-authorization-r2-retry.md`. Exit code 1, 3.233 seconds. Raw
  summary: `Ran 8 tests in 3.184s; FAILED (failures=8)`. All eight methods
  failed in shared setup at line 107. The diagnostic CLI stdout said
  `handoff FD-001-000002-design-ready pending worker`; the temporary receipt
  list contained `000002-design-ready.json` targeting Worker and
  `000002-design-requested.json` targeting Planner. The fixture's filename
  sort selected the Planner receipt because both used sequence 2 and
  `design-requested` sorts after `design-ready`.
- Root cause of that collision: fixture FD text hardcoded `**Revision:** 1`
  while the actual Planner receipt created by `new` carried revision 2.
  Tester changed the fixture to use `planner['fd_revision']`.
- Third authorized command was the same exact command; record
  `FD-014-test-authorization-r2-human-retry.md` cites the user's explicit
  one-run approval after the automatic retry budget was exhausted. Exit code
  0, 7.622 seconds. Raw result:

  ```text
  test_authorization_for_other_implementation_is_rejected ... ok
  test_executed_evidence_requires_authorization ... ok
  test_independent_policy_routes_through_tester_and_pm ... ok
  test_legacy_fd_routes_directly_to_reviewer ... ok
  test_pm_cannot_accept_failed_executed_case ... ok
  test_pm_rejection_returns_to_worker_and_next_implementation_retests ... ok
  test_report_requires_labelled_tester_session ... ok
  test_reviewer_cannot_bypass_pending_tester ... ok
  Ran 8 tests in 7.572s
  OK
  ```

  The test code reads
  root `plugins/aiw-fd.py` and `docs/features/TEMPLATE.md`, copies them into
  a `tempfile.TemporaryDirectory`, runs `git init --quiet` only inside that
  temporary directory to satisfy the CLI worktree precondition, then invokes
  the copied CLI through Python subprocesses. It creates and deletes only
  temporary fixture and `.git` files; it removes `AIW_FD_ROLE_RUNNER` from
  subprocess environments and disables bytecode writes. There is no network
  or privilege operation in the test file. Planner must inspect the invoked
  CLI code before classifying the full command as low risk. Each subprocess
  has a 15-second timeout; a hang can make the entire eight-case run longer
  than the normal estimate.
- All three commands have matching Planner authorization records in the same
  order as the command list. No authorization for branch coverage or further
  test execution was given. This report requests PM disposition on the 60%
  requirement coverage and unavailable branch coverage; it does not claim
  the 100% targets were met.
