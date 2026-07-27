## Why

AIW maintains canonical Skills under the repository-root `skills`, but users cannot safely
install one of those Skills through a consistent `aiw skills` command. The
existing installer defaults to Codex-specific locations and can replace whole
directories without managed ownership or integrity records.

## What Changes

- Add an `aiw skills` plugin with catalog and single-Skill install commands.
- Install Portable Skills to the current project's standard agent Skill
  location by default.
- Validate required Skill metadata before changing the destination.
- Stage and hash copied content before committing an installation.
- Record AIW-managed installation state in a versioned manifest.
- Protect unmanaged same-name destinations and make identical reinstallations
  idempotent.
- Provide dry-run, human-readable, and JSON command results with stable exit
  behavior.
- Package canonical Skills beside `program` and `plugins` so Skill content is
  not represented as executable program code.

## Capabilities

### New Capabilities

- `skill-installation`: Safely list and install one canonical Portable Skill
  into the current project with managed ownership and integrity metadata.

### Modified Capabilities

None.

## Impact

- Adds a new external AIW plugin command surface.
- Reads canonical Skill sources and writes only the selected project Skill plus
  the managed installation manifest.
- Adds CLI-level tests that execute through the same plugin boundary users
  invoke.
- Moves the canonical source from `program/skills` to the repository and
  release root `skills` directory.
- Does not change `aiw-flow`, existing Skill discovery, user-scope targets,
  Codex-specific targets, bundles, links, or capability integration.
- Adds no third-party dependency.
