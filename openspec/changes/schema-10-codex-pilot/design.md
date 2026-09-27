# Design: opt-in Schema 10 Codex pilot

## Boundary

The pilot is a separate CLI adapter, never a fallback in `NextRunnerOutcome`. A new Task is eligible only with one mapped WorkItem, no active Attempt/request/lease, an explicit input/output Token budget, a frozen compile plan, and a configured Codex profile. Activation evidence must be verified before migration. Schema 9 remains the default.

The implementation Task `schema-10-codex-pilot` is not the trial Task. A later disposable trial Task must be created explicitly; this change's checklist is not reduced to one WorkItem.

The CLI surface is `aiw wf pilot <task-id> status|activate|run`. `status` is read-only and lists unmet preflight conditions, while `activate` is the only migration entry and may not call the model. `run` resumes only the current Coder or compile stage: it records normalized Provider usage idempotently before consuming a terminal Coder result, validates the implementation report, and then runs the frozen compile targets. A missing or inconclusive compile receipt remains unknown and is never rerun. Passing compilation stops at Tester; no acceptance, tests, delivery, or cleanup occur. There is no implicit activation in `supervise start` and no migration-on-read.

Activation refuses a Task with a different schema, more than one mapped WorkItem, any active/past Attempt, a write lease, pending event, Supervisor lease or prior Stop, an unresolved Gate, an unavailable Codex profile, a missing frozen compile plan, or absent positive input/output limits. It also refuses if activation evidence cannot be persisted and independently verified. These checks precede migration; failure leaves the Schema 9 Task state unchanged (an identical content-addressed evidence artifact may remain for reconciliation). An already migrated Task is not silently treated as a fresh pilot.

After successful migration, the initial budget is snapshotted through the ordered Task event path before any dispatch. A crash between these commits leaves a migrated-but-unconfigured Task blocked; `status` must offer explicit reconciliation, not infer a budget or replay the migration. The pilot adapter refuses all Tester, test-run, acceptance, delivery, and cleanup requests. A passing compile leaves the item in `PhaseTester` and unaccepted.

## Controlled sequence

`preflight → explicit activation/migration → BeginExecution → frozen Coder request → ClaimStageDispatch → Codex Session turn → persist terminal/unknown observation and Provider usage → report validation → compile request/result → stop at Tester boundary`.

Reuse the existing Session Codex adapter and JSONL usage parser; do not create another token parser. A real host must enforce request identity, write scope, process timeout/output limits, and read-only reconciliation of the original invocation. After interruption, recovery is automatic when durable host evidence proves the original invocation reached a terminal state and its result can be replayed idempotently through the existing Store event path. Human reconciliation is an optional fallback, not a mandatory step. An absent receipt, uncertain process-tree liveness, or incomplete terminal record remains unknown and never permits redispatch; the same invocation must be reconciled or the operator must explicitly stop/escalate it. Only the existing ordered Store event path may update Task state. A failed compile may use the bounded repair sequence only after its stage evidence is recorded.

The pilot runs in a locally trusted environment. OS sandboxing, network isolation, and hostile-code confinement are explicitly out of scope; the operator accepts that Codex may access the host according to the CLI's normal local permissions. The host still enforces a bounded call, durable process identity and output evidence, terminal/process-tree reconciliation, Token budget/Stop, and no redispatch of unknown work. The existing full `ExecutionServices` activation interface remains fail-closed. Pilot-only stages must not claim Tester, acceptance, or delivery capability. Do not connect empty success callbacks or fabricate activation evidence. Activation must still verify the Task, budget, model, compile plan, and host wiring before migration.

## Budget and safety

Use `[ai.usage_budget]` defaults only when explicitly snapshotted for the pilot Task. Input/output totals count Provider-reported values; cache/reasoning are subsets. A pending Gate prevents the next generation. A single call may exceed the remaining budget because its final Token count is not known before it returns; do not claim a per-call hard cap unless the host actually enforces it. Unknown usage remains unknown. No money or credit estimate is required.

## Validation

Compile-only validation is allowed after implementation. Focused tests may be written but are not run by default. A real Codex call needs separate authorization because it consumes account resources and may edit the pilot workspace.

%% NEEDS_INPUT: Verify with the authorized real Codex pilot that the host-owned journal can durably correlate a StageRequest to a terminal turn and prove process-tree exit after a crash. JSONL turn events alone are not proof that a process tree stopped. If this cannot be established, keep that request unknown; never replay the model call. OS sandbox and network-isolation evidence are out of scope by operator decision.

## Current implementation inventory

- Reuse `internal/ai/cli.go` for `codex exec --json`, `internal/ai/usage.go` for Provider Token fields, `internal/session/backend.go` for atomic turn outputs, and `internal/workflow/usage_event.go` for idempotent Task accounting.
- Reuse `execution.RunStage` and `Store.ClaimStageDispatch` to preserve the original request as unknown before crossing the process boundary. `ConsumeStageResult` advances Coder to report validation and compile to Tester; it never directly accepts the WorkItem.
- Reuse the frozen compile-plan selection and the existing Stage `PhaseCompile` contract. The pilot must supply a real StageExecutor and activation services; the normal Runner explicitly blocks Schema 10.
- The Codex pilot service composer wires the real Coder receipt validators and existing generation budget, requires a caller-supplied activation verifier, and rejects Acceptance. `pilot status|activate` performs preflight, persists immutable activation evidence, acquires the durable Task lock, migrates explicitly, and configures the initial input/output Token budget. `pilot run` connects the durable Coder receipt, Provider usage ledger, report validation, and frozen compile stage; it stops before Tester execution. The host-owned invocation journal, process identity/termination checks, and read-only reconciliation are implemented but not exercised against a real Codex process. Human reconciliation is an optional escape hatch only when an authorized evidence source can resolve an otherwise unknown state; it is not required when automatic reconciliation has conclusive evidence.
- The existing live JSONL file is created/truncated by the Session adapter and does not by itself prove terminal process exit after a crash. `Session.StateRunning` is not terminal evidence. Re-running `codex exec` would be a new model call and is forbidden for an unknown request. Recovery may automatically ingest the same terminal result once durable process and receipt evidence are complete; otherwise preserve unknown state and offer optional human reconciliation.

%% Verification (2026-09-28): Static call-path review only. No model invocation, test, or compile was run for the pilot inventory.

%% TODO: Confirm activation and initial Token-budget configuration with a real, separately authorized disposable pilot Task; compile evidence alone is not runtime proof.
