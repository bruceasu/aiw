# Schema 10 Codex CLI pilot

## Why

Schema 10 has durable stage and usage contracts, but the normal Workflow Runner deliberately refuses to dispatch them. This prevents a user from trying the real Codex CLI path. A narrowly scoped, explicit pilot is needed without presenting partial execution as production readiness.

## What changes

- Add an opt-in pilot for a new, single-WorkItem Task using Codex CLI with the user's existing ChatGPT login.
- Connect a real Coder generation, report validation, and the repository's frozen compile-only plan to Schema 10 stage records, usage accounting, and a Task-wide input/output Token budget.
- Preserve dispatch identity, automatic idempotent recovery from conclusive durable evidence, unknown-result safety, Stop, and budget authorization. Human reconciliation is optional; inconclusive process or receipt state must fail closed without redispatch.
- Stop after controlled compilation. Tester, test execution, acceptance, delivery, and cleanup remain unavailable in this pilot.

## Out of scope

- Migrating an existing active Task, automatic acceptance/delivery, monetary or credit estimates, Copilot support, and full E01–E04 production activation.
- Running a real model call or end-to-end validation during implementation without separate runtime authorization.

%% TODO: Confirm a real Codex host can satisfy the frozen input, output, process, and reconciliation contract before enabling the pilot command.
