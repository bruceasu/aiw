## ADDED Requirements

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
