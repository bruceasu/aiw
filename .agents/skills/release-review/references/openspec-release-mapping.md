# OpenSpec-lite TOML Mapping for Release Review

> Explains how to place the output of `release-review` inside a host
> repository that uses the OpenSpec-lite TOML workflow (see `AGENTS.md` in
> the tangram-trade-mt5 family of repos).

## Two Layers

1. **Logical profile** ── what this skill emits:
   - `release.md` (always, when running standalone)

2. **On-disk location** ── decided by the host repo's OpenSpec variant.

## Release Docs Are Change-Scoped, Not Long-Lived Specs

A release review is tied to a specific change and launch window. It belongs
under `openspec/changes/<change-id>/release.md`, **not** under
`openspec/specs/`.

Long-lived rules that survive many releases (e.g., "all exports must be
audited") belong in `openspec/specs/<capability>/audit.md` (owned by
`eng-review-finance`).

```text
openspec/
  changes/
    <change-id>/
      task.toml
      task.md
      tasks.md
      design.md          # from eng-review-finance
      release.md         # produced here by release-review
  specs/
    <capability>/
      metrics.md         # from metrics-review, referenced by release.md
      permissions.md     # from eng-review-finance, referenced by release.md
      audit.md           # from eng-review-finance, referenced by release.md
```

## Section-to-File Mapping

| Output Format section | Target location | Notes |
|---|---|---|
| `## Decision` | `release.md` head | Also written to `task.toml` `[release]` if the host uses TOML status tracking. |
| `## 1. Scope` | `release.md` | 1:1. |
| `## 2. Release Checklist` | `release.md` | 1:1. Canonical launch gate. |
| `## 3. Schema and Migration` | `release.md` | 1:1. Deep dive lives in `references/migration-risk-template.md`. |
| `## 4. Data Impact` | `release.md` | 1:1. |
| `## 5. Metrics and Reporting Impact` | `release.md` | Reference `metrics.md` entries by name; do not duplicate metric definitions. |
| `## 6. Permission Impact` | `release.md` | Reference `permissions.md` rows; do not duplicate the permission matrix. |
| `## 7. Audit Impact` | `release.md` | Reference `audit.md` rows; do not duplicate audit fields. |
| `## 8. Rollback Plan` | `release.md` | 1:1. |
| `## 9. Observability and Operations` | `release.md` | 1:1. |
| `## 10. Open Risks` | `release.md` + `openspec/changes/<change-id>/tasks.md` | Each `blocker` risk becomes a task; each non-blocker becomes a follow-up. |
| `## 11. Final Recommendation` | `release.md` tail | Includes named business + engineering risk owners. |

## Change-ID Naming

Follow the host repo convention. In the tangram-trade-mt5 family this is:

```text
<yyyy-mm-dd>-<kebab-topic>
```

Example: `2026-07-07-daily-withdraw-summary`.

## Cross-Reference Discipline

- `release.md` **references** but does not **redefine**:
  - Metric formulas (owned by `metrics.md`).
  - Permission matrix (owned by `permissions.md`).
  - Audit fields and retention (owned by `audit.md`).
  - Architecture and failure modes (owned by `design.md`).

- If a release review needs a metric / permission / audit rule that does not
  yet exist in a spec file, that is a `NO GO` ── route back to the owning
  skill (`metrics-review` or `eng-review-finance`) first.

## Language Rule

- File names, section headings, table headers, decision enums, severity
  enums: **English**.
- Body content: match user language (中文正文 default in this repo family).

## Validator Note

The included `scripts/validate_release_review.py` validates a **single**
`RELEASE_REVIEW.md` (or `release.md`) file against the Output Format
headings. It does not walk the whole `openspec/` tree.

## Backward Compatibility

- Never delete a heading from `release.md` even when the section is
  intentionally empty; leave `TODO` in place so the validator and downstream
  reviewers both see the gap.
- When a `GO WITH RISK` release later needs a follow-up review, append a
  `## 11.1 Follow-up Review <yyyy-mm-dd>` sub-section instead of rewriting
  section 11 ── the audit trail matters.
- Archived releases should stay in `openspec/changes/<change-id>/release.md`
  even after launch. Do not move them into a separate history directory.