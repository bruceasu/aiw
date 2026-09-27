# Codex pilot capability

## Requirements

### Requirement: Explicit and isolated activation

The pilot MUST require an explicit command on a new Task with exactly one mapped WorkItem, an input/output Token budget, a Codex profile, verified activation evidence, and a frozen compile plan. It MUST NOT automatically migrate or dispatch any existing Task. Failed preflight MUST leave Schema 9 state unchanged.

#### Scenario: Existing Task or incomplete host

- **WHEN** activation is requested for an active Task or the controlled host cannot prove a required capability
- **THEN** activation is rejected with a specific reason and no Schema 10 migration event is written

### Requirement: One real controlled Coder and compiler path

The pilot MUST use the existing Codex CLI Session adapter for one Stage Coder request, persist the original request and Provider usage, validate its implementation report, and run the frozen compile-only plan after a verified terminal Coder result. An unknown invocation MUST remain bound to its original request and MUST NOT be silently redispatched. Stop or an open budget Gate MUST prevent further generation.

#### Scenario: Codex terminal usage

- **WHEN** Codex returns a terminal JSONL event with usage
- **THEN** the Task records Provider-reported input, output, cached-input, and reasoning-output Tokens for the exact Session turn before moving to compilation

#### Scenario: Interrupted dispatch

- **WHEN** the process exits or the supervisor restarts without a verified terminal receipt
- **THEN** the request remains unknown and requires read-only reconciliation of that invocation; no second model call is made

### Requirement: Deliberate stop after compilation

The pilot MUST stop at the Tester boundary even if compilation passes. It MUST NOT mark the WorkItem accepted or run tests, delivery, or cleanup.

#### Scenario: Compile passes

- **WHEN** controlled compilation succeeds
- **THEN** the pilot reports Coder and compile evidence, keeps later phases unavailable, and tells the operator that the WorkItem is not accepted

%% Verification: Pending implementation and focused review. No real Codex call is authorized by this specification.
