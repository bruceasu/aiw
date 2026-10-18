# FD-084: Build publishing and target catalog

**Status:** In Progress
**Priority:** Medium
**Revision:** 2
**Evidence policy:** Dual

## Problem

Build targets mix components and groups. Docs is listed but cannot be built.
Publishing a freshly built version to a local repository needs a direct command.
Machine-specific settings must come from the environment and a local `.env`.

## Decision

Keep compile/build/install/export compatible. Add `publish <package> <version>`
and `publish-all <version>`. Both compile and package the selected dependency
closure before publishing the newly returned paths through the existing Repo
engine. Never select a previous release through the latest manifest.

Use one target catalog with programs, discovered plugins, skills, resources,
collections and compatibility groups. Accept canonical package names as aliases
for single-package publishing. `all` includes docs and all current packages.
Keep bin/plugins/skills and existing short component names compatible.

Build docs as `aiw-docs`, kind resource, from docs/usage with README.md when
present, installed under docs/reference/usage and docs/reference/README.md
to avoid collisions with program-owned usage documents. Add `scripts/packages/docs.py` using the existing independent script
pattern. Do not package feature reports, local config or build output.

Load root `.env` without new dependencies, shell execution or interpolation.
Accept blank lines, comments, optional export prefix and quoted/unquoted values.
Existing process variables win. Use AIW_PACKAGE_REPO for the required publish
destination, AIW_VERSION for the existing default build version,
AIW_GO and AIW_NODE for tool paths, and existing AIW_INSTALL_DIR for installation.
Keep the fixed dual-platform and amd64 package contract.
Resolve relative repository paths against the checkout root. Reject URL
destinations for the new local publish commands before compilation.
Check in a safe `.env.example`; local `.env` records
`AIW_PACKAGE_REPO=d:/playground/aiw-packages` and stays ignored.

Alternative: reuse export after a build. Rejected because latest manifests can
refer to another version. Alternative: add a dotenv dependency. Rejected because
the required subset is small and network/dependency work is out of scope.

## Scope and compatibility

Only build orchestration, package definitions, docs packaging and documentation.
No dependency, repository format, public package-engine API or architecture change.
No actual package build, publish, install, network access or tests in this task.
Existing Repo immutability and partial multi-package publication semantics apply;
publish-all is not a repository transaction. No automatic overwrite or rollback.

## Work Items

- [ ] 1.1 Add environment loading, local settings and coherent target catalog.
  Size: small; difficulty: medium; dependencies: none. Completion: env precedence,
  tool overrides and categorized targets implemented. Evidence: static call trace.
- [ ] 1.2 Add fresh-build publish commands and independent docs packaging.
  Size: small; difficulty: medium; dependencies: 1.1. Completion: exact-version
  paths published and docs script exists. Evidence: static dependency/path trace.
- [ ] 1.3 Update usage/stable spec, compile source, record static evidence and hand off review.
  Size: small; difficulty: low; dependencies: 1.2. Completion: consistent docs,
  compile-only result and authored report. Evidence: compile result and final diff.

## Acceptance

- A named package plus version builds its dependencies, then publishes those exact ZIPs.
- publish-all uses the complete deduplicated target closure, including docs.
- Missing/remote repository fails before any build side effect.
- Process variables override .env; direct docs packaging and build docs work.
- Existing compile/build/install/export syntax and defaults remain compatible.

## TODO

- Implement 1.1-1.3 in the dedicated FD worktree after plan readiness.
- Independent review, local delivery and archive after gates pass.

## Verification

Planning: statically inspected build.py, package definitions, independent package
script pattern, existing compile helper, ignore rules and FD-083 package contract.
Independent design review: reviews/FD-084-design.md. Incorporated platform,
documentation ownership and Work Item metadata findings before implementation.
Tests, builds, publish/install and network commands have not run.

## Risks

%% Multi-package publication can partially succeed; use existing Repo error reporting.
%% Runtime build/publish behavior will remain unverified under the zero-test/build budget.
