## 1. Design gates and contracts

- [x] 1.1 Resolve the Provider usage envelope with field-level `known`, `unknown`, and `invalid` states, Provider-only cost/currency evidence, and partial-field semantics; remove the corresponding `%% NEEDS_INPUT` note from `design.md`.
- [x] 1.2 Define the deterministic WorkItem difficulty escalation signal as an unresolved, unaccepted Agent round and update the Profile-level selection decision and acceptance scenarios.
- [ ] 1.3 Confirm the normalized Schema 10 persistence and event compatibility contract for existing Tasks and Sessions.

## 2. Provider and Session evidence

- [ ] 2.1 Add a versioned Provider usage envelope carrying Token, monetary cost, availability status, Provider, Model, timestamps, and raw-response evidence.
- [ ] 2.2 Populate the envelope from Codex CLI, Copilot CLI, OpenAI SDK, and other supported Provider adapters using returned values only.
- [ ] 2.3 Persist normalized usage evidence with each Session turn while preserving existing output, stderr, lifecycle, and one-call override behavior.
- [ ] 2.4 Extend `[ai.profiles.<id>]` parsing with explicit level and reasoning-intensity settings without exposing credentials through Profile records.

## 3. Workflow accounting and budget control

- [ ] 3.1 Add the Schema 10 Task usage ledger, cumulative budget projection, unknown-usage counters, and backward-compatible state loading.
- [ ] 3.2 Record one idempotent usage event bound to Task, WorkItem, Attempt, Session turn, Provider, Model, Profile, difficulty, parameters, and outcome.
- [ ] 3.3 Aggregate known Token and monetary values across the whole Task and preserve raw usage records without converting unknown values to zero.
- [ ] 3.4 Open a human authorization gate when either known budget reaches its limit; implement simultaneous default 30 percent increases, explicit overrides, repeated approvals, and audit history.
- [ ] 3.5 Implement human termination as `BLOCKED` and preserve usage, budget, approval, and recovery history across restart and repeated recovery.
- [ ] 3.6 Implement deterministic Profile selection by level and Profile name, next-higher-level fallback, reasoning-intensity changes, and difficulty audit records.

## 4. Reporting and operator surfaces

- [ ] 4.1 Add the read-only `aiw wf usage <task-id>` command with bounded WorkItem, Attempt, Provider, Profile, and RFC3339 time-range filters plus table and JSON formats.
- [ ] 4.2 Report known Token totals, known monetary totals by currency, call counts, `usage_unknown` counts, budget approvals, difficulty changes, and outcomes separately through the Workflow facade.
- [ ] 4.3 Keep `aiw wf status` limited to budget state, pending authorization, BLOCKED reason, selected Profile, and partial-usage diagnostics; do not expose raw Provider evidence in the normal report.

## 5. Focused verification

- [ ] 5.1 Add Provider and Session tests for complete usage, partial usage, `usage_unknown`, raw evidence, and backward-compatible records.
- [ ] 5.2 Add Workflow tests for Task-wide aggregation, idempotent replay, concurrent approval serialization, repeated 30 percent increases, override limits, and BLOCKED recovery.
- [ ] 5.3 Add Profile and difficulty tests for explicit levels, same-level name ordering, next-higher fallback, reasoning-intensity changes, and audit records.
- [ ] 5.4 Add report tests for every supported dimension and separate known/unknown totals.
- [ ] 5.5 Compile the changed packages with the repository-authorized narrow compile command; run tests only after explicit authorization.

## 6. Documentation and migration

- [ ] 6.1 Document the Profile level/reasoning-intensity configuration, Task Token/monetary budgets, approval behavior, and `usage_unknown` semantics.
- [ ] 6.2 Document Schema 10 migration and recovery behavior for existing Tasks, Sessions, in-flight Attempts, and Provider adapters.
- [ ] 6.3 Update TODO, Verification evidence, unresolved `%%` notes, and the implementation handoff without changing OpenSpec-owned checklist prose from Workflow synchronization.

## TODO

- [x] 7.1 Resolve all design-readiness gates before implementation handoff.
- [ ] 7.2 Record any Provider capability that cannot return monetary cost as an explicit unsupported/unknown capability; never add local pricing inference.

## Verification

- [ ] 8.1 Confirm proposal capabilities, design decisions, delta specs, and checklist scope are consistent.
- [ ] 8.2 Confirm Task ID, `task.toml.id`, and OpenSpec change directory are identical.
- [ ] 8.3 Confirm Workflow Core mapping is synchronized after this checklist is accepted and that no Attempt, lease, or worktree was created during specification.
