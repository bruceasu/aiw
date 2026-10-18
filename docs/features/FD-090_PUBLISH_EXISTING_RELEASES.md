# FD-090: Publish existing releases without rebuilding

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem and decision

The user observed publish-all printing Using snapshot output and Built because
publish still calls build_packages. Under the clarified independent stages,
publish/publish-all must publish already built artifacts, never compile or package.

Resolve an exact existing release from the five dist categories using the local
package adapter. Named publication selects the canonical root ZIP at the requested
version and its exact declared dependency/member closure. publish-all selects all
ZIPs under those categories for that exact version, including packages whose source
was removed later, and resolves their declared closure. Do not use current expand(all)
or latest manifests, which may refer to another release or changed membership.

Validate all selected ZIP identities, directory/filename/version consistency, links,
duplicate ambiguity, dependency/member completeness and entry hashes before the
first repository write. Missing release files fail with a build-first instruction;
Reject duplicate identities even when their contents are equal. Materialize and
fetch-validate the complete closure into an original-path list before publication;
do not defer validation of later ZIPs until earlier packages have been published.
no implicit compile, build or fallback version. Only explicit publication writes
the configured AIW_PACKAGE_REPO. Fetch returns original paths. Manifest timestamps,
snapshot records and package files remain unchanged during selection/publication.
Keep existing Repo immutability and partial multi-package publication behavior.

Implement a LocalPackages release factory and share its exact-file lookup with
the existing legacy install adapter; do not change the shared package engine API.
Existing build/install/clean/export syntax and behavior stay unchanged.

The reported installed aiw pkg discovery failure is a separate diagnosis: plugin
files currently exist as ordinary files, but the supplied error is EvalSymlinks
on its directory. No source defect or safe correction is confirmed yet. Do not
claim this publication FD fixes that symptom or modify installed files.

## Work Items

- [x] 1.1 Add existing-release discovery and rewire publish commands.
  Size small, difficulty medium, no dependencies. Completion: exact version selection,
  complete preflight before writes, original ZIPs and no build/compile calls traced.
- [x] 1.2 Update usage/spec, compile-only and independent review evidence.
  Size small, difficulty low, depends on 1.1. Completion: independent phases,
  failure behavior, syntax compilation and independent static review handoff.

## Acceptance

- publish-all 1.0.0 publishes existing 1.0.0 ZIPs without snapshot/manifest/package writes.
- Named publish uses canonical aliases and metadata-declared exact release dependencies.
- Newer latest manifests and source catalog changes do not change the selected old release.
- Missing/incomplete/unsafe/ambiguous release fails before publication and never rebuilds.
- Existing explicit destination, immutability and partial publication semantics remain.

## TODO

- Implementation and two-file Python compile-only complete; independent static review passed.
- Local delivery, cleanup and archive after managed pass, then separate plugin correction.

## Verification

Planning: read publish->build_packages path, LocalPackages closure/fetch and existing
Repo publication. Static installed-plugin observations are separate from this change.
No product CLI, tests, builds, installations, publication or network executed.
Independent design report reviews/FD-090-design.md and JSON: ready without blockers.
Implementation report reports/FD-090-implementation.md and JSON. Python compile-only
passed for build.py/scripts/local_packages.py with no imports or retained artifacts;
scripts/compile.py is Go-only. One scoped git diff and documentation rg batch against
1851532 traced exact existing files, full preflight before first publication and
absence of build/compile calls. No actual publication or packaging was run.
Independent Reviewer report reviews/FD-090-review.md and JSON: passed against
7551f5d/base 1851532, source FD-090-000004-review-requested; no remaining findings.
Help-text inconsistency corrected within the same review. No runtime publication validation
was performed; installed-plugin failure remains a separate follow-up.

### Separate installed-plugin diagnostic authorization

The user explicitly approved one `& 'C:\green\aiw\aiw.exe' pkg --help` from the
primary checkout, about 5 seconds, offline help/discovery only. Inspected manifest,
discovery resolver and Python argparse help path: no install/publication or network
operation requested. Budget: one run; no unchanged retry or wider test/build scope.
This is diagnostic evidence for the separate symptom, not publication acceptance.
Result: exit 1, same installed plugin-directory EvalSymlinks error reproduced.
No retry performed. The directory and files were statically observed to exist.

## Risks

%% Publication changes from implicit build to existing-file consumption as requested.
%% Multi-package publication can partially succeed; immutable identities remain.
%% aiw pkg runtime failure remains undetermined and outside this FD's acceptance.
