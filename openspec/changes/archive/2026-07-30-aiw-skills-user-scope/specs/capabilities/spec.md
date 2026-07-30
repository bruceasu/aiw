## ADDED Requirements

### Requirement: Select a Skill installation scope

The system SHALL support `project` and `user` scopes for Skill catalog
management, with `project` as the default. User scope SHALL resolve to the
current user's `.agents/skills` directory.

#### Scenario: Preserve the default project scope

- **WHEN** a Skill management command is run without `--scope`
- **THEN** it operates on the current project's `.agents/skills` directory

#### Scenario: Select user scope

- **WHEN** a supported command is run with `--scope user`
- **THEN** it operates on the current user's `.agents/skills` directory

### Requirement: Apply scope consistently and safely

The system SHALL apply the selected scope to install, discover, adopt, and
sync, and SHALL retain existing validation, staging, digest, manifest,
idempotency, and unmanaged-destination protections at both scopes.

#### Scenario: Preview a user installation

- **WHEN** `aiw skills install <skill> --scope user --dry-run` is run
- **THEN** it reports the user destination and makes no filesystem changes

#### Scenario: Maintain independent manifests

- **WHEN** the same Skill is installed in project and user scope
- **THEN** each destination root maintains its own managed manifest

#### Scenario: Exclude agent-specific directories

- **WHEN** a user-scope operation resolves its destination
- **THEN** it does not write to or require `~/.codex/skills` or
  `~/.copilot/skills`

