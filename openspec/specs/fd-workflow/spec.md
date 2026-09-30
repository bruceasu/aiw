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

Worker MUST resolve all scoped Work Items before requesting review. An
independent Reviewer MUST record a report for either `changes-requested` or
`verification-passed`. Completion MUST NOT imply an unrun test passed.
Archiving as Complete MUST require the current Reviewer's
`verification-passed` event with the same FD revision and content digest,
not only editable FD status text.

#### Scenario: Review fails

- **WHEN** Reviewer emits `changes-requested` with a report
- **THEN** the FD returns to In Progress and Worker receives the report.

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

- **WHEN** the FD is not archived and Complete, the reason is invalid, or the
  latest acknowledged Reviewer pass is missing
- **THEN** the request fails without creating an event or changing the FD

### Requirement: Git and workspace boundary

An FD MAY have an isolated worktree once its plan is committed. Local focused
commits MAY be made when authorized. Commit MUST NOT imply push, merge,
release, deployment, worktree deletion, or archive.

### Requirement: Legacy records remain readable

Existing Task and Workflow Core records MUST NOT be rewritten as FD events or
evidence. The old Task and `aiw wf` paths MAY remain available as explicit
compatibility commands while new FD work avoids Supervisor.
