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
python build.py compile aiw --version 1.2.3
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
python build.py compile aiw
python build.py latest-compile aiw
python build.py build aiw
python build.py clean
```

Compile writes only bin, including compile provenance. Build packages selected
components and reuses compatible recorded compilation when no --version is given;
otherwise it compiles first. Install/export consume an existing release manifest and never compile,
assemble or download. Plugins and Skills have individual scripts under
scripts/packages; shared definitions live in scripts/package_definitions.py.
plugin-meta.toml and skills-meta.toml define core/extension membership. Main includes
agent-templates; shared root Skill documents have unique resource package owners.
Individual Skills depend on needed resources rather than an aggregate support ZIP.
Usage documents belong to their program/plugin; development docs are not installed.

For `compile`, version priority is `--version`, a nonempty
`AIW_VERSION`, then a generated UTC timestamp such as
`snapshot-20261018-130000123456Z` (year/month/day, hour/minute/second and
microseconds). The command prints the selected version before starting and uses
that one value for the entire invocation. Publish commands still require their
explicit positional version. Independent package scripts still require `--version`
unless `AIW_VERSION` is configured.

For `build` without `--version`, first reuse the latest successful compile record
covering the requested target. If `AIW_VERSION` is set, only that version qualifies.
Both the recorded version and native binaries are reused, without calling the
compilers again. For example, `compile aiw` then `build aiw` shares one version;
`compile all` can also serve `build aiw`. If no covering record exists, build selects
the configured/default snapshot version and compiles as before. An explicit
`build --version` and publish commands always compile afresh.

Compile records live in `bin/.compile.json`. A new compile invalidates overlapping
native records before changing any output and records success only after completion.
Reuse compares the complete native file inventory and SHA-256 values, including
both Go platforms and added/removed TypeScript files. Missing/changed output fails
with an error; compile again before packaging. Old output without provenance is not
reused. `latest-compile [target]` prints the newest validated covering compile
version; `latest [target]` continues to query successful package builds.

Reuse deliberately packages the recorded binaries with current source/resources;
it does not detect source changes. Run compile again when updated native code is
needed. Compilation, build and clean assume a single writer.

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
after packaging and local repository staging succeed, with a completion timestamp;
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

`publish <package> <version>` compiles and packages that package plus its exact
dependencies/members, then publishes those newly built ZIPs. Use `publish-all
<version>` for every package. These commands require an explicit version and a
local `AIW_PACKAGE_REPO`; they never reuse an older release manifest. They retain
the build staging repository as well. Publication stops on the first error;
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
| `AIW_VERSION` | Explicit default version; constrains compile reuse for build | Latest compatible compile for build; otherwise UTC snapshot for build.py. Independent scripts require `--version` when unset |
| `AIW_GO` | Go executable path or command name | `go` on PATH |
| `AIW_NODE` | Node executable path or command name | `node` on PATH |
| `AIW_INSTALL_DIR` | Existing default installation root; `install --dest` wins | Platform default |

Go builds continue to produce both Windows/Linux amd64 binaries. Tool paths and
installation/repository paths can vary by machine without changing that package contract.

Packages live under dist/<category>/<version>/; raw main/program binaries are in
dist/bin/<platform>-amd64. A build also stages a local dist/repository with index.json.
Repeated publication of identical normalized content succeeds without rewriting;
different content under the same name/version is rejected: choose a new version.
There is no force override. Bootstrap through build.py install or the standalone
Python package entry point with --repo pointing at the staged repository.

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
