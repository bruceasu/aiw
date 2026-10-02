# FD-014: Independent testing and PM acceptance in FD workflow

**Status:** Complete
**Revision:** 23
**Priority:** High
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

The numbered FD workflow treats implementation and independent review as its
main execution and verification stages. It does not consistently require an
independent Tester to decide which behavior needs active tests, author and run
those tests, and report evidence. A Worker could author tests for its own work
or report test success without an independent test record. Static review and
compile-only checks do not establish that business behavior or API contracts
work.

This affects PMs deciding whether an FD meets acceptance, Reviewers assessing
the evidence, and users relying on behavior changes. Some work (for example,
documentation-only edits) does not warrant active tests, while public APIs,
business rules, and integrations need behavior coverage. Requirements scenario
coverage and code coverage are separate measures: a suite can cover all stated
requirements without executing all code, and can execute all code without
checking correct behavior.

## Options and decision

1. Keep tests optional and leave their scope and evidence to Worker. This is
   low ceremony but does not provide independent test authorship or consistent
   acceptance evidence.
2. Require active testing for every Work Item. This is uniform but adds test
   work to changes with no meaningful executable behavior and conflicts with
   repository validation limits.
3. Add a risk-based, independent Tester stage. Tester derives test scope from
   FD acceptance and changed behavior, owns test case authorship/execution and
   reporting, and records a reason when active tests do not apply. PM decides
   whether the report satisfies the FD's acceptance policy; Reviewer remains
   independent and respects the recorded PM disposition.

Decision: use option 3. PM creates the FD, handles user decisions, dispatches
roles and manages handoffs; after review passes, PM owns merge and archive.
Planner owns design completion. Worker implements in the assigned branch or
worktree and ensures compilation succeeds. Tester is independent of Worker
and Reviewer, with a separate role/session and assigned branch or worktree.
Planner also reviews each exact Tester validation command before execution.
For a focused, offline command whose invoked test code and side effects are
inspectable and confined to the assigned workspace or temporary files,
Planner records a low-risk approval without asking the human. If the command
may alter unrelated data, use secrets, network or external services, download
dependencies, elevate privileges, create release artifacts, or has unknown
effects, Planner requests human approval and records the result. An approval
is bound to the implementation event, FD revision/digest, and Tester session.
Tester may start in parallel with Worker by reading requirements and public
interface contracts and preparing black-box cases, without seeing
implementation details. Reviewer is also independent and inspects the FD,
actual diff and evidence, then recommends pass or changes.

Tester enumerates all applicable requirement scenarios and maps each to
executed tests or marks it uncovered. Requirement scenario coverage is the
share of applicable scenarios exercised by at least one executed test; the
uncovered scenarios remain in the denominator. All executed behavior cases
must pass. Public/external API changes require suitable automated
contract or integration cases when the changed boundary permits them. The
report distinguishes tests prepared from tests executed and must not present
unavailable credentials, services, or authorization as a pass. Requirements
scenario coverage and business-code branch coverage are reported separately.
Each has a 100% target, not a hard pass condition. When each measure is at
least 70%, coverage may be considered sufficient for a pass if all executed
behavior tests pass and evidence is valid. If either measure is below 70%,
Reviewer requests PM's decision rather than deciding acceptance on coverage
alone. PM may accept the gap or require more work, but cannot change recorded
test output or describe failed/unrun tests as passed.

Recommended code-coverage boundary: Tester does not inspect implementation or
write white-box tests. A fixed project coverage tool measures the frozen
revision and emits raw evidence independently of Tester. Worker may add unit
tests to exercise uncovered code; Reviewer checks whether those tests assert
meaningful outcomes, not merely execute lines. Coverage evidence alone is not
evidence that behavior is correct.

Ideal code coverage target: 100% branch coverage of business code. Generated
code and test code are excluded from the denominator. Unreachable business
branches are annotated in the report and excluded from the denominator. The
ideal requirements scenario coverage target is also 100%, reported separately.
Neither 100% target is a hard pass condition: at least 70% on each measure may
support a pass when all executed behavior tests pass. If either measure is
below 70%, Reviewer requests PM's decision; PM may accept the gap or require
more work.

## Solution

New evidence format decision: reports created after this revision have a
Chinese Markdown file for human readers and a same-version JSON sidecar for
CLI/AI consumers. The Markdown names its JSON sidecar in a machine-readable
comment. The JSON names its Markdown file, FD ID, evidence kind, source event,
and structured data. The CLI validates the JSON for Tester reports, PM
decisions, and Planner authorization records; it requires a matching sidecar
for future Worker and Reviewer handoffs as well. Older Markdown-only reports
remain historical evidence. Closing an FD archives both formats together.

Add an independent Tester handoff that can start alongside Worker
implementation and must finish before final Review and completion. Tester
uses a separate role/session from Worker, receives the current FD, acceptance
criteria, public interface contract, and applicable validation rules, but not
implementation source or details. Tester may prepare black-box behavior
cases while Worker implements. Tester writes only within assigned test paths
in the designated branch/worktree, and executes tests when the implementation
is available, subject to repository instructions and user authorization.
Tester writes a report with:

- acceptance scenario to test case mapping and uncovered scenarios;
- test files authored or changed by Tester, limited to black-box tests using
  the public contract;
- exact commands and actual results, including failures and skipped tests;
- requirements scenario coverage percentage and evidence;
- separately collected code coverage percentage, measurement tool/command,
  scope, exclusions, and raw report reference;
- external service, framework, credential, or authorization limits;
- a clear recommendation: pass, fail, or blocked, with residual risks.

Case preparation can run concurrently with Worker implementation because it
depends on the approved FD and public contract. If that contract changes,
Tester reconciles affected cases before execution. Concurrent writes must
remain within separately assigned paths/workspaces; Tester does not edit
Worker production files. Worker-authored unit tests may help exercise code for coverage evidence, but
do not substitute for Tester-authored black-box behavior cases. Tester does
not inspect implementation to pursue code coverage. A deterministic tool
measures code execution separately. Reviewer inspects the FD, actual diff,
test assertions and reports, coverage evidence, and PM's recorded disposition.
Reviewer may report factual implementation, test-quality, or evidence
findings; PM owns the acceptance decision and merge/archive. A conflict is
returned to PM for a recorded decision rather than silently changing the
disposition.

Active behavior tests target public interfaces, business rules, and external
API calls. Work with no meaningful executable behavior may be marked not
applicable with a reason and PM disposition; compile-only evidence and static
review remain separate evidence types. Test execution follows the
Planner authorization rule: Tester proposes the exact command, scope,
expected time, and side effects. Planner inspects invoked test code and
records an approval before execution. A low-risk approval allows that command
once in the assigned scope; dangerous or unclear commands need explicit human
approval first. A Tester handoff grants no execution permission. Worker
remains responsible for compile success.

Agreed PM record: append a versioned Test Report Decision to FD evidence,
including report path and FD revision/digest, `accepted` or `rejected`, both
coverage results, rationale, any exception and residual risk, PM identity, and
decision time. A decision cannot rewrite the Tester report or raw tool output.
The Reviewer report cites this decision and records its findings separately.

### Managed test and acceptance states

New FDs use `**Test policy:** Independent` in the template. An existing FD
without this marker retains its direct Worker-to-Reviewer path, preserving
its current handoff and archive behavior. FD-014 opts into the new path.

For an independent-policy FD, Worker `implementation-ready` changes status to
`Pending Test` and targets Tester. That event carries the claimed Worker
session reference. Tester claims in a different session, prepares black-box
cases from the FD and public contract, and emits `test-report-ready` with a
report; this changes status to `Pending Test Acceptance` and targets PM. The
report identifies the implementation event, tested revision/digest, scenario
inventory and covered/uncovered counts, executed/passed/failed/unrun cases,
both coverage measures and raw evidence or a reason they are unavailable,
authored test files, exact commands, and residual risks. Prepared cases are
never counted as executed.

PM writes a separate, versioned Test Report Decision identifying the Tester
report and tested FD revision/digest, disposition (`accepted` or `rejected`),
both coverage measures, rationale, exceptions, residual risk, PM identity,
and decision time. PM `test-accepted` moves the FD to `Pending Verification`
and hands the decision and Tester evidence to Reviewer. PM `test-rejected`
returns it to `In Progress` and Worker. Both cite the exact
`test-report-ready` event. An accepted decision with either coverage measure
below 70% or unavailable must name an exception and residual risk; failed
executed behavior tests cannot be accepted. Reviewer is a third session,
distinct from Worker and Tester, and may still report factual findings.
`changes-requested` returns to Worker; the next `implementation-ready` starts
a new Tester round rather than reusing an earlier decision.

Optional parallel case preparation is a separate PM assignment in an
isolated test path/workspace. It does not advance the FD's single canonical
handoff stream, grant test execution permission, or let Tester write Worker
production files. The canonical Tester event begins after Worker completion.
The source `fd-workflow` Skill directs a separate Tester session for this
stage. Its report is the evidence for PM acceptance; Worker-authored tests
alone cannot replace it.

### Recovery of an outdated Worker handoff

PM may revise an `Open` or `In Progress` FD after Planner has emitted
`design-ready`. The existing Worker receipt then fails the revision/digest
claim gate, even if it is still pending. Add a managed
`aiw fd refresh-worker <fd-id> --reason <text>` operation for this case. It
requires a one-line reason, an active FD, and the latest receipt to be a
pending Worker handoff whose revision or digest differs from the current FD.
It rejects a matching handoff, any claimed/launching/dispatched handoff, and
an archived FD. PM is the producer of a new `work-requested` event addressed
to Worker; this records a fresh instruction without pretending that Planner
finished another design stage or Worker already implemented the change.

The operation increments the FD revision, binds the new receipt to the updated
FD digest, records the old event ID in `supersedes`, cancels the old pending
receipt, and updates the index. Its file/receipt writes must roll back on
failure or leave no claimable orphan. Worker claims only the new event and
cites it as `--source-event` on `implementation-ready`. The original receipt
remains available as historical evidence. FD-015 implemented and independently
verified this operation; use it to replace FD-014's stale revision-6 handoff.

## Scope

In scope: numbered FD role sequence and handoff; Tester independence and
artifact ownership; test selection and report format; PM acceptance evidence;
Reviewer interaction; stable FD workflow spec, source and installed
`fd-workflow` Skills, and usage documentation.

Out of scope: automatic execution without a recorded Planner decision;
implementing a specific external test
framework or provider integration; modifying legacy Task/Workflow Core
lifecycles unless separately designed; claiming code coverage where no
measured coverage evidence exists.

Compatibility: preserve existing FD events and manual commands unless the
approved design identifies the smallest necessary lifecycle extension. No
tests, builds, network access, or external API calls are implied by this FD.

## Work items

- [x] 1.1 Select code-coverage metric, exclusions, and unreachable-code
  handling before finalizing acceptance gates.
- [x] 1.2 Design the Tester handoff, independent test authorship/execution
  boundary, report schema, failure/blocked recovery path, and managed recovery
  of an outdated pending Worker handoff against existing FD event semantics.
- [x] 1.2.1 Implement the managed `refresh-worker` transition, including
  collision, digest, ownership, rollback, and supersession checks. The
  separately reviewed prerequisite FD-015 supplies the bootstrap path.
- [x] 1.3 Update the stable FD workflow spec, source and installed Skills, and
  usage documentation consistently; retain repository authorization limits.
- [x] 1.4 Prepare independent review evidence for the workflow, including
  proof that Worker cannot self-certify required tests and Reviewer respects
  PM's recorded report disposition. The actual Reviewer result follows the
  Worker handoff.
- [x] 1.5 Add Planner review of exact Tester commands, automatic low-risk
  authorization, human escalation for dangerous or unclear commands, and
  revision-bound authorization evidence in reports and workflow guidance.
- [x] 1.6 Address Reviewer round-1 findings in Worker code and guidance:
  reject nonapproval values for human-approved authorization and require a
  fine-grained Tester scenario inventory in the next round. The independent
  Tester owns the new inventory and negative public CLI cases; PM must issue
  a new decision.
- [x] 1.7 Add Chinese human-readable Markdown plus version-linked JSON
  machine evidence for future FD reports, decisions, and authorizations;
  validate the sidecar in CLI handoffs and archive both files together.

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## TODO

- [x] Reconcile the pending Worker handoff with the updated PM coverage rule
  through the reviewed `refresh-worker` operation before implementation.
- [x] Synchronize the installed `fd-workflow` Skill and verify its `SKILL.md`
  matches the source by SHA-256.
- [x] Complete Work Items 1.3–1.4 and prepare the Tester handoff.
- [x] Complete the independent Tester, PM decision, and Reviewer stages.
- [x] Complete Work Item 1.5 and prepare a new Tester round against the
  revised FD. The first Tester report was blocked and PM rejected it because
  the new authorization policy had not yet been implemented.
- [x] Resolve `FD-014-review-r1.md` findings with a new Tester report and PM
  decision before requesting another independent review.
- [x] Complete the dual evidence policy and prepare a new Tester round. The
  revision-17 Tester report was blocked and PM rejected it without running
  tests because the policy had not yet been implemented.

## Acceptance

1. PM creates FDs, resolves user decisions, dispatches roles, manages handoffs,
   and merges/archives after review; Planner designs; Worker implements and
   compiles; Tester and Reviewer are independent roles/sessions from Worker
   and each other. Tester may prepare cases in parallel from the FD and public
   contract, then executes against the available implementation. Reviewer
   independently reviews the FD, actual diff, and evidence.
2. Tester enumerates every applicable requirement scenario and reports which
   scenarios have executed tests and which remain uncovered. Scenario coverage
   is measured as covered scenarios divided by all applicable scenarios.
   Requirements scenario coverage and business-code branch coverage each have
   a 100% target and a 70% pass threshold. If either measure is below 70%,
   Reviewer requests PM's decision. Every executed behavior test must pass;
   unrun, failed, or unavailable tests cannot be reported as passing.
3. Tester uses requirements and public interfaces without source access.
   Tester-authored repository tests live under the root `tests/` directory.
   Code coverage is collected independently and separately from requirements
   coverage. Worker may add white-box unit tests for coverage; Reviewer checks
   assertions and evidence quality.
4. Code coverage is measured as business-code branch coverage. Generated code
   and test code are excluded; unreachable branches are annotated and excluded
   from the denominator. Raw evidence is retained. The 100% target is ideal;
   70% or higher may support a pass when behavior tests pass. Below 70%
   requires PM's decision. Code coverage cannot substitute for requirements
   coverage or passing behavior tests.
5. The test report preserves exact commands and actual results, identifies
   Tester-authored tests, reports both coverage measures, and separates
   executed tests from prepared tests.
6. PM disposition uses a version-linked decision record. Reviewer cites and
   respects the PM decision while reporting factual findings for PM
   resolution.
7. Planner reviews the exact proposed Tester command, invoked code, scope,
   duration, side effects, and network/permission risk. It records a
   revision-bound low-risk approval for a focused, offline, confined command
   without human review. Dangerous or unclear commands require human approval
   before any execution. Tester reports the authorization artifact and exact
   command; CLI rejects executed-test evidence without a matching approval.
   A Tester handoff alone grants no execution permission.
8. Source Skill, installed Skill, stable spec, and usage guide describe the
   same role sequence and acceptance rules.
9. An outdated pending Worker handoff can be replaced by a PM-produced,
   digest-bound `work-requested` event through a managed operation. The old
   receipt is cancelled with an explicit successor; in-flight handoffs cannot
   be replaced, and no claimable orphan survives a failed mutation.
10. The first Tester round remains historical blocked evidence; a Worker
    reimplementation after PM rejection starts a new Tester round and does
    not reuse an approval from the earlier FD digest.
11. Human-approved authorization with an explicit `denied` or `pending` value
    is rejected. The Tester report enumerates distinct applicable scenarios,
    not just broad numbered acceptance items; partial behavior remains
    uncovered in the scenario denominator.
12. Reports created under the Dual evidence policy use Chinese Markdown for
    people and structured JSON for CLI/AI. The two files cross-reference each
    other and the exact source event. Missing or inconsistent JSON blocks a
    role handoff; closing the FD archives both files without overwriting
    earlier evidence. Historical Markdown-only reports remain readable.

## Verification

- Design stage: static review of `openspec/specs/fd-workflow/spec.md`,
  `docs/usage/aiw-fd.md`, `docs/features/archive/FD-007/FD-007_BOUNDED_AUTOMATIC_FD_WORKFLOW.md`,
  and the current Coder/Tester/Verifier boundaries in
  `openspec/specs/agent-session/spec.md` and
  `openspec/specs/workflow-supervision/spec.md`.
- Implementation stage: static comparison of the spec, both Skill copies,
  usage docs, event transitions, and test-report acceptance gate. A Tester
  command may run after a recorded Planner low-risk authorization or explicit
  human authorization for dangerous effects; the FD handoff alone grants none.
- Current status: PM clarified role ownership, separate 100% targets with a
  70% pass threshold for requirements scenarios and business-code branches,
  coverage exclusions, unreachable-branch reporting, and agreed to the PM
  decision record format;
- The Worker handoff created for revision 6 is stale because PM's coverage
  threshold and scenario-coverage denominator were added at revisions 7–8.
  FD-015's independent Reviewer passed the new managed `refresh-worker`
  operation, and FD-015 was archived Complete. FD-014's old handoff must be
  superseded through that command before Worker claims a new event.
- At the earlier design stage, no lifecycle implementation, tests, compile,
  or runtime validation had been performed.
- PM used reviewed `refresh-worker` after FD-015's archive. The old
  `FD-014-000006-design-ready` receipt was cancelled; new
  `FD-014-000009-work-requested` was claimed by this Worker session. No old
  receipt was edited by hand.
- Source changes now add opt-in independent Tester and PM decision states,
  report/decision templates, new-FD policy, role separation checks, and
  legacy direct-review compatibility. `python scripts/compile.py` passed
  with network disabled and no final binary. This compile script covers Go;
  it does not exercise the new Python lifecycle. Python syntax was parsed
  successfully without creating a bytecode artifact. `git diff --check` on
  the changed tracked paths reported no whitespace error. No tests, final builds,
  coverage runs, or external calls were performed.
- The project Skill manager reported `fd-workflow` already installed. Its
  installed `SKILL.md` and source `SKILL.md` have the same SHA-256 digest,
  `296017CDF6D92A0AED8B8B872C7C306FFAB498CDAC598467C97949A1B622254B`.
  Worker report:
  `docs/features/reports/FD-014-implementation.md`.
- The first independent Tester report `FD-014-test-report-r1.md` recorded six
  prepared cases, zero executed cases, 0/9 scenario coverage, and unmeasured
  branch coverage. PM rejected the report in `FD-014-test-decision-r1.md` and
  claimed `FD-014-000012-test-rejected` for the revised authorization policy.
  No unrun test was counted as passed.
- Work Item 1.5 adds revision-bound Planner authorization evidence to the
  Tester report gate and a low-risk/human escalation rule to repository
  instructions, stable spec, Skills, templates, and usage docs. Source and
  project-installed `fd-workflow` and `fd-review` Skills were synchronized.
  The next Tester round must use a fresh authorization record; the first
  round's blocked report remains historical evidence.
- User assigned Tester-authored repository tests to the root `tests/`
  directory. The first Tester report keeps its original path as historical
  evidence; the next Tester round uses the relocated file and command.
- `python scripts/compile.py` passed with the Go toolchain local and proxy
  disabled, retaining no final binary. This does not exercise the Python
  authorization gate. Worker round-2 evidence is
  `docs/features/reports/FD-014-implementation-r2.md`.
- Reviewer round 1 recorded two findings in
  `docs/features/reviews/FD-014-review-r1.md` and returned
  `FD-014-000016-changes-requested` to Worker. Worker now requires an
  affirmative structured human approval reference and strengthened the
  scenario inventory guidance. `python scripts/compile.py` passed with
  local Go toolchain and proxy off; no Python behavior tests ran in this
  repair round. Worker round-3 evidence is
  `docs/features/reports/FD-014-implementation-r3.md`. The next Tester round
  must add a public negative authorization case and a new fine-grained
  scenario mapping; PM then records a fresh decision.
- The revision-17 Tester handoff produced `FD-014-test-report-r3.md`, which
  recorded 0/30 scenarios and 11 prepared but unrun cases because the user
  added the dual evidence format before testing. PM rejected that report in
  `FD-014-test-decision-r3.md` and Worker claimed
  `FD-014-000019-test-rejected`. Worker implemented Chinese Markdown plus
  same-basename structured JSON for future FD evidence, CLI cross-reference
  and scenario validation, and paired archive moves. Historical reports
  remain unchanged. `python scripts/compile.py` passed with the local Go
  toolchain and proxy disabled; it does not run Python behavior tests.
  Worker round-4 evidence is `FD-014-implementation-r4.md` and its JSON
  sidecar. Independent Tester/PM/Reviewer evidence is still pending.
- Independent Reviewer round 1 claimed `FD-014-000015-test-accepted` in
  session `fd014-reviewer-20261002-c8261b92` and requested changes in
  `docs/features/reviews/FD-014-review-r1.md`: the human-approval gate accepts
  explicit nonapproval text, and the Tester report's item-level denominator
  does not enumerate all applicable scenarios. PM's 60% and unmeasured-branch
  exceptions remain recorded; neither resolves these findings.
- Independent Tester round 4 reported a corrected 15/15 passing rerun and
  26/34 distinct applicable scenarios with passing evidence. PM accepted the
  report with an explicit exception for unmeasured business-code branch
  coverage and recorded eight uncovered scenarios in
  `docs/features/reports/FD-014-test-decision-r4.md` and its JSON sidecar.
  Independent Reviewer round 2 used session
  `fd014-reviewer-20261002-c8261b92` for `FD-014-000022-test-accepted` and
  passed the static code and evidence review in
  `docs/features/reviews/FD-014-review-r2.md` with a matching JSON sidecar.
  This Reviewer did not rerun tests, compile, coverage, or builds.

## Sources

- User decisions: PM creates the FD, handles user decisions, dispatches roles,
  manages handoffs, and merges/archives after review; Planner designs; Worker
  implements in the assigned branch/worktree and ensures compile success;
  Tester is independent from Worker and may prepare cases in parallel from
  requirements/public contracts before testing in the assigned branch/worktree;
  Reviewer independently checks the FD, actual diff, and evidence.
  Requirements scenario coverage and business-code branch coverage each
  target 100%, reported separately, but 100% is not a hard pass condition;
  70% or above may pass when executed behavior tests pass, while anything
  below 70% requires PM's decision. Generated and test code are excluded;
  unreachable branches are annotated but excluded from the denominator.
  Tester uses public interfaces and requirements, not implementation details.
- PM agreed to the versioned, revision-linked Test Report Decision format
  containing disposition, both coverage results, rationale, exceptions and
  residual risk, PM identity, and decision time.
- User direction: add proactive testing for material interfaces/business
  behavior, consider automated integration tests for public/external API
  boundaries, assign independent test authoring/execution and reporting to
  Tester, require PM acceptance of the test report, and have Reviewer respect
  PM's decision.
- `openspec/specs/fd-workflow/spec.md`
- `openspec/specs/agent-session/spec.md`
- `openspec/specs/workflow-supervision/spec.md`
- `docs/usage/aiw-fd.md`
- `docs/features/archive/FD-007/FD-007_BOUNDED_AUTOMATIC_FD_WORKFLOW.md`
- `skills/work-management.md`
- `docs/features/archive/FD-015/FD-015_REFRESH_STALE_WORKER_HANDOFFS_FOR_ACTIVE_FDS.md`

**Completed:** 2026-10-01
