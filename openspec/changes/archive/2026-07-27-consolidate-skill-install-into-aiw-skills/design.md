## Context

AIW already has a managed Skill installation flow in `aiw-skills`. The older
`aiw-install-skill` plugin can unpack more input shapes, but it does not
participate in managed ownership. The desired end state is a single visible
installation command with consistent semantics: any installed Skill is
tracked, verifiable, and protected from accidental overwrite.

## Goals / Non-Goals

**Goals:**

- Keep a single user-facing installation verb: `aiw skills install`.
- Support canonical Skill names and local source paths in the same command.
- Preserve safe managed installation semantics for every installed Skill.
- Allow path sources to represent a single Skill directory, a zip file, or a
  bundle directory that contains multiple Skills.
- Remove the need for a separate import-only plugin.

**Non-Goals:**

- Introducing a new `import` subcommand.
- Changing Skill discovery or `/skills` invocation behavior.
- Adding support for non-local remote sources.
- Reworking the managed manifest schema unless required by the unified input
  flow.

## Decisions

### One install command, two source classes

`aiw skills install` will accept either a canonical Skill name or a local path.
The command will resolve names against the canonical repository `skills/`
collection and resolve existing paths as import sources. The CLI will not add a
new verb for path sources.

### Managed installation is mandatory

All installed Skills must go through the existing managed pipeline: validation,
staging, content digesting, atomic publish, and manifest updates. The old
unmanaged copy behavior is not retained.

### Bundle handling remains an input shape, not a separate mode

Bundle directories and bundle zip layouts are just another way to describe one
or more Skills to install. Each resulting Skill is still installed as an
individually managed entry. The command output may report multiple installed
Skills from one invocation.

### Deprecate before removal

The separate `aiw-install-skill` plugin should either become a thin
compatibility wrapper that forwards to `aiw skills install` with a deprecation
message or be removed once all internal references are migrated. The exact
transition can be staged to reduce breakage.

## Risks / Trade-offs

- [Risk] Resolving a positional argument as either a name or a path can be
  ambiguous. Mitigation: prefer existing filesystem paths first, then fall back
  to canonical Skill name lookup.
- [Risk] Bundle installs may partially succeed if one skill fails after others
  have already been published. Mitigation: keep each installed Skill atomic and
  report failures clearly; add rollback only if later requirements demand it.
- [Risk] Some downstream scripts may still call `aiw-install-skill` directly.
  Mitigation: keep a short-lived compatibility shim and document the deprecation
  window.
- [Risk] More permissive input handling can complicate tests and help text.
  Mitigation: add focused CLI tests for canonical, path, zip, and bundle
  sources.

## Migration Plan

1. Extend `aiw-skills install` to resolve path-based sources.
2. Reuse the existing managed install logic for every source type.
3. Add tests for canonical names, single directories, zip files, and bundles.
4. Update docs and help text to show one install command.
5. Deprecate `aiw-install-skill` and migrate internal callers.
6. Remove `aiw-install-skill` after compatibility is no longer needed.

## Open Questions

- Should bundle installs be allowed to partially succeed, or should the command
  fail as a unit if any member Skill fails validation?
- Should the compatibility shim for `aiw-install-skill` remain for one release
  cycle or be removed immediately after migration?
