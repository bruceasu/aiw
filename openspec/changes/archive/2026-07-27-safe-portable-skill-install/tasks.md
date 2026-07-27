## 1. Public CLI Foundation

- [x] 1.1 Add a failing CLI test for top-level HELP and canonical Skill listing
- [x] 1.2 Add the `aiw skills` plugin entry point and minimum list behavior
- [x] 1.3 Add invalid metadata listing coverage and diagnostics

## 2. Safe Single-Skill Installation

- [x] 2.1 Add a failing CLI test for default project installation
- [x] 2.2 Implement metadata and filesystem-entry validation before writes
- [x] 2.3 Implement dry-run with zero filesystem side effects
- [x] 2.4 Implement staging, deterministic directory hashing, and atomic publish
- [x] 2.5 Implement atomic managed manifest writes and rollback on manifest failure
- [x] 2.6 Protect unmanaged destinations and make identical managed reinstalls idempotent

## 3. Script-Stable Results

- [x] 3.1 Add JSON success and operational error behavior through the CLI seam
- [x] 3.2 Document commands, constraints, examples, and manifest behavior in Easy English

## 4. Verification

- [x] 4.1 Run focused CLI tests and Python compile checks
- [x] 4.2 Run the complete plugin and repository test suites
- [x] 4.3 Validate the OpenSpec change strictly and record verification evidence
- [x] 4.4 Run two-axis standards/spec review and resolve actionable findings

## 5. Canonical Source Layout

- [x] 5.1 Add a failing packaged-layout CLI test for root `skills`
- [x] 5.2 Move canonical Skill sources from `program/skills` to root `skills`
- [x] 5.3 Update plugin resolution, documentation, and release copy scripts
- [x] 5.4 Run complete verification and confirm no active `program/skills` references remain
- [x] 5.5 Review and commit the canonical source layout migration

## Verification

- Focused `aiw skills` CLI tests: 25 total, 22 passed, 3 skipped because the
  Windows account cannot create file or directory symlinks.
- Python compile checks passed for the plugin and focused test module.
- Existing Python suites passed: `program/aiw-flow` 51,
  `plugins/aiw-flow` 57, `plugins/aiw-git` 11, finance Skill validators 65.
- `go test ./...`, `go vet ./...`, and `go build ./...` passed.
- Real `go run . skills list --json` plugin dispatch passed with non-ASCII Skill
  metadata after the Windows UTF-8 regression fix.
- `openspec validate safe-portable-skill-install --strict` passed.
- Two-axis Standards and Spec review passed after resolving metadata parsing,
  symlink, unmanaged-target, fixture-duplication, and staging validation
  findings. No blocking finding remains.
- Packaged-layout CLI coverage passed with canonical content at root `skills/`.
- Windows release seam coverage passed for full and selected plugin installs;
  `build.bat` and `install-plugin.bat` also propagate Skill copy failures.
- Canonical Skill and finance validator paths now execute from root `skills/`;
  no active code or user documentation references `program/skills`.
- Final Standards and Spec review passed after aligning release fixtures with
  real documentation assets and making optional per-plugin docs explicit.
- %% The repository does not provide `ruff` or `mypy`; no dependency was added.
  Syntax coverage uses `py_compile`, with behavior covered through the public
  CLI seam.
