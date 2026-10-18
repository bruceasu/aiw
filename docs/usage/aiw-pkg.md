# Packages and repositories

Python 3.11+ is required. Supported installation platforms: Windows/Linux amd64.
Use a new, clean destination for this release. Back up personal configuration and
custom Skills, exit running programs, then install into a new directory. Do not
copy old `.aiw` state or overwrite old aggregate packages. The installer never
migrates ownership, deletes an old installation, or converts old backup state.

## Repository configuration

```toml
[repo]
url = "https://packages.example.com/aiw"
publish_url = "ssh://repo-host/srv/packages/aiw"
```

Reads use url; writes use publish_url, defaulting to url. Local directories,
HTTP(S), S3 and SSH use the same index and package rules. HTTP supports reads
only. Relative paths are resolved from the config file; `--repo ./packages`
is relative to the current directory and overrides both endpoints. No server
type flag, network probe or fallback is used. The nearest project aiw.toml or
.aiw.toml wins; AIW_ROOT/aiw.toml is the fallback. Expand ~ for local paths.

```text
aiw pkg list [--all] [--repo URL]
aiw pkg search QUERY
aiw pkg info NAME [--version VERSION]
aiw pkg list --installed [--kind plugin] [--dest ROOT]
aiw pkg show NAME [--dest ROOT]
aiw pkg entries --json
aiw pkg install NAME [--version VERSION] [--dest ROOT] [--dry-run]
aiw pkg install ./NAME-VERSION.zip --repo ./packages --dest ROOT
aiw pkg update [--core | --all] [--dry-run] [--dest ROOT]
aiw update [--core | --all] [--dry-run] [--dest ROOT]
aiw pkg remove NAME [--dest ROOT]
aiw pkg purge NAME [--dest ROOT]
aiw pkg mark manual NAME [--dest ROOT]
aiw pkg autoremove [--dry-run] [--dest ROOT]
aiw pkg publish NAME-VERSION.zip VERSION [--repo URL]
aiw pkg unpublish NAME VERSION [--repo URL]
aiw pkg index ./packages
aiw pkg pack WORKSPACE --output NAME-VERSION.zip
```

Destination: --dest, then AIW_INSTALL_DIR, then C:/green/aiw on Windows or
~/apps/aiw on Linux. Each root has separate .aiw/packages state and a write lock.
Local package input errors never trigger repository access. Local input ZIPs
are retained; downloads use an operation temporary directory and are cleaned.

Versions use SemVer major.minor.patch for latest, with stable above prerelease;
dev sorts below numeric releases. Other identifiers require --version. Equal
precedence with different identifiers is ambiguous and rejected.

## Collections and updates

Collections contain exact member references and no member payload. Nested
collections are deduplicated; missing dependencies, cycles, incompatible exact
versions, platform mismatches and conflicting paths fail before file changes.
Changes then use per-package transactions; failure reports completed packages
and stops. There is no collection-wide rollback or hook external-effect recovery.

Default update selects latest versions of installed packages only, including
installed extensions. New collection members are not installed or recorded as
installed. Required runtime dependencies are shown and may be installed. Conflicting
exact requirements, including requirements from unchanged installed packages,
fail before mutation; no implicit downgrade or old-release fallback is selected.
--core fills aiw-core; --all fills the fixed aiw-all release manifest, never every
package in a third-party repository. Neither option uninstalls out-of-scope
packages. --core and --all are mutually exclusive. Dry run never writes installation
files/state or invokes hooks.

Direct installs and explicitly selected root collections are manual. Dependencies
and members are auto; an existing manual reason is preserved. Removing a collection
retains its members and prints orphan cleanup/mark instructions. autoremove removes
only auto packages unreachable from all remaining manual roots. It retains config.
mark manual changes only the reason. Other packages cannot be removed while a
retained installed package requires them.

remove uninstalls managed program files and keeps registered config for a later
purge. purge also removes registered config and its own state, never unrelated
user files or collection members. No previous-version backup or rollback command
exists. Only the current transaction keeps recovery snapshots; success cleans them.
Incomplete recovery blocks subsequent writes and reports the transaction path.

## Skill management

```text
aiw pkg skill list
aiw pkg skill discover --scope project
aiw pkg skill adopt --scope project
aiw pkg skill install fd-workflow --scope project --dry-run
aiw pkg skill sync fd-workflow --scope user
```

skill-list/discover/adopt/install/sync are equivalent spellings. Canonical package
installation and project/user Skill copies are different operations. Existing
.aiw-skills.json records, local-change checks, unmanaged-directory protection and
shared-document materialization remain in place. aiw skills and aiw plugin were
removed. Use pkg entries for plugin entry metadata, not the repository inventory.

## Build and bootstrap

```text
python build.py compile aiw
python build.py build aiw-core --version 1.2.3
python build.py install aiw-core --dest NEW_ROOT
python build.py export aiw-core --repository ./packages
python build.py build docs --version 1.2.3
python scripts/packages/docs.py --version 1.2.3
python build.py publish aiw-git 1.2.3
python build.py publish-all 1.2.3
python build.py build docs
python build.py latest
python build.py latest aiw-docs
python build.py latest-compile aiw
python build.py build aiw --version 1.0.1
python build.py clean
```

Compile writes fixed snapshot output and provenance only to bin. Build packages
that snapshot, automatically compiling when required output is missing.
Install/export consume an existing release manifest and never compile,
assemble or download. Plugins and Skills have individual scripts under
scripts/packages; shared definitions live in scripts/package_definitions.py.
plugin-meta.toml and skills-meta.toml define core/extension membership. Main includes
agent-templates; shared root Skill documents have unique resource package owners.
Individual Skills depend on needed resources rather than an aggregate support ZIP.
Usage documents belong to their program/plugin; development docs are not installed.

Build writes versioned ZIPs and dependency-closed release manifests only. It does
not publish, copy ZIPs into `dist/repository`, or print `Published`. Install resolves
dependencies and collection members directly from original dist ZIPs through an
in-memory catalog, with no repository index or extra downloaded/archive copies.
For a single component or collection, only the requested package is an explicit
installation root; dependencies retain the engine's auto/manual rules. Compatibility
groups keep their existing explicit-root behavior.

Older incomplete component manifests are supported by exact declared name/version
lookup in the versioned mist/plugins/skills/resources/collections dist categories.
Missing, ambiguous or mismatched ZIPs fail before installation; no repository/network
fallback occurs. All source archives are validated, then rechecked when fetched.
Before installation, every selected payload target is checked against all original
ZIPs and their ancestors, preventing one package from overwriting another input.
Installation still uses the shared dependency graph checks and transaction staging.

`compile <target>` always refreshes one fixed `snapshot` output slot. It takes no
version parameter and ignores `AIW_VERSION`. Versioned Go executables embed the
literal `snapshot`; native output paths remain under bin. The old `compile --version`
and `build --compile-version` options have been removed.

`build <target> --version 1.0.1` reuses validated snapshot output and creates package
1.0.1, with matching dependency/member references. Missing provenance, target coverage
or ordinary native files causes automatic snapshot compilation before packaging.
Explicit package versions never force native recompilation. `compile all` can serve
`build aiw`; no historical compilation selector or artifact archive is needed.

Package version priority is `build --version`, nonempty `AIW_VERSION`, then one UTC
timestamp such as `snapshot-20261018-130000123456Z`. This timestamp versions the
packages only. Publish commands require an explicit positional package version
and consume existing packages without requiring snapshot output. Independent package scripts still require
`--version` unless `AIW_VERSION` is configured.

Compile records live in `bin/.compile.json`. New compilation invalidates overlapping
native records before changing output and records success only after completion.
Reuse validates complete native paths and SHA-256 values, including both Go platforms
and every TypeScript output file. Missing files trigger automatic compilation only
after the remaining inventory is checked; modified/extra files, links and filesystem
errors fail clearly. Run compile again to refresh modified output. Historical records
whose compile version differs from `snapshot` are readable but not reusable; build
automatically replaces the requested compilation with snapshot output.

Successful build manifests record `version` (package) and `compile_version=snapshot`.
`latest [target]` queries the latest package build. The compatible read-only
`latest-compile [target]` validates current snapshot output and prints `snapshot`,
or reports an error if none is usable; it never compiles automatically.

Snapshot reuse packages the recorded native binaries with current source/resources
and does not detect source edits. Run `compile <target>` to refresh changed native
code. Compilation, build and clean assume a single writer. Immutable publication
rules still require a new package version when packaged contents change.

`clean` removes only this checkout's `bin/` and `dist/`, including compile/build
records and `dist/repository`. It preserves the external `AIW_PACKAGE_REPO`, source,
dependencies, installed files and workflow state. Both output trees are preflighted
for confinement, regular files and links/junctions before any deletion; unsafe
trees fail without starting deletion. Missing output directories are harmless.
An operating-system deletion failure can leave partial cleanup and returns an error.

`python build.py latest` prints only the most recently built version across local
release manifests. Add a target or full package alias to query that target.
This is chronological build order, separate from repository SemVer `latest`.
Compile-only commands do not update the build record. New manifests are written
after packaging succeeds, with a completion timestamp;
old manifests fall back to filesystem modification time. Copying an old manifest
can therefore affect its order. The command checks the selected manifest's package
paths and reports an error for missing records/artifacts or invalid selected data.
It does not compile, package, publish or install anything.

The independent `aiw-docs` resource package installs usage documents under
`docs/reference/usage` and the root README under `docs/reference/README.md`.
It does not take ownership of the usage files installed by individual packages.

Targets come from one catalog: programs (`aiw`, `gateway`, `http-openai-proxy`),
discovered plugins, Skills, shared resources (including `docs` and `plugin-runtime`),
and release collections (`plugins-core/all`, `skills-core/all`, `aiw-core/all`).
Compatibility groups `bin`, `plugins`, `skills`, and `all` remain available.
`all` includes every package, including docs, and expands dependencies once.
Full package names such as `aiw-main`, `aiw-git`, `aiw-skill-fd-workflow` and
`aiw-docs` are aliases for their component targets. Collection names stay unchanged.

`publish <package> <version>` publishes that version's existing root ZIP and exact
metadata-declared dependencies/members. `publish-all <version>` publishes all ZIPs
already built under that version's five dist category directories, including packages
removed from the current source catalog. These commands never compile, package or
update build manifests; missing release files fail with an instruction to build first.
They validate the complete selected closure, filenames, identities and hashes before
the first repository write, rejecting ambiguous duplicate identities even if identical.
Selection uses the requested version's artifacts, independently of newer latest
manifests. Both commands require a local `AIW_PACKAGE_REPO`. For example:

```text
python build.py build --version 1.0.0 all
python build.py publish-all 1.0.0
```

The second command prints publication results without `Using snapshot output` or
`Built` messages and sends the original ZIPs to the configured repository. Explicit
`export` publishes existing manifest ZIPs to its requested repository. Publication stops on the first error;
earlier packages may already have been published. Existing versions cannot be
overwritten with changed contents.

Copy `.env.example` to the checkout root `.env` for local settings. `build.py`
and independent package scripts load that file, regardless of the invoking
directory. Existing process environment variables win over `.env` values.
The loader accepts `KEY=value`, optional `export`, quotes, blank lines and
comments; it does not execute commands, interpolate variables, or decode escapes.
Use quotes around paths containing ` #`. Local `.env` stays ignored by Git.

| Variable | Purpose | Default |
| --- | --- | --- |
| `AIW_PACKAGE_REPO` | Required local destination for publish commands; relative paths use the checkout root | No default; example: `d:/playground/aiw-packages` |
| `AIW_VERSION` | Default package version only; never changes compilation | UTC timestamp package version for build.py; independent scripts require `--version` when unset |
| `AIW_GO` | Go executable path or command name | `go` on PATH |
| `AIW_NODE` | Node executable path or command name | `node` on PATH |
| `AIW_INSTALL_DIR` | Existing default installation root; `install --dest` wins | Platform default |

Go builds continue to produce both Windows/Linux amd64 binaries. Tool paths and
installation/repository paths can vary by machine without changing that package contract.

Packages live under dist/<category>/<version>/; raw main/program binaries are in
dist/bin/<platform>-amd64. Only explicit publish/export actions write repositories.
An older dist/repository is left untouched and unused; clean removes it with dist.
Repeated publication of identical normalized content succeeds without rewriting;
different content under the same name/version is rejected: choose a new version.
There is no force override. Bootstrap through build.py install or the standalone
Python package entry point with --repo pointing at an explicitly published repository.

If plugin discovery reports `resolve plugin directory` although plugin.toml and
its entrypoint exist, the ordinary-path resolver avoids an additional Windows case
enumeration step after proving the entire path has no links. Links and irregular
objects still require physical resolution and must stay inside the plugin directory.
Windows junctions such as `C:\green` pointing to `D:\green` may appear as
`ModeIrregular` in Go. The resolver explicitly reads symlink/junction targets and
rechecks the resulting path, with a bounded link count. Unknown irregular objects
that do not provide a readable link target fail with the affected path in the error.
This correction is in the native launcher, so updating only the Python pkg files
does not update it: recompile the main snapshot, build/install the main package,
then verify the newly installed launcher separately. Keep the original failure output.

On Windows, exit programs that lock installation files. To update aiw itself,
exit aiw and invoke Python directly, for example:

```text
python C:/green/aiw/plugins/aiw-pkg/aiw-pkg.py install ./aiw-main-1.2.3.zip --repo ./packages --dest C:/green/aiw
```

The running EXE cannot replace itself on Windows. No updater, process termination,
lock detection, forced replacement or reboot scheduling is provided. Failed
recovery is reported as incomplete, with material location; preserve it before retry.
The official aiw-pkg package has no hooks; the engine preloads modules and package
bytes before changing its own files. User hooks run from independent staging and
remain responsible for their external effects.

## Publishing backends

Local writes use an exclusive lock directory and atomic index replacement. Inspect
stale locks manually; no automatic takeover occurs. Size, ZIP SHA-256, identity and
normalized content fingerprint are distinct checks. Hashes do not authenticate a
publisher. Use trusted repository access and system credentials.

S3 needs an existing aws CLI with conditional put-object support (--if-none-match).
Credentials remain in system configuration. A conditionally created lock object
serializes writers. Package upload precedes index commit; an uncertain index result
is re-read before cleanup. Cleanup/lock failures are reported, not hidden.

SSH needs existing ssh/scp locally and Python 3 remotely. Host aliases, IP and DNS
names use system SSH settings. Operators must install archive.py and repository.py
from src/plugins/aiw-pkg into REPO/.aiw-repo before use and grant write access.
Uploads are temporary and remote publication invokes the same filesystem writer.
The HTTP read URL and SSH write path must identify the same logical repository;
AIW does not infer or verify that mapping. HTTP write is unsupported in this release;
the future POST contract is defined in openspec/specs/aiw-packages/spec.md.

No tests, builds, downloads or publish operations are implied by reading this guide.

### Package usage document ownership

Program packages include their own guides under `docs/usage`. If package names
share a prefix, the longest matching name owns the guide: for example,
`aiw-file-convert.md` belongs to `aiw-file-convert`, not `aiw-file`.
The `aiw-docs` resource installs its reference copy under `docs/reference`.
Existing ZIPs keep their original metadata; rebuild affected packages explicitly
before installing them again. Updating this source does not modify installed
ownership records or previously built ZIPs.

### Local install planning

`python build.py install all` plans all selected roots together. Shared dependencies
and collection members are processed once. All selected roots retain manual-install
status; dependency-only packages retain the normal dependency reason rules.
Unchanged fingerprints skip temporary payload materialization, while archive,
platform, file ownership and final dependency checks still apply.

The existing per-operation 512 MiB aggregate input-ZIP limit applies to the entire
selection. A larger selection fails before installation; install smaller explicit
targets separately. Transactions remain per package: a later failure does not roll
back earlier successful packages. No benchmark result is implied by this change.
