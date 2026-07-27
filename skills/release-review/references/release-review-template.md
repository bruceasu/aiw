# Release Review

> Mirror of `SKILL.md` Output Format. Copy this file, fill each section, then
> run `python scripts/validate_release_review.py <this-file>` to check
> structural coverage. `--strict` also enforces section order.

## Decision
GO / GO WITH RISK / NO GO

<!-- Any failed gate in SKILL.md "Review Gates" forces NO GO. Document each
     gate outcome in the Release Checklist below. -->

## 1. Scope

- Feature or change summary:
- Affected services / jobs / tables / dashboards / APIs / admin screens:
- Customer-facing / internal-only / regulatory impact:
- Feature flag or staged rollout availability:
- Target launch window:

## 2. Release Checklist

| Area | Status | Evidence | Risk | Required Action | Owner |
|---|---|---|---|---|---|
| Schema and migration        |  |  |  |  |  |
| Backfill                    |  |  |  |  |  |
| Data reconciliation         |  |  |  |  |  |
| Metrics and reporting       |  |  |  |  |  |
| Permissions                 |  |  |  |  |  |
| Audit                       |  |  |  |  |  |
| Rollback                    |  |  |  |  |  |
| Observability               |  |  |  |  |  |
| On-call / ownership         |  |  |  |  |  |

<!-- Status enum: PASS / GO WITH RISK / FAIL / N/A -->

## 3. Schema and Migration

- DDL changes:
- Index changes:
- Migration lock risk:
- Backward compatibility (old app reads new schema):
- Forward compatibility (new app reads old schema):
- Migration verification query:

## 4. Data Impact

- Historical data impact:
- Real-time data impact:
- Recalculation / backfill need:
- Backfill idempotency:
- Data freshness expectation:
- Reconciliation plan + tolerance:

## 5. Metrics and Reporting Impact

- Changed metric definitions (link to `metrics-review` entries):
- Changed aggregation / dedup logic:
- Time-window / cut-off changes:
- Currency / precision / rounding / settlement changes:
- Finance / risk / management-reporting conflicts:

## 6. Permission Impact

- New / changed view permission:
- New / changed export permission:
- New / changed edit permission:
- New / changed approval permission:
- Field-level restriction changes:
- Data-scope restriction changes:
- Role migration / default access risk:

## 7. Audit Impact

- Sensitive actions introduced or changed:
- Actor / timestamp / action / target captured:
- Before + after values captured:
- Reason captured:
- Request ID or trace ID captured:
- Retention policy:
- Audit queryability (who queries, latency budget, filters):

## 8. Rollback Plan

- Application rollback:
- Schema rollback:
- Data rollback:
- Feature flag fallback:
- Manual recovery procedure:
- Rollback owner:
- Rollback verification (how do we prove rollback succeeded):

## 9. Observability and Operations

- Release health dashboard:
- Logs / metrics / alerts / tracing:
- Data freshness checks:
- Data quality checks:
- Permission error monitoring:
- Audit write failure monitoring:
- On-call rota + first responder:

## 10. Open Risks

| Risk | Severity | Owner | Mitigation | Release Blocker? |
|---|---|---|---|---|
|      |          |       |            |                  |

<!-- Severity enum: blocker / high / medium / low
     Release Blocker enum: yes / no -->

## 11. Final Recommendation

- Decision: GO / GO WITH RISK / NO GO
- Rationale:
- Named risk owner (business):
- Named risk owner (engineering):
- Next review date if `GO WITH RISK` or `NO GO`: