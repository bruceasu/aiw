# FD-first workflow specification

## Purpose

Define the new numbered Feature Design path. Legacy AIW Task and Workflow Core
records remain readable during migration; they are not the execution source
for a new numbered FD.

## Requirements

### Requirement: FD owns authored progress

The system MUST allocate a stable `FD-XXX` ID and store the design, numbered
Work Items, status, and Verification in `docs/features/FD-XXX/FD-XXX_SLUG.md`. Each FD folder MUST own an index.md navigation page.
FEATURE_INDEX.md MUST be fixed navigation without a shared mutable status
table; aiw fd list MUST derive live status from FD files. A new FD MUST NOT require a Task or
OpenSpec change.

#### Scenario: Create a new FD

- **WHEN** a title is supplied to `aiw fd new`
- **THEN** a non-reused ID, FD folder, Design record and local index are created.
- **AND** a Planner handoff is added only for --handoff or a configured role runner.

### Requirement: Navigation with optional bounded consultation

ask-asu MUST default to current-state navigation without extra model calls,
artifact writes or role dispatch. It MAY obtain a second opinion only for a
concrete question under actual consultation authority. The host MUST choose
one bounded channel: one read-only non-recursive host agent or one single-turn
aiw ask invocation with a known fixed provider. An unresolved auto-provider
fallback MUST NOT be represented as one bounded model call. Failed or unknown
outcomes MUST NOT trigger automatic retries or channel/model switching.

The consultation MUST receive explicit necessary context/access, not assumed
conversation inheritance. Side effects including ask history and applicable
network/data access MUST fit authorization. Model/reasoning settings and provider
capabilities MUST be reported as actually known; an agent/profile label MUST NOT
be claimed to guarantee stronger reasoning. Consultation MUST remain advice,
not human approval or independent FD review, and MUST NOT reset consumed budgets,
restart diagnosis-only or change workflow state. Navigation MUST remain usable
without any consultation capability.

#### Scenario: Routine workflow navigation

- WHEN the current stage and next action can be established from existing evidence
- THEN ask-asu MUST recommend that action without dispatching it or calling another model

#### Scenario: Authorized second opinion

- WHEN a concrete question has authority for one consultation
- THEN only the chosen bounded channel MAY be used with explicit evidence/context limits
- AND its actual outcome and limitations MUST return to the owning discussion or FD
- AND the response MUST NOT substitute for a managed Reviewer verdict

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

Worker MUST resolve all scoped Work Items and record actual Verification
evidence before whole-FD completion. Skipped checks MUST NOT be labelled passed.
The FD SHOULD hold routine progress, self-checks, risks and blockers; a separate
Worker report is required only for a real delegated/machine handoff or an
explicit report request. Small FDs MAY use one implementation commit.

New FDs MUST declare Review policy: Self and default to Evidence policy:
Markdown. Independent review MUST be selected for material public-interface,
permission/auth, persistence/migration, concurrency or cross-module effects,
uncertain impact, or an explicit user review request. The selection needs only
a short reason in Verification. Role handoffs and Work Items MUST NOT each
trigger review. Independent review occurs once before delivery; rechecks focus
on findings, fixes and affected behavior. Historical FDs without a review
policy and legacy Independent Tester FDs MUST retain independent review.

A Self FD with resolved numbered Work Items, no blocking NEEDS_INPUT notes,
actual Verification evidence and no unresolved role execution MAY close
Complete from Open, In Progress or Complete without a Reviewer event or force.
Self-check completion MUST NOT generate a Reviewer verification-passed event.
For managed Self work, implementation-ready MUST route to PM/Complete.
For Independent work it MUST route to Reviewer/Pending Verification, whose
current unforced verification-passed event MUST match FD revision and content
before normal Complete archive. Existing review requests/findings MUST NOT
be silently downgraded or represented as self-checked acceptance.

Reviewer request/outcome history MUST preserve that FD's independent-review
requirement across Worker refresh/recover chains. Editing the Review policy
to Self MUST NOT erase the recorded gate. Explicit force recovery remains
separately audited and MUST NOT become normal Self completion.

Without a runner, new MUST create a Design FD without a role handoff by default.
An explicit --handoff or configured runner MUST retain managed Planner/Worker
dispatch and receipt ownership. Single-host work MAY continue directly from
the FD without stage claims or a separate Worker report. Request-review MUST
accept a ready active Open/In Progress/Complete FD, select Independent and
create the Reviewer handoff without requiring a forced status operation.
Self-completed archives MUST remain available for later explicit review.

New FDs MUST NOT declare Test policy: Independent by default. Tests, coverage,
Tester reports, PM test votes and risk assessors MUST NOT become default
acceptance gates. Explicit test support selects suitable core-rule TDD and
independent fd-test support before readiness, under exact command/revision,
language and side-effect authorization and finite budgets. Assigned tests
belong under root tests/. Reports supply facts without status changes,
automatic thresholds or acceptance voting. Confirmed defects and material
unknowns MUST NOT be ignored. Existing Independent Tester FDs retain their
legacy event/refresh-tester validation and independent-review route.

Existing or explicitly selected Dual evidence MUST retain Chinese Markdown
plus same-basename JSON with one aiw-data reference, schema aiw.fd.evidence.v1,
matching FD, kind, source event and Markdown filename, and machine fields under
data. CLI handoffs MUST reject invalid/missing/mismatched required sidecars.
Routine blockers belong in the FD; separate feedback reports are needed only
when requested, required by a consumer/repository rule, or necessary for
substantial delivery-recovery evidence. Historical evidence stays unchanged.

Closing MUST archive the design under
docs/features/archive/<FD-ID>/<FD-ID>_SLUG.md and move matching reports/reviews
and sidecars into that FD directory. Reject collisions before moving files,
preserve other FDs and historical evidence, and keep both sides of dual reports
together. Local archive MUST NOT imply release or deployment approval.

#### Scenario: Routine completion uses self-checks

- WHEN a new Self FD resolves all Work Items and records actual Verification
- THEN normal Complete archive requires no independent Reviewer or force
- AND no independent pass is invented; incomplete scope and unknowns still block

#### Scenario: Risk or explicit request requires independent review

- WHEN a ready FD requires Independent review or the user requests it
- THEN one separate Reviewer inspects the combined implementation and evidence
- AND current matching independent evidence gates normal completion
- AND a role handoff or completed Work Item alone does not request another review

#### Scenario: Optional managed handoffs retain ownership

- WHEN new is invoked without a runner or --handoff
- THEN only a Design record and local navigation are created
- WHEN managed handoffs are selected
- THEN explicit events, exact claims and in-flight ownership remain enforced

#### Scenario: Historical policies remain compatible

- WHEN an existing FD has no Review policy or uses legacy Independent testing
- THEN independent review remains required
- AND existing Dual evidence and legacy Tester validation remain available

#### Scenario: Refresh cannot discard required review

- WHEN a failed review is followed by policy edits and Worker refresh/recovery
- THEN the historical Reviewer request/outcome still requires independent review
- AND Self implementation-ready or close cannot bypass that requirement

#### Scenario: Optional test support supplies facts

- WHEN test support is explicitly requested
- THEN its scoped evidence and authorized results inform acceptance
- AND no automatic coverage threshold, test vote or default execution is added

#### Scenario: Archive preserves evidence

- WHEN a Self or Independent FD satisfies its applicable completion gate
- THEN matching evidence is archived together and destination collisions block
- AND a later review of a Self-completed archive retains its historical evidence

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

Routine blockers MUST be recorded in the FD's TODO or Verification with facts,
actual attempts, unresolved questions and next actions. Blocking unknowns MUST
use NEEDS_INPUT and stay unresolved until evidence supports resolution. Separate
feedback reports are required only when requested, required by a repository
rule or machine consumer, or needed for substantial delivery-recovery evidence.
Selected Dual policy retains matching Chinese Markdown/JSON sidecars.

Resume and pre-archive checks MUST inspect existing notes/reports and preserve
relevant failure history. Feedback MUST NOT edit a foreign/in-flight role's FD
or receipts, authorize retries, tests, rule changes or gate bypasses, or falsely
claim a resolution. Existing feedback follows its normal per-FD archive layout.

#### Scenario: Ordinary blocked work

- WHEN a routine single-host FD cannot continue
- THEN the host records the blocker and next step in that FD
- AND no standalone feedback pair is added without a concrete consumer need
- AND resolution updates preserve useful observed failure history

### Requirement: Diagnose failed delivery before parent synchronization

The normal isolated FD lifecycle MUST preserve implementation and applicable
self-check/independent verification, attempt `local-merge` directly, and MUST NOT proactively sync the parent
into every FD. A clean parent advancing alone MUST NOT invalidate unchanged
reviewed FD work or force the parent to become an ancestor of the FD branch.
After a delivery failure the host MUST preserve full output and diagnose the
state before choosing recovery. Existing CLI conflict recovery remains intact.
If it already started the parent merge in the FD worktree, the host MUST NOT
blindly repeat sync. A separately needed sync MUST name the recorded parent
explicitly. For actual conflicts in the full FD lifecycle the host MUST load
`resolving-merge-conflicts` for one bounded attempt at resolutions supported by
the FD and both histories. Missing Skill, unclear intent or any conflict that
cannot safely be resolved MUST stop recovery, preserve the in-progress merge,
remaining conflict entries and documented resolved edits, and produce blocker
feedback. The host MUST NOT abort, reset, force a side, commit a partial merge,
or bypass stage limits and runtime budgets. Staging or committing MUST NOT be
represented as semantic conflict resolution.

After recovery, the host MUST record the recovery commit/diff and refresh Self
verification evidence. If Independent work changed outside its current pass,
it MUST obtain fresh focused independent review through request-review before
retrying delivery. It MUST preserve historical reports, stage limits, exact role
ownership and the cumulative active-cycle review limit. A dirty workspace or
failed cleanup MUST NOT be treated as a reason to blindly sync or create a
second squash. Successful local-merge cleans Git resources; the host MUST
separately close and commit the FD archive.

#### Scenario: Clean direct delivery

- **WHEN** an FD has its applicable self-check/independent evidence and its recorded parent is clean
- **THEN** the host attempts local-merge without a mandatory parent sync
- **AND** successful verified delivery/cleanup is followed by close and archive commit.

#### Scenario: Conflict recovery changes the implementation

- **WHEN** a failed delivery leaves a parent merge conflict in the FD worktree
- **THEN** the host loads resolving-merge-conflicts for one bounded resolution attempt
- **AND** uncertain or unsafe conflicts preserve the merge state and stop recovery
- **AND** it inspects and resolves each conflict before staging or committing
- **AND** recovery refreshes Self evidence or obtains fresh required Independent review
  before retry; a no-op sync or successful commit is not proof of a correct resolution.

### Requirement: One FD skill entry owns implementation routing

The host MUST use `fd-workflow` for numbered FD design and implementation.
Its Worker stage MUST load `references/implementation.md`; a separate Worker
role prompt is needed only for a real delegated or managed handoff. The implementation method MUST NOT be a separately discoverable
Skill or call back into a duplicate implementation entry. The retired standalone
entry MUST NOT retain a compatibility alias in canonical or project skills.
Current routing, installation examples and the project installed-skills index
MUST point to the unified workflow. Historical FD evidence and user-level
installations MUST NOT be rewritten as part of this repository consolidation.

Natural-language implementation-only, partial-item, review-only and no-merge
requests MUST preserve their existing stage/scope boundaries before any
next-role dispatch. No new CLI flags are implied. Normal full-FD implementation
MUST retain self-checks or required independent review, automatic local completion
and bounded conflict recovery; design discussion alone MUST NOT implement.

#### Scenario: Implement only through the unified entry

- **WHEN** the user requests only implementation of one ready FD through fd-workflow
- **THEN** Worker loads its implementation reference, records scoped evidence and commits
- **AND** it stops before Reviewer dispatch, delivery or archive.

#### Scenario: Normal implementation through the unified entry

- **WHEN** the user requests a normal complete FD implementation through fd-workflow
- **THEN** Worker uses the same reference and the host continues the existing lifecycle
- **AND** no standalone implementation Skill or second auto request is required.

### Requirement: Bounded host-operated automatic FD lifecycle

A full numbered-FD implementation request MUST include its authorized normal
local lifecycle without another auto or merge confirmation. The host plans,
implements, self-checks, obtains independent review when required, and continues
delivery/cleanup/archive when the applicable gates pass. Explicit planning-only,
implementation-only, review-only, partial-item, no-commit and no-merge limits MUST
take priority. Incomplete scope MUST NOT be delivered or archived as complete.

Commits MUST contain coherent scoped changes; a small FD MAY use one
implementation commit. Neither branch is rebased. Isolated delivery MUST
squash to the recorded parent as a single-parent commit with FD-Source equal
to the clean FD HEAD, verify coordinates and clean workspaces, and safely
clean up only verified resources. The host then closes and commits the archive
when authorized. Direct-workspace execution skips worktree creation/delivery.

Before delivery, require complete scope, no unresolved blocker, actual self-check
evidence or a current matching independent pass, and applicable workspace gates.
Existing managed handoffs MUST retain exact claims, source events and matching
revisions/digests. Single-host work without handoffs MUST NOT invent them.
Required Reviewer uses a separate session; its host MUST NOT fabricate that
verdict. Missing independent capability blocks only work that needs it.

Tests remain explicitly selected and separately authorized. Preserve optional
TDD/fd-test budgets and legacy Tester compatibility. Independent review history
and the maximum of three failed review outcomes MUST survive resumes; repairs
and rechecks focus on findings and affected behavior. A third failed review or
no safe actionable repair stops further automatic repair.

Missing decisions/authority, foreign or unknown in-flight ownership, stale
required evidence, dirty delivery workspaces, unresolved conflicts or failed
delivery/cleanup/archive stop the affected stage. Record blockers proportionately.
For actual conflicts, the authorized bounded recovery may proceed; refresh Self
evidence or obtain fresh required independent review before retry. A successful
delivery with failed cleanup MUST NOT cause a second squash or rollback.

This scope excludes restricted runtime checks, network/downloads, push, release,
deployment, privilege changes and evidence-gate overrides.

#### Scenario: Ordinary implementation completes automatically

- WHEN a full FD has complete scope, applicable self-check/independent evidence,
  no blocker and verified clean delivery coordinates
- THEN host continues delivery, cleanup and archive without another confirmation
- AND explicit Git/stage limits still take priority

#### Scenario: Independent findings require a focused recheck

- WHEN Reviewer requests changes and Worker repairs them
- THEN recheck covers those findings, fixes and affected behavior
- AND refresh/recover or policy edits do not erase the recorded review requirement
- AND the third failed outcome stops automatic repair without a fabricated pass

#### Scenario: Required independent capability is unavailable

- WHEN an Independent FD cannot obtain a separate Reviewer session
- THEN its review stops with preserved state
- AND unrelated Self work may continue under its own authorization

### Requirement: Re-review a completed FD

An operator MUST be able to request an explicit review of an archived Complete
FD with a one-line reason. For Independent or unmarked historical work, the operation MUST require an
acknowledged latest Reviewer `verification-passed` receipt. A Self-completed
archive does not need prior Reviewer evidence. Both routes MUST select
Independent, create a new `review-requested` event
for the current FD content, increment its revision, return it to the active
feature directory as `Pending Verification`, and update its local navigation. It MUST
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
  required latest acknowledged Reviewer pass is missing
- **THEN** the request fails without creating an event or changing the FD

### Requirement: Reopen a closed or deferred FD

An operator MUST be able to reopen an archived `Closed` or `Deferred` FD with
a one-line reason. The operation MUST preserve its ID, prior disposition
record, Work Item checkboxes, and archived evidence; increment its revision;
return the FD to the active directory as `In Progress`; update its local navigation;
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

An FD MAY have an isolated worktree once its plan is committed. A normal full
numbered FD implementation request authorizes the focused local lifecycle
above unless explicitly limited. A standalone commit or review MUST NOT imply
delivery or archive, and no local request implies push, release or deployment.
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
the Agent. Status override MUST increment revision and update its local navigation without
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
revision, update its local navigation, and move the plan and matching evidence through the
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

- WHEN an operator forces an Independent FD Complete without normal review evidence
- THEN status/revision/index and audit change, without review evidence or archive
- AND normal Complete close still rejects absent or forced review evidence

#### Scenario: Force archive an FD as Complete without review evidence

- WHEN an operator closes an active FD of any defined status as Complete with
  `--force` and a valid reason
- THEN the archive transaction sets Complete, increments revision, moves only
  matching evidence, updates its local navigation, and records the skipped status and
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

### Requirement: Unified staged methods without legacy ticket planning

Current host Skills MUST use the Issue for business scope/approval and the FD
for the single engineering design, Work Items and execution plan. A standalone
legacy ticket-mapping Skill MUST NOT be required or retained in canonical or
project Skill catalogs. Retiring those instructions MUST NOT delete historical
FD evidence, mutate user-level installations or claim removal of existing CLI
implementations. Existing CLI compatibility specifications remain descriptive.

Planner MUST use proportionate planning guidance, compare options when a real
decision needs them, keep acceptance and Work Items clear, resolve prerequisites and load
specification-writing methods only when settled capability rules change.
Stable specs MUST distinguish proposed rules from implemented behavior. New
OpenSpec changes require explicit requests; tasks.md MUST NOT be a second plan.
A later material decision change MUST reconcile FD/specs/items before renewed
implementation review. Discussion or design-only authority MUST NOT implement.

Independent design feedback MAY be selected for an explicit human request
or a concrete unresolved design question. Risk alone MUST NOT add a second
mandatory review before the final implementation review. Material decisions
and findings MUST be resolved before design-ready. Missing requested capability
MUST remain explicit. Design feedback MUST NOT produce implementation
verification events or grant runtime/high-risk authority.

fd-review MUST inspect actual implementation and relevant call paths, covering
Correctness, Standards, Spec and Evidence with the shared method. It MUST own
the managed FD verdict without invoking standalone code-review or dispatching
its two-axis agents for every FD. code-review MUST remain usable for independent
branch/PR scope without changing FD state or implying release approval.
Methods MUST NOT themselves grant agents, review rounds or runtime checks.

A local FD pass/merge/archive MUST NOT require release gate execution. Only a
planned release calls for fd-release-gate operational evidence assessment.
Its GO/GO_WITH_RISK/NO_GO MUST NOT authorize deployment or publishing.

Release assessment evidence MUST identify its release candidate and target
environment, separating observed results, planned or unrun checks, unknowns
and justified inapplicable areas. Missing critical evidence MUST result in an
incomplete assessment and NO_GO recommendation. GO_WITH_RISK MUST be limited
to non-critical residual risks with an owner and evidence of acceptance by an
authorized decision maker; the model MUST NOT accept risk on their behalf.
An FD reference is optional for assessment of other implemented changes.

#### Scenario: Release candidate changes after a readiness assessment

- WHEN the candidate or target environment changes after a GO recommendation
- THEN affected evidence MUST be reassessed for that candidate and environment
- AND the previous recommendation MUST NOT be treated as deployment permission

#### Scenario: Large business request versus engineering work

- WHEN an Issue combines independently acceptable business outcomes
- THEN Issue Management prepares child scopes with lineage and separate approval
- AND Planner later splits each ready FD into engineering Work Items without
  a duplicate ticket or OpenSpec execution checklist.

#### Scenario: Design feedback precedes implementation

- WHEN an explicit request or unresolved design question selects design review
- THEN one bounded independent Reviewer supplies focused design feedback
- AND Planner resolves material findings before design-ready, without emitting
  an implementation verification-passed event from that design report.

#### Scenario: Review and release remain separate

- WHEN a complete FD passes independent implementation review
- THEN the host may continue authorized local delivery/archive without running
  a second standalone code-review or a release-readiness assessment
- AND a later planned release is assessed against operational evidence separately.


### Requirement: Reviewable test design and version-bound evidence

Selected TDD/fd-test work MUST derive important scenarios from confirmed
acceptance or contracts before using implementation as adaptation material.
Each key scenario MUST record rule reference, input/preconditions, expected
result and independent derivation, key assertions and plausible fault detected.
Characterization MUST be distinguished from confirmed requirement correctness.
Static fault reasoning MUST NOT be represented as executed mutation evidence.
Reused tests and implementation access MUST be disclosed. Reports MUST bind
actual implementation/test commits and dirty-diff identities, FD revision,
commands, authorization, environment, exit codes and raw output. Observed status
MUST be separate from product/test/environment/undetermined cause. Unrun, partial
or stale evidence MUST NOT be reported as passed. Historical reports stay intact.

#### Scenario: Expected output is uncertain

- WHEN the rounding or permission rule needed for a case is unconfirmed
- THEN the affected case is blocked with the missing rule
- AND independent cases continue without copying production output as the oracle

#### Scenario: Meaningful TDD Red

- WHEN an authorized predeclared TDD cycle observes its target assertion fail
- THEN the failure may be recorded as Red, followed by scoped implementation
  and authorized Green with the same behavioral assertion
- AND import/configuration failures are not target Red
- AND absence of executed Red/Green is not represented as completed TDD

### Requirement: Semantic test support and bounded diagnosis fallback

The host MUST recognize a request for test support by intent in any language,
not require literal with test syntax. Negation, cases-only, no-TDD, no-execution
and stage limits MUST take precedence. Planner MUST record scope, appropriate
TDD slices, independent acceptance round, authority and finite budget in FD
Verification. No extra CLI flag, FD state, Tester event or engineering plan is
introduced. Worker MUST NOT emit implementation-ready before selected test
support has been accounted for. Independent testing MUST use a different
session without concurrent workspace writers; missing capability is explicit.

Each planned TDD Red/Green phase MUST have authority. Expected Red is not an
unexpected repair retry. Unexpected failures MUST follow repository budgets:
here one focused execution and at most one rerun after a relevant correction,
with stricter limits prevailing. No unchanged failure rerun or allowance reset
through new commands, cases, sessions, review rounds or resume is permitted.
At exhaustion, no safe fix or unclear cause, host MUST automatically stop the
affected effort's test-driven repair and execution, perform bounded static
diagnosis-only, retain failures/versions/ledger and continue independent work.
Renewed repair/execution requires explicit direction and valid authority.

#### Scenario: Natural-language support request

- WHEN a user requests FD auto with test support in equivalent English, Chinese
  or another language
- THEN suitable core-rule TDD and independent fd-test are selected
- AND exact command/revision authority and local language prohibitions remain
- AND a narrower request for cases only does not authorize execution

#### Scenario: Repeated failure degrades without losing the scene

- WHEN a focused test fails and the authorized rerun after correction also fails
- THEN further affected repair/test execution stops automatically
- AND bounded static diagnosis records observations, cause or uncertainty,
  candidate correction, budget and remaining work
- AND independent work continues without silently resetting retries on resume

#### Scenario: Diagnosis does not hide an acceptance defect

- WHEN diagnosis-only retains a confirmed acceptance defect or material unknown
- THEN the host MUST NOT emit full-FD readiness, merge or archive Complete
- AND it may complete other authorized work and scoped static feedback
- AND environment failure alone is not classified as a product defect or pass

#### Scenario: Changes invalidate relevant test evidence

- WHEN review repair or merge recovery changes previously tested behavior
- THEN affected results are marked stale and unrelated evidence is reused only
  with justification
- AND diagnosis-only does not grant automatic reruns or further test repairs


### Requirement: Conditional diagnosis and architecture methods

FD design MAY load codebase-design for an actual interface/responsibility or
test-seam question, and improve-codebase-architecture for an observed structural
obstacle. Implementation and selected testing MAY use diagnosing-bugs for an
unclear failure. These MUST remain independently usable methods, not mandatory
FD stages, extra role events or a replacement for fd-review's independent verdict.
Decisions/evidence MUST return to the current FD/report; out-of-scope refactors
MUST remain follow-up proposals rather than automatic implementation.

Diagnosis MUST permit static evidence without a previously run failing command.
It MUST separate observations from product, oracle/test, fixture/setup,
environment or undetermined causes and distinguish confirmed cause from
hypothesis. Runtime experiments MUST retain the existing command/revision
authority and budget. In diagnosis-only, the method MUST use one bounded static
analysis or reference the analysis already recorded, preserve evidence/ledger
and propose corrections without applying them. Changing methods MUST NOT restart
repair/execution or waive unresolved acceptance defects.

Architecture exploration MUST default to scoped Markdown and local analysis.
It MUST NOT require HTML, agents, a browser, CDN access or grilling. Requested
HTML MUST be offline, uniquely stored under workspace .ai/tmp and not opened
automatically. Alternative interfaces SHOULD be compared in the current session;
explicitly authorized independent comparison MUST use at most two bounded static
agent passes total without recursive delegation or extra runtime authority.
Design principles MUST be contextual guidance, not adapter-count requirements,
project terminology bans or permission to delete existing tests automatically.

#### Scenario: Static bug diagnosis without runtime authority

- WHEN Worker has a reported symptom and relevant source/output but no runtime authorization
- THEN a bounded static diagnosis MAY identify a cause or supported hypothesis
- AND not-run reproduction MUST remain explicit rather than blocking all source analysis

#### Scenario: Testing budget already exhausted

- WHEN the selected testing effort is already diagnosis-only
- THEN diagnosing-bugs MUST reuse the existing analysis or consume preserved evidence once
- AND it MUST NOT execute, instrument or apply the candidate repair through another Skill
- AND independent work MAY continue while unresolved acceptance defects remain blocking

#### Scenario: Structural obstacle during FD design

- WHEN a caller/interface or structural issue prevents a suitable scoped design
- THEN Planner MAY use the relevant design/architecture method and record the result in this FD
- AND broader redesign MUST remain a proposal without automatic implementation, HTML or agents

### Requirement: Scoped domain and decision methods support FD design


Planner MAY load Domain clarification and Decision dependencies from the
entry-declared shared workflow-methods resource for actual design gaps.
It MUST preserve user-goal -> software-acceptance -> engineering-decision links.
Issue owns business rules/approval; the FD remains the only engineering plan
and Work Item checklist. Material business gaps return to the Issue owner and
material engineering unknowns MUST block design-ready. Independent ready work
may continue without inventing a full-FD readiness result.

Confirmed vocabulary belongs in CONTEXT, stable capability rules in specs,
current choices/dependencies/items in the FD and durable non-obvious trade-offs
in ADRs when justified. Cross-links MUST prevent competing copies of rules.
Discussion/modeling alone MUST NOT authorize stable writes, new ADRs, automatic
research/prototype agents or runtime checks. No wayfinder stage or label machine
is introduced; historical maps and user-global installs are not auto-migrated.

#### Scenario: Domain discussion reaches an engineering decision

- WHEN scoped modeling resolves a term and a related state invariant
- THEN authorized vocabulary edits hold the concise definition and link to the
  owning stable/proposed rule, without duplicating a state specification
- AND the FD records current engineering choices and unresolved gates

#### Scenario: Dependent design spans sessions

- WHEN several engineering questions depend on prior decisions
- THEN Planner records prerequisites and answer links in the same FD, resumes
  from settled evidence and preserves the sole Work Item plan
- AND resolving one question neither limits the whole session to one ticket
  nor grants implementation authority for unresolved decisions

### Requirement: Independent feature folders and compatibility

New active FDs MUST use docs/features/FD-ID/ with their original numbered
design filename, index.md, reports/ and reviews/. Archived folders MUST remain
docs/features/archive/FD-ID/. Existing flat designs and evidence directories
MUST remain readable by FD and worktree tooling. Evidence inspection and shell
completion MUST recognize both layouts. Historical documents, JSON and issued
event digests MUST NOT be rewritten during this structural migration.

Status transitions MUST NOT update a shared tracked status table or another
FD's navigation. New/reopened/re-reviewed records MUST use the folder layout.
Local navigation MUST link all historical designs when a duplicate historical
FD ID exists, without collapsing or overwriting them. ID selection MUST retain
existing ambiguity errors and active-first precedence.

Missing legacy artifact paths MAY resolve to a unique exact filename in the
FD's canonical active/archive locations. Ambiguous candidates MUST fail. This
CLI compatibility MUST NOT be represented as a browser redirect. Consumers
of the former FEATURE_INDEX.md status tables MUST migrate to FD files or list.

Archive MUST move evidence from both active layouts, reject duplicate target
filenames before moving any evidence, and restore both local indexes on failure.
Only AIW-generated local navigation MAY be replaced. Reopening and requesting
archived review MUST preserve prior evidence and update both local indexes with
rollback on failure.

#### Scenario: Independent concurrent FD state changes

- WHEN two worktrees advance different FDs
- THEN their status/navigation writes affect only their respective FD folders
- AND no shared generated status table is changed
- AND genuine overlapping source changes retain normal Git conflict reporting.

#### Scenario: Legacy and folder evidence collide

- WHEN two evidence sources would archive to the same filename
- THEN archive fails clearly before moving evidence
- AND neither source is silently selected or overwritten.
