# Engineering Review

> Mirror of `SKILL.md` Output Format. Copy this file, fill each section, then
> run `python scripts/validate_eng_review.py <this-file>` to check structural
> coverage. `--strict` also enforces section order.

## Status
READY / INCOMPLETE / HIGH RISK

<!-- Two axes are mandatory (see SKILL.md "Status Model"):
     - Review Status (this line): READY | INCOMPLETE | HIGH RISK
     - Release Readiness (in section 12): GO | GO WITH RISK | NO GO | NOT YET REVIEWED
-->

## 1. Context

<!-- What is being built and why. One paragraph. Link business decision (from
     business-review) and metric registry (from metrics-review). -->

## 2. System Boundary

| In Scope | Out of Scope | External Dependency | Owner |
|---|---|---|---|
|          |              |                     |       |

## 3. Module Responsibilities

| Module | Responsibility | Owner | Upstream | Downstream | Backward Compatibility Risk |
|---|---|---|---|---|---|
|        |                |       |          |            |                             |

## 4. Data Contracts

| Producer | Consumer | Contract (schema / API / topic) | Version | Compatibility Risk | Owner |
|---|---|---|---|---|---|
|          |          |                                 |         |                    |       |

## 5. Data Flow

```text
Source system
  -> ingestion / ETL / stream processor
  -> warehouse / mart / operational store
  -> service API
  -> admin UI / dashboard / report
```

### 5.1 Data Flow Table

| Step | Input | Processing | Output | Owner | Failure Mode | Validation |
|---|---|---|---|---|---|---|
|      |       |            |        |       |              |            |

### 5.2 Data Freshness

| Dataset | Expected Refresh | SLA | Alert Threshold |
|---|---|---|---|
|         |                  |     |                 |

### 5.3 Data Quality

| Check | Rule | Severity | Owner |
|---|---|---|---|
|       |      |          |       |

## 6. Permissions

### 6.1 Roles

<!-- List every role that can reach any surface built here. Include default
     access for new users. -->

### 6.2 Permission Matrix

| Role | View | Export | Edit | Approve | Field-level Restriction | Data-scope Restriction |
|---|---|---|---|---|---|---|
|      |      |        |      |         |                         |                        |

### 6.3 Field-level Restrictions

<!-- Which columns are masked / redacted / hashed for which role. -->

### 6.4 Data-scope Rules

<!-- Which rows a role may see: by customer, by desk, by region, by legal entity. -->

## 7. Audit Requirements

### 7.1 Audited Actions

| Action | Actor | Timestamp | Target | Before | After | Reason | Trace / Request Id | Queryable | Retention |
|---|---|---|---|---|---|---|---|---|---|
|        |       |           |        |        |       |        |                    |           |           |

### 7.2 Retention Policy

<!-- Storage duration by action class; regulatory driver if any. -->

### 7.3 Audit Query Requirements

<!-- Who queries the audit log, how fast, with what filters. -->

## 8. Failure Modes

| Failure Mode | Impact | Detection | Mitigation | Owner |
|---|---|---|---|---|
| Data delay              |  |  |  |  |
| Duplicate message       |  |  |  |  |
| Missing event           |  |  |  |  |
| Partial backfill        |  |  |  |  |
| Rollback failure        |  |  |  |  |
| Permission misconfig    |  |  |  |  |
| Stale dashboard         |  |  |  |  |
| Currency / precision    |  |  |  |  |
| Cut-off boundary        |  |  |  |  |

## 9. Observability

| Signal | Tool | Threshold | Alert Channel | Owner |
|---|---|---|---|---|
| Logs           |  |  |  |  |
| Metrics        |  |  |  |  |
| Alerts         |  |  |  |  |
| Tracing        |  |  |  |  |
| Freshness      |  |  |  |  |
| Data quality   |  |  |  |  |

## 10. Testing Strategy

| Layer | Scope | Tool | Coverage Target | Owner |
|---|---|---|---|---|
| Unit                       |  |  |  |  |
| Integration                |  |  |  |  |
| E2E                        |  |  |  |  |
| Data validation            |  |  |  |  |
| Permission tests           |  |  |  |  |
| Audit log tests            |  |  |  |  |
| Migration + rollback tests |  |  |  |  |

## 11. Risks

<!-- Free-form list of engineering risks that do not fit the failure-modes
     table (people, process, third-party, licensing, capacity). -->

## 12. Release Readiness Impact

- Schema or migration risk:
- Data impact:
- Permission impact:
- Audit impact:
- Rollback considerations:
- Monitoring required before release:
- Handoff to `release-review`: GO / GO WITH RISK / NO GO / NOT YET REVIEWED

## 13. Required Decisions Before Implementation

<!-- One bullet per open decision. Include: who decides, by when, what blocks
     if unresolved. Every INCOMPLETE / HIGH RISK trigger must appear here. -->