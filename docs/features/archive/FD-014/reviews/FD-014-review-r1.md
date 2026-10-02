# FD-014 independent review, round 1

- Source event: `FD-014-000015-test-accepted`, claimed by Reviewer session
  `fd014-reviewer-20261002-c8261b92`.
- Reviewed base: `de92691` plus the scoped working-tree changes to
  `plugins/aiw-fd.py`, FD-014 evidence, root `tests/`, workflow instructions,
  stable spec, templates, and usage guide. Unrelated workspace changes and
  OpenSpec deletions were excluded.
- Outcome: **changes-requested**.

## Findings

1. **Approval gate accepts an explicit nonapproval.** In
   `plugins/aiw-fd.py:299-306`, `Basis: human-approved` rejects only `none` and
   `not required` for `Human approval`. A record with `Human approval: denied`
   or `pending`, alongside the other required fields, therefore passes
   `validate_test_authorization`. The CLI can then accept executed-test
   evidence as human-approved despite the field explicitly saying approval
   was not granted. This conflicts with FD-014 Acceptance 7 and the stable
   spec's requirement that dangerous or unclear commands wait for explicit
   human approval. Reject nonapproval values and require an auditable approval
   reference. Add a public CLI negative case for a denied or pending decision.
   This finding does not dispute the genuine user `confirm` cited by the
   current round-2 retry record.
2. **Scenario coverage does not use a complete scenario inventory.**
   `docs/features/reports/FD-014-test-report-r2.md:7-10,31-46` uses the ten
   broad numbered acceptance items as the entire applicable-scenario
   denominator, counts six as covered, and states that several counted items
   are covered only in part. For example, item 7 counts CLI missing/stale
   authorization checks while the low-risk Planner decision and dangerous
   command escalation paths are not separate mapped scenarios. The report
   also names the untested 70% boundary within a counted item. FD-014
   Acceptance 2 requires enumeration of every applicable requirement
   scenario and coverage against that inventory. Break multi-behavior items
   into observable scenarios, map each executed case or uncovered scenario,
   recalculate the denominator and percentage, then obtain a revised PM
   decision. The PM's recorded exception to the 70% threshold permits a
   lower result; it does not establish the accuracy of the inventory.

## Evidence reviewed

- The round-2 Tester report lists three exact invocations of
  `python -B -m unittest tests.test_fd014_blackbox -v`. Each has a separate
  Planner record matching implementation event `FD-014-000013-implementation-ready`,
  FD revision 13 and digest, Tester session, command, and scope. The first
  two attempts failed in fixture setup; the third is reported as eight
  passing public CLI cases. The records' decision times precede the reported
  results. The third record cites the user's explicit one-run approval.
- The test file is under root `tests/`, copies the CLI and template into
  temporary Git projects, removes `AIW_FD_ROLE_RUNNER`, and invokes the CLI
  through subprocesses. Its visible operations match the Planner's confined,
  offline risk assessment. Source and installed `fd-workflow` and `fd-review`
  Skills have matching SHA-256 hashes.
- The PM's round-2 decision explicitly accepts 60% item-level coverage and
  unavailable business-code branch coverage for this handoff. No branch
  result or unrun case is treated as passed here. The earlier Tester report
  and PM rejection remain separate historical evidence; the new
  implementation event started a fresh Tester round. FD-015's archived
  review covers the prerequisite `refresh-worker` behavior.
- Static inspection found independent Tester/PM/Reviewer handoffs, session
  separation checks, revision/digest linkage, a current PM acceptance gate,
  and the direct Reviewer route for legacy FDs without the policy marker.

## Commands and limits

Ran read-only `Get-Content`, `rg`, `Get-FileHash`, `git status --short`,
`git diff`, and `git log` on relevant paths, plus `python plugins/aiw-fd.py
show FD-014`. Claimed the exact event with `python plugins/aiw-fd.py claim
FD-014 FD-014-000015-test-accepted --session
fd014-reviewer-20261002-c8261b92`. The initial `status` subcommand was a
command-spelling error and changed nothing. No tests, coverage commands,
compile checks, builds, network calls, or final artifacts were run by this
Reviewer. The reported eight-case result was inspected as Tester evidence,
not independently rerun. Business-code branch coverage remains unknown under
the PM's explicit exception.
