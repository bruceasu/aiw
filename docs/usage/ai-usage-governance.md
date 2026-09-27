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

Input and output Token limits belong to an individual Schema 10 Task. Both
must be positive. Monetary limits are optional; credits are not required.
When a Provider supplies an actual amount and currency, AIW retains that
evidence separately without converting currencies. Existing Tasks do not
receive an inferred budget merely by being read. Historical total-Token and
monetary budgets remain readable and keep their original limits.

Set the initial budget explicitly after the Task has entered Schema 10:

```text
aiw wf budget <task-id> configure --input-tokens 100000 --output-tokens 20000
```

To reuse defaults across Tasks, put them in `aiw.toml` and explicitly snapshot
them with `aiw wf budget <task-id> configure` after migration:

```toml
[ai.usage_budget]
input_tokens = 100000
output_tokens = 20000
```

Reading a Task never installs these defaults. Changing `aiw.toml` later does
not rewrite a Task's persisted budget.

An optional `--cost USD=25` can be supplied when actual monetary usage is
available. This operation is allowed only once for a Task and requires both
positive Token limits. The legacy `--tokens` form remains available for old
total-Token budgets, but Codex CLI does not report a total-Token field and
should use the separate input/output form. Inspect the result with `aiw wf status
<task-id>` or the bounded `aiw wf usage <task-id>` report.

Workflow aggregates known Provider-reported input and output Tokens separately
over every WorkItem and Attempt in the Task. Reaching either configured limit
opens a human authorization gate and pauses new budget-consuming dispatches.
For Codex CLI, `cached_input_tokens` is recorded as a subset of input, and
`reasoning_output_tokens` as a subset of output; neither is added to its parent
count. The report shows both subsets independently. Codex's missing monetary
and total-Token fields remain unknown, not locally estimated.
Generic Gate resolution cannot approve a budget increase. A human approval
without replacement limits raises both Token limits and any configured currency
limits by 30 percent. An explicit replacement must increase both Token limits
and every existing currency limit. Further breaches require further decisions;
approvals preserve the old and new limits, actor, time, reason, and source usage
digest. A human may instead terminate at the gate, which records the decision
and leaves the Task `BLOCKED`.

Across all approvals, no configured dimension may rise above twice its initial
limit. This applies independently to input Tokens, output Tokens, and each
configured currency. An automatic 30 percent increase or explicit override
that crosses the ceiling is rejected without changing the pending Gate;
additional currency dimensions cannot be introduced by approval without an
initial limit. A human may terminate the Task at that point.

When authorization is pending, approve the default simultaneous 30 percent
increase with:

```text
aiw wf budget <task-id> approve --reason "approved for the next iteration"
```

The actor is the current OS user unless `--by <actor>` is supplied. To choose
explicit new limits, provide `--input-tokens` and `--output-tokens` and any
previously configured `--cost`; every existing dimension must increase. To stop instead:

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
against its input or output limit even when monetary cost is unknown. Only known monetary
amounts count toward their currency limit; unknown amounts remain visible in
the usage report and do not establish that a monetary limit is satisfied.
Usage reports separate known totals from unknown counts and do not expose raw
Provider response evidence.

%% TODO: The generic supervisor does not yet enable Schema 10 execution.

%% Verification (2026-09-28): Initial-limit approval ceilings now have focused
%% tests for exact-limit acceptance, over-limit rejection, and restart state.
%% Tests were not run under the default budget. One compile-only attempt failed
%% on a big.Rat argument type error; the error was corrected without rerunning
%% compilation, so this change has not passed compile-only validation.

%% Verification (2026-09-28): `python scripts/compile.py` exited 0 after the
%% Token-only budget, Codex subset extraction, and config-default changes.
%% Focused tests were added but not run under the repository's default test
%% budget. Profile effort/fast dispatch and the Schema 9/10 execution choice
%% remain separate work; these changes alone do not activate Schema 10.

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
