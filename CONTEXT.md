# AIW Workflow

AIW manages local engineering Tasks and their execution context while OpenSpec
owns requirement and implementation-checklist artifacts.

## Language

**Primary Workspace**:
The AIW project root in the repository's primary Git checkout.
_Avoid_: Current directory, main worktree

**Isolated Workspace**:
A linked Git worktree explicitly created and managed for one AIW Task.
_Avoid_: Task workspace, temporary checkout

**Unassigned Workspace**:
A Task state in which no writable execution workspace is currently bound.
_Avoid_: Missing worktree

**Workspace Binding**:
The explicit association between a Task, its workspace kind, workspace path,
and branch.
_Avoid_: Worktree mapping

**Task Completion**:
The state in which the Task checklist and Verification record are complete.
_Avoid_: Delivery, merge completion

**Git Delivery**:
The optional process of integrating isolated Task changes into the recorded
parent branch.
_Avoid_: Task completion, archive
