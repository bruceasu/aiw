# Work Management

OpenSpec is the canonical local source of truth for this repository.

Local work lives under:

- `openspec/changes/<change-id>/` for active changes and their task artifacts.
- `openspec/specs/` for stable capability requirements.
- `openspec/archive/` for archived changes.
- `.wt/<task-id>/` for AIW task worktrees when a task declares one.

The normal workflow does not ask users to choose a Local Markdown, GitHub, or
GitLab tracker. Existing `.scratch` content is legacy user data and is left
untouched; new canonical specifications and tickets are not written there.

GitHub and GitLab are optional external projections. They are used only after an
explicit request, and OpenSpec remains authoritative for requirements, task
progress, and local status.

For Skill behavior and context resolution, read `skills/work-management.md`.
