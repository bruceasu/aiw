## 1. Design gates and contracts

- [x] 1.1 Resolve the Provider usage envelope with field-level `known`, `unknown`, and `invalid` states, Provider-only cost/currency evidence, and partial-field semantics; remove the corresponding `%% NEEDS_INPUT` note from `design.md`.
- [x] 1.2 Define the deterministic WorkItem difficulty escalation signal as an unresolved, unaccepted Agent round and update the Profile-level selection decision and acceptance scenarios.
- [x] 1.3 Confirm the normalized Schema 10 persistence and event compatibility contract for existing Tasks and Sessions.

## 2. Provider and Session evidence

- [x] 2.1 Add a versioned Provider usage envelope carrying Token, monetary cost, availability status, Provider, Model, timestamps, and raw-response evidence.
- [x] 2.2 Populate the envelope from Codex CLI, Copilot CLI, OpenAI SDK, and other supported Provider adapters using returned values only.
- [x] 2.3 Persist normalized usage evidence with each Session turn while preserving existing output, stderr, lifecycle, and one-call override behavior.
- [x] 2.4 Extend `[ai.profiles.<id>]` parsing with explicit level and reasoning-intensity settings without exposing credentials through Profile records.

## 3. Workflow accounting and budget control

- [x] 3.1 Add the Schema 10 Task usage ledger, cumulative budget projection, unknown-usage counters, and backward-compatible state loading.
- [x] 3.2 Record one idempotent usage event bound to Task, WorkItem, Attempt, Session turn, Provider, Model, Profile, difficulty, parameters, and outcome.
- [x] 3.3 Aggregate known Token and monetary values across the whole Task and preserve raw usage records without converting unknown values to zero.
- [x] 3.4 Open a human authorization gate when either known budget reaches its limit; implement explicit initial Task budget configuration, simultaneous default 30 percent increases, explicit overrides, repeated approvals, and audit history.
- [x] 3.5 Implement human termination as `BLOCKED` and preserve usage, budget, approval, and recovery history across restart and repeated recovery.
- [x] 3.6 Implement deterministic Profile selection by level and Profile name, next-higher-level fallback, reasoning-intensity changes, and difficulty audit records.
- [x] 3.7 Sanitize and bound raw Provider usage evidence, retain it for 90 days by default, and expire only the raw fragment while preserving its digest and normalized history.

## 4. Reporting and operator surfaces

- [x] 4.1 Add the read-only `aiw wf usage <task-id>` command with bounded WorkItem, Attempt, Provider, Profile, and RFC3339 time-range filters plus table and JSON formats.
- [x] 4.2 Report known Token totals, known monetary totals by currency, call counts, `usage_unknown` counts, budget approvals, difficulty changes, and outcomes separately through the Workflow facade.
- [x] 4.3 Keep `aiw wf status` limited to budget state, pending authorization, BLOCKED reason, selected Profile, and partial-usage diagnostics; do not expose raw Provider evidence in the normal report.

## 5. Focused verification

- [x] 5.1 Add Provider and Session tests for complete usage, partial usage, `usage_unknown`, raw evidence, and backward-compatible records.
- [x] 5.2 Add Workflow tests for Task-wide aggregation, idempotent replay, concurrent approval serialization, repeated 30 percent increases, override limits, and BLOCKED recovery.
- [x] 5.3 Add Profile and difficulty tests for explicit levels, same-level name ordering, next-higher fallback, reasoning-intensity changes, and audit records.
- [x] 5.4 Add report tests for every supported dimension and separate known/unknown totals.
- [x] 5.5 Compile the changed packages with the repository-authorized narrow compile command; run tests only after explicit authorization.
- [x] 5.6 Add focused tests for sanitized raw-evidence retention expiry and preservation of normalized history and evidence digests.

## 6. Documentation and migration

- [x] 6.1 Document the Profile level/reasoning-intensity configuration, Task Token/monetary budgets, approval behavior, and `usage_unknown` semantics.
- [x] 6.2 Document Schema 10 migration and recovery behavior for existing Tasks, Sessions, in-flight Attempts, and Provider adapters.
- [x] 6.3 Update TODO, Verification evidence, unresolved `%%` notes, and the implementation handoff without changing OpenSpec-owned checklist prose from Workflow synchronization.

## TODO

- [x] 7.1 Resolve all design-readiness gates before implementation handoff.
- [x] 7.2 Record each supported Provider adapter's Token and monetary-cost capability as conditional on returned evidence; retain `unknown` when cost or currency is absent and never add local pricing inference. The capability matrix is documented in `design.md` for Codex CLI, Copilot CLI, OpenAI Responses API, Gemini REST API, and OpenAI-compatible/Ollama/llama.cpp adapters.
- [x] 7.3 Complete the remaining usage-report aggregates and audit histories in item 4.2.
- [x] 7.4 Add focused tests for the implemented raw Provider evidence retention and expiry contract (checklist item 5.6).

## Verification

- Implementation evidence for WI 3.7: Provider usage evidence is reduced to
  allowlisted scalar usage fields and bounded to 4 KiB. Session turn usage
  records retain a SHA-256 digest and a default 90-day expiry; reading an
  expired turn removes only its raw fragment and persists the normalized
  fields and digest. Managed Workflow usage events store normalized evidence
  and the digest without duplicating the raw fragment. Added
  `TestReadExpiredTurnUsageRemovesOnlySanitizedRawEvidence` to verify allowlist
  sanitization, the default expiry, digesting the sanitized fragment, and
  removal of only raw evidence on read while normalized fields and the digest
  survive in the persisted sidecar. The test was added but not run; the
  supervisor owns compile-only validation.

- Identity confirmation for WI 8.2: the managed Task ID, the `id` value in
  `.ai/tasks/schema-10-ai-usage-governance/task.toml`, and the OpenSpec change
  directory are all `schema-10-ai-usage-governance`.

- Consistency review for WI 8.1: proposal capabilities, design decisions, and
  both delta specs agree on Provider-only usage, field-level availability,
  Task-wide budgets, deterministic Profile escalation, usage reporting, and
  backward-compatible Task/Session persistence. The review found that the
  90-day sanitized raw-evidence retention and expiry requirement was specified
  but had no implementation or test checklist item; items 3.7 and 5.6 now
  cover that work. TODO 7.4 tracks it as outstanding. This checklist update
  aligns the scope; it does not claim that retention behavior is implemented.
-  WI 5.5 compile-only validation passed in the isolated worktree with
    `python scripts/compile.py`; result was exit code 0. On Windows the script
    directs Go output to `NUL`, so the complete `go build ./cmd/aiw` performs
    compilation without retaining an executable. Tests were not run.
- Implementation evidence for WI 5.2: Windows Workflow tests cover Task-wide
  aggregation across WorkItems, idempotent usage-event replay, serialized
  concurrent default approvals and two consecutive 30 percent increases,
  rejection of an override that fails to raise every configured budget
  dimension, recording a human termination while authorization is pending,
  and recovery after reopening the Store while retaining usage and termination
  history. Tests were added but not run; the supervisor owns compile-only
  validation.
- Implementation evidence for WI 5.3: Profile parsing asserts explicit level
  and reasoning-intensity values. Workflow CLI tests cover deterministic
  same-level Profile-name ordering, selection of the next higher configured
  level, difficulty escalation after an ended unaccepted Agent round, changed
  reasoning intensity, previous/selected Profile audit provenance, and the
  no-higher-Profile fallback reason. Tests were added but not run; the
  supervisor owns compile-only validation.
- Implementation evidence for WI 5.4: `internal/workflow/usage_report_windows_test.go`
  covers WorkItem, Attempt, Provider, Profile, and inclusive RFC3339 time-range
  filtering. It verifies known Token totals and per-currency monetary totals
  separately from `usage_unknown` calls, outcome counts, and omission of raw
  Provider evidence from the report. Tests were added but not run; the supervisor
  owns compile-only validation.
- Implementation evidence for WI 5.1: Provider mapper tests cover complete
  Token and monetary usage, exact usage-object evidence without unrelated
  output, partial Token fields, amount without currency remaining unknown,
  malformed fields remaining invalid, and absent usage remaining unknown.
  Session tests cover persisting and reading the normalized envelope alongside
  existing final output, events, and stderr, plus a legacy turn without a
  usage sidecar returning unknown without rewriting the Session. Tests were
  added but not run; the supervisor owns compile-only validation.
- Implementation evidence for WI 4.1: `aiw wf usage <task-id>` parses the
  WorkItem, Attempt, Provider, Profile, RFC3339 time bounds, and table/JSON
  options, then calls the read-only Workflow usage-report facade. The report
  filters normalized call summaries and omits Provider raw evidence. Aggregate
  totals and audit histories were added in 4.2; runtime behavior and compilation
  were not run in this supervised turn.
- Implementation evidence for WI 4.2: the Workflow Store usage-report facade
  applies supported filters while aggregating only matching normalized usage
  events into known input/output/total Token totals, exact known monetary
  totals by currency, call and `usage_unknown` counts, and outcome counts. It
  separately returns time-bounded budget approvals and terminations plus
  per-call difficulty/Profile changes, without exposing Provider raw evidence.
  The table CLI renders aggregate and audit summaries, while JSON exposes the
  structured report. Tests and compile were not run; the supervisor owns
  compile-only validation.
- Implementation evidence for WI 4.3: `aiw wf status <task-id>` now renders a
  bounded operational view from the normalized Task usage ledger: budget
  limits and known usage, pending authorization, the most recently selected
  Profile, field-level partial-usage counters, and a blocked reason with
  recovery guidance when the Task is BLOCKED. A missing ledger is reported as
  historical usage unknown. The status surface does not serialize usage
  records or Provider evidence. Tests were not run; the supervisor owns
  compile-only validation.
- Implementation evidence for WI 7.2: the Provider capability matrix in
  `design.md` records the usage evidence inspected by each supported adapter.
  Codex CLI, Copilot CLI, OpenAI Responses, Gemini REST, and
  OpenAI-compatible/Ollama/llama.cpp adapters accept only returned Token and
  monetary fields. Monetary cost remains `unknown` unless both amount and
  currency are present; no adapter infers a price. The current Gemini adapter
  uses the REST API directly rather than an official SDK. No tests or compile
  were run for this documentation-only update; the supervisor owns
  compile-only validation.
- Implementation evidence for WI 6.1: added
  `docs/usage/ai-usage-governance.md` describing Profile levels and reasoning
  intensity, Task-scoped Token and per-currency monetary limits, human approval
  and termination behavior, and field-level `usage_unknown` semantics. Linked
  it from the execution and verification overview in `docs/auto-coding.md`.
  Static inspection confirmed the Workflow budget model, Profile selection,
  and that Schema 10 is not the default CLI path. No tests or compile command
  were run; the supervisor owns compile-only validation. Provider-specific
  capability notes are now recorded under TODO 7.2.
- Implementation evidence for WI 6.3: updated the TODO and Verification
  records, removed the resolved Provider capability `%% NEEDS_INPUT`, and
  recorded the Codex CLI, Copilot CLI, OpenAI Responses, Gemini REST, and
  OpenAI-compatible/Ollama/llama.cpp capability matrix in `design.md`.
  The implementation handoff is intentionally stored at the shared runtime
  path `.ai/tasks/schema-10-ai-usage-governance/artifacts/handoff.md`, because
  `RuntimeTaskDir` is rooted at the primary repository so isolated Agents and
  the supervisor share one lineage. The handoff is runtime-owned and is now
  explicitly read-only for Agents; lifecycle updates are recorded by the
  supervisor. No tests were run; compile-only validation is owned by the
  supervisor.
- Implementation evidence for WI 2.1: `internal/ai.Response` can carry a
  versioned Provider usage envelope with independent field states, Provider
  values, timestamps, and raw response evidence.
- Implementation evidence for WI 2.2: Codex CLI and Copilot CLI decode usage
  objects present in returned JSONL, OpenAI Responses maps its returned usage
  object, and OpenAI-compatible and Gemini adapters map their returned usage
  objects. The shared mapper preserves only the raw usage object, maps Token
  fields independently, and leaves cost unknown unless returned amount and
  currency are both present; no Token or cost values are inferred. Runtime
  adapter behavior remains unverified because tests were not run.
- Implementation evidence for WI 2.3: Session turn persistence writes the
  normalized Provider usage envelope to a per-turn `-usage.json` sidecar after
  preserving the existing final output, event output, and optional stderr
  files. Missing usage is materialized as field-level `unknown` with the
  effective one-call Provider and Model and response timestamps; it does not
  alter persisted Session Provider/model settings or lifecycle updates.
- Implementation evidence for WI 2.4: `internal/ai.LoadProfiles` reads optional
  numeric `level` and `reasoning_intensity` values from `[ai.profiles.<id>]`.
  Invalid configured levels return a profile-specific error; omitted values
  retain zero/empty defaults for compatibility. The returned `Profile` contains
  only name, Provider, Model, level, and reasoning intensity, while credentials
  and transport configuration remain in the separate `Config` path. The focused
  parsing test was updated but not run.
- Implementation evidence for WI 3.1: Schema 10 `ExecutionProtocol` now has an
  optional, versioned Task usage ledger with normalized record storage,
  cumulative known Token totals, known monetary totals by currency, call and
  `usage_unknown` counts, and per-field known/unknown/invalid counters. The
  protocol validator rejects unsupported ledger versions, negative or
  over-counted projections, malformed records, and invalid monetary totals.
  Older Schema 10 states without `usage_ledger` continue loading with a nil
  ledger; loading does not manufacture a zero balance or rewrite state. Event
  recording and projection updates remained for WI 3.2–3.3. Tests and the
  supervisor-owned compile were not run in that turn.
- Implementation evidence for WI 3.2: after validating the frozen managed
  Session and expected turn, the supervisor reads that turn's normalized usage
  sidecar (or represents a legacy missing sidecar as unknown) and commits one
  `protocol.usage.recorded` event plus an immutable ledger record through the
  existing conditional Task event path. The deterministic event ID binds
  Task, WorkItem, Attempt, Session, and turn; replay of identical evidence is
  a no-op, while reuse of that identity with different evidence is rejected.
  The record carries Provider, Model, Profile, level, reasoning intensity,
  non-secret parameter digest, outcome, and the normalized Provider envelope.
  Tests and the supervisor-owned compile were not run in this turn.
- Implementation evidence for WI 3.3: each newly recorded usage event updates
  the Task-wide projection in the same ordered Task event mutation. The
  projection independently sums Provider-reported input, output, and total
  Tokens, and accumulates known monetary amounts by the Provider-returned
  currency using exact rational arithmetic. It increments per-field
  known/unknown/invalid counters and a per-call `usage_unknown` count when any
  primary usage field or availability is not known. The immutable serialized
  event, including normalized and raw Provider usage evidence, remains in the
  ledger; unknown or invalid values never enter numeric totals or become zero.
  Tests were not run, and the supervisor owns compile-only validation.
- Implementation evidence for WI 3.4: the Schema 10 usage ledger stores
  explicit Task Token limits, monetary limits by currency, pending
  authorization, and approval history. Recording usage opens the existing
  authorization Gate when known total Tokens or a known per-currency amount
  reaches its limit; stage begin, preparation, and dispatch claim reject new
  calls while that Gate is open. Generic Gate resolution cannot bypass the
  approval path. Human approvals record actor, time, reason, previous and new
  limits, and the source usage digest. An approval without replacement limits
  raises Token and each monetary limit simultaneously by 30%; explicit limits
  must increase each existing dimension. Repeated breaches create new pending
  decisions without resetting usage or prior approvals. Tests were not run;
  the supervisor owns compile-only validation.
  The Workflow facade now exposes the missing operator seam: `aiw wf budget
  <task-id> configure --tokens <n> --cost <CURRENCY=AMOUNT> ...` explicitly
  installs the initial budget once; `approve` records the current OS user by
  default and applies either the simultaneous 30 percent increase or an
  explicit increasing replacement; `terminate` records the human decision and
  leaves the Task blocked. No budget is inferred from Profiles, Providers, or
  historical usage.
- Implementation evidence for WI 3.5: a human can terminate execution while
  Task usage-budget authorization is pending. Workflow records the actor, time,
  reason, and source-usage digest in the versioned usage ledger, persists the
  decision evidence through the existing protocol artifact path, and commits
  the typed execution Stop through the ordered Task event path. This Stop
  derives Task execution as `BLOCKED`; usage, limits, prior approvals, pending
  authorization, and stage recovery history remain in the durable protocol.
  Existing explicit Stop recovery can resume execution without deleting the
  termination audit record. Tests were not run; the supervisor owns
  compile-only validation.
- Implementation evidence for WI 3.6: supervised dispatch counts prior ended,
  unaccepted Agent Attempts for the same unresolved WorkItem as its requested
  level increase. Configured Profiles are selected by the smallest level at or
  above that level, with Profile name as the stable tie-breaker; if no higher
  Profile exists, the last selection is retained and the audit records
  `no_higher_profile_available`. Each usage event now retains the requested
  level, previous and selected Profile/provider/model/level/reasoning intensity,
  and adjustment reason. Runtime Profile selection and report rendering remain
  unverified because tests and the supervisor-owned compile were not run. The
  compiler repair for WI-0013 added the missing standard-library `errors`
  import used by the explicit usage-budget Gate guard; the supervisor owns the
  follow-up compile.
- [x] 8.1 Confirm proposal capabilities, design decisions, delta specs, and checklist scope are consistent.
- [x] 8.2 Confirm Task ID, `task.toml.id`, and OpenSpec change directory are identical.
- [x] 8.3 Confirm Workflow Core mapping is synchronized after this checklist is accepted and that no Attempt, lease, or worktree was created during specification.
