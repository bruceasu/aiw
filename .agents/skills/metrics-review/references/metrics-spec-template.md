# Metrics Spec

> Mirror of `SKILL.md` Output Format. Copy this file, fill each section, then
> run `python scripts/validate_metrics_review.py <this-file>` to check
> structural coverage. `--strict` also enforces section order.

## Status
READY / INCOMPLETE / CONFLICT

<!-- Multi-metric documents: Status here reflects the WORST status across the
     registry. Per-metric status lives in the Metric Registry table. -->

## 1. Context

<!-- Which admin platform / dashboard / report / decision flow consumes these
     metrics. One paragraph. -->

## 2. Metric Registry

| Metric | Business Definition | Formula | Unit | Time Dim | Refresh | Currency | Precision | Rounding | Cut-off | Business Owner | Technical Owner | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
|        |                     |         |      |          |         |          |           |          |         |                |                 |        |

<!-- Formula cells should use business terms, not SQL. Prefer:
       sum of successful settled withdrawals in the natural day
     over:
       SELECT SUM(amount) FROM settlement.withdraw_daily WHERE ... -->

## 3. Source Mapping

| Metric | System | Table | Field | Transformation | Filter (test / frozen / closed / internal / abnormal excluded?) | Notes |
|---|---|---|---|---|---|---|
|        |        |       |       |                |                                                                  |       |

## 4. Financial Correctness

| Metric | Currency + Base Reporting Currency | FX Source | FX Conversion Timestamp | Precision (decimals) | Rounding Mode | Timezone | Business Day Cut-off | Settlement / Value Date | Snapshot vs Transaction | Dedup Rule |
|---|---|---|---|---|---|---|---|---|---|---|
|        |                                     |           |                          |                       |                |          |                        |                          |                          |             |

## 5. Consistency Review

| Metric | Existing Similar Metric | Finance Definition | Risk Definition | Regulatory Definition | Management Reporting | Conflict? | Resolution Owner | Target Decision Date |
|---|---|---|---|---|---|---|---|---|
|        |                          |                     |                  |                        |                        |            |                    |                        |

## 6. Ownership

| Metric | Business Owner | Technical Owner | Escalation Path |
|---|---|---|---|
|        |                 |                  |                  |

## 7. Known Limitations

<!-- Free-form: known blind spots, precision loss, latency limits,
     late-arriving-data effects, coverage gaps. -->

## 8. Open Issues

| Issue | Impact | Owner | Required Decision | Target Date |
|---|---|---|---|---|
|       |        |       |                    |             |