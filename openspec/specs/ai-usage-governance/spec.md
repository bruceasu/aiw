# ai-usage-governance Specification

## Purpose
TBD - created by archiving change schema-10-ai-usage-governance. Update Purpose after archive.
## Requirements
### Requirement: Schema 10 persistence compatibility

Schema 10 usage accounting MUST extend the existing Task Workflow state,
ordered event log, and Session turn persistence seams. Existing Schema 9 Tasks
MUST remain readable without eager migration, and existing Session schema 1
records and turns without usage MUST remain readable without rewrite. Explicit
Task migration MUST preserve the source snapshot and existing event sequence;
it MUST NOT infer historical usage or initialize unknown historical balances
as zero. New Task accounting transitions MUST be committed through the
existing Task lock and ordered Workflow event path, and repeated recovery of a
committed Provider call MUST remain idempotent on its frozen Session/turn
identity. Session usage evidence MUST be an additive optional turn field and
MUST preserve existing output, stderr, event-output, lifecycle, and one-call
override behavior.

#### Scenario: Existing Task without usage ledger

- **WHEN** a pre-Schema-10 Task is read before explicit managed migration
- **THEN** its existing state and event history remain readable, no usage or
  budget is inferred, and no migration is performed as a read side effect

#### Scenario: Explicit Task migration

- **WHEN** an eligible Schema 9 Task passes the existing migration gates
- **THEN** migration preserves its original snapshot and event sequence,
  appends the Schema 10 migration event through the ordered event path, and
  leaves historical usage unknown

#### Scenario: Existing Session turn without usage

- **WHEN** a Session schema 1 record or prior turn has no usage envelope
- **THEN** the record remains readable, the missing usage remains unknown, and
  output, stderr, event-output, lifecycle, and saved Provider/model settings
  retain their existing meaning

#### Scenario: Recover a committed usage event

- **WHEN** recovery encounters a Provider call already committed to the Task
  ledger for its frozen Session and turn identity
- **THEN** it does not append or aggregate that call a second time

### Requirement: Provider usage evidence

AIW MUST persist one normalized usage record for every managed AI call. The
record MUST bind the Task, WorkItem, Attempt, Session turn, Provider, Model,
Profile, difficulty level, reasoning intensity, start/completion timestamps,
duration, outcome, and parameter-adjustment state. Each usage field MUST carry
`known`, `unknown`, or `invalid` state. Token and monetary values MUST come
only from the Provider response; unavailable or malformed fields MUST remain
explicitly unknown/invalid and MUST NOT be estimated or converted to zero.
AIW MUST NOT derive total Tokens from component fields or convert currencies.

#### Scenario: Provider returns complete usage

- **WHEN** a managed Provider call returns Token and monetary usage
- **THEN** AIW persists the values with the exact Provider, Model, Profile,
  Task, WorkItem, Attempt, Session turn, and outcome identity

#### Scenario: Provider returns partial usage

- **WHEN** a Provider returns known Token usage but no monetary cost
- **THEN** AIW records the known Token value, records monetary cost as
  `usage_unknown`, and continues execution without inventing a cost

#### Scenario: Provider call fails after returning usage

- **WHEN** a Provider call fails after returning a response containing usage
  evidence
- **THEN** AIW persists that usage evidence with the failed outcome and does
  not discard it

#### Scenario: Monetary currency is absent

- **WHEN** a Provider returns a monetary amount without a currency
- **THEN** AIW records monetary cost as `usage_unknown` and MUST NOT apply a
  price assumption or currency conversion

### Requirement: Task cumulative AI budget

AIW MUST aggregate known Token and monetary usage across the entire Task,
including every WorkItem and Attempt. Token and monetary limits MUST be
evaluated independently. Call count MAY be reported as an auxiliary metric but
MUST NOT replace either primary budget.

The initial Task budget MUST be supplied explicitly after Schema 10 migration;
AIW MUST NOT infer it from a Profile, Provider, historical usage, or a global
configuration. The Workflow facade MUST provide an operator operation that
accepts one positive Token limit and at least one positive per-currency
monetary limit, and MUST persist it through the ordered Task event path.

#### Scenario: Explicit initial budget

- **WHEN** an operator configures a migrated Task with a positive Token limit
  and one or more positive currency limits
- **THEN** AIW persists those limits as the Task budget and makes them
  available to the cumulative usage Gate

#### Scenario: Missing initial budget dimension

- **WHEN** an operator omits the Token limit or all monetary limits
- **THEN** AIW rejects configuration and does not create a partial budget

The initial Task budget MUST be supplied explicitly after Schema 10 migration;
AIW MUST NOT infer it from a Profile, Provider, historical usage, or a global
configuration. The Workflow facade MUST provide an operator operation that
accepts one positive Token limit and at least one positive per-currency
monetary limit, and MUST persist it through the ordered Task event path.

#### Scenario: Explicit initial budget

- **WHEN** an operator configures a migrated Task with a positive Token limit
  and one or more positive currency limits
- **THEN** AIW persists those limits as the Task budget and makes them
  available to the cumulative usage Gate

#### Scenario: Missing initial budget dimension

- **WHEN** an operator omits the Token limit or all monetary limits
- **THEN** AIW rejects configuration and does not create a partial budget

Initial Token and per-currency monetary limits MUST be read from application
configuration and snapshotted into the Task when Schema 10 accounting is
initialized. For each dimension, the cumulative human-approved increase MUST
NOT exceed 100% of its initial configured limit, so the effective limit MUST
NOT exceed twice the initial limit. Repeated approvals share this ceiling. A
default increase proposal MUST be capped at the remaining allowance, and an
explicit override MUST be rejected if it exceeds the ceiling.

#### Scenario: Initialize a Task budget from configuration

- **WHEN** a Task enters Schema 10 accounting
- **THEN** AIW copies the configured initial Token and per-currency monetary
  limits into that Task's budget baseline

#### Scenario: Repeated overrides reach the cumulative ceiling

- **WHEN** prior approvals have already increased a budget by its initial
  amount
- **THEN** AIW rejects any further increase to that dimension, including an
  increase included in a repeated approval

#### Scenario: Usage crosses one budget

- **WHEN** cumulative known Token or monetary usage reaches its Task limit
- **THEN** AIW opens an authorization gate for additional budget and prevents
  new budget-consuming dispatches until a human decision is recorded

#### Scenario: Unknown monetary usage

- **WHEN** a call has known Token usage and unknown monetary cost
- **THEN** known Token usage counts against the Token limit, monetary usage
  remains unknown, and execution continues

### Requirement: Repeated budget approval and termination

AIW MUST propose a simultaneous 30 percent increase to both Task limits when a
budget increase is requested. A human MUST be able to replace the proposed
limits, approve additional increases repeatedly, or terminate the Task. Each
decision MUST record the actor, time, previous limits, new limits, reason, and
source usage digest. Human termination MUST set the Task to `BLOCKED`.

#### Scenario: Default budget increase

- **WHEN** a human approves an increase without replacement limits
- **THEN** AIW increases both Token and monetary limits by 30 percent and
  records the approval

#### Scenario: Human override or termination

- **WHEN** a human supplies different limits or terminates the Task
- **THEN** AIW records the explicit decision; a termination results in
  `BLOCKED`, and no automatic approval is inferred

The approval and termination operations MUST require a reason. If an actor is
not supplied, the CLI MUST record the current OS user as the actor; failure to
determine that identity MUST reject the operation. An explicit approval MUST
increase every existing budget dimension, while an approval without replacement
limits MUST apply the simultaneous 30 percent increase.

#### Scenario: Repeated approval

- **WHEN** a later budget breach occurs after a previous approval
- **THEN** AIW creates a new authorization decision for the same Task without
  resetting prior usage or approval history

### Requirement: Difficulty and Profile escalation

AIW MUST support an explicit numeric `level` and reasoning-intensity setting
under each configured `ai.profiles.<id>`. Only configured Profiles MAY be
selected. Eligible Profiles MUST be ordered by level and then Profile name in
ascending order. If the requested level is unavailable, AIW MUST select the
next higher available level and persist the selected Profile and reasoning
intensity with the call record.

#### Scenario: Same-level Profile selection

- **WHEN** more than one configured Profile has the selected level
- **THEN** AIW selects the Profile with the lexicographically smallest name

#### Scenario: Missing requested level

- **WHEN** the requested difficulty level has no configured Profile
- **THEN** AIW selects the next higher available level and records the actual
  selected level

#### Scenario: Unresolved Agent round raises difficulty

- **WHEN** one managed Agent round completes but the WorkItem remains unresolved
  and its outcome is not accepted by Workflow
- **THEN** AIW records the previous and selected Profile, model, reasoning
  intensity, and the unresolved-round adjustment reason before the next dispatch

### Requirement: Usage and difficulty statistics

AIW MUST provide bounded statistics over the immutable usage records for Task,
WorkItem, Attempt, Provider, Model/Profile, and time-range dimensions. Reports
MUST separate known Token totals, known monetary totals, call counts, and
`usage_unknown` counts, and MUST retain budget approval and difficulty-change
history.

The operator CLI MUST expose the read-only command
`aiw wf usage <task-id>`. It MUST support WorkItem, Attempt, Provider, Profile,
and RFC3339 time-range filters and `table` and `json` formats. The Workflow
facade MUST expose the equivalent read-only usage-report query to adapters
without requiring them to read ledger storage directly. `aiw wf status` MUST
remain an operational summary rather than a replacement for the usage report.

Normalized usage events, budget decisions, difficulty changes, and evidence
digests MUST remain available for the lifetime of the Task and its archive.
AIW MUST NOT automatically delete those records by age. Raw Provider evidence
MAY be retained separately as a bounded, sanitized usage-related fragment for
90 days by default; after expiry AIW MUST retain its digest and normalized
metadata while removing the raw fragment. Raw evidence MUST NOT include
credentials, full prompts, or full output, and MUST NOT be returned by the
normal usage summary.

#### Scenario: Filtered usage report

- **WHEN** an operator requests statistics for a Task and any supported
  WorkItem, Attempt, Provider, Profile, or time range filter
- **THEN** the report returns only matching records with known totals and
  explicit unknown counts

#### Scenario: Audit a difficulty change

- **WHEN** an operator inspects a call whose Profile or reasoning intensity was
  adjusted
- **THEN** the report shows the prior selection, new selection, difficulty
  level, adjustment reason, usage, duration, and outcome

#### Scenario: Request the machine-readable usage report

- **WHEN** an operator runs `aiw wf usage task-1 --format json` with supported
  filters
- **THEN** AIW returns a bounded JSON summary through the Workflow facade with
  separate known totals, unknown counts, budget approvals, difficulty changes,
  and outcomes

#### Scenario: Retain normalized history after raw evidence expiry

- **WHEN** bounded raw Provider evidence reaches its retention boundary
- **THEN** AIW removes only the raw fragment and preserves the evidence digest,
  normalized usage fields, timestamps, availability states, and accounting
  event in the Task archive

