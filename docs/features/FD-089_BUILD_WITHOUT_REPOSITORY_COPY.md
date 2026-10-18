# FD-089: Build and install without a repository copy

**Status:** In Progress
**Revision:** 2
**Priority:** Medium
**Evidence policy:** Dual

## Problem and decision

The user confirms build should not publish or duplicate package ZIPs into an
internal repository. Remove build's export to dist/repository. Only explicit
publish/publish-all/export write a repository. Build completes after packaging and
successful manifest writes; existing snapshot and package-version rules stay intact.

Use a build-local LocalPackages adapter with the existing Operation index/fetch
interface. It reads original ZIPs under dist, builds an in-memory exact identity
catalog and returns those original paths from fetch, without copying or writing
index.json. Keep the shared installation engine/transactions unchanged.
Validate package metadata and declared dependency/member closure before Manager
construction. Also preflight all host-selected payload targets against all original
ZIPs: reject an equal path or file target that is an ancestor of any source ZIP.
Resolve the destination root as Manager does; preserve allowed destination aliases,
then use safe_path below it. This preserves input archives across package boundaries.
Fetch revalidates path and complete entry metadata/hash before use;
Operation continues graph/version/cycle/platform/target preflight and byte pinning.

New component manifests must include their full dependency/member closure, even
when produced by build all. For existing incomplete component manifests, resolve
only exact declared name/version ZIPs under the five existing versioned dist
categories (mist/plugins/skills/resources/collections). Do not scan repositories,
use latest, use network or fall back to dist/repository. Reject missing, mismatched,
conflicting duplicate identities and ambiguous fallback paths before installation.
Dependencies not declared by selected packages are not discovered or installed.

Do not delete old dist/repository automatically; clean remains the explicit way
to remove previous outputs. Keep catalog closure separate from install roots:
single components install only their canonical package as root; collections install
only themselves. Compatibility groups retain their existing manifest-root behavior.
Preserve dependency auto/manual rules and explicit
export/publish behavior. Installation extraction/transaction backups remain needed;
this change removes duplicate repository/download ZIPs, not transaction safeguards.

Alternative: prepare dist/repository during install. Rejected because it still
copies ZIPs. Alternative: change Operation to accept a new package map API.
Rejected because a local adapter preserves the shared engine boundary.

## Work Items

- [ ] 1.1 Add direct-source closure adapter and cross-source target preflight.
  Size small, difficulty medium, no dependencies. Completion: source confinement,
  exact closure, original-path fetch verification and source-overwrite protection traced.
- [ ] 1.2 Wire build/install and closed component manifests with separate roots.
  Size small, difficulty medium, depends on 1.1. Completion: no implicit repository,
  declared root identity, dependency auto/manual rules and legacy lookup traced.
- [ ] 1.3 Update usage/spec, compile-only and independent review evidence.
  Size small, difficulty low, depends on 1.2. Completion: coherent build/publish/install
  boundaries, syntax compilation and independent static review.

## Acceptance

- build creates packages/manifests without Repo.publish, dist/repository or Published messages.
- install resolves collection/dependency ZIPs directly under dist without duplicate ZIP writes.
- New component manifests are closed; legacy incomplete manifests resolve exact declared ZIPs.
- Missing/conflicting/unsafe artifacts fail before destination writes; engine graph gates remain.
- A payload cannot overwrite any input ZIP or replace an ancestor of any input ZIP.
- Single component/collection dependency closure does not become manual install roots.
- Explicit publication/export still writes requested repositories; clean handles old staging output.

## TODO

- Design D1-D3 incorporated: separate roots, small items and cross-source preflight.
- Implement at .wt/FD-089 on feature/FD-089.
- Independent implementation review, local delivery, cleanup and archive.

## Verification

Planning: inspected build/install/export dispatch, manifest generation, Repo.index/fetch,
Operation dependency/member selection and Manager preflight. No runtime/install/build,
publication, deletion, network or tests performed.
Independent design evidence reviews/FD-089-design.md and JSON recorded D1-D3;
all findings incorporated above before implementation, with no repeated design review.

## Risks

%% Local adapter adds metadata/hash reads but no duplicate archive writes.
%% Runtime install behavior remains unverified; shared transaction protections must be preserved.
%% Legacy fallback is exact and confined to standard categories; missing artifacts require rebuild.
