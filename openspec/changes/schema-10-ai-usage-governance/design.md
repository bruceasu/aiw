## Context

AIW already separates Provider execution (`internal/ai`), persisted Session
turns (`internal/session`), and Task/WorkItem orchestration (`internal/workflow`).
The current Provider response records output, events, stderr, exit code, and
timestamps, while Workflow Core records Attempts and gates. There is no shared
accounting record that binds a Provider call to its Task, WorkItem, Attempt,
Session turn, selected Profile, difficulty, resource usage, and outcome.

Schema 10 must preserve the existing Session and Workflow seams. Provider
responses are authoritative for usage and monetary cost; AIW must never infer a
price from a model name, Profile, or local token guess.

## Goals / Non-Goals

**Goals:**

- Persist one auditable usage record for every managed AI call.
- Aggregate known Token and monetary usage at the Task boundary across all
  WorkItems and Attempts.
- Gate budget increases behind explicit human authorization, with repeatable
  approvals and a `BLOCKED` termination path.
- Persist difficulty, Profile, model, reasoning intensity, parameter changes,
  result, and usage together so later reports can correlate them.
- Keep Provider usage gaps explicit as `usage_unknown` while continuing to run.
- Preserve readable behavior for pre-Schema-10 Tasks, Sessions, and Providers.

**Non-Goals:**

- Local pricing tables, estimated monetary cost, or inferred Token usage.
- Replacing Codex CLI, Copilot CLI, OpenAI SDK, or other Provider adapters.
- Automatic approval of a budget increase.
- A new independent store or a second Task lifecycle.
- Deciding business policy for model quality beyond configured Profile levels.

## Decisions

### Provider result and Session evidence

Extend the Provider-neutral response with a versioned usage envelope. The
envelope carries Provider-reported input/output/total Tokens when present,
Provider-reported monetary cost and currency when present, a per-field state
of `known`, `unknown`, or `invalid`, Provider and Model identity, and the call
start/completion timestamps. A missing or malformed field is represented as
unknown/invalid, never as zero.

AIW MUST NOT derive `total_tokens` by adding input and output fields. Monetary
cost is known only when the Provider supplies both an amount and a currency;
AIW MUST NOT infer prices or convert between currencies. Task monetary budgets
are therefore maintained per currency. A failed Provider call may still retain
usage when the response contains usage evidence.

The Session adapter persists the raw Provider response evidence and the
normalized usage envelope beside the existing turn output. The existing
Session lifecycle remains authoritative for turn execution and output; it does
not become the Task budget owner.

### Provider capability evidence

Provider capability is evidence-driven: an adapter reports a field as
`known` only when that field is present and valid in the Provider response.
The following matrix records the current adapter contract; it does not claim
that a Provider always returns every optional field.

| Adapter | Provider usage evidence inspected | Token capability | Monetary-cost capability |
| --- | --- | --- | --- |
| Codex CLI | JSONL stdout, including `last_token_usage` or nested `usage` | Conditional; input/output/total fields are recorded when returned | Conditional; otherwise `unknown`; no local pricing |
| Copilot CLI | JSONL/stdout response and nested `usage` | Conditional; fields are recorded when returned | Conditional; otherwise `unknown`; no local pricing |
| OpenAI Responses API adapter | OpenAI response `usage` object | Conditional; `input_tokens`, `output_tokens`, and `total_tokens` are accepted when returned | Conditional; normal Responses usage without Provider cost remains `unknown` |
| Gemini API adapter | REST response `usageMetadata` | Conditional; `promptTokenCount`, `candidatesTokenCount`, and `totalTokenCount` are accepted when returned | Conditional; normally `unknown` unless amount and currency are returned |
| OpenAI-compatible/Ollama/llama.cpp | JSON response `usage` object | Conditional; recognized Provider fields are recorded | Conditional; `unknown` unless both amount and currency are returned |

The current Gemini implementation uses the Gemini HTTP API directly; it is
not an official Gemini SDK integration. If a future SDK adapter is added, it
must use the same envelope and capability evidence rules. CLI or API
availability, successful model output, and execution duration do not count as
Token or monetary usage evidence. Missing usage remains `unknown`, malformed
usage is `invalid`, and AIW never derives cost from model, Profile, or a local
price table.

### Workflow accounting ownership

Workflow Core owns the immutable Task-level usage ledger and budget projection.
Each record binds `TaskID`, `WorkItemID`, `AttemptID`, `SessionID`, Session
turn/request identity, Provider, Model, Profile, difficulty level, reasoning
intensity, Token/cost values and availability, duration, outcome, and parameter
adjustment status.

The accounting event is idempotent on the frozen managed-call identity. A
recovered or repeated projection MUST NOT count the same Provider call twice.
The ledger stores known totals, explicit unknown counters, and the original
usage records; unknown values are not substituted with zero.

### Task budget and authorization

Budget limits and cumulative usage are Task-scoped and cover every WorkItem and
Attempt. Token and monetary limits are independently evaluated. If either
known value reaches its limit, Workflow Core opens an authorization gate and
pauses new budget-consuming dispatches until a human decision is recorded.

The initial budget is explicit operator configuration, not a global default or
an inferred value. The Workflow facade exposes:

```text
aiw wf budget <task-id> configure --tokens <positive> \
  --cost <CURRENCY=AMOUNT> [--cost <CURRENCY=AMOUNT> ...]
```

Configuration is allowed once for a migrated Schema 10 Task and is committed
through the ordered Workflow event path. The same facade exposes `approve` and
`terminate` for a pending budget Gate. Both require a non-empty reason; the
actor defaults to the current OS user when `--by` is omitted. `approve` without
replacement limits applies the simultaneous 30 percent increase. An explicit
replacement must provide a Token limit and all existing currencies, and the
durable Store remains responsible for checking that every dimension increases.
`terminate` records a human decision and leaves execution `BLOCKED`.

The default approval proposal increases both limits by 30%. A human decision
may provide different new limits. Approvals may repeat and each decision is
recorded with actor, time, previous limits, new limits, reason, and source
usage digest. Human termination records a Task `BLOCKED` outcome; recovery is
not tied to one command and does not require a separate recovery event before
execution.

Provider calls with unknown monetary cost continue. Known Token usage still
counts against the Token limit. `usage_unknown` remains visible in reports and
cannot silently satisfy or clear a monetary budget gate.

### Difficulty and Profile selection

Profile configuration remains under `ai.profiles.<id>`. Each Profile may carry
an explicit numeric `level` and reasoning-intensity setting, for example:

```toml
[ai.profiles.fast]
provider = "openai"
model = "small-model"
level = 1
reasoning_intensity = "standard"
```

Only configured Profiles are eligible. Selection orders eligible Profiles by
level and then Profile name ascending. If the requested level is unavailable,
the selector chooses the next higher available level. The selected Profile,
level, and reasoning intensity are frozen into the managed-call record.

The difficulty escalation signal is the existing Workflow outcome boundary: if
one managed Agent round completes but the WorkItem is not resolved and its
outcome is not accepted, the next dispatch raises the requested difficulty
level. A Provider process failure is recorded separately; it raises difficulty
only when the resulting Agent round is also not accepted by Workflow.

### Reporting seam

Reporting reads the immutable ledger and exposes bounded filters for Task,
WorkItem, Attempt, Provider, Model/Profile, and time range. It returns known
Token and monetary totals separately from unknown counts and includes the
budget history, approvals, difficulty changes, and outcomes needed to explain
why a Task was blocked or escalated.

The stable operator command is:

```text
aiw wf usage <task-id> [--work-item <id>] [--attempt <id>]
  [--provider <id>] [--profile <id>] [--from <RFC3339>] [--to <RFC3339>]
  [--format table|json]
```

The default table is a bounded summary. JSON is the machine-readable contract
for automation and contains the same summary fields, including separate known
Token totals, known monetary totals by currency, call counts,
`usage_unknown` counts, budget approvals, difficulty changes, and outcomes.
`aiw wf status` remains an operational view and reports only budget state,
pending authorization, selected Profile, BLOCKED reason, and recovery
guidance; it does not expand into the complete usage report.

The Workflow facade exposes the corresponding read-only seam:

```go
GetUsageReport(ctx context.Context, taskID string, query UsageReportQuery) (UsageReport, error)
```

CLI, HTTP, and MCP adapters, if added later, use this seam rather than reading
the ledger storage directly. Raw Provider response evidence is not included in
the normal report.

### Usage retention boundary

Normalized usage events, budget decisions, difficulty changes, and evidence
digests are retained for the lifetime of the Task and its archive. They are
not automatically deleted by age because they are the authoritative cost,
budget, and recovery history. Explicit purge of an archived Task is the only
operation that removes this normalized history.

Raw Provider evidence is diagnostic data, not the accounting authority. AIW
retains a bounded and sanitized usage-related fragment for 90 days by default,
with a size limit and without credentials, full prompts, or full output. After
expiry, AIW removes the raw fragment but keeps the evidence digest, Provider
request identity, normalized usage fields, timestamps, and availability
states. The normal usage report never exposes the raw fragment.

### Compatibility and recovery

Schema 10 is an additive runtime state evolution, entered only through the
existing explicit managed migration after its service and activation evidence
gates pass. Existing Schema 9 Tasks remain readable and are not eagerly
rewritten or assigned a new budget. Migration preserves the original state
bytes as its source artifact, retains existing WorkItems, Attempts, and event
sequence, and does not infer historical usage; pre-ledger usage remains
unknown. A migration is rejected while a legacy Attempt, lease, prepared
request, or pending event is unresolved.

The normalized Task ledger and budget projection belong to the existing
versioned Workflow protocol state. Each committed accounting or budget
transition uses the existing Task lock and ordered `events.jsonl` commit path,
with its normal sequence and Schema 10 event metadata. Existing events are
preserved byte-for-byte and in order; Schema 10 events are appended only after
the explicit migration. Usage records bind to the frozen Task, WorkItem,
Attempt, Session, and turn identity, so replay after a crash is idempotent and
cannot change historical accounting. The ledger does not create a second
Task store or event journal.

Session usage is an optional additive field on the persisted result for its
turn. Existing Session schema 1 records, turns without usage, output files,
stderr files, event-output files, lifecycle state, and one-call Provider/model
overrides remain readable and retain their current meaning. Reading an older
turn yields absent/unknown usage; it does not trigger Session migration or
rewrite. The Session copy preserves normalized usage and bounded Provider
evidence, while Workflow remains authoritative for Task aggregation and
accounting decisions. Existing Provider implementations remain valid; a
Provider adapter that does not return a field records it as unknown.

A crash before the accounting event is reconciled from the frozen
Session/Attempt identity; a repeated reconciliation is idempotent. Usage
evidence is committed with the turn result before the corresponding Task
accounting projection is advanced, so recovery can complete or replay the
projection without losing or double-counting the Provider result.

## Design Readiness

**FD_NOT_REQUIRED** — the material Provider, budget, Profile, and Workflow
seams are explicit in this design and remain within the existing AI, Session,
and Workflow boundaries. No independent feature-design artifact is needed.

## Risks / Trade-offs

- [Provider omits monetary cost] → Continue with `usage_unknown`, preserve the
  evidence, and keep the unknown amount visible rather than estimating it.
- [Known Token usage exceeds while monetary cost is unknown] → Enforce the
  Token gate independently and report the monetary dimension as unknown.
- [Repeated recovery or projection] → Use a frozen call identity and ordered
  idempotent accounting event before updating aggregates.
- [Profile level has no exact match] → Select the next higher configured level;
  if none exists, retain the current selection and expose that no escalation
  was available.
- [Concurrent budget approvals] → Serialize Task budget events and reject a
  stale approval whose previous-limit digest no longer matches.

## Migration Plan

1. Add optional Schema 10 fields and readers without changing existing Task or
   Session records.
2. Add Provider adapters' usage envelopes and persist raw/normalized evidence.
3. Initialize new Task budget projections only when a Task first enters Schema
   10 accounting; do not reinterpret historical usage as zero.
4. Enable Profile level selection and Task ledger reporting behind the existing
   managed Workflow path.
5. On rollback, preserve usage evidence and leave unknown/unmigrated fields
   readable; do not delete ledger or approval history.

## Open Questions

No unresolved design questions remain for the report surface or retention
boundary. Exact HTTP or MCP transport details remain implementation work and
must reuse the Workflow facade seam above.
