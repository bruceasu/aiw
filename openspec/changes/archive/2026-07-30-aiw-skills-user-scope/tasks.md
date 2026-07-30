## TODO

- [x] Add a scope-aware destination resolver for project and user `.agents/skills` roots.
- [x] Add `--scope project|user` to install, discover, adopt, and sync commands.
- [x] Thread the selected scope through manifest loading, safe publication, and result reporting.
- [x] Preserve project-scope behavior and protect unmanaged destinations at user scope.
- [x] Update `plugins/aiw-skills/README.md` and the user-facing CLI help with user-scope examples.
- [x] Add focused CLI tests for both scopes, dry-run/JSON output, and per-scope manifests.

## Verification

- [x] Static review confirms the default project target remains unchanged.
- [x] Static review confirms no Codex-specific `~/.codex/skills` target is introduced.
- [X] Focused tests are not run by default; ask the user before running one focused command.

## Risks



