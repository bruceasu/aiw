## Problem Statement

AIW runs Python plugins with whichever interpreter it discovers beside the AIW
executable or first on the current process `PATH`. This can make the same AIW
installation use different Python versions in different shells or on different
machines. Users cannot currently select an interpreter through AIW configuration,
and editor-specific interpreter settings do not affect the AIW process.

## Solution

Allow users and distributors to configure an explicit Python executable. Program
configuration provides defaults, user configuration overrides those defaults,
and an environment variable provides the highest-priority temporary override.
When no explicit value is configured, AIW keeps its existing bundled-Python and
system-`PATH` discovery. Reading configuration remains side-effect free, so AIW
does not create a user configuration file merely because it is absent.

## User Stories

1. As an AIW user, I want to select a specific Python executable, so that Python
   plugins run with a predictable Python version.
2. As an AIW user, I want my user configuration to override program defaults, so
   that I can customize an installation without editing distributed files.
3. As an AIW distributor, I want to provide a default Python path beside AIW, so
   that a packaged installation works consistently by default.
4. As a shell user, I want an environment variable override, so that I can test
   or temporarily switch Python without editing configuration.
5. As a CI operator, I want to select Python through the environment, so that a
   job can use its provisioned runtime deterministically.
6. As a Windows user, I want AIW to find configuration under the standard
   application configuration directory, so that configuration follows platform
   conventions.
7. As an existing Windows user of the home `.config` convention, I want that
   location to remain a fallback, so that existing configuration can continue to
   work.
8. As an XDG user, I want AIW to respect `XDG_CONFIG_HOME`, so that AIW follows
   my configured directory layout.
9. As a user with more than one possible user configuration file, I want AIW to
   choose one deterministically, so that settings are not merged unexpectedly.
10. As a user opening an unfamiliar repository, I do not want repository
    configuration to select a local executable, so that project files do not
    cross an unnecessary execution trust boundary.
11. As an existing AIW user with no runtime setting, I want current bundled and
    `PATH` discovery to keep working, so that the change is backward compatible.
12. As a user with a broken configured path, I want a clear error that identifies
    the configuration source, so that I can fix the correct setting.
13. As a user who explicitly configured Python, I do not want AIW to silently
    fall back to another version, so that configuration mistakes are visible.
14. As a user with no AIW configuration, I do not want ordinary commands to
    create files in my profile, so that read-only use has no filesystem side
    effects.
15. As a portable-installation user, I want a bundled Python runtime to remain a
    fallback, so that AIW can run without a system Python installation.
16. As a maintainer, I want Python configuration isolated from other runtime
    families, so that Perl, Java, Bash, JavaScript, and PowerShell behavior does
    not change accidentally.
17. As a maintainer, I want the behavior covered at the interpreter-resolution
    boundary, so that tests remain stable while internal helpers evolve.
18. As a maintainer, I want no new dependency for this focused setting, so that
    the change remains small and reviewable.

## Implementation Decisions

- Add a core runtime-configuration reader that is independent of the standalone
  commit-message command's configuration code.
- Use this Python interpreter priority: environment override, user configuration,
  program-directory configuration, bundled interpreter, `python` on `PATH`, then
  `python3` on `PATH`.
- Use `AIW_PYTHON` as the environment override.
- Use `runtime.python` in `aiw.toml` as the file-based setting.
- Treat program-directory configuration as defaults and user configuration as an
  override.
- Use the platform user-configuration directory as the canonical user location.
- On Windows, use the roaming application configuration directory, not the local
  application-data directory.
- On XDG platforms, respect `XDG_CONFIG_HOME` and otherwise use the home `.config`
  convention.
- Check the home `.config` convention as a compatibility fallback when it differs
  from the canonical platform path.
- Stop after finding the first user configuration file instead of merging
  multiple user files.
- Do not read project-root configuration for interpreter selection.
- Require explicit interpreter values to be absolute paths to existing files.
- Treat an empty or whitespace-only value as unset.
- Report invalid explicit paths and do not silently continue to fallback
  discovery.
- Do not create configuration directories or files during interpreter
  resolution.
- Do not add or upgrade a TOML dependency for this change.
- Keep all non-Python interpreter behavior unchanged.

## Testing Decisions

- Test observable interpreter selection and errors at the existing
  interpreter-resolution boundary rather than testing private helper structure.
- Cover the full priority chain from the environment override through the final
  `python3` lookup.
- Cover user-over-program precedence and program-default behavior.
- Cover Windows canonical configuration, XDG configuration, home compatibility
  fallback, canonical-file priority, and absent configuration.
- Verify that discovery reads at most one user configuration file.
- Verify that absent configuration does not create a directory or file.
- Cover empty values, unrelated sections, comments, quoting, and malformed
  relevant configuration values.
- Cover relative paths, missing paths, directories, and valid executable files.
- Verify errors identify whether the value came from the environment, user
  configuration, or program configuration.
- Preserve and extend the existing interpreter-resolution tests as prior art.
- Run focused plugin tests, the complete Go test suite, vet, build, formatting,
  OpenSpec validation, and scoped diff inspection.

## Out of Scope

- Automatically creating a user configuration file.
- Adding `aiw config init` or `aiw config set` commands.
- Installing, downloading, upgrading, or managing Python.
- Creating or activating virtual environments.
- Allowing relative interpreter paths.
- Reading interpreter selection from a checked-out project's configuration.
- Configuring non-Python interpreters.
- Reworking all existing AIW configuration into one general framework.
- Adding a third-party TOML dependency.

## Further Notes

The change is backward compatible for users who do not configure an interpreter.
The intended GitHub triage label is `ready-for-agent`. The OpenSpec change
`configure-python-interpreter` contains the implementation design, normative
scenarios, and task checklist.
