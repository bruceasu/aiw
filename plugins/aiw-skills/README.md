# aiw skills

Use `aiw skills` to list, discover, adopt, and safely install canonical AIW Skills.

This release installs one Portable Skill at a time. It always uses the current
project and the standard `.agents/skills` target. Managed reinstall, adoption,
and discovery are supported; user scope, Codex-specific targets, removal, and
links are separate follow-up features.

## Quick start

```text
aiw skills list
aiw skills discover
aiw skills install tdd --dry-run
aiw skills install tdd
```

查看完整命令帮助：

```text
aiw skills --help
aiw skills list --help
aiw skills discover --help
aiw skills install --help
```

Use `--json` after the subcommand when a script needs one machine-readable
result, for example `aiw skills list --json`.

## List Skills

```text
aiw skills list
aiw skills list --json
```

`list` reads canonical Skill metadata and does not change the project. Invalid
candidates are reported but are not listed as installable.

## Discover installed Skills

```text
aiw skills discover
aiw skills discover --json
```

`discover` inspects `.agents/skills` and reports which installed Skills are
already managed by AIW and which are unmanaged.

## Adopt existing Skills

```text
aiw skills adopt
```

`adopt` writes a managed manifest for valid existing Skill directories under
`.agents/skills`. Use this before reinstalling Skills that already exist but
are currently unmanaged.

## Preview an installation

```text
aiw skills install tdd --dry-run
```

The preview validates the Skill and shows the source and destination. It does
not create the target root, staging content, or a managed manifest.

## Install one Skill

```text
aiw skills install tdd
```

The installer:

1. validates the Skill name, description, and filesystem entries;
2. hashes the complete source directory;
3. copies it to staging on the destination filesystem;
4. verifies the staged hash;
5. publishes the directory;
6. atomically records managed ownership.

The managed manifest is `.agents/skills/.aiw-skills.json`. It records schema
version, source identity, source revision when available, copy mode, and the
SHA-256 content digest.

An unmanaged same-name directory is protected and will not be replaced by
default. An already managed Skill may be republished from canonical source,
which makes reinstall and upgrade flows possible. Reinstalling an identical
managed Skill is a successful no-op.

## Canonical source

Normal execution reads the repository or release root's maintained `skills`
collection, beside `program` and `plugins`. Tests and trusted development
environments can set
`AIW_SKILLS_SOURCE_ROOT` to an alternative canonical fixture directory.

Treat that override as trusted code input. Installed Skills can contain
instructions and executable scripts.

## Exit behavior

- `0`: the command succeeded, including dry-run, adoption, sync, and
  idempotent reinstall.
- `1`: an operational error occurred, such as an unknown Skill or protected
  destination.
- `2`: command syntax was invalid.

In JSON mode, operational results are written as one JSON object on stdout.
Human-readable operational errors are written to stderr.
