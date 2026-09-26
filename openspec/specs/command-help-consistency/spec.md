# command-help-consistency Specification

## Purpose
TBD - created by archiving change command-help-consistency. Update Purpose after archive.
## Requirements
### Requirement: CLI help reflects the public command surface

AIW MUST expose help text whose command names, aliases, subcommand forms, argument shapes, and documented options match the commands accepted by the source-code dispatch and validation paths.

#### Scenario: Workflow help lists every callable operation

- **WHEN** a user requests workflow help through `aiw wf`
- **THEN** the help lists every publicly dispatchable workflow operation, including planning, execution, recovery, delivery, focused-test, routing, reporting, and metadata-repair operations

#### Scenario: Workflow option documentation matches parsing

- **WHEN** a workflow command accepts command-line options
- **THEN** help documents the supported options and their constraints, including execution, primary-workspace, provider/model, and dry-run options where applicable

#### Scenario: Task and top-level help remain consistent

- **WHEN** a user requests top-level or Task help
- **THEN** the listed commands and example invocations refer only to supported commands and use the same workflow entry points and naming as the dispatcher

### Requirement: README and shell completion match the CLI surface

AIW MUST keep README command summaries/examples and shell completion command lists aligned with the source-code command surface and the canonical help output.

#### Scenario: README workflow documentation is complete

- **WHEN** a user follows the workflow command summary or examples in README
- **THEN** each referenced command and option is accepted by the CLI, and the summary does not omit a publicly callable workflow operation that is required for command discovery

#### Scenario: Completion exposes documented workflow commands

- **WHEN** a user requests completion after `wf`
- **THEN** completion offers the same public workflow operation names represented in help, without stale or invented names

#### Scenario: Consistency regression is detected

- **WHEN** a public command or supported option is added or changed without updating the related help, completion, or README inventory
- **THEN** the repository's consistency check fails with an actionable indication of the missing or mismatched surface

