# OpenSpec Finance Extension

Use this profile when generating or reviewing OpenSpec-style documentation for financial admin, operations, reporting, risk, finance, or analytics systems.

> **Layering note**: the file layout below is the **logical** profile. The **actual on-disk location** is decided by the host repository. For repositories using the OpenSpec-lite TOML workflow, see `openspec-lite-toml-mapping.md`.

## Required Files

```text
specs/
requirements.md
design.md
tasks.md
metrics.md
permissions.md
audit.md
release.md
```

## Finance Gates

Each gate lists the **required headings** the file must contain. Headings must appear at the specified level (`h1` / `h2`). Case-insensitive substring match; you may append qualifiers (e.g., `## Metric Registry (Golden Definitions)`).

### Decision Gate — `requirements.md`

`requirements.md` must explain who sees what, what judgment they make, what action follows, and what business loss or risk exists without the change.

Required headings:

- `# Requirements` (h1)
- `## Decision Flow` (h2) — contains a table with columns: Actor, Situation, Sees, Decides, Acts, Downstream Impact.

### Metrics Gate — `metrics.md`

`metrics.md` must define business definition, formula, unit, time dimension, refresh frequency, source system/table/field mapping, owner, currency, precision, rounding, cut-off, and consistency checks.

Required headings:

- `# Metrics` (h1)
- `## Metric Registry` (h2)
- `## Source Mapping` (h2)
- `## Financial Correctness` (h2)
- `## Consistency Review` (h2)

### Permission Gate — `permissions.md`

`permissions.md` must distinguish view, export, edit, approval, field-level, and data-scope permissions.

Required headings:

- `# Permissions` (h1)
- `## Roles` (h2)
- `## Permission Matrix` (h2)
- `## Field-level Restrictions` (h2)
- `## Data Scope Rules` (h2)

### Audit Gate — `audit.md`

`audit.md` must cover sensitive actions with actor, timestamp, action, target, before value, after value, reason, request id or trace id, queryability, and retention.

Required headings:

- `# Audit` (h1)
- `## Audited Actions` (h2)
- `## Retention Policy` (h2)
- `## Audit Query Requirements` (h2)

### Release Gate — `release.md`

`release.md` must cover schema, data, metrics, permissions, audit, rollback, observability, operational support, and final `GO / GO WITH RISK / NO GO` decision.

Required headings:

- `# Release` (h1)
- `## Decision` (h2)
- `## Release Checklist` (h2)
- `## Rollback Plan` (h2)
- `## Open Release Risks` (h2)

## Design & Tasks (Minimum)

- `design.md` must contain at least `# Design` (h1).
- `tasks.md` must contain at least `# Tasks` (h1).

## Validation

Use the bundled validator to check structural coverage:

```bash
python scripts/validate_openspec_finance.py <specs-dir>            # coverage only
python scripts/validate_openspec_finance.py <specs-dir> --strict    # also enforce heading order
python scripts/validate_openspec_finance.py <specs-dir> --json      # machine-readable report
```

The validator only checks presence and (optionally) order of the headings listed above. It does not validate business correctness — that requires human review by the corresponding sibling skill's owner.