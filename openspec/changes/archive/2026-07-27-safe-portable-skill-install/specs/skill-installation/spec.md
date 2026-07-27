## ADDED Requirements

### Requirement: List canonical Portable Skills
The system SHALL list valid canonical Portable Skills through the `aiw skills`
CLI without modifying the project.

#### Scenario: List valid Skills
- **WHEN** the canonical source contains Skills with valid single-line `name`
  and `description` frontmatter
- **THEN** the command displays each Skill name and description in stable name
  order

#### Scenario: Report an invalid candidate
- **WHEN** a canonical candidate is missing valid required frontmatter
- **THEN** the command reports the candidate as invalid without listing it as
  installable

### Requirement: Resolve the packaged canonical Skill root
The system SHALL use the `skills` directory beside `program` and `plugins` as
the default canonical Skill source in repository and release layouts.

#### Scenario: Run from a packaged plugin layout
- **WHEN** `aiw-skills.py` runs from
  `<install-root>/plugins/aiw-skills/aiw-skills.py` without a source override
- **THEN** the command reads canonical Skills from `<install-root>/skills`

### Requirement: Install one Skill to the default project target
The system SHALL copy one named canonical Portable Skill into the current
project's `.agents/skills` directory when no scope or target is specified.

#### Scenario: Install a valid Skill
- **WHEN** the user installs a valid canonical Skill into a project where its
  destination does not exist
- **THEN** the complete Skill directory is copied to the default project target
  and becomes discoverable there

#### Scenario: Reject an unknown Skill
- **WHEN** the user names a Skill that is not in the valid canonical catalog
- **THEN** the command fails without creating the destination root or manifest

### Requirement: Validate before writing
The system MUST validate required Skill metadata and all copyable source
entries before it changes the destination.

#### Scenario: Reject invalid metadata
- **WHEN** the selected canonical candidate has a missing or invalid `name` or
  `description`
- **THEN** the command fails without writing project state

#### Scenario: Reject unsupported filesystem entries
- **WHEN** the selected Skill contains a symlink or non-regular entry
- **THEN** the command fails before publishing an installed Skill

### Requirement: Preview installation without side effects
The system SHALL provide a dry-run that reports the resolved source,
destination, and intended action without changing the filesystem.

#### Scenario: Dry-run a new installation
- **WHEN** the user installs a valid Skill with `--dry-run`
- **THEN** the command reports that the Skill would be installed and creates no
  destination directory, staging directory, or manifest

### Requirement: Stage and verify copied content
The system MUST stage a new installation on the destination filesystem and
verify its deterministic SHA-256 digest before publishing it.

#### Scenario: Publish verified staged content
- **WHEN** the source and staged content have the same deterministic digest
- **THEN** the system atomically renames the staged directory to the final
  destination

#### Scenario: Source changes during installation
- **WHEN** the source and staged content digests differ
- **THEN** the system removes staging, reports failure, and does not publish the
  destination

### Requirement: Record managed ownership
The system SHALL atomically maintain a versioned JSON manifest that records
each installed Skill's source identity, source revision when available,
installation mode, and content digest.

#### Scenario: Record a new installation
- **WHEN** a Skill is successfully published
- **THEN** the managed manifest contains a copy-mode entry whose digest matches
  the installed content

#### Scenario: Manifest update fails
- **WHEN** the manifest cannot be committed after a new Skill is published
- **THEN** the system removes only that newly published Skill and reports
  failure

### Requirement: Protect existing destinations
The system MUST NOT overwrite or delete a same-name destination that lacks a
matching managed manifest entry.

#### Scenario: Unmanaged destination exists
- **WHEN** a same-name destination exists without matching AIW ownership
- **THEN** the command fails and preserves the destination unchanged

#### Scenario: Identical managed installation exists
- **WHEN** the destination, source, and managed manifest all have the same
  digest
- **THEN** the command succeeds as an idempotent no-op without rewriting files

### Requirement: Provide script-stable command results
The system SHALL provide human-readable results by default, one structured JSON
result when requested, and stable success or failure exit behavior.

#### Scenario: Request JSON success output
- **WHEN** a list, dry-run, new install, or idempotent install succeeds with
  `--json`
- **THEN** stdout contains one valid JSON object describing the action and
  stderr is empty

#### Scenario: Request JSON operational error output
- **WHEN** an install operation fails with `--json`
- **THEN** stdout contains one valid JSON error object and the process returns
  a nonzero operational exit code
