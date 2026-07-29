# OpenSpec-lite TOML Host Mapping

Use this mapping when the host repository uses the **OpenSpec-lite TOML** workflow, where changes live in `openspec/changes/<change-id>/` and long-lived capability specs live in `openspec/specs/<capability>/`.

## File Mapping

| Logical Profile File | OpenSpec-lite On-Disk Location |
|---|---|
| `requirements.md` | `openspec/changes/<change-id>/task.md` (or the `[requirements]` block in `task.toml`) |
| `design.md` | `openspec/changes/<change-id>/design.md` |
| `tasks.md` | `openspec/changes/<change-id>/tasks.md` |
| `metrics.md` | `openspec/specs/<capability>/metrics.md` |
| `permissions.md` | `openspec/specs/<capability>/permissions.md` |
| `audit.md` | `openspec/specs/<capability>/audit.md` |
| `release.md` | `openspec/changes/<change-id>/release.md` |

## Rules

- **One change directory per change-id.** Do not mix multiple changes in one folder.
- **Governance docs live under `openspec/specs/<capability>/`** and are updated in place, not per-change.
- **`task.toml` is the source of truth** for status, title, and links between artifacts. Keep the human-readable `task.md` / `tasks.md` in sync with the TOML block titles.
- **English titles/keywords, localized body.** Titles and section headings in `task.toml`, `task.md`, and `tasks.md` stay in English so tooling can parse them; body content follows the user's language.
- **Section coverage is checked in-place.** The validator does not re-resolve paths — run it once against the logical profile folder (e.g., a temp staging dir) or against the capability specs dir for governance docs.
- **`%%` markers for uncertainties** — use inline `%%` notes so open questions are searchable and never silently accepted.

## Change-id Naming

- Lowercase, hyphen-separated, verb-first: `add-abnormal-fund-adjust`, `fix-fx-rounding`, `remove-legacy-report-x`.
- No dates in the id (the folder mtime + `task.toml` timestamps are the audit trail).
- Match the id in `task.toml` `id = "..."` field.

## Example

For a change with id `add-abnormal-fund-adjust`, capability `risk-ops`:

```text
openspec/
├── changes/
│   └── add-abnormal-fund-adjust/
│       ├── task.toml           # source of truth: id, title, status, owner, links
│       ├── task.md             # human-readable → profile requirements.md
│       ├── tasks.md            # ordered task list → profile tasks.md
│       ├── design.md           # → profile design.md
│       └── release.md          # → profile release.md
└── specs/
    └── risk-ops/
        ├── metrics.md          # long-lived, updated by this change
        ├── permissions.md
        └── audit.md
```

## Validator Notes

- The bundled validator (`scripts/validate_openspec_finance.py`) validates one folder at a time and expects the seven logical files by name.
- To validate an OpenSpec-lite change end-to-end, stage the mapped files into a temp folder or run the validator twice: once against the change folder (for `requirements.md` / `design.md` / `tasks.md` / `release.md`, renamed as needed) and once against the capability spec folder (for `metrics.md` / `permissions.md` / `audit.md`).
- A future iteration may add a `--host openspec-lite-toml --change-id <id> --capability <name>` mode that reads directly from the on-disk layout.