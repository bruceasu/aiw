# Schema 10 Codex CLI pilot checklist

- [x] 1.1 Inspect existing Codex Session receipts, StageExecutor, migration, budget, and compile contracts; record exact reuse and missing host guarantees.
- [x] 1.2 Define explicit pilot activation and refusal rules for a new one-WorkItem Task; keep Schema 9 and existing Tasks unchanged.
- [ ] 2.1 Implement a real Codex Coder host with frozen request/Session identity, bounded execution, durable terminal evidence, and read-only unknown-result reconciliation.
- [ ] 2.2 Connect verified activation and Schema 10 Store services without empty-success callbacks; expose a narrow opt-in CLI entrypoint.
- [ ] 2.3 Wire the original Coder result and Provider usage to the Task ledger, validate the implementation report, then run the frozen compile-only stage; stop at Tester without acceptance or delivery.
- [ ] 2.4 Enforce configured input/output Token Gate and Stop before each new model dispatch; preserve unknown usage and existing approvals.
- [ ] 3.1 Add focused tests for refusal paths, identity/restart, unknown result, usage replay, budget Gate, and compile stop; do not run them without authorization.
- [ ] 3.2 Run `python scripts/compile.py`, document the exact exit status, and provide a manual real-Codex pilot command requiring separate approval before invocation.

## Verification

%% 1.1 complete by static review of Codex JSONL, Session persistence, StageExecutor/Store, usage, and compile paths. The Task and OpenSpec change have been created; no model call, test, final build, or Git delivery has been run.
%% 1.2 complete in design.md: explicit status/activate/run boundary and fail-closed preflight; no runtime behavior has been implemented or verified.

## Risks

%% NEEDS_INPUT: Prove that the real Codex host journal can correlate a StageRequest to a terminal turn and confirm process-tree exit after interruption. Conclusive evidence enables automatic idempotent recovery; human reconciliation is optional. Ambiguous liveness or missing evidence stays unknown and must never trigger redispatch.
