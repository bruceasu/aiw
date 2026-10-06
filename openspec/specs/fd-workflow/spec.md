# FD-first workflow specification

## Purpose

Define the new numbered Feature Design path. Legacy AIW Task and Workflow Core
records remain readable during migration; they are not the execution source
for a new numbered FD.

## Requirements

### Requirement: FD owns authored progress

The system MUST allocate a stable `FD-XXX` ID and store the design, numbered
Work Items, status, and Verification in `docs/features/FD-XXX_SLUG.md`. The
index MUST be a projection of FD files. A new FD MUST NOT require a Task or
OpenSpec change.

#### Scenario: Create a new FD

- **WHEN** a title is supplied to `aiw fd new`
- **THEN** a non-reused ID, FD file, index row, and Planner handoff are created.

### Requirement: Explicit role handoffs

Only an explicit stage operation MAY create a handoff event. The event MUST
identify its FD, revision, type, producer, target role, and an existing
project-relative artifact. A file save or Git commit MUST NOT dispatch a role.

#### Scenario: Design becomes ready

- **WHEN** Planner emits `design-ready` from a complete Design FD
- **THEN** the FD becomes Open and a Worker handoff is recorded.
- **AND** unresolved material `%% NEEDS_INPUT` notes block the transition.

### Requirement: No duplicate unknown work

The dispatcher MUST save a receipt before starting a role. A pending receipt
MAY be resumed. A launching or dispatched receipt MUST NOT start a second
writer automatically. The operator MUST reconcile its original process or
session before retrying an unknown result.
Receipt content digests MUST treat CRLF and LF line endings as equivalent;
other FD content changes MUST still invalidate the handoff.

A host Agent that handles a pending receipt MUST atomically bind the exact
event ID to its Session before writing. Repeating the claim for the same
Session MUST be harmless; a different Session MUST NOT claim it. The next
handoff MUST cite the claimed event ID.

#### Scenario: Resume during in-flight work

- **WHEN** `aiw fd resume` sees a launching or dispatched event
- **THEN** it reports the original event and does not redispatch it.

#### Scenario: Host claims a pending handoff

- **WHEN** a host Session claims the latest pending role event
- **THEN** the receipt records that Session and later resume reports it.
- **AND** another Session cannot claim the same event.

### Requirement: Evidence-based completion

Worker MUST resolve all scoped Work Items before handing off implementation. An
independent Reviewer MUST record a report for either `changes-requested` or
`verification-passed`. Completion MUST NOT imply an unrun test passed.
New FDs MUST declare `**Test policy:** Independent`. Existing FDs without the
marker MAY use their current direct Worker-to-Reviewer path without rewriting
history. For independent-policy FDs, `implementation-ready` MUST create a
`Pending Test` Tester handoff rather than a Reviewer handoff. Tester MUST use a
different session from Worker, derive black-box cases from requirements and
public contracts, place repository test code under root `tests/`, and author a
report that separates prepared, executed,
failed, and unrun tests. The report MUST inventory applicable scenarios,
covered and uncovered scenarios, requirements coverage, business-code branch
coverage, raw coverage evidence or an unavailable reason, exact commands,
test-file ownership, and residual risk. Each broad acceptance item with
multiple observable behaviors MUST be split into distinct scenarios before
computing the denominator. A case covering only part of an item MUST NOT
mark its other behaviors covered. Static-only requirements need a stated
exclusion reason, not a fabricated test result. Coverage targets are 100%;
70% on each applicable measure MAY support acceptance if executed behavior
tests pass. Tester execution requires a recorded Planner authorization for each
exact command. Planner MUST inspect the invoked test code. Planner MAY approve
a focused, offline, inspectable command confined to assigned or temporary
paths without human review. Destructive or unrelated writes, secrets,
network/external services, downloads, elevated privileges, release artifacts,
or unclear effects MUST be escalated for explicit human approval. Planner
records the command, scope, risk assessment, approval basis, decision time,
implementation event, FD revision/digest, and Tester session before execution.
For a human-approved command, the record MUST contain an affirmative,
auditable approval reference; `denied`, `pending`, or an unanswered request
MUST NOT satisfy the CLI gate.
Tester MUST cite matching authorization records for executed tests or measured
branch coverage. The CLI MUST reject such evidence without matching records.
An earlier FD revision's approval cannot be reused.
New FDs MUST also declare `**Evidence policy:** Dual`. For a Dual evidence FD,
each new Worker, Tester, PM, and Reviewer report and Planner authorization
MUST have a Chinese Markdown file for people and a same-basename JSON sidecar
for CLI/AI. Markdown MUST contain one `aiw-data` reference. JSON MUST use
schema `aiw.fd.evidence.v1`, name the FD, evidence kind, exact source event,
and Markdown filename, and place machine fields under `data`. Both sides MUST
use same-directory filenames so their links remain valid after archive.
The CLI MUST reject
missing, invalid, or mismatched JSON before a role handoff. For a Tester
report, JSON `scenarios` MUST enumerate each applicable behavior with a
unique ID and status; its passed count MUST equal reported covered scenarios.
Historical Markdown-only evidence and FDs without the marker retain their
existing format and validation path.

Tester `test-report-ready` MUST move the FD to `Pending Test Acceptance` and
route to PM. PM MUST record a versioned decision bound to the Tester report
and FD revision/digest, with accepted/rejected disposition, both coverage
results, rationale, exception, residual risk, identity, and decision time.
PM MUST NOT accept failed executed behavior tests. An accepted decision below
70% or with unavailable coverage MUST name an exception and residual risk.
`test-accepted` routes to an independent Reviewer; `test-rejected` returns to
Worker. Reviewer MUST use a third session, respect PM's recorded exception,
and still report factual implementation or evidence defects. A later Worker
revision starts another Tester round rather than reusing an earlier decision.
Archiving as Complete MUST require the current Reviewer's
`verification-passed` event with the same FD revision and content digest,
not only editable FD status text.
Closing an FD MUST store its design at
`docs/features/archive/<FD-ID>/<FD-ID>_SLUG.md` and move its matching Markdown
and JSON implementation and review evidence into that directory's `reports/` and
`reviews/` subdirectories. It MUST reject a destination collision before
moving any file and MUST leave other FDs' files in place. Re-review of an
archived FD MUST retain earlier archived evidence; a later close moves any
newly written evidence into the same FD directory.

#### Scenario: Review fails

- **WHEN** Reviewer emits `changes-requested` with a report
- **THEN** the FD returns to In Progress and Worker receives the report.

#### Scenario: Dual evidence handoff and archive

- **WHEN** a Dual evidence role emits a report with matching Chinese Markdown
  and structured JSON
- **THEN** the CLI validates their references and event binding, records the
  Markdown artifact, and a later close archives both files together
- **AND** missing or mismatched JSON blocks the handoff without changing FD
  state; historical Markdown-only reports remain intact

#### Scenario: Independent test evidence precedes review

- **WHEN** Worker completes an independent-policy FD and Tester claims its
  handoff in a separate session
- **THEN** Tester reports actual scenario/test/coverage evidence and PM
  records an acceptance or rejection before any Reviewer handoff
- **AND** a failed executed behavior test cannot be accepted as passing

#### Scenario: Planner authorizes a low-risk Tester command

- **WHEN** Tester proposes an exact focused command and Planner confirms its
  invoked code is offline, inspectable, and confined to assigned or temporary
  paths
- **THEN** Planner records a revision-bound low-risk approval without asking
  the human, and Tester may execute only that command

#### Scenario: Tester command has dangerous or unclear effects

- **WHEN** Planner identifies external, destructive, privileged, secret-bearing,
  release-producing, or unknown effects
- **THEN** Planner requests human approval and Tester waits; an approved
  command cites the human decision in its authorization record

#### Scenario: Existing FD retains direct review

- **WHEN** an FD predating the test policy has no Independent marker
- **THEN** its current `implementation-ready` route remains Reviewer

#### Scenario: Close archives its evidence

- **WHEN** an FD is closed with the required lifecycle evidence
- **THEN** its available implementation and review reports are archived with
  the FD, while reports for other FDs stay in the active directories.
- **AND** an existing archive destination prevents the close from overwriting
  evidence.

### Requirement: Per-FD native archive layout

Native archived FD designs MUST be stored at
`docs/features/archive/<FD-ID>/<FD-ID>_SLUG.md`. Their reports and reviews
MUST be stored under the same FD directory. FD lookup, listing, close,
request-review, and reopen MUST resolve this layout, and the generated index
MUST link to the actual nested paths. Migration of existing flat native FD
archives MUST preflight all destinations and MUST roll back partial file moves
and path rewrites on filesystem errors. Event receipt artifact paths MUST
remain unchanged. Task-owned FD archive layouts MUST remain unchanged.

#### Scenario: List an archived FD in its own directory

- **WHEN** an archived native FD has its design and evidence under
  `archive/<FD-ID>/`
- **THEN** `aiw fd list` and `aiw fd show` resolve it and the index links to
  the nested design path.

#### Scenario: Reject an archive migration collision

- **WHEN** migration finds an occupied destination for a design or evidence
  file
- **THEN** it fails before moving files or rewriting path references.

### Requirement: Bounded host-operated automatic FD lifecycle

When a human explicitly requests `$fd-workflow auto`, the host agent MAY
create or resume one numbered FD, design and split its Work Items, implement
them, request independent review, repair findings, archive a passed FD, and
remove its clean worktree and delivered branch after the archive commit.
It MUST commit each completed, independently reviewable Work Item with only
that item's changes and MUST NOT rebase either branch as part of the workflow.
After review, local delivery MUST squash the FD result into one single-parent
commit on the recorded parent branch, with the delivered FD HEAD recorded as
`FD-Source`. The parent MUST NOT inherit the FD's individual Work Item commits.
After archive, branch removal MUST require the current FD HEAD to match that
trailer in the parent's delivery history.
The operation MUST use the existing role receipt, exact claim, source-event,
revision, and digest gates. Tester and Reviewer MUST each run in separate
subagents for an independent-policy FD; the PM/Planner/Worker host MUST NOT
emit their results. No more than three
Reviewer outcome events MAY occur in one active implementation cycle, even if
the auto operation is resumed. After a third `changes-requested`, the FD MUST
remain active with its findings. The operation MUST stop for missing human
decisions, authorization, subagent capability, or an in-flight/foreign-claimed
handoff. It MUST NOT infer permission to run restricted validation or Git
delivery operations from the auto request.

#### Scenario: Review findings are repaired and the next review passes

- **WHEN** the first independent Reviewer subagent requests changes and the
  Worker repairs the findings under the next claimed handoff
- **THEN** a new implementation-ready event requests another independent
  review, and a current verification-passed event permits Complete archive

#### Scenario: Three reviews request changes

- **WHEN** the third Reviewer outcome for an active implementation cycle is
  changes-requested
- **THEN** automatic repair/review stops, preserves all reports, and does not
  claim the new Worker handoff, archive, or synthesize a passing event

#### Scenario: Auto resumes after the third failed review

- **WHEN** a later auto invocation finds an In Progress FD whose latest
  changes-requested handoff was left pending after the review limit
- **THEN** it checks the prior Reviewer outcomes and report before claim,
  leaves the Worker handoff pending, and reports the human-directed Gate

#### Scenario: Reviewer subagent cannot start

- **WHEN** the host lacks a separate subagent capability or the Reviewer
  handoff is already owned by another session
- **THEN** auto stops at the recorded handoff without self-review or a
  duplicate writer

### Requirement: Re-review a completed FD

An operator MUST be able to request an explicit review of an archived Complete
FD with a one-line reason. The operation MUST require an acknowledged latest
Reviewer `verification-passed` receipt, create a new `review-requested` event
for the current FD content, increment its revision, return it to the active
feature directory as `Pending Verification`, and rebuild the index. It MUST
preserve the prior completion date and MUST NOT modify Work Item checkboxes.
The Reviewer MUST use the normal claim/source-event protocol and report through
the existing `verification-passed` or `changes-requested` event.

#### Scenario: Request review after an archived FD is updated

- **WHEN** an operator runs `aiw fd request-review FD-001 --reason "..."` for
  an archived Complete FD with an acknowledged Reviewer pass
- **THEN** the current FD content is revisioned and a Reviewer handoff records
  the reason and content digest
- **AND** the FD returns to the active index as Pending Verification without
  changing its Work Items
- **AND** a Reviewer pass can be closed as Complete again

#### Scenario: Reject re-review without completion evidence

- **WHEN** an archived FD is not Complete, the reason is invalid, or the
  latest acknowledged Reviewer pass is missing
- **THEN** the request fails without creating an event or changing the FD

### Requirement: Reopen a closed or deferred FD

An operator MUST be able to reopen an archived `Closed` or `Deferred` FD with
a one-line reason. The operation MUST preserve its ID, prior disposition
record, Work Item checkboxes, and archived evidence; increment its revision;
return the FD to the active directory as `In Progress`; rebuild the index;
and create a `reopen-requested` Worker handoff for the current FD digest.
The Worker MUST claim that exact event before editing and cite it when
emitting `implementation-ready`. An archived `Complete` FD MUST continue
through `request-review`. An active FD during an ordinary reopen, pending or in-flight role, invalid
reason, existing destination, or event collision MUST be rejected without
creating a claimable orphan handoff.
New reports and reviews after reopening MUST use filenames distinct from
earlier archived evidence so a later `close` cannot overwrite that evidence.
Before the new Worker handoff is claimed, `reopen --correct-reason` MAY correct
an unreadable or mistaken reason. It MUST verify the current FD digest and
event identity, update the authored reason and receipt digest together, retain
the previous reason in correction history, and leave the handoff pending. It
MUST reject a claimed or otherwise changed handoff.

#### Scenario: Resume a previously closed FD

- **WHEN** an operator runs `aiw fd reopen FD-004 --reason "continue review"`
  for an archived `Closed` FD
- **THEN** it becomes active `In Progress` with its earlier close reason
  preserved and a new pending Worker handoff.
- **AND** prior reports and reviews remain in the archive.

#### Scenario: Reject an unsupported reopen

- **WHEN** the FD is active, archived `Complete`, or has an unresolved role
  handoff
- **THEN** `reopen` fails before changing the FD or event store.

#### Scenario: Correct an unclaimed reopen reason

- **WHEN** an operator runs `aiw fd reopen FD-004 --reason "continue review"
  --correct-reason` while its `reopen-requested` Worker event is pending
- **THEN** the FD reason and handoff receipt use the corrected value and
  matching digest, the old value remains in correction history, and the
  Worker can still claim the same event.

### Requirement: Refresh a stale Worker handoff

PM MAY run `aiw fd refresh-worker <fd-id> --reason <text>` for an active
`Open` or `In Progress` FD whose latest pending Worker receipt has a different
revision or normalized content digest. The one-line reason MUST be retained.
The operation MUST increment the current FD revision, preserve its status and
Work Items, cancel the old receipt with a successor link, and create a
PM-produced `work-requested` Worker handoff with the new FD digest and a
`supersedes` link. The Worker MUST claim the new event and cite it when
emitting `implementation-ready`. A current, claimed, launching, dispatched,
non-Worker, or archived handoff MUST NOT be replaced. A failed mutation MUST
leave no claimable orphan and restore the prior FD and receipt state.

#### Scenario: PM updates acceptance after Planner handoff

- **WHEN** an active FD has changed since its latest pending Worker event and
  PM refreshes the handoff with a valid reason
- **THEN** the stale event is cancelled, and a new pending Worker event is
  bound to the current FD content with preserved supersession provenance

#### Scenario: Worker already owns the handoff

- **WHEN** the latest Worker event is current, claimed, or in flight
- **THEN** refreshing is rejected without changing the FD or receipts

### Requirement: Refresh a stale Tester handoff

PM MAY run `aiw fd refresh-tester <fd-id> --reason <text> --artifact <report>`
for an active independent `Pending Test` FD whose latest unclaimed pending
Tester receipt has a different revision or normalized content digest. The
one-line reason MUST be retained and limited to 500 characters. The report
MUST exist inside the repository; Dual evidence MUST validate its worker-report
sidecar and FD identity. If the report declares `data.fd_revision`, it MUST
match the FD before the refresh. A known Worker session MUST be retained.

The operation MUST increment the FD revision, preserve status and Work Items,
cancel the old receipt with `superseded_by`, and create a PM-produced
`test-requested` receipt with the current digest, supplied report, `supersedes`,
Worker session and original `implementation_event` provenance. Repeated
refreshes MUST retain that original implementation reference. This operation
MUST NOT represent a Worker result, Tester result, or test authorization.
Current, claimed, launching, dispatched, wrong-role and archived handoffs MUST
be rejected before writes. Failed writes MUST restore the prior FD, receipt
and index and leave no claimable orphan; incomplete rollback MUST be reported.

Tester MUST claim the new event and cite it as `source_event` and evidence's
`Implementation event`; `test-report-ready` MUST accept this PM recovery
handoff while retaining all existing revision, identity, coverage and PM
decision checks. Any execution requires fresh authorization for this event
and revision. Earlier test approvals or PM decisions MUST NOT be reused.

#### Scenario: Recover after acceptance revision

- **WHEN** an independent Pending Test FD has changed after its unclaimed
  Tester handoff and PM supplies the current implementation report
- **THEN** the old receipt is cancelled, the new receipt binds the current
  revision/report, and independent Tester can claim it and report normally

#### Scenario: Reject unsafe replacement

- **WHEN** the handoff is current or owned/in flight, the FD is not independent
  Pending Test, or the report/Worker identity is missing or invalid
- **THEN** refresh is rejected without changing the FD or receipts

#### Scenario: Refresh write failure

- **WHEN** a mutation fails after a preparing receipt was written
- **THEN** rollback restores the prior FD, receipt and index and removes the
  new receipt; any incomplete rollback is explicitly reported

### Requirement: Recover an active FD review handoff

An operator MAY request review of an active Pending Verification FD when its
prior pending handoff is stale or missing. The operation MUST create a new
`review-requested` receipt tied to the current FD content and reason, preserve
the FD status and Work Items, and record which prior event it supersedes. A
superseded pending receipt MUST be cancelled. A launching or dispatched receipt
MUST NOT be superseded, and an identical pending Reviewer handoff MUST NOT be
duplicated. The Reviewer MUST use the normal claim/source-event protocol;
unresolved `%% NEEDS_INPUT` notes MUST still prevent `verification-passed`.

#### Scenario: Recover a stale Planner handoff

- **WHEN** an active Pending Verification FD changed after an unclaimed Planner
  event, and an operator runs `aiw fd request-review FD-005 --reason "..."`
- **THEN** a new pending Reviewer event records the current FD digest and the
  superseded Planner event, without claiming that Worker emitted a result
- **AND** the Reviewer may claim the new event and report changes requested

#### Scenario: Preserve in-flight review

- **WHEN** the latest event is launching or dispatched, or a current Reviewer
  event is already pending
- **THEN** the request fails without creating a duplicate event

### Requirement: Git and workspace boundary

An FD MAY have an isolated worktree once its plan is committed. Local focused
commits MAY be made when authorized. Commit MUST NOT imply push, merge,
release, deployment, worktree deletion, or archive.
Before `aiw wt add` creates a worktree or metadata, it MUST confirm that the
FD worktree path and `.ai` metadata path are ignored by Git. If either path is
not ignored, add MUST fail without changing Git state and tell the user which
ignore rule to add and commit.

### Requirement: Legacy records remain readable

Existing Task and Workflow Core records MUST NOT be rewritten as FD events or
evidence. New engineering work MUST use numbered FDs. The removed `aiw wf`
command MUST NOT be presented as an available path. The `aiw wt` plugin MUST
be the sole worktree command surface for FDs; `wt add` directly creates the
recorded worktree and workspace metadata. After review, `wt local-merge`
MUST squash the FD result into one single-parent commit on its recorded parent,
including `FD-Source: <FD-HEAD>` in the commit message. On a parent-side squash
content conflict, it MUST reset the failed squash, verify parent recovery, and
merge the parent into the FD worktree for resolution. Delivery MUST require a
second explicit `local-merge` invocation after that resolution is committed.

#### Scenario: Squash multiple Work Item commits

- **WHEN** a clean FD branch contains several completed Work Item commits
- **THEN** `local-merge` creates one commit on the recorded parent whose sole
  parent is the prior parent HEAD and whose `FD-Source` names the FD HEAD
- **AND** the Work Item commit IDs do not become ancestors of the parent branch

#### Scenario: Recover a squash conflict

- **WHEN** the parent and FD branch change the same content incompatibly
- **THEN** the failed squash leaves the parent HEAD and worktree unchanged,
  merges the parent into the FD worktree, and requires an explicit retry
