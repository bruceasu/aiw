# Migration Risk Template

> Companion to `release-review-template.md` sections 3 (Schema and Migration)
> and 4 (Data Impact). Use this when the release includes DDL, index changes,
> backfill, or reconciliation.

## Migration Summary

- Tables affected:
- Columns affected:
- Indexes affected:
- Jobs or services affected:
- Dashboards or reports affected:
- Regulatory scope (does this touch reported figures?):

## Compatibility

| Check | Status | Notes |
|---|---|---|
| Backward compatible                          | pass / fail |  |
| Forward compatible                           | pass / fail |  |
| Old app can read new schema                  | pass / fail |  |
| New app can read old schema                  | pass / fail |  |
| Safe rollout order (expand → migrate → contract) | pass / fail |  |
| Long-running DDL avoided or online-safe      | pass / fail |  |

## Backfill

- Required: yes / no
- Idempotent: yes / no
- Idempotency key (column or hash):
- Batch size:
- Expected duration:
- Lock risk:
- Retry plan:
- Verification query:
- Verification tolerance (numeric):
- Owner of backfill execution:
- Sign-off of verification (business + engineering):

## Rollback

- Application rollback (steps + expected duration):
- Schema rollback (steps + irreversibility risk):
- Data rollback (steps + acceptable data loss window):
- Feature flag fallback (which flag, who flips):
- Manual correction plan (who runs, from where):
- Rollback verification (how do we prove rollback succeeded):
- Rollback owner:

## Reconciliation

| Dataset | Expected Count / Amount | Actual Count / Amount | Tolerance | Timing (T+0 / T+1) | Owner |
|---|---|---|---|---|---|
|         |                          |                        |           |                    |       |

<!-- Tolerance must be a number (e.g., "0 rows", "≤ 0.01 CNY", "≤ 5bps").
     "small" or "close enough" is not acceptable for financial data. -->

## Risk Register

| Risk | Likelihood | Impact | Detection | Mitigation | Owner | Blocker? |
|---|---|---|---|---|---|---|
|      |            |        |           |            |       |          |

## Sign-off

- Business owner (name + date):
- Engineering owner (name + date):
- Risk / compliance owner (name + date, when regulatory):