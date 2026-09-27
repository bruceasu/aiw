## Why

The Codex pilot implementation needs one real, tightly scoped invocation to verify that a new Schema 10 Task can persist Provider-reported usage and complete its frozen compile step. A disposable Task makes that evidence independent of the implementation Task and avoids accepting or delivering the trial result.

## What Changes

- Create a one-WorkItem trial Task whose Coder writes one small Markdown proof file in its isolated worktree.
- Run the opt-in Codex pilot only after explicit preflight and budget configuration; record the original Session usage and compile evidence.
- Stop at Tester. Do not run tests, accept the WorkItem, deliver changes, or clean up the worktree.

## Capabilities

### New Capabilities
- `codex-pilot-trial`: A disposable one-WorkItem operational trial of the Schema 10 Codex pilot.

### Modified Capabilities

## Impact

Uses the existing `aiw wf pilot` CLI, Codex Session adapter, Task usage ledger, and frozen compile plan. The only intended workspace edit is `docs/codex-pilot-trial.md` in the isolated trial worktree. The real Codex invocation consumes account resources and is not sandboxed from the local host or network.
