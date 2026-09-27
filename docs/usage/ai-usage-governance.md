# Schema 10 AI usage governance

Schema 10 records Provider-reported usage for managed AI calls and accounts
for it across the whole Task. This controlled Workflow path is separate from
the default CLI execution loop; configuring a Profile does not enable Schema
10 execution or migrate an existing Task.

## Profile levels and reasoning intensity

Profiles remain under `[ai.profiles.<name>]`. Keep the required `provider` and
`model` fields, and add an integer `level` and optional `reasoning_intensity`
to describe the configured choice:

```toml
[ai.profiles.fast]
provider = "codex"
model = "your-fast-model"
level = 1
reasoning_intensity = "low"

[ai.profiles.reasoning]
provider = "codex"
model = "your-reasoning-model"
level = 3
reasoning_intensity = "high"
```

Only complete configured Profiles are eligible. Workflow selects the Profile
at the requested level, breaking same-level ties by Profile name in ascending
order. If that level is unavailable, it selects the next higher configured
level. If there is no higher level, it retains the previous selection and
records that escalation was unavailable. An unresolved, unaccepted Agent round
raises the requested level for the next dispatch. The selected Profile,
reasoning intensity, and any adjustment are recorded with usage evidence.

## Task budgets and approval

Token and monetary limits belong to an individual Schema 10 Task, not to the
global `[ai]` configuration or a Profile. Configure a positive Token limit and
at least one positive monetary limit keyed by currency. Monetary totals and
limits are kept separately for each currency; amounts are not converted.
Existing Tasks do not receive an inferred budget merely by being read.

Set the initial budget explicitly after the Task has entered Schema 10:

```text
aiw wf budget <task-id> configure --tokens 100000 --cost USD=25
```

Multiple currencies may be configured by repeating `--cost`. This operation is
allowed only once for a Task and requires both a positive Token limit and at
least one positive monetary limit. Inspect the result with `aiw wf status
<task-id>` or the bounded `aiw wf usage <task-id>` report.

Workflow aggregates known Provider-reported total Tokens and monetary amounts
over every WorkItem and Attempt in the Task. Reaching either configured limit
opens a human authorization gate and pauses new budget-consuming dispatches.
Generic Gate resolution cannot approve a budget increase. A human approval
without replacement limits raises the Token limit and each configured currency
limit by 30 percent. An explicit replacement must increase the Token limit and
every existing currency limit. Further breaches require further decisions;
approvals preserve the old and new limits, actor, time, reason, and source usage
digest. A human may instead terminate at the gate, which records the decision
and leaves the Task `BLOCKED`.

When authorization is pending, approve the default simultaneous 30 percent
increase with:

```text
aiw wf budget <task-id> approve --reason "approved for the next iteration"
```

The actor is the current OS user unless `--by <actor>` is supplied. To choose
explicit new limits, provide `--tokens` and repeat `--cost`; every existing
Token and currency dimension must increase. To stop instead:

```text
aiw wf budget <task-id> terminate --reason "stop at the approved limit"
```

## Missing or partial Provider usage

`usage_unknown` means the Provider did not supply a usable value for one or
more usage fields. Each field retains its own `known`, `unknown`, or `invalid`
state. Missing Token values and monetary cost are not replaced with zero or
estimated locally. Total Tokens are not derived from input and output counts.
Monetary cost is known only when the Provider supplies both an amount and a
currency.

Unknown usage does not stop execution by itself. Known Token usage still counts
against the Token limit even when monetary cost is unknown. Only known monetary
amounts count toward their currency limit; unknown amounts remain visible in
the usage report and do not establish that a monetary limit is satisfied.
Usage reports separate known totals from unknown counts and do not expose raw
Provider response evidence.

## Schema 10 migration and recovery

Schema 10 is entered only through the existing explicit managed migration
after its service and activation evidence gates pass. Reading a Schema 9 Task
does not migrate it, create a budget, rewrite its state, or infer historical
usage. Eligible migration preserves the original state bytes as its source
artifact, retains WorkItems, Attempts, and the existing event sequence, then
appends the migration event through the ordered Task event path. Historical
usage remains unknown. The normalized usage ledger and budget projection stay
in the existing versioned Workflow protocol state; migration does not create a
second Task store or event journal.

Migration is rejected while a legacy Attempt, write lease, prepared request, or
pending event is unresolved. Reconcile or finish that work through the normal
Workflow recovery path first; do not discard an in-flight record to make the
migration eligible. Existing Task event bytes and order are preserved, and
Schema 10 events are appended only after migration.

Session usage is additive. Existing Session schema 1 records and turns without
usage remain readable without migration or read-side rewrites. Their missing
usage is reported as unknown. Existing output, stderr, event-output, lifecycle,
and saved Provider/model settings keep their existing meaning. New turn usage
is persisted beside the turn output; the Workflow ledger remains authoritative
for Task totals and budget decisions.

After a crash, Workflow reconciles usage from the frozen Task, WorkItem,
Attempt, Session, and turn identity. The Session turn result and its usage
evidence are committed before the corresponding Task accounting projection is
advanced. Recovery can therefore append or replay the projection, while the
identity-based idempotency check prevents the same Provider call from being
counted twice. Reusing that identity with different evidence is rejected and
requires reconciliation rather than a replacement call.

Existing Provider adapters remain usable. An adapter that does not return a
usage field records that field as unknown; malformed returned values are
invalid. Token counts, cost, and currency come only from returned Provider
evidence. Cost is known only when both amount and currency are returned, and
AIW does not estimate prices or convert currencies. Raw Provider evidence is
bounded diagnostic data: it is sanitized, excludes credentials, full prompts,
and full output, and is retained for 90 days by default. Expiry removes only
the raw fragment; the digest, normalized usage, timestamps, availability, and
accounting history remain with the Task archive.
