# Data Flow Template

> Companion to `architecture-review-template.md` section 5. Use this when the
> change is data-pipeline heavy (ingestion, ETL, warehouse, mart, reporting).

## Canonical Path

```text
Source system
  -> ingestion / ETL / stream processor
  -> warehouse / mart / operational store
  -> service API
  -> admin UI / dashboard / report
```

## Data Flow Table

| Step | Input | Processing | Output | Owner | Failure Mode | Validation |
|---|---|---|---|---|---|---|
|      |       |            |        |       |              |            |

## Data Freshness

| Dataset | Expected Refresh | SLA | Alert Threshold | Cut-off Semantics |
|---|---|---|---|---|
|         |                  |     |                 |                    |

<!-- Cut-off semantics example: "T+0 03:00 UTC+8 based on trade_date; late
     records are appended to next-day partition, dashboard uses trade_date." -->

## Data Quality

| Check | Rule | Severity | Owner |
|---|---|---|---|
| Row count vs source           |         | blocker  |       |
| Duplicate key                 |         | blocker  |       |
| Null / negative on amount     |         | blocker  |       |
| Currency present + valid      |         | blocker  |       |
| Precision (decimals) matches  |         | warning  |       |
| Timezone consistency          |         | warning  |       |
| Reference-data completeness   |         | warning  |       |

## Backfill and Reprocessing

| Scenario | Trigger | Idempotency Guarantee | Rollback Path | Owner |
|---|---|---|---|---|
| Historical backfill      |  |  |  |  |
| Late-arriving records    |  |  |  |  |
| Source correction        |  |  |  |  |
| Metric formula change    |  |  |  |  |
| Recalc after outage      |  |  |  |  |

## Cross-System Consistency

| Metric / Dataset | System A | System B | Reconciliation Rule | Owner | Alert Threshold |
|---|---|---|---|---|---|
|                  |          |          |                     |       |                 |

<!-- Reconciliation examples: 客户余额 vs 会计总账、订单表 vs 结算表、
     实时仪表盘 vs T+1 报表。差异容忍度必须写数字。 -->