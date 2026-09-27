# Schema 10 Requirement Plan draft

> This is a review draft. It is not yet the formal Requirement Plan.

## Facts and confirmed rules

- AI usage cost and resource consumption must be recorded per AI call.
- Cost and usage values come only from the Provider response. AIW must not
  estimate or invent monetary cost.
- The Task is the cumulative budget boundary across all WorkItems and Attempts.
- Token and monetary limits are both enforced. Call count is an auxiliary
  statistic.
- When either limit is reached, both limits increase by 30% by default. A
  human may replace the increase. Repeated approvals are allowed.
- A human may terminate the Task; the resulting state is `BLOCKED`. Recovery is
  not restricted to one command and does not require a recovery event as a
  precondition.
- A Provider response with unavailable usage produces `usage_unknown` and does
  not stop execution. Known Token usage still counts against the Token budget;
  unknown values are not converted to zero.
- Difficulty changes may select a stronger model and higher reasoning
  intensity, using only configured Profiles.
- Profile difficulty is explicitly configured through the existing profile
  namespace, for example `ai.profiles.fast.level = 1`. Same-level Profiles are
  ordered by Profile name. If a target level is unavailable, use the next
  higher available level.

## Goals

- Make Task-level AI budget consumption observable and enforceable.
- Require explicit human control when cumulative budget must increase.
- Make difficulty, Profile selection, parameter changes, usage, cost, and
  outcomes auditable and analyzable together.
- Preserve execution continuity when a Provider reports incomplete usage data.

## In scope

- Provider usage/cost result contract and normalization.
- Task cumulative Token and monetary budget accounting.
- Repeated budget-increase approval and BLOCKED transitions.
- WorkItem difficulty and per-call parameter/outcome records.
- Profile-level configuration and deterministic level selection.
- Statistics by Task, WorkItem, Attempt, Provider, Model/Profile, and time range.
- Explicit `usage_unknown` reporting.

## Out of scope

- Locally inferred pricing or estimated monetary cost.
- Automatic approval of budget increases.
- Selecting models outside the configured Profile allowlist.
- Replacing the existing Provider implementations.

## Acceptance examples

1. A Task aggregates Token and monetary usage across all WorkItems and
   Attempts, and a Token or monetary limit breach requests human approval.
2. The default approved increase raises both limits by 30%; a human override
   is recorded and can be repeated.
3. A call with known Tokens and unknown cost increments the Token total,
   records `usage_unknown` for cost, and continues execution.
4. A difficulty increase selects the configured Profile at the target level;
   same-level ties use Profile name order, and an unavailable level falls
   forward to the next available level.
5. Reports can filter and aggregate by Task, WorkItem, Attempt, Provider,
   Model/Profile, and time range, while retaining unknown-usage counts.
6. Each call record includes difficulty level, Profile, model, reasoning
   intensity, Token usage, monetary cost or unknown status, duration, result,
   and parameter-adjustment status.
7. Human termination puts the Task in `BLOCKED`; a later allowed recovery path
   can resume it without a mandatory recovery-event gate.

## Remaining engineering decisions

- Exact normalized Provider usage schema and capability/error contract.
- Exact budget and approval event schema, including actor and audit fields.
- Persistence and concurrency rules for cumulative Task accounting.
- Exact report/query API and retention policy.
- Definition of difficulty measurement and the trigger for level escalation.
