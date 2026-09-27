# Schema 10 Codex CLI pilot checklist

- [x] 1.1 Inspect existing Codex Session receipts, StageExecutor, migration, budget, and compile contracts; record exact reuse and missing host guarantees.
- [x] 1.2 Define explicit pilot activation and refusal rules for a new one-WorkItem Task; keep Schema 9 and existing Tasks unchanged.
- [x] 2.1 Implement a real Codex Coder host with frozen request/Session identity, bounded execution, durable terminal evidence, and read-only unknown-result reconciliation.
- [x] 2.2 Connect verified activation and Schema 10 Store services without empty-success callbacks; expose a narrow opt-in CLI entrypoint.
- [x] 2.3 Wire the original Coder result and Provider usage to the Task ledger, validate the implementation report, then run the frozen compile-only stage; stop at Tester without acceptance or delivery.
- [x] 2.4 Enforce configured input/output Token Gate and Stop before each new model dispatch; preserve unknown usage and existing approvals.
- [x] 3.1 Add focused tests for refusal paths, identity/restart, unknown result, usage replay, budget Gate, and compile stop; do not run them without authorization.
- [x] 3.2 Run `python scripts/compile.py`, document the exact exit status, and provide a manual real-Codex pilot command requiring separate approval before invocation.

## Verification

%% 1.1 complete by static review of Codex JSONL, Session persistence, StageExecutor/Store, usage, and compile paths. The Task and OpenSpec change have been created; no model call, test, final build, or Git delivery has been run.
%% 1.2 complete in design.md: explicit status/activate/run boundary and fail-closed preflight; no runtime behavior has been exercised against a real Task.
%% 2.1 code implemented: managed Codex `exec --json` path, frozen StageRequest journal, process-tree identity/termination checks, bounded/synced JSONL capture, and idempotent Session recovery. The pilot no longer depends on an injected sandbox VerificationBoundary by operator decision. `python scripts/compile.py` exited 0 on 2026-09-28 after this host change. No model call or tests were run.
%% 2.2 implemented: `execution.ConnectCodexPilot` installs actual Coder verification and generation-budget services, requires a real activation verifier, rejects Acceptance, and limits generation profiles to Codex. `pilot status|activate` performs preflight, snapshots immutable activation evidence, prepares the durable Task lock, migrates explicitly, and configures the input/output Token budget. Preflight refuses prior Stop and freezes the selected Codex profile plus eligible higher-level Codex profiles. Compile result is recorded below. `pilot run` belongs to 2.3 with the durable Stage/report/compile pipeline; no model call or tests were run.
%% Verification 2026-09-28: `python scripts/compile.py` exited 0 after 2.2 changes. No tests or real Codex invocation were run.
%% 2.3 implemented: `aiw wf pilot <task-id> run` resumes the exact active Coder/compile request, persists normalized Provider usage idempotently before stage consumption, validates the Coder report, runs only frozen compile targets, and stops at Tester without acceptance, test, delivery, or cleanup. Unknown compile state is not rerun. Verification (2026-09-28): `python scripts/compile.py` exited 0. No tests or real Codex invocation were run.
%% 2.4 implemented by reusing the ordered Task usage ledger and Stage Stop checks. Pending usage authorization now blocks Coder/Tester generation at both request preparation and dispatch claim, while deterministic report validation and local compile can proceed. Unknown usage remains unknown; approvals and Stop records are not reset. Verification (2026-09-28): `python scripts/compile.py` exited 0 after correction. No tests or real Codex invocation were run.
%% 3.1 added focused Windows workflow coverage for Stop-before-claim, budget-Gate dispatch refusal, compile continuation through a pending Gate and stop-at-Tester projection; added pilot tests for activation refusal, frozen invocation identity, restart/no-redispatch, and unknown compile receipts. Existing usage-ledger replay and approval tests cover idempotency. Static review only; tests are explicitly not run in this task.
%% Manual real-Codex pilot requires a separate operator decision and a disposable Task: `aiw wf pilot <disposable-task-id> status`, then `aiw wf pilot <disposable-task-id> activate`, then `aiw wf pilot <disposable-task-id> run`. Do not run `activate` or `run` until separately authorized; `run` may invoke Codex and edit the selected workspace.
%% 3.2 Verification (2026-09-28): `python scripts/compile.py` exited 0. This compile script does not execute or compile Go test files. Focused tests were not run as directed; no real Codex invocation was made.

## Risks

%% RISK: Verify in the separately authorized real Codex pilot that the host journal correlates a StageRequest to a terminal turn and confirms process-tree exit after interruption. Compile evidence is not runtime proof. Conclusive evidence enables automatic idempotent recovery; human reconciliation is optional. Ambiguous liveness or missing evidence stays unknown and must never trigger redispatch. The pilot intentionally provides no OS sandbox or network isolation; only use it in the locally trusted workspace the operator selected.
