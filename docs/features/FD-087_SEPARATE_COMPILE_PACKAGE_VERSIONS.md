# FD-087: Separate compile and package versions

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem and decision

The user needs compiled snapshot artifacts packaged as a separate release version.
Add `build <target> --compile-version <source> --version <package>`.
Select the existing validated covering compile record by exact source version and
skip compilation. Missing, invalidated or changed source output fails before any
packaging/staging/manifest writes; never silently compile the package version.

--version and AIW_VERSION select only the package identity when --compile-version
is explicit. With neither package version setting, use the selected compile
version. Without --compile-version, preserve FD-086 default reuse/fresh compilation
rules. Publish commands remain fresh builds and do not accept --compile-version.

All package ZIP names, info versions, exact dependencies/members and build-manifest
version fields use the package version. Native files and embedded version strings
retain the compile version; do not rewrite or relink binaries. Do not mutate compile
records while repackaging. Print both identities and add optional compile_version
to successful build manifests for provenance. Latest remains the package version;
latest-compile remains the compile version. Historical manifests remain readable.

Use existing bin output and provenance rather than introducing versioned artifact
storage. Previously overwritten/invalidated compilations are unavailable: fail
clearly instead of claiming historical artifacts exist. Current source/resources
are still packaged with the selected native output; no source freshness guarantee.

Alternative: make build --version always rename latest output. Rejected because it
changes existing explicit-version fresh compilation. An explicit source flag is
reviewable and preserves existing command behavior.

## Work Items

- [x] 1.1 Add explicit source selection and record dual version provenance.
  Size small, difficulty medium, no dependencies. Completion: static trace of
  source failure before writes, unchanged compile cache/binaries and package-version
  propagation through artifacts/dependencies/collections/manifests.
- [x] 1.2 Update usage/spec, compile-only and independent review evidence.
  Size small, difficulty low, depends on 1.1. Completion: consistent examples,
  syntax compilation and final scoped diff; independent Reviewer handoff.

## Acceptance

- compile aiw snapshot-X then build aiw --compile-version snapshot-X --version 1.0.1
  uses unchanged native files, packages/dependencies at 1.0.1 and records both versions.
- Explicit source overrides environment source filtering; package env defaults still work.
- Unknown, stale or uncovered source versions fail without rebuilding/packaging.
- Default reuse, explicit fresh build, publish, clean and latest behaviors stay compatible.

## TODO

- Implementation, Python compile-only and independent static review complete.
- Local delivery, cleanup and archive after managed review passes.

## Verification

Planning: inspected current compile provenance/coverage/hash checks, build dispatch,
version fallback, package scripts/dependencies and successful-manifest writes.
No tests, target builds, packaging, publishing, installation or network execution.
Independent design evidence: reviews/FD-087-design.md and JSON, ready without blockers.
Implementation: reports/FD-087-implementation.md and JSON. Python compile-only for
build.py passed without imports or output artifacts; scripts/compile.py is Go-only.
One final scoped git diff against fb67c60 traced exact source selection before writes,
package version propagation, manifest compile_version for components/target/groups,
default/env compatibility and unchanged fresh publish. No test, target compilation,
packaging, runtime CLI validation, cleanup, installation, publishing or network run.
Independent implementation review: reviews/FD-087-review.md and JSON; no blockers
against commit 0b238d3, base fb67c60. Runtime checks remain unperformed.

## Risks

%% Binary-reported version can intentionally differ from package identity.
%% Only still-valid current output is available; no historical artifact archive added.
%% Repackaging uses current source/resources and retains immutable repository rules.
