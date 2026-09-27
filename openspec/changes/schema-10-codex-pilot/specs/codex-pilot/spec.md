# Codex pilot capability

## Requirements

### Requirement: Explicit pilot activation

The pilot MUST require an explicit command on a new Task with exactly one mapped WorkItem, an input/output Token budget, a Codex profile, verified activation evidence, and a frozen compile plan. It MUST NOT automatically migrate or dispatch any existing Task. Failed preflight MUST leave Schema 9 state unchanged. The pilot is for a locally trusted workspace and does not require an OS sandbox or network isolation; it MUST still bound execution time and output, preserve exact invocation identity, and refuse to redispatch an unknown invocation.

#### Scenario: Existing Task or incomplete host

- **WHEN** activation is requested for an active Task or the controlled host cannot prove a required capability
- **THEN** activation is rejected with a specific reason and no Schema 10 migration event is written

### Requirement: One real local Coder and compiler path

The pilot MUST use the existing Codex CLI Session adapter for one Stage Coder request, persist the original request and Provider usage, validate its implementation report, and run the frozen compile-only plan after a verified terminal Coder result. An unknown invocation MUST remain bound to its original request and MUST NOT be silently redispatched. Stop or an open budget Gate MUST prevent further model generation; a budget Gate alone MUST NOT block deterministic report validation or local compile-only work. The operator accepts that the local Codex process is not isolated from the host or network.

#### Scenario: Codex terminal usage

- **WHEN** Codex returns a terminal JSONL event with usage
- **THEN** the Task records Provider-reported input, output, cached-input, and reasoning-output Tokens for the exact Session turn before moving to compilation

#### Scenario: Interrupted dispatch

- **WHEN** the process exits or the supervisor restarts without a verified terminal receipt
- **THEN** the request remains unknown and requires read-only reconciliation of that invocation; no second model call is made

### Requirement: Report, usage, and compile are one recoverable sequence

The pilot MUST account only Provider-reported normalized usage for the exact terminal Session turn, idempotently, before consuming the Coder result. It MUST validate the implementation report before compiling and execute only the frozen compile plan. Missing or ambiguous compile completion evidence MUST remain unknown and MUST NOT trigger a repeated compile request.

#### Scenario: Validated Coder report and compile

- **WHEN** the original Coder turn has a verified terminal receipt and its report passes validation
- **THEN** the Task ledger records the Provider usage once, the frozen compile targets run, and their evidence is persisted against the original request

### Requirement: Deliberate stop after compilation

The pilot MUST stop at the Tester boundary even if compilation passes. It MUST NOT mark the WorkItem accepted or run tests, delivery, or cleanup.

#### Scenario: Compile passes

- **WHEN** controlled compilation succeeds
- **THEN** the pilot reports Coder and compile evidence, keeps later phases unavailable, and tells the operator that the WorkItem is not accepted

%% Verification: focused refusal, frozen-identity/restart, unknown-receipt, usage-replay, budget-Gate, and compile-stop tests are authored. They have not been run. No real Codex call is authorized by this specification.
