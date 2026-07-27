# Value Matrix

> Detailed rules that back the six-dimension Value Matrix in `SKILL.md`
> section 2. Use this file when a reviewer disputes a `HOLD` verdict or when
> classifying evidence as `strong` vs `weak`.

## The Six Dimensions

A request must hit **at least one** value dimension. If none, default to
`HOLD`. All evidence must reference a named person, incident id, metric
name, or regulatory citation ── never anonymous sentiment.

| Value Dimension | Strong Evidence | Weak Evidence | Review Question |
|---|---|---|---|
| Increase revenue                        | Direct conversion, retention, trading volume, deposit, AUM, or paid-product impact tied to a named funnel step or SKU. | Generic growth claim, "will help our numbers". | Which revenue path changes? By how much per week / month? |
| Reduce risk                             | Prevents a specific financial loss, compliance exposure, fraud pattern, operational error class, or unauthorized-access path. Cites incident id or risk register entry. | Vague safety improvement, "makes things safer". | What risk event becomes less likely or less severe? Cite incident or risk register. |
| Reduce manual labor cost                | Replaces a repeated manual workflow with measurable frequency and duration (× per week × minutes per case × number of staff). | One-off convenience, "would be nice". | How many hours per month are saved, for which named team? |
| Improve decision efficiency             | Speeds up a specific operational decision from N minutes / hours to M, or improves decision accuracy measurable against a ground truth. | Passive information display, "management wants to see". | Which decision becomes faster or better? By whom? Measured how? |
| Satisfy regulatory / audit / compliance | Tied to an explicit rule id, audit finding id, or mandatory control cited by name. Compliance owner named. | Nice-to-have traceability, "generally good for audit". | What rule / finding / control is addressed? Cite the source. |
| Improve customer experience             | Reduces a measurable customer delay, complaint volume, failure rate, or confusion class. Cites complaint id or SLA metric. | Internal-only preference, "our team prefers". | What customer pain is reduced? Measured how? |

## Strong vs Weak Evidence

Evidence is **strong** when it contains at least one of:

- A named person or team who feels the pain (not "everyone").
- An incident id, ticket id, or complaint id.
- A metric with a current value and a target value.
- A regulatory rule id, audit finding id, or compliance control id.

Evidence is **weak** when it is:

- Anonymous ("users say", "management wants").
- Feature-centric ("we should have a dashboard").
- Comparative without number ("faster", "safer", "better").
- Speculative ("could increase revenue").

A row with only weak evidence contributes **zero** value points and cannot
justify `APPROVE` by itself.

## Rating Decision Tree

```text
Any dimension has strong evidence?
├── yes → continue to cost / scope / owner gates
└── no  → any dimension has weak evidence?
         ├── yes → NEEDS_VALIDATION (lightweight measurement first)
         └── no  → HOLD (no value dimension met)
```

## Common Anti-Patterns

- Claiming "reduce risk" for a change that only *shows* risk (dashboards do
  not reduce risk unless they change a decision that changes behavior).
- Claiming "improve decision efficiency" without naming the decision.
- Claiming "satisfy regulatory" without citing a rule id ── this is the
  single most common false-positive.
- Claiming "reduce labor cost" from a one-off task rather than a repeating
  workflow.
- Bundling many small weak claims to look like a strong overall case ──
  reviewer must rate per dimension, not by count.

## Quantification Templates

- Labor cost saved per month = frequency × duration × loaded cost of person.
- Revenue impact per month = affected customer count × per-customer delta.
- Risk reduction = (incident probability × loss per incident) delta.
- Decision efficiency = (old decision time − new decision time) × decisions
  per period × count of deciders.
- CX improvement = complaint volume delta or SLA breach volume delta.

Use these when the reviewer needs to challenge a `strong` rating.