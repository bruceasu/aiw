## 1. Unified install input handling

- [x] 1.1 Extend `aiw-skills install` to accept canonical Skill names and
  existing path inputs.
- [x] 1.2 Reuse the existing managed install pipeline for directories, zip
  files, and bundle layouts.
- [x] 1.3 Add tests for canonical installs, single-skill path installs, zip
  installs, and bundle installs.

## 2. Managed ownership and safety

- [x] 2.1 Preserve staging, digest verification, and atomic publish for every
  installed Skill.
- [x] 2.2 Preserve unmanaged target protection and identical reinstall
  idempotence.
- [x] 2.3 Ensure multi-skill bundle installs write managed records for each
  skill.

## 3. Deprecation and cleanup

- [x] 3.1 Add a compatibility/deprecation path for `aiw-install-skill` or
  remove its references from documentation and release scripts.
- [x] 3.2 Update README and help text to describe the single `aiw skills
  install` entry point.
- [x] 3.3 Remove the old plugin after callers have migrated.

## 4. Verification

- [x] 4.1 Run focused plugin tests for managed installs and path sources.
- [x] 4.2 Run repository test and build verification.
- [x] 4.3 Record any remaining compatibility risks or follow-up questions in the
  change notes.
