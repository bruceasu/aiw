# Design: opt-in Schema 10 Codex pilot

## Boundary

The pilot is a separate CLI adapter, never a fallback in `NextRunnerOutcome`. A new Task is eligible only with one mapped WorkItem, no active Attempt/request/lease, an explicit input/output Token budget, a frozen compile plan, and a configured Codex profile. Activation evidence must be verified before migration. Schema 9 remains the default.

The implementation Task `schema-10-codex-pilot` is not the trial Task. A later disposable trial Task must be created explicitly; this change's checklist is not reduced to one WorkItem.

The proposed CLI surface is `aiw wf pilot <task-id> status|activate|run`. `status` is read-only and lists unmet preflight conditions. `activate` is the only migration entry and may not call the model; `run` may execute only the current controlled Coder, report-validation, or compile stage. There is no implicit activation in `supervise start` and no migration-on-read.

Activation refuses a Task with a different schema, more than one mapped WorkItem, any active/past Attempt or generation request, a write lease, pending event, Supervisor lease, Stop, an unresolved Gate, unavailable Codex host, missing frozen compile plan, or absent positive input/output limits. It also refuses if the selected profile is not Codex, the real host cannot enforce its execution boundary, or activation evidence cannot be persisted and independently verified. These checks precede migration; failure leaves the Schema 9 Task unchanged. An already migrated Task is not silently treated as a fresh pilot.

After successful migration, the initial budget is snapshotted through the ordered Task event path before any dispatch. A crash between these commits leaves a migrated-but-unconfigured Task blocked; `status` must offer explicit reconciliation, not infer a budget or replay the migration. The pilot adapter refuses all Tester, test-run, acceptance, delivery, and cleanup requests. A passing compile leaves the item in `PhaseTester` and unaccepted.

## Controlled sequence

`preflight → explicit activation/migration → BeginExecution → frozen Coder request → ClaimStageDispatch → Codex Session turn → persist terminal/unknown observation and Provider usage → report validation → compile request/result → stop at Tester boundary`.

Reuse the existing Session Codex adapter and JSONL usage parser; do not create another token parser. A real host must enforce request identity, write scope, process timeout/output limits, and read-only reconciliation of the original invocation. An absent receipt or interrupted process is unknown, not a failure that permits another dispatch. Only the existing ordered Store event path may update Task state. A failed compile may use the bounded repair sequence only after its stage evidence is recorded.

The existing full `ExecutionServices` activation interface remains fail-closed. Pilot-only stages must not claim Tester, acceptance, or delivery capability. Do not connect empty success callbacks or fabricate activation evidence. If the host cannot meet a required contract, activation stays unavailable and the operator sees the exact missing capability.

## Budget and safety

Use `[ai.usage_budget]` defaults only when explicitly snapshotted for the pilot Task. Input/output totals count Provider-reported values; cache/reasoning are subsets. A pending Gate prevents the next generation. A single call may exceed the remaining budget because its final Token count is not known before it returns; do not claim a per-call hard cap unless the host actually enforces it. Unknown usage remains unknown. No money or credit estimate is required.

## Validation

Compile-only validation is allowed after implementation. Focused tests may be written but are not run by default. A real Codex call needs separate authorization because it consumes account resources and may edit the pilot workspace.

%% NEEDS_INPUT: Whether the current Codex CLI can expose a durable request/turn identifier and terminal receipt sufficient for read-only reconciliation after a crash. If not, the pilot must remain disabled rather than replaying the call.

## Current implementation inventory

- Reuse `internal/ai/cli.go` for `codex exec --json`, `internal/ai/usage.go` for Provider Token fields, `internal/session/backend.go` for atomic turn outputs, and `internal/workflow/usage_event.go` for idempotent Task accounting.
- Reuse `execution.RunStage` and `Store.ClaimStageDispatch` to preserve the original request as unknown before crossing the process boundary. `ConsumeStageResult` advances Coder to report validation and compile to Tester; it never directly accepts the WorkItem.
- Reuse the frozen compile-plan selection and the existing Stage `PhaseCompile` contract. The pilot must supply a real StageExecutor and activation services; the normal Runner explicitly blocks Schema 10.
- Missing today: a host-owned durable Codex invocation journal tied to a StageRequest, proof that a timed-out or crashed process tree stopped, read-only reconciliation of the same invocation, a production `ExecutionServices` factory, and an explicit pilot CLI entrypoint.
- The existing live JSONL file is created/truncated by the Session adapter and does not by itself prove terminal process exit after a crash. `Session.StateRunning` is not terminal evidence. Re-running `codex exec` would be a new model call and is forbidden for an unknown request.

%% Verification (2026-09-28): Static call-path review only. No model invocation, test, or compile was run for the pilot inventory.

%% TODO: Check the proposed CLI shape against the concrete host implementation before documenting copy/paste operator commands; do not expose `activate` until all preflight and reconciliation guarantees exist.
