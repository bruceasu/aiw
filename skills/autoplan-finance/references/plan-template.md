# PLAN.md Template

Copy this template verbatim when starting a new plan. Do not delete section headings even if empty; leave `TODO` in place so gaps stay visible. All headings are English so the validator and downstream tooling can parse them.

## Status

APPROVE / REDUCE / HOLD / BLOCKED

## 1. Problem

> One paragraph. What is broken today? Who feels the pain? What is the cost of inaction?

## 2. Decision Flow

| Actor | Situation | Sees | Decides | Acts | Downstream Impact |
|---|---|---|---|---|---|

## 3. Stakeholders

| Role | Team | Responsibility | Consulted / Informed |
|---|---|---|---|

## 4. Business Value

> Revenue, risk reduction, labor saving, decision efficiency, regulatory need, or CX benefit. Quantify when possible.

## 5. Scope

### Must Have

### Should Have

### Nice To Have

### Explicit Non-Goals

## 6. Metrics

### 6.1 Metric Registry

| Metric | Definition | Formula | Unit | Time Dim | Refresh | Source (system.table.field) | Owner | Currency | Precision | Rounding | Cut-off | Consistency Check |
|---|---|---|---|---|---|---|---|---|---|---|---|---|

### 6.2 Source Mapping

> Upstream systems, tables, extraction cadence, latency SLA.

### 6.3 Financial Correctness Notes

> Rounding rules, FX conversion source & timing, T+0/T+1 boundaries, base currency.

### 6.4 Consistency Review

> Cross-check against existing finance/risk definitions. List any drift and its owner.

## 7. Data Definition

> Entities, fields, PII/sensitivity class, retention, lineage.

## 8. Architecture

> Components, data flow, sync vs async, storage, external dependencies. One diagram or bullet list is enough.

## 9. Permissions

### 9.1 Roles

| Role | Description |
|---|---|

### 9.2 Permission Matrix

| Role | View | Export | Edit | Approve | Field-level Restrictions | Data Scope |
|---|---|---|---|---|---|---|

### 9.3 Field-level Restrictions

> Masked / redacted fields per role. Compliance basis.

### 9.4 Data Scope Rules

> Row-level filters (own team only / own region / own customer segment / etc.).

## 10. Audit Requirements

### 10.1 Audited Actions

| Action | Actor | Timestamp | Target | Before | After | Reason | Trace/Request Id | Queryable | Retention |
|---|---|---|---|---|---|---|---|---|---|

### 10.2 Retention Policy

> Retention duration by action class, storage tier, deletion policy, legal hold rules.

### 10.3 Audit Query Requirements

> Who can query? By what dimensions? SLA for query response?

## 11. Release Readiness

GO / GO WITH RISK / NO GO / NOT YET REVIEWED

### 11.1 Release Checklist

- [ ] Schema migration reviewed and dry-run passed
- [ ] Data backfill / reconciliation script tested
- [ ] Metrics dashboards updated
- [ ] Permission matrix applied
- [ ] Audit hooks wired
- [ ] Rollback verified
- [ ] Monitoring & alerts configured
- [ ] Ops runbook updated

### 11.2 Rollback Plan

> Trigger conditions, rollback steps, data recovery approach, communication plan.

### 11.3 Open Release Risks

## 12. Testing Strategy

> Unit / integration / data-quality / permission / audit / regression / UAT scope.

## 13. Risks

| Risk | Likelihood | Impact | Mitigation | Owner |
|---|---|---|---|---|

## 14. Open Questions

- [ ] TODO

## 15. Milestones

| Milestone | Target Date | Owner | Exit Criteria |
|---|---|---|---|

## 16. Next Review

> Date, attendees, gating decision.