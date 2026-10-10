# AIW Workflow

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
