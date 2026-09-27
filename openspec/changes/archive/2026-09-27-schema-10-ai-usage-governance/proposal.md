## Why

AIW currently records whether a Session or WorkItem ran, but it does not provide
an authoritative Task-level view of Provider-reported AI usage, monetary cost,
budget approvals, or the relationship between difficulty and model parameters.
Schema 10 is needed so supervised execution can control cumulative resource use
and leave auditable evidence for later analysis without inventing cost data.

## What Changes

- Add a Provider usage envelope for Token counts, monetary cost, usage status,
  duration, Provider, Model, Profile, difficulty level, reasoning intensity,
  outcome, and parameter adjustments.
- Add cumulative Task-level Token and monetary budgets across all WorkItems and
  Attempts, with explicit repeated human approvals for increases.
- Default approved budget increases to 30% for both Token and monetary limits;
  allow human overrides and Task termination into `BLOCKED`.
- Add deterministic Profile difficulty selection using explicit Profile levels,
  Profile-name ordering for ties, and the next higher available level fallback.
- Preserve and report `usage_unknown` without converting it to zero or stopping
  execution; known Token usage remains budget-accountable.
- Add bounded statistics by Task, WorkItem, Attempt, Provider, Model/Profile,
  and time range.

## Capabilities

### New Capabilities

- `ai-usage-governance`: Provider usage records, Task budgets, approval gates,
  difficulty/Profile escalation, and usage statistics.

### Modified Capabilities

- `agent-session`: Session turn results expose the Provider usage envelope while
  preserving existing prompt, output, and lifecycle evidence.

## Impact

- Affected boundaries include `internal/ai` Provider responses and Profile
  configuration, `internal/session` turn persistence, and Workflow Core Task,
  WorkItem, Attempt, Gate, and reporting projections.
- Existing Providers remain responsible for their own returned usage data;
  AIW normalizes and persists only values actually returned by the Provider.
- Existing Tasks and Sessions must remain readable when no Schema 10 ledger or
  Provider usage data exists.
- No new pricing table, locally inferred cost, or replacement Provider is in
  scope.
