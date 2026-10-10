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
After confirming a Worker session has stopped, PM MAY explicitly recover its
latest dispatched handoff using the exact event ID and session reference.
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

Worker MUST resolve all scoped Work Items before handing off implementation.
An independent Reviewer MUST record a report for either changes-requested or
verification-passed. Completion MUST NOT imply an unrun check passed.

New FDs MUST declare Evidence policy: Dual and MUST NOT declare
Test policy: Independent. For a new/default FD, implementation-ready MUST
route directly to Pending Verification and an independent Reviewer. Before
handoff, Worker MUST perform the repository-authorized compile-only check and
static review. The default workflow MUST NOT require test execution, coverage
metrics, a Tester report, a PM test-report decision, or risk-assessor reports
as acceptance evidence.

Users MAY invoke the standalone fd-test Skill to derive black-box scenarios,
write test code under root tests/, and execute specifically authorized test
or coverage commands. It produces a factual report that MAY contain scenario
counts and coverage measurements. The Skill MUST NOT emit FD workflow events,
change FD status, add acceptance thresholds, or send its report to PM,
assessors, or Reviewer for evaluation. Reviewer MUST NOT inspect or assess an
optional fd-test report.

For a Dual-evidence FD, each new Worker and Reviewer report MUST have a Chinese
Markdown file and a same-basename JSON sidecar. Markdown MUST contain one
aiw-data reference. JSON MUST use schema aiw.fd.evidence.v1, name the FD,
evidence kind, exact source event, and Markdown filename, and place machine
fields under data. Both sides MUST use same-directory filenames so their
links remain valid after archive. The CLI MUST reject missing, invalid, or
mismatched JSON before a role handoff. Historical Markdown-only evidence
remains intact.

Existing FDs that already declare Test policy: Independent MAY continue using
their recorded legacy CLI Tester events and validation behavior, including
refresh-tester. Those compatibility paths MUST NOT become the default for new
FDs, and their test reports are not Reviewer inputs. Archiving as Complete
without an explicit force override MUST require the current Reviewer's
verification-passed event with the same FD revision and content digest, not
only editable FD status text.

Closing an FD MUST store its design at
docs/features/archive/<FD-ID>/<FD-ID>_SLUG.md and move its matching Markdown
and JSON implementation and review evidence into that directory's reports/
and reviews/ subdirectories. It MUST reject a destination collision before
moving any file and MUST leave other FDs' files in place. Re-review of an
archived FD MUST retain earlier archived evidence; a later close moves any
newly written evidence into the same FD directory.

#### Scenario: Default implementation handoff goes to review

- WHEN Worker completes a new/default FD and emits implementation-ready
  after compile-only and static checks
- THEN the CLI routes directly to an independent Reviewer
- AND the workflow requires no Tester report, coverage threshold, PM
  test-report decision, or assessor report

#### Scenario: Optional fd-test remains outside acceptance

- WHEN a user invokes fd-test for an FD
- THEN it records requested scenarios and any authorized test results in
  an optional factual report
- AND it does not change the FD status or create an acceptance handoff
- AND Reviewer does not inspect or evaluate that report

#### Scenario: Legacy Tester CLI remains compatible

- WHEN an existing FD already carries Test policy: Independent
- THEN its existing Tester events and refresh-tester command remain
  available under the legacy CLI contract
- AND newly created FDs do not receive that policy by default

#### Scenario: Review fails

- WHEN Reviewer emits changes-requested with a report
- THEN the FD returns to In Progress and Worker receives the report.

#### Scenario: Dual evidence handoff and archive

- WHEN a Dual evidence role emits a report with matching Chinese Markdown
  and structured JSON
- THEN the CLI validates their references and event binding, records the
  Markdown artifact, and a later close archives both files together
- AND missing or mismatched JSON blocks the handoff without changing FD
  state; historical Markdown-only reports remain intact

#### Scenario: Close archives its evidence

- WHEN an FD is closed with the required lifecycle evidence
- THEN its available implementation and review reports are archived with
  the FD, while reports for other FDs stay in the active directories
- AND an existing archive destination prevents the close from overwriting
evidence.

### Requirement: Read-only FD status and evidence inspection

The FD CLI MUST provide read-only status and evidence inspection. `aiw fd show`
MUST preserve its existing FD body output and show the current status, recorded
worktree path and branch when available, current pending or in-flight handoff,
and latest event. Missing optional metadata or event records MUST be stated
explicitly. Handoff and event details MUST include their timestamps and
formatted receipt JSON.

`aiw fd show-report` and `aiw fd show-review` MUST list and display matching
Markdown evidence from the current branch, the FD worktree and recorded parent
branch when those sources can be verified, plus the FD native archive evidence
directory. Only Markdown filenames prefixed by the requested FD ID are
matching evidence. They MUST resolve branch and worktree sources through recorded
metadata and Git worktree/ref inspection; they MUST NOT construct filesystem
paths by concatenating branch names. Duplicate copies of identical evidence
MUST be collapsed by evidence kind and content digest while retaining all
visible source paths, even when identical copies have different relative paths.
Lists MUST show source and modification time, sort newest first, and contain no
more than 20 entries. For checked-out files, modification time is filesystem
mtime; for branch-only files, it is the latest commit time that changed the
path. Displayed timestamps MUST use UTC.
A present same-basename JSON sidecar MUST be identified alongside its Markdown.

With `--last`, the command MUST display the newest match without prompting.
Without `--last`, interactive selection MAY read input only when both stdin and
stdout are terminals. A blank or quit selection MUST exit safely. In a
non-interactive environment the command MUST print the available list and an
actionable usage hint without waiting for stdin. No matches MUST produce a
clear empty result. These inspection commands MUST NOT modify the FD, index,
workspace metadata, receipts, or evidence.

#### Scenario: Inspect FD status and handoff records

- **WHEN** a user runs `aiw fd show` for an FD with status, worktree metadata,
  and event receipts
- **THEN** the existing FD body remains visible and the summary shows status,
  verified worktree and branch values, and the latest event JSON and time
- **AND** a pending or in-flight handoff is shown separately, while absent
  optional values are identified as not set or not recorded

#### Scenario: Browse reports and reviews across visible sources

- **WHEN** a user runs either evidence command for an FD with evidence in the
  current branch, a verified FD worktree or parent branch, or its native archive
- **THEN** the command lists up to 20 unique Markdown items newest first with
  modification time and source path, and can display the selected body
- **AND** identical copies are deduplicated and a same-basename JSON sidecar is
  identified when present

#### Scenario: Select the latest evidence without interaction

- **WHEN** a user runs `aiw fd show-report FD-001 --last` or
  `aiw fd show-review FD-001 --last`
- **THEN** the newest matching Markdown is displayed without reading stdin

#### Scenario: Avoid blocking when selection is not interactive

- **WHEN** an evidence command has matches but stdin or stdout is not a terminal
  and `--last` was not supplied
- **THEN** it prints the bounded list and a usage hint, then exits without
  waiting for input

#### Scenario: Cancel or find no evidence safely

- **WHEN** a user cancels an interactive evidence selection or no match exists
- **THEN** the command exits safely with a cancellation or empty-result message
  and leaves FD and evidence state unchanged

#### Scenario: Inspection is read-only

- **WHEN** a user runs any FD inspection command
- **THEN** FD files, index, receipts, workspace metadata, and evidence remain
  unchanged

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

### Requirement: Persistent automatic workflow blocker feedback

When an automatic FD lifecycle stops at a Gate, its operator MUST record the
observed blocker in a separate FD-prefixed Markdown report and same-basename
JSON sidecar under `docs/features/reports/`. The pair MUST identify FD,
stage, role, source event when one exists, observed facts, confirmed cause or
unknown, actual recovery attempts, resolved or unresolved state, human
decision need, and a reusable improvement suggestion or none. An unresolved
blocker MUST remain explicitly unresolved until its actual resolution is
recorded. A later automatic resume and pre-archive check MUST inspect the FD's
feedback and identify reusable improvements. Feedback MUST NOT require editing
the FD body or receipt while a handoff is active, nor authorize a Gate bypass,
test execution, or rule change. FD-prefixed feedback files MUST follow the
existing per-FD evidence archive path; historical evidence remains valid.

#### Scenario: Stop, resume, and archive

- **WHEN** automatic processing stops at a Gate and later resumes
- **THEN** the operator records known facts and unresolved status, then
  checks the feedback on resume and adds any evidenced resolution.
- **AND** closing the FD moves its Markdown and JSON feedback with its other
  `reports/` evidence, without changing an earlier handoff digest.

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
revision, and digest gates. The Reviewer MUST run in a separate subagent; the
PM/Planner/Worker host MUST NOT emit the review result. The default lifecycle
MUST NOT dispatch a Tester or test-report assessor. Legacy Tester CLI events
remain available only for FDs that already use that compatibility path. No more than three
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

### Requirement: Recover an abandoned Worker handoff

PM MAY run `aiw fd recover-worker <fd-id> --expected-event <event-id>
--expected-session <session-ref> --reason <text>` after confirming the named
Worker session has stopped. The FD MUST be active and Open or In Progress; the
named event MUST be its latest dispatched Worker handoff with the exact stored
session reference. The reason MUST be one line and at most 500 characters.
The operation MUST preserve the FD status and Work Items, increment its
revision, cancel the old receipt with the reason and successor link, and create
a PM-produced pending `work-requested` event bound to the current FD digest.
The new event MUST identify the superseded event and abandoned session. A
failed mutation MUST restore the prior FD and receipt and leave no claimable
orphan. The new Worker MUST claim the event under its own session and cite it
when reporting implementation.

#### Scenario: PM confirms an abandoned Worker session

- **WHEN** PM supplies the latest dispatched Worker event, its stored session,
  and a reason after confirming that session has stopped
- **THEN** the old receipt is cancelled with provenance and a new pending
  Worker handoff is available for a different session

#### Scenario: Recovery targets another event or session

- **WHEN** the event ID, session, role, state, or active FD does not match
- **THEN** recovery is rejected without changing the FD or receipts

### Requirement: Refresh a stale Tester handoff (legacy compatibility)

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
Before `aiw git wt add` creates a worktree or metadata, it MUST confirm that the
FD worktree path and `.ai` metadata path are ignored by Git. If either path is
not ignored, add MUST fail without changing Git state and tell the user which
ignore rule to add and commit.

### Requirement: Legacy records remain readable

Existing Task and Workflow Core records MUST NOT be rewritten as FD events or
evidence. New engineering work MUST use numbered FDs. The removed `aiw wf`
command MUST NOT be presented as an available path. The `aiw git wt` command
provided by `aiw-git` MUST be the sole FD worktree interface. A standalone
`aiw wt` command MUST NOT be provided. `wt add` directly creates the recorded
worktree and workspace metadata. After review,
`wt local-merge` MUST squash the FD result into one single-parent commit on its
recorded parent, including `FD-Source: <FD-HEAD>` in the commit message. On a
parent-side squash content conflict, it MUST reset the failed squash, verify
parent recovery, and merge the parent into the FD worktree for resolution.
Delivery MUST require a second explicit `local-merge` invocation after that
resolution is committed. After a successful squash and validation of the
single-parent commit, recorded coordinates, clean FD worktree, and matching
`FD-Source`, `local-merge` MUST remove that FD worktree and local branch. Before
Git removes a worktree whose `.ai` is a junction or symbolic link, the command
MUST verify that the link targets the primary workspace `.ai` and remove only
the link itself. Unknown reparse points or mismatched targets MUST stop cleanup
without deleting the target. Cleanup MUST preserve `.ai/fd/<FD-ID>` receipts.
If cleanup partially fails, the command MUST report the completed steps and
recovery action without rolling back the delivered commit.

#### Scenario: Squash multiple Work Item commits

- **WHEN** a clean FD branch contains several completed Work Item commits
- **THEN** `local-merge` creates one commit on the recorded parent whose sole
  parent is the prior parent HEAD and whose `FD-Source` names the FD HEAD
- **AND** the Work Item commit IDs do not become ancestors of the parent branch

#### Scenario: Recover a squash conflict

- **WHEN** the parent and FD branch change the same content incompatibly
- **THEN** the failed squash leaves the parent HEAD and worktree unchanged,
  merges the parent into the FD worktree, and requires an explicit retry

#### Scenario: Clean up after verified delivery

- **WHEN** `local-merge` creates the expected single-parent squash commit and
  its `FD-Source` matches the clean FD worktree HEAD
- **THEN** the command removes only the recorded `.wt/<FD-ID>` worktree and
  `feature/<FD-ID>` branch
- **AND** it preserves `.ai/fd/<FD-ID>` receipts and other runtime records
- **AND** an unsafe `.ai` junction or partial cleanup failure is reported while
  preserving all resources that have not yet been safely removed

### Requirement: Explicit operator force recovery

An operator MAY cancel the exact latest dispatched receipt with `cancel-event`,
set an active FD to any defined status with `set-status`, or emit any event in
ALLOWED with `force-emit`, bypassing normal workflow preconditions. These
operations MUST require a non-empty single-line reason (at most 500 characters)
and declared operator identity (at most 200 characters), and MUST reject archived
FDs, undefined statuses/events, invalid producer/target-role pairings, and unsafe
or missing artifacts. Declared identity MUST NOT be presented as authentication.

Cancellation MUST preserve session/pid/history and state that it does not stop
the Agent. Status override MUST increment revision and update the index without
fabricating review evidence, changing Work Items, moving files, or changing
receipts. Force emit MUST preserve defined event routing, including legacy
Independent implementation routing, advance beyond existing receipt revisions,
mark its new receipt forced, cancel unfinished prior receipts with successor
provenance, and leave the new receipt pending without launching a role runner.
For decision-recorded, Planned/Design route to planner, Open/In Progress to
worker, Pending Test to tester, Pending Test Acceptance and terminal states to
pm, and Pending Verification to reviewer.

Each operation MUST persist an audit under shared `.ai/fd/<FD-ID>/operations/`
with operator, local user, reason, time, old state/event, result, and skipped
checks. Mutations MUST hold the existing FD lock and restore changed FD, index,
receipts, and audit on write failure, removing any new claimable orphan; an
incomplete rollback MUST be reported. This does not stop uncoordinated writes
from the original Agent. Normal workflow validation MUST remain enabled.
A forced verification-passed receipt MUST NOT satisfy normal Complete archive
or archived request-review evidence requirements.

An operator MAY invoke `close <id> Complete --force` (`-f`) with a non-empty,
single-line `--reason` of at most 500 characters to archive an active FD from
any defined status. Forced close MUST set status to Complete, increment the FD
revision, update the index, and move the plan and matching evidence through the
existing rollback-capable archive transaction. Its audit MUST record the prior
status/event, resulting revision, skipped status and Reviewer checks, and
`review_verified: false`; it MUST NOT create or alter a Reviewer verification
receipt. Forced close MUST reject other outcomes, archived FDs, unsafe or
colliding destinations, and a latest launching/dispatched receipt. Normal
Complete, Deferred, and Closed behavior remains unchanged without `--force`.

#### Scenario: Cancel only the receipt

- WHEN the operator supplies the exact latest dispatched event and a reason
- THEN its receipt becomes cancelled and its session history remains visible
- AND output states that the Agent has not been stopped
- AND a mismatching event or non-dispatched receipt is rejected without writes

#### Scenario: Override terminal state truthfully

- WHEN an operator forces Complete without normal review evidence
- THEN status/revision/index and audit change, without review evidence or archive
- AND normal Complete close still rejects absent or forced review evidence

#### Scenario: Force archive an FD as Complete without review evidence

- WHEN an operator closes an active FD of any defined status as Complete with
  `--force` and a valid reason
- THEN the archive transaction sets Complete, increments revision, moves only
  matching evidence, updates the index, and records the skipped status and
  Reviewer checks in an operation audit
- AND the archive contains no fabricated Reviewer verification result
- AND a latest pending non-`verification-passed` receipt is cancelled in the
  same transaction and restored if the archive rolls back
- AND any `verification-passed` receipt remains unchanged
- AND an unknown launching/dispatched role result still prevents the archive

#### Scenario: Emit outside the normal stage

- WHEN an operator force-emits a defined event from another defined FD status
- THEN workflow preconditions are skipped but artifact and role structure remain valid
- AND the new forced receipt is pending with an audit and no runner is launched
