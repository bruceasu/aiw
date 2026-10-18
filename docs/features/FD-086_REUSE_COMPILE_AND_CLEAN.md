# FD-086: Reuse compiled output and clean build directories

**Status:** In Progress
**Revision:** 2
**Priority:** Medium
**Evidence policy:** Dual

## Problem and decision

The user wants separate compile/build commands to share the compiled version and
artifacts, plus a clean command to remove previous output.

Record successful compile invocations in bin/.compile.json: target, version,
expanded components, completion time and SHA-256 inventory of native output.
Before touching compiled output, invalidate records with overlapping native
components (and the same target). Failed compilations must not remain reusable.
Write success only after the entire invocation completes. Inventory includes both
Go platforms and every TypeScript output file; reject links and missing/changed files.
Before reuse, recompute the complete native output inventory and require exact
path-set and hash equality, including rejection of newly added TypeScript files.

build without --version reuses the newest covering compile record (including a
superset such as compile all -> build aiw) and skips compilation. Nonempty
AIW_VERSION still selects a version: reuse only matching-version records; otherwise
compile that version. If there is no matching record, keep the current fresh
snapshot/compile behavior. A selected record with missing/changed artifacts fails
clearly, requiring compile again; never silently package stale output.
Explicit --version and publish always compile afresh and record success.
Resource-only compile invocations also record their version with an empty native
inventory. No source freshness guarantee: reuse means package the compiled native
artifacts plus current source/resources. Keep existing latest build behavior and
add latest-compile [target] for chronological compile lookup.

clean removes only checkout bin/ and dist/ (including compile/build records and
dist/repository), never the configured external repository or installed files.
Preflight both exact resolved output paths and all descendants before deleting
anything; reject symlinks/junctions or unexpected non-directory outputs. Missing
directories are harmless. No prompts, downloads or actual clean execution in this task.

Alternative: reuse only a version string. Rejected because overlapping/failed
compiles could silently package binaries from another version. Alternative: a new
versioned binary layout. Rejected as a broader packaging-path change.
No concurrency/locking contract added; existing build commands remain single-writer.

## Work Items

- [ ] 1.1 Add compile provenance, invalidation, default build reuse and compile lookup.
  Size medium, difficulty medium, no dependencies. Completion: safe coverage,
  version selection and exact full-inventory equality traced statically.
- [ ] 1.2 Add confined clean, update usage/spec and compile-only evidence.
  Size small, difficulty medium, depends on 1.1. Completion: cleanup preflight,
  docs consistency, Python compile-only and independent static review.

## Acceptance

- compile aiw then build aiw uses the same version and skips Go compilation.
- A covering compile all can serve a subset; incompatible targets/versions compile anew.
- Explicit build versions and publish keep fresh compilation.
- Failed/overlapping compilation cannot authorize stale output reuse.
- Added, removed or changed native output files prevent reuse of that record.
- latest-compile [target] reports a validated reusable compile version.
- clean preflights and removes only bin/dist, is repeatable and preserves external repo.

## TODO

- Design feedback D1 incorporated: exact full output inventory comparison required.
- Implement in .wt/FD-086 on feature/FD-086.
- Compile-only, independent implementation review, local delivery and archive.

## Verification

Planning: read existing compile/build dispatch, output layout, manifest ordering,
archive path/link helpers and current version/query contracts. No runtime/build/test,
clean, publishing or network execution authorized or performed.
Independent design report reviews/FD-086-design.md: its single blocking D1 was
incorporated above before implementation; no repeated design review required.

## Risks

%% Reuse does not inspect source edits; run compile again when new binaries are wanted.
%% Artifact hashing adds local disk reads; no concurrent compile/build/clean guarantee.
%% clean can partially complete if filesystem deletion fails after preflight; report errors.
