## ADDED Requirements

### Requirement: One isolated real-Codex trial

The trial SHALL contain exactly one mapped WorkItem in an isolated worktree that includes the Schema 10 Codex pilot implementation. It SHALL use the explicit configured Codex profile and positive Task input/output Token limits. The Coder SHALL create only `docs/codex-pilot-trial.md`; after validated report and frozen compile-only evidence, the workflow SHALL stop at Tester without tests, acceptance, delivery, or cleanup.

#### Scenario: Real invocation and compile finish
- **WHEN** the operator explicitly activates the eligible trial Task and runs its current WorkItem
- **THEN** the original Codex turn and Provider-reported usage are recorded, the specified Markdown proof file is created, the frozen compile plan runs, and the WorkItem remains unaccepted at Tester

#### Scenario: Preflight or execution is inconclusive
- **WHEN** the profile, limits, workspace, process state, usage, or terminal evidence cannot be verified
- **THEN** the workflow fails closed, preserves the original Task/request evidence, does not dispatch a second model call, and asks the operator to review the specific blocker
