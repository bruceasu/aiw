# Metrics Governance Rules

> Detailed rules that back the gates in `SKILL.md`. Use this file when a
> reviewer disputes a `INCOMPLETE` or `CONFLICT` verdict.

## Non-Negotiable Rule

Do not only ask how to query the data. First define what the number means in
business terms. A metric without a one-sentence business definition is not a
metric ── it is a query result.

## Required Fields

Every metric row in the registry must define:

- Metric name
- Business definition (one sentence, business terms, no SQL)
- Formula (business terms, references to registry entries or ledger accounts)
- Unit (count / amount / ratio / duration / rate / bps)
- Time dimension (event time / trade date / value date / snapshot date / natural day)
- Refresh frequency (real-time / N minutes / hourly / T+0 daily / T+1 daily / monthly)
- Source system
- Source table
- Source fields
- Currency (if monetary) + base reporting currency
- Precision (decimals) and rounding mode
- Timezone and business day cut-off
- Business owner (accountable for definition and reconciliation)
- Technical owner (accountable for pipeline and freshness)
- Known limitations
- Status: `READY`, `INCOMPLETE`, or `CONFLICT`

## Financial Correctness

Check every monetary or time-sensitive metric against:

- **Currency**: which currency is stored? Which is displayed? Which is
  reported to finance?
- **FX source and conversion timestamp**: same rate for all rows? Which day's
  rate (T-1 close, intraday, event-time)?
- **Precision and rounding**: decimals stored vs decimals displayed vs
  decimals reported. Rounding mode (half-up / half-even / truncate).
- **Timezone and business day cut-off**: UTC vs local. Cut-off column (event
  time / trade date / booking date / value date). Late-arriving policy.
- **Settlement cycle or value date**: T+0 / T+1 / T+2 / T+n. Which cycle
  applies for which product?
- **Snapshot vs ledger transaction**: is the number a state at a point in time
  (snapshot) or a sum of movements (transactions)? These are not the same
  even when they should reconcile.
- **Deduplication rule**: primary key of a fact row. What defines a duplicate?
- **Inclusion / exclusion rules** for the following classes of accounts:
  test, frozen, closed, internal, corporate, market-maker, staff, abnormal
  (KYC failed / AML flagged / manual hold).

## Consistency Review

Before approving a metric, check whether it conflicts with:

- Existing metric registry definitions
- Finance reporting definitions (GL, PnL, balance sheet)
- Risk exposure definitions (VaR, position, limit usage)
- Management reporting definitions (weekly / monthly / quarterly deck)
- Regulatory reporting definitions (local regulator, group compliance)
- Existing dashboard labels or aliases (same label, different formula is the
  most common finance-vs-ops incident)

Every conflict must have a named resolution owner and a target decision date
before the metric may leave `CONFLICT` status.

## Status Rules

- Use `READY` only when the definition, formula, source mapping, financial
  correctness fields, both owners, and consistency review are all complete
  with no unresolved conflict.
- Use `INCOMPLETE` when source fields, owners, refresh frequency, time
  dimension, currency / precision / cut-off, or any financial correctness
  rule is missing.
- Use `CONFLICT` when finance, risk, regulatory, or management reporting
  definitions disagree and no owner has resolved the difference.

## Blockers

Block downstream engineering review (`eng-review-finance`) when:

- Metric formula is missing.
- Source table or field mapping is missing.
- Currency, precision, rounding, or FX source is undefined for monetary
  values.
- Time window or business day cut-off is undefined.
- Snapshot-vs-transaction semantics are undefined for a metric that could be
  interpreted either way (e.g., 客户余额 vs 客户资金流水汇总).
- Dedup rule is undefined.
- Inclusion / exclusion rule for test / frozen / closed / internal / abnormal
  accounts is undefined.
- Business owner or technical owner is missing.
- Finance or risk conflict remains unresolved.

Block downstream release review (`release-review`) when any of the above is
true and the release would surface the metric to users or downstream systems.

## Common Anti-Patterns

- Reusing a familiar label ("净流入", "活跃客户") without checking whether the
  existing formula matches.
- Silent currency conversion in the aggregation layer.
- Silent precision loss (e.g., aggregating `decimal(18,4)` into `double`).
- Timezone drift between event-time storage and dashboard-time display.
- Snapshot vs transaction confusion in reconciliation with finance.
- "Real-time" metric that in fact reflects T+1 due to upstream batch.