# AIW — AI Worker

AIW means **AI Worker**: an AI participant that works from an explicit goal,
follows a recorded plan, hands off scoped work, and reports real evidence.
The AIW CLI supplies workflow records and tool operations. Skills guide the
host Agent; the host and configured providers supply AI execution.

Humans decide goals, scope, and authorization. PM manages workflow state and
human decisions. Planner defines the design and ordered Work Items. Worker
implements the ready scope. An independent Reviewer checks the implementation
against the FD. The Worker role is one stage of the broader AI Worker workflow.
Automatic role dispatch requires a configured runner; otherwise handoffs remain
pending for a Skill or a later resume operation.

AIW manages numbered Feature Designs (FDs), role handoffs, and their execution
context. An Issue captures intent and approval before FD design. The numbered
FD owns design decisions, Work Items, acceptance, status, and Verification;
OpenSpec owns stable capability specs under `openspec/specs/`. An OpenSpec
change is optional and is created only when explicitly requested.

New Issues use IDs such as `ISSUE-00001` under `docs/issues/<id>/`; existing REQ records
keep their IDs and `docs/requirements/` paths. New FDs live under
`docs/features/FD-XXX_SLUG.md`. `FEATURE_INDEX.md` is a derived index;
`.ai/fd/<fd-id>/` stores receipts, workspace metadata, and operator audits.
Neither owns FD status. Archived designs and evidence live under
`docs/features/archive/<FD-ID>/`.

The default handoff sequence is Planner → Worker → independent Reviewer.
Optional `$fd-test` reports do not gate FD acceptance. Existing Task and
Workflow Core records remain available for compatibility.

## Language

**AI Worker**:
The meaning of AIW and the project's organizing idea: AI work with an explicit
scope, a maintained plan, clear handoffs, and traceable evidence.
_Avoid_: Assuming that all roles run automatically or share one session

**Worker**:
The implementation role in the FD lifecycle. It reports actual changes and
verification limits; it does not author its own independent review.
_Avoid_: Using Worker as a synonym for PM, Planner, or Reviewer


**Primary Workspace**:
The AIW project root in the repository's primary Git checkout.
_Avoid_: Current directory, main worktree

**Isolated Workspace**:
A linked Git worktree created and managed for one numbered FD, using
`feature/<fd-id>` and `.wt/<fd-id>`. Legacy Tasks keep their Task-based naming.
_Avoid_: Task workspace, temporary checkout

**Unassigned Workspace**:
A Task state in which no writable execution workspace is currently bound.
_Avoid_: Missing worktree

**Workspace Binding**:
The explicit association between an FD or legacy Task, its workspace path,
and branch. For an FD, `.ai/fd/<fd-id>/workspace.json` also records the parent
branch used for delivery.
_Avoid_: Worktree mapping

**FD Completion**:
The FD's scoped Work Items and Verification are resolved, with a current
independent Reviewer pass for normal Complete archive. Operator overrides
remain identified as overrides and do not fabricate review evidence.
_Avoid_: Delivery, merge completion

**Legacy Task Completion**:
The state in which the Task checklist and Verification record are complete.
_Avoid_: Delivery, merge completion

**Git Delivery**:
The process of integrating isolated FD changes into the recorded parent branch
with `aiw git wt local-merge <fd-id>`. Successful delivery creates a verified
squash commit and removes the clean FD worktree and branch while preserving
receipts. FD archive follows delivery when isolated delivery was requested.
_Avoid_: Task completion, archive
