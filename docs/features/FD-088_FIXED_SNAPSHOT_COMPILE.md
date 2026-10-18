# FD-088: Fixed snapshot compilation and package-only versions

**Status:** In Progress
**Revision:** 2
**Priority:** Medium
**Evidence policy:** Dual

## Problem and decision

The user replaces compile-version selection with one fixed snapshot output slot.
compile <target> has no version option and always generates snapshot binaries.
build <target> --version <package> packages validated snapshot output; if no covering
snapshot record or required output is missing, compile automatically as snapshot.
The package version never changes binary compilation or embedded snapshot strings.
Publish uses the same build helper and snapshot policy, then publishes fresh packages.

Package version selection stays --version, nonempty AIW_VERSION, then one UTC
snapshot-YYYYMMDD-HHMMSSffffffZ per packaging invocation. AIW_VERSION never controls
compilation or filters snapshot reuse. Remove compile --version and build
--compile-version, explicitly superseding FD-086/087 parameter rules per user intent.
Keep latest-compile [target] as a compatible status query returning snapshot only
when validated recorded output exists; latest still returns the package version.

Retain provenance/inventory protection, overlap invalidation before native writes,
single-writer policy and clean boundaries. Legacy non-snapshot compile records are
readable but not eligible for reuse; build recompiles them into the snapshot slot.
Missing regular artifacts trigger automatic compilation for build/publish; unsafe
links, malformed records or modified/extra native output still fail clearly,
requiring explicit compile/clean. Do not swallow filesystem/security errors.
Scan all remaining native output before a missing-file fallback; a missing file
must not hide modified/extra output, links or permission errors in the same record.
Keep compile_version=snapshot in successful manifests for historical readability;
do not retain historical compiler versions or source freshness tracking.

Alternative: keep version selection hidden behind aliases. Rejected because the
user explicitly wants one snapshot model. Alternative: package version passed to
the compiler. Rejected because it repeats the original ambiguity and recompiles.

## Work Items

- [ ] 1.1 Simplify compile/build/publish to the fixed snapshot slot.
  Size small, difficulty medium, no dependencies. Completion: no compile version
  options, deterministic snapshot linker, coverage reuse and missing-output fallback.
- [ ] 1.2 Reconcile usage/spec, compile-only and independent review evidence.
  Size small, difficulty low, depends on 1.1. Completion: one coherent command model,
  accurate old-record handling, syntax compile and one independent implementation review.

## Acceptance

- compile aiw always compiles snapshot regardless of AIW_VERSION.
- build aiw --version 1.0.1 reuses snapshot without compilation and sets package refs 1.0.1.
- Missing snapshot provenance, coverage or native files causes automatic snapshot compilation.
- Explicit compile refreshes snapshot; native changes/unsafe paths do not silently authorize reuse.
- Package fallback keeps timestamp snapshots; latest/latest-compile and clean remain coherent.
- Publish uses the same snapshot prerequisite and explicit package version.

## TODO

- Independent design ready; full-inventory scan before missing fallback incorporated.
- Implement at .wt/FD-088 on feature/FD-088.
- Independent implementation review, local delivery, cleanup and archive.

## Verification

Planning: inspected compile provenance/coverage/hash checks, compiler linker and
build/publish dispatch, version fallback and stable package/usage requirements.
No target compile, tests, package builds, cleanup, publication or network run.
Independent design evidence: reviews/FD-088-design.md and JSON; ready without blockers.

## Risks

%% Compile --version and build --compile-version intentionally removed under the new request.
%% Reuse checks binaries, not source freshness; explicit compile refreshes source changes.
%% Same package version with changed content still fails immutable repository publication.
