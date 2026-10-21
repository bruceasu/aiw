# FD templates and lifecycle

## FEATURE_INDEX.md

```markdown
# Feature Design Index

Each feature owns a folder with index.md, its FD design, reports/ and reviews/.
Browse docs/features/ and docs/features/archive/; run aiw fd list for status.
```

## FD file template

```markdown
# FD-XXX: Title

**Status:** Open
**Priority:** Low | Medium | High
**Effort:** Low (< 1 hour) | Medium (1-4 hours) | High (> 4 hours)
**Impact:** Brief description of what this enables

## Problem

What we're solving and why it matters.

## Solution

How to implement it. Be specific about approach.

## Files to Create/Modify

| File | Action | Purpose |
|------|--------|---------|
| `path/to/file` | CREATE / MODIFY | What and why |

## Verification

How to test that it works. Concrete steps.

## Related

- Links to related FDs, docs, or issues
```

## CHANGELOG.md skeleton

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project uses Semantic Versioning unless existing project conventions say otherwise.

## [Unreleased]
```

When closing a Complete FD, add one concise entry under Added, Changed, Fixed, or Removed. End the entry with `(FD-XXX)`.

## AGENTS.md section

Append or merge this section into the repository's `AGENTS.md`:

```markdown
---

## Feature Design (FD) Management

Features are tracked in `docs/features/`. Each FD owns `docs/features/FD-XXX/`, with `index.md`, its design, and evidence. `docs/features/FEATURE_INDEX.md` is fixed navigation; use `aiw fd list` for status.

### FD Lifecycle

| Stage | Description |
|-------|-------------|
| **Planned** | Identified but not yet designed |
| **Design** | Actively designing |
| **Open** | Designed and ready for implementation |
| **In Progress** | Currently being implemented |
| **Pending Verification** | Code complete, awaiting verification |
| **Complete** | Verified working and ready to archive |
| **Deferred** | Postponed or blocked |
| **Closed** | Will not be implemented |

### FD Operations

Use the `fd-workflow` skill for:

- `init` — initialize or repair FD tracking
- `new` — create a feature design
- `explore` — inspect project, FD history, and recent activity
- `deep` — four-angle deep analysis with claim verification
- `status` — reconcile and report FD status
- `verify` — review and verify implementation
- `close` — complete/close/defer and archive an FD

### Conventions

- FD files: `docs/features/FD-XXX/FD-XXX_TITLE.md`
- Default commit format: `FD-XXX: Brief description` unless repository history shows another convention
- Numbering: highest known FD number + 1
- Source of truth: FD file status; indexes contain navigation only
- Archive: Complete, Closed, and Deferred FDs move to `docs/features/archive/`

### Folder indexes

Each FD owns its navigation page. Keep status in the FD design and read it
through `aiw fd list`; never maintain a shared status table during execution.

### Inline annotations (`%%`)

Lines beginning with `%%` in FD/design documents are direct user annotations. Address every annotation; remove it after it is resolved. Preserve and report annotations that require an unresolved decision rather than guessing.
```
