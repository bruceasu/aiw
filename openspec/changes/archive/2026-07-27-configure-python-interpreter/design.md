## Context

Python plugins are currently executed by resolving `python` from a directory
beside the AIW executable and then from the process `PATH`. AIW has no shared
runtime configuration loader, while the existing `aiw.toml` behavior in the
standalone `cz` program is command-specific. The change must therefore add a
small, core configuration boundary without coupling plugin execution to `cz`.

The relevant stakeholders are users who need reproducible plugin execution,
administrators or distributors who provide program defaults, and existing users
who rely on bundled Python or `PATH` discovery.

## Goals / Non-Goals

**Goals:**

- Make Python interpreter selection explicit and deterministic when configured.
- Let user configuration override program-directory defaults.
- Let an environment variable override file-based configuration.
- Use platform-appropriate user configuration discovery while retaining the
  existing `$HOME/.config/aiw` convention as a compatibility fallback.
- Keep existing bundled-interpreter and `PATH` behavior when configuration is
  absent.
- Keep ordinary configuration reads free of filesystem writes.
- Provide actionable errors for invalid explicit interpreter paths.

**Non-Goals:**

- Adding configuration commands such as `aiw config init` or `aiw config set`.
- Automatically creating or rewriting user configuration.
- Selecting Python from project-root configuration.
- Managing virtual environments or installing Python.
- Configuring the Perl, Java, Bash, JavaScript, or PowerShell runtimes.
- Adding or upgrading a TOML dependency.

## Decisions

### Use layered runtime configuration

Interpreter selection will use this precedence, from highest to lowest:

1. `AIW_PYTHON`
2. `[runtime].python` from the user configuration
3. `[runtime].python` from `aiw.toml` beside the resolved AIW executable
4. `python/python.exe` or the platform-equivalent bundled interpreter beside AIW
5. `python`, then `python3`, from `PATH`

The implementation will read lower-priority program defaults before applying
the user value. This follows the common model of program defaults overridden by
user preferences, while preserving an explicit environment override for shells,
CI, and one-off execution.

An alternative was to give the executable-directory file priority for portable
installations. That would prevent users from overriding distributor defaults and
was rejected.

### Discover one user configuration file

The canonical user configuration directory will come from the platform's
standard user-config directory:

- Windows: `%APPDATA%\aiw\aiw.toml`
- Linux and other XDG platforms: `$XDG_CONFIG_HOME/aiw/aiw.toml`, falling back
  to `$HOME/.config/aiw/aiw.toml`
- macOS: the platform user-config directory returned by the operating system

For compatibility, `$HOME/.config/aiw/aiw.toml` will be checked if it differs
from the canonical path and the canonical file does not exist. Discovery stops
at the first existing user file; multiple user files are not merged.

`%LOCALAPPDATA%` is not a configuration default because it is conventionally
used for non-roaming machine-local data. Project-root `aiw.toml` is excluded
because allowing a checked-out repository to select an executable would create
an avoidable trust boundary.

### Treat configured values as executable paths

`AIW_PYTHON` and `[runtime].python` will represent an explicit filesystem path,
not a command name. The configured value must be absolute and identify an
existing non-directory file. An invalid explicit value will return an
actionable error naming the source and will not silently fall back to another
runtime.

Relative paths were considered but rejected because their base would differ
between environment variables, program configuration, and user configuration.

An empty or whitespace-only value is treated as unset so a documented default
configuration can contain `python = ""`.

### Add a focused core configuration reader

Core AIW code will own discovery and reading of the runtime setting. It will
read only the `[runtime]` section needed by this change and will not depend on
the standalone `cz` command's configuration implementation.

The reader will not mutate files. A missing configuration file is normal and
produces no error. A present file with an invalid `runtime.python` value or
malformed relevant syntax produces a source-specific error.

A third-party TOML library was considered, but adding a dependency is outside
the approved scope. The parser will stay deliberately narrow and covered by
behavioral tests.

### Test at the interpreter-resolution boundary

Tests will exercise the existing interpreter-resolution behavior with injected
executable, environment, user-config, and path lookup boundaries. Assertions
will cover the command selected or the user-visible error, rather than internal
helper structure. This is the highest existing seam that covers configuration
and fallback without starting external Python processes.

## Risks / Trade-offs

- [A focused TOML reader may not support every TOML construct] → Limit the
  contract to a string value in `[runtime]`, document it, and test whitespace,
  comments, unrelated sections, quoting, and malformed relevant values.
- [A configured interpreter can be deleted after validation] → Return the normal
  process-start error; eliminating this filesystem race is out of scope.
- [Platform config conventions may differ from existing user habits] → Use the
  platform-standard location and retain `$HOME/.config/aiw` as a fallback.
- [Strict invalid-path handling can stop commands that previously fell back] →
  Apply strictness only when the user or administrator explicitly configured a
  path and include the configuration source in the error.
- [Resolving configuration for every plugin execution adds filesystem checks] →
  The bounded set of small local files makes the cost negligible for CLI use.

## Migration Plan

No migration is required. Existing installations without `[runtime].python` or
`AIW_PYTHON` continue through bundled-interpreter and `PATH` discovery.

The feature can be rolled back by removing the explicit setting; no persistent
data transformation is involved.

## Open Questions

None.
