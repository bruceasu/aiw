### Requirement: Install canonical Skills into the project target
The system SHALL install a canonical Portable Skill into the current project's
`.agents/skills` directory.

#### Scenario: Install into an empty destination
- **WHEN** the user installs a valid canonical Skill into a project where the
  destination does not exist
- **THEN** the complete Skill directory is copied to the default project target
  and becomes discoverable there

#### Scenario: Preserve unmanaged destination content
- **WHEN** a same-name destination exists without a matching managed manifest
  entry
- **THEN** the command fails and preserves the destination unchanged

#### Scenario: Reinstall a managed Skill
- **WHEN** a same-name destination exists with a matching managed manifest
  entry
- **THEN** the command replaces the managed destination with the selected
  canonical source and refreshes the manifest entry

### Requirement: Adopt an existing installed Skill
The system SHALL record a valid existing Skill directory under
`.agents/skills` as AIW-managed without changing the directory content.

#### Scenario: Adopt a valid installed Skill
- **WHEN** the user adopts an installed Skill directory that contains valid
  Skill metadata and copyable filesystem entries
- **THEN** the system writes a managed manifest entry for that Skill and leaves
  the directory content unchanged

### Requirement: Discover installed Skills
The system SHALL provide a discovery command that reports installed Skills and
their managed status without modifying the filesystem.

#### Scenario: Report installed status
- **WHEN** the user runs the discovery command
- **THEN** the system reports each installed Skill as managed or unmanaged

### Requirement: Synchronize a managed Skill
The system SHALL provide a sync behavior that republishes the canonical source
for a managed Skill into its destination and updates the managed digest.

#### Scenario: Sync a managed Skill after source changes
- **WHEN** the canonical source for a managed Skill has changed since the last
  installation
- **THEN** the sync operation replaces the managed destination with the new
  source content and records the new digest

#### Scenario: Reject sync for unmanaged destinations
- **WHEN** a same-name destination exists without a matching managed manifest
  entry
- **THEN** the sync operation fails without modifying the destination
