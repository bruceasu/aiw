# Workflow execution

`workflow` owns runtime models, Store transitions, leases, and compiler rules.
`execution` owns the foreground Supervisor loop, Session outcome validation,
compile repair sequencing, and delivery eligibility. It does not import command
packages or terminal UI.

`task` owns Task metadata, workspace resolution, checklist adaptation, and
handoff artifacts. Its dependency remains directed toward Workflow Core.
`commands/task` constructs execution dependencies and renders terminal output.

## Current migration scope

- [x] Move the Supervisor loop and outcome consumption into execution.
- [x] Move frozen-plan compile and repair orchestration into execution.
- [x] Move Task checklist, workspace preflight, and handoff operations into task.
- [x] Keep terminal rendering in commands/task.
- [x] Move outcome, compile repair, and scoped Git environment tests with their implementation.
- [ ] Migrate the shared single-step Runner behind Supervisor.RunStep.
- [ ] Migrate the Git merge implementation behind Supervisor.Merge.

The last two callbacks are temporary migration seams, not the target ownership
for execution logic. Both still use existing command implementations. This slice
does not claim that all of commands/task has been reduced to entry and UI code.

## Behavior preserved

Dispatch, Session validation, compile, outcome recording, and delivery keep their
existing order. Gate handling, retry limits, frozen requests, compile repair
ownership, and stop semantics remain unchanged. The unused command heartbeat
helper was removed; no heartbeat or cancellation behavior was added to Start.

Report and Delivered are CLI presentation callbacks. Report retains its error
return because Task metadata read failures already interrupt execution.

## Verification

Static review covers moved call sites, package dependencies, and state update
ordering. Existing tests are relocated; no tests or Git delivery are run during
this migration. Compile verification remains unconfirmed because the earlier
compile-only command encountered denied access to the configured Go cache.

%% The shared Runner and Git Delivery adapters still need a subsequent migration.
%% This extraction does not fix the existing long-running operation lease gap.
