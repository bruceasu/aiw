# Release Gate Checklist

> Detailed rules backing the gates in `SKILL.md`. Use this file when a
> reviewer disputes a `NO GO` verdict or wants to justify `GO WITH RISK`.

## Mandatory Gates (any unresolved item = `NO GO`)

- Rollback plan is absent or untested (application + schema + data + feature flag must all be covered, even if some are "not applicable" with justification).
- Permission impact is unknown or default access for new roles / users is undefined.
- Audit impact is missing for sensitive actions (any action that changes money, limits, fees, permissions, workflow status, or exports customer data).
- Migration verification is undefined (no query, no reconciliation, no manual sign-off).
- Data reconciliation is undefined for financial or reporting data (tolerance must be a number, not "small").
- Monitoring is missing for the changed critical path (data freshness, data quality, permission errors, audit write failures).
- Owner is missing for release, rollback, or operational support.
- Any surfaced metric has status `INCOMPLETE` or `CONFLICT` from `metrics-review`.
- Any documented `HIGH RISK` or `INCOMPLETE` item from `eng-review-finance` remains unresolved.

## GO Criteria

- Scope is bounded and documented (services, jobs, tables, dashboards, APIs, admin screens).
- Schema changes are backward compatible or safely sequenced (expand → migrate → contract).
- Backfills are idempotent, restartable, and verifiable with a documented query.
- Historical and real-time data impacts are understood and bounded.
- Metrics changes are reconciled with finance and risk owners, with named sign-off.
- Permissions cover view / export / edit / approval / field-level / data-scope; default-deny for new roles / users.
- Audit captures actor, timestamp, action, before value, after value, reason, request id or trace id, source context, and retention policy.
- Rollback covers application, schema, data, and feature-flag fallback, with a verification step for each.
- Observability covers logs, metrics, alerts, tracing, data freshness, data quality, permission errors, and audit write failures.
- On-call rota is defined and the first responder has been briefed.

## GO WITH RISK Criteria

- Risks are enumerated and limited in blast radius (bounded users, bounded data, bounded time window).
- Recovery is manual but documented step-by-step and reachable within an agreed RTO.
- Monitoring is sufficient to detect failure quickly (within an agreed MTTD).
- Business owner explicitly accepts the residual risk (named person, not team).
- Engineering owner is on standby for release support (named person, not team).
- A next-review date is set to close the residual risk.

## NO GO Criteria

- Missing rollback for data or schema changes.
- New permissions are ambiguous, overly broad, or lack default-deny.
- Export or edit operations are unaudited or capture only after-value.
- Financial metrics can change without reconciliation, tolerance, or named finance sign-off.
- Migration can corrupt or lock critical production tables (long-running DDL without safe rollout).
- No verification exists for backfill correctness.
- No owner is assigned for rollback or post-release monitoring.
- Any dependency on a `NO GO` or `INCOMPLETE` upstream review (`metrics-review`, `eng-review-finance`).

## Common Anti-Patterns That Justify `NO GO`

- "Rollback = redeploy old version" without acknowledging the schema is now incompatible.
- "Backfill is idempotent" without an idempotency key or written test.
- Permission change routed only through code review, not through the permission matrix.
- Audit added only to the happy path; error path silently drops sensitive actions.
- Monitoring exists but no alert routes anywhere.
- Feature flag exists but has never been flipped in staging.