---
id: payment
type: spec
status: active
created: 2026-05-28
updated: 2026-05-28
---
# Payment Domain Spec
# Purpose
Handle payment processing and retry safely.
# Invariants
- payment must never duplicate charge
- retries must be idempotent
- retry count max = 5
# APIs
## POST /payment/retry
...
# Notes
- retry uses delayed queue
- poison messages go to DLQ
