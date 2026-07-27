# Permission and Audit Checklist

> Companion to `architecture-review-template.md` sections 6 (Permissions) and
> 7 (Audit Requirements). Use this to expand each row when the change touches
> customer data, money, limits, or workflow approvals.

## Design Principles

- **Default deny**: new roles and new users get no access unless explicitly granted.
- **Separation of duty**: view / export / edit / approve must be independently grantable; the same person should not both edit and approve a money-moving change.
- **Least privilege by scope**: prefer field-level and data-scope restrictions over全表 read.
- **Auditable by default**: any action that changes money, limits, fees, rates, permissions, or workflow status must be audited with before + after values.
- **Reversible where possible**: sensitive edits should support review + rollback within a bounded window.

## Permission Model

Define all permission layers, not only page access:

| Action | View | Export | Edit | Approve | Field-Level Restriction | Data-Scope Restriction |
|---|---|---|---|---|---|---|
|        |      |        |      |         |                         |                        |

## Role and Scope Review

For each role, define:

- Role purpose
- Allowed actions
- Denied actions
- Data scope (by customer / desk / region / legal entity)
- Field masking rules (which columns are masked, hashed, or hidden)
- Export limits (row cap, rate cap, watermarking)
- Approval requirements (who approves, four-eyes required?)
- Default access for new users of this role
- Onboarding + offboarding path (how permission is granted / revoked)

## Sensitive Actions

Treat these as sensitive by default:

- Customer asset view (balance, positions, holdings)
- Customer personal information view (name, id, contact, KYC docs)
- Export customer data (any format, any row count)
- Edit balances, limits, rates, fees, settlement, or risk flags
- Override workflow status (approve / reject / reopen)
- Approve financial or operational changes
- Change roles or permissions
- Trigger backfills, recalculations, or data corrections
- Read or modify audit log itself

## Audit Requirements

Each sensitive action must capture:

- Actor (user id + display name at time of action)
- Timestamp (UTC + local timezone)
- Action (canonical action name from a fixed enum)
- Target entity (id + type)
- Before value (JSON snapshot of touched fields)
- After value (JSON snapshot of touched fields)
- Reason (free text, required for edits and overrides)
- Request ID or trace ID
- Source IP / session / device context when available
- Retention policy (years, plus legal hold behavior)

## Audit Query Requirements

- Who can read audit records (usually risk / compliance / internal audit).
- Latency budget for audit queries (e.g., last 90 days interactive, > 90 days batch).
- Standard filters: actor, target customer, action name, time range, trace id.
- Export path for regulator requests (format, sign-off).

## Blockers

Mark the design `INCOMPLETE` or `HIGH RISK` when any of the following is true:

- Export permission is not separated from view permission.
- Edit permission is not separated from approval permission.
- Field-level or data-scope restrictions are undefined for sensitive data.
- Sensitive actions do not capture before **and** after values.
- Audit retention is undefined or shorter than the applicable regulatory floor.
- Permission defaults for new roles or users are unclear (must be default-deny).
- Audit log is writable by the same role that performs sensitive actions.
- Bulk export has no row cap, rate cap, or watermarking.