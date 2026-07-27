## Why

AIW currently selects the Python interpreter for `.py` plugins from a bundled
location or the process `PATH`, so the same installation can silently run under
different Python versions across machines and shells. Users need an explicit,
predictable way to select the interpreter without ordinary AIW commands creating
configuration files as a side effect.

## What Changes

- Add a `[runtime].python` setting to AIW configuration for an explicit Python
  executable path.
- Add an `AIW_PYTHON` environment variable as the highest-priority override.
- Treat the executable-directory configuration as program defaults and allow the
  platform user configuration to override it.
- Resolve platform user configuration from the standard configuration directory,
  with the existing `$HOME/.config/aiw` convention as a compatibility fallback.
- Preserve bundled-interpreter and system-`PATH` discovery when no explicit
  setting is present.
- Keep configuration reads side-effect free; a missing user configuration file
  is not created automatically.
- Reject an explicitly configured interpreter that is missing or not a file
  instead of silently selecting a different Python runtime.

## Capabilities

### New Capabilities

- `python-interpreter-configuration`: Defines explicit Python interpreter
  configuration, platform configuration discovery, precedence, validation, and
  fallback behavior for Python plugins.

### Modified Capabilities

None.

## Impact

- Affected behavior: Python plugin command construction and interpreter
  resolution.
- Affected configuration: executable-directory and user-level `aiw.toml`, plus
  the `AIW_PYTHON` environment variable.
- Affected documentation: plugin interpreter discovery and configuration
  precedence.
- No dependency, persistent data, plugin CLI, or public Go API changes are
  required.
- Existing bundled Python and `PATH` behavior remains backward compatible when
  no explicit setting is supplied.
