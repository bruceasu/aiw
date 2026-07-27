## Why

AIW currently exposes two different skill installation paths:

- `aiw-skills`, which provides managed installation for canonical Skills
- `aiw-install-skill`, which can import folders, zip files, and bundles but
  does not track ownership or installed state

That split makes the installation story unclear. If a Skill is being installed
into a workspace, it should be managed. Otherwise the operation is closer to a
copy or unpack command than an install command.

## What Changes

- Deprecate and remove `aiw-install-skill` as a separate plugin entry point.
- Extend `aiw-skills install` so it can accept either:
  - a canonical Skill name from the repository `skills/` collection, or
  - a local path to a Skill directory, zip file, or bundle directory
- Keep all installed Skills under managed ownership with digest and manifest
  tracking.
- Preserve safe install behavior: staging, atomic publish, unmanaged target
  protection, and idempotent reinstall of identical managed Skills.
- Keep the user-facing installation vocabulary to a single verb: `install`.

## Capabilities

### New Capabilities

- `managed-skill-source-unification`: Install canonical Skills and external
  bundles through the same managed `aiw skills install` command.

### Modified Capabilities

- `aiw-skills`: Expand `install` to handle path-based sources in addition to
  canonical Skill names.

### Removed Capabilities

- `aiw-install-skill`: Separate untracked import/install entry point.

## Impact

- Affected Python plugin behavior in `plugins/aiw-skills`.
- Affected plugin packaging and release copy scripts.
- Affected README and usage documentation.
- Affected tests for install, bundle parsing, deprecation, and managed
  ownership.
- No new third-party dependencies are required.
