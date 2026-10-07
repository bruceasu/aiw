# command-help-consistency Specification

## Purpose
TBD - created by archiving change command-help-consistency. Update Purpose after archive.
## Requirements
### Requirement: CLI help reflects the public command surface

AIW MUST expose help text whose command names, aliases, subcommand forms, argument shapes, and documented options match the commands accepted by the source-code dispatch and validation paths.

#### Scenario: FD worktree help matches the command surface

- **WHEN** a user requests `aiw git wt` help
- **THEN** help lists only supported FD operations, including add, status,
  commit, local-merge, and list
- **AND** examples describe conflict recovery in the FD worktree
- **AND** successful local delivery documents automatic worktree and branch cleanup

#### Scenario: Task and top-level help remain consistent

- **WHEN** a user requests top-level or Task help
- **THEN** the listed commands and example invocations refer only to supported commands and use the same workflow entry points and naming as the dispatcher

### Requirement: README and shell completion match the CLI surface

AIW MUST keep README command summaries/examples and shell completion command lists aligned with the source-code command surface and the canonical help output.

#### Scenario: README FD worktree documentation is complete

- **WHEN** a user follows README FD workflow commands
- **THEN** each command and option is supported and uses FD IDs
- **AND** the removed `aiw wf` and `aiw fd worktree` commands are not presented
  as available commands


#### Scenario: Consistency regression is detected

- **WHEN** a public command or supported option is added or changed without updating the related help, completion, or README inventory
- **THEN** the repository's consistency check fails with an actionable indication of the missing or mismatched surface

