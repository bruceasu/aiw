## ADDED Requirements

### Requirement: Explicit Python environment override

AIW SHALL use the absolute executable path in `AIW_PYTHON` for Python plugins
when the environment variable contains a non-empty value.

#### Scenario: Environment overrides file configuration

- **WHEN** `AIW_PYTHON` names a valid Python executable and both user and program
  configuration specify different Python executables
- **THEN** AIW executes the Python plugin with the executable named by
  `AIW_PYTHON`

#### Scenario: Empty environment value is unset

- **WHEN** `AIW_PYTHON` is absent, empty, or contains only whitespace
- **THEN** AIW continues to file-based interpreter configuration

### Requirement: Layered AIW runtime configuration

AIW SHALL read `[runtime].python` from the user `aiw.toml` before using the
program-directory default, while logically treating the user value as an
override of the program value.

#### Scenario: User configuration overrides program defaults

- **WHEN** the user configuration and the `aiw.toml` beside the AIW executable
  specify different valid Python executable paths
- **THEN** AIW executes the Python plugin with the user-configured executable

#### Scenario: Program configuration supplies a default

- **WHEN** no environment or user value is configured and the `aiw.toml` beside
  the AIW executable specifies a valid Python executable
- **THEN** AIW executes the Python plugin with the program-configured executable

#### Scenario: Empty configured value is unset

- **WHEN** a discovered `[runtime].python` value is empty or whitespace-only
- **THEN** AIW continues to the next interpreter source

### Requirement: Platform user configuration discovery

AIW SHALL discover at most one user configuration file using the platform
configuration location and the existing home-directory compatibility location.

#### Scenario: Windows canonical user configuration

- **WHEN** AIW runs on Windows and `%APPDATA%\aiw\aiw.toml` exists
- **THEN** AIW uses it as the user configuration file

#### Scenario: XDG user configuration

- **WHEN** AIW runs on an XDG platform, `XDG_CONFIG_HOME` is set, and
  `$XDG_CONFIG_HOME/aiw/aiw.toml` exists
- **THEN** AIW uses it as the user configuration file

#### Scenario: Home compatibility fallback

- **WHEN** the canonical user configuration does not exist and
  `$HOME/.config/aiw/aiw.toml` exists at a different path
- **THEN** AIW uses the home compatibility file

#### Scenario: Canonical file prevents compatibility merge

- **WHEN** both the canonical and home compatibility user configuration files
  exist
- **THEN** AIW uses only the canonical file and does not merge the compatibility
  file

### Requirement: Configuration reads have no write side effects

AIW MUST NOT create a user configuration directory or file while resolving a
Python interpreter.

#### Scenario: User configuration is absent

- **WHEN** no user configuration directory or file exists and AIW resolves a
  Python interpreter
- **THEN** AIW leaves the user filesystem unchanged

### Requirement: Explicit interpreter validation

AIW MUST require every non-empty explicitly configured Python interpreter to be
an absolute path to an existing non-directory file.

#### Scenario: Configured interpreter is valid

- **WHEN** the selected explicit value is an absolute path to an existing file
- **THEN** AIW uses that file as the Python interpreter

#### Scenario: Configured interpreter is relative

- **WHEN** the selected explicit value is a relative path
- **THEN** AIW stops before starting the plugin and reports which configuration
  source contains the invalid value

#### Scenario: Configured interpreter is missing

- **WHEN** the selected explicit value is an absolute path that does not identify
  an existing non-directory file
- **THEN** AIW stops before starting the plugin and reports which configuration
  source contains the invalid value

### Requirement: Backward-compatible interpreter fallback

AIW SHALL preserve bundled and system interpreter discovery when no explicit
Python interpreter is configured.

#### Scenario: Bundled Python is available

- **WHEN** no explicit interpreter is configured and a supported Python
  executable exists in the `python` directory beside the AIW executable
- **THEN** AIW uses the bundled executable

#### Scenario: System Python is used

- **WHEN** no explicit or bundled interpreter is available and `python` is
  available through `PATH`
- **THEN** AIW uses the resolved `python` executable

#### Scenario: Python3 is the final command candidate

- **WHEN** no explicit or bundled interpreter is available, `python` is absent
  from `PATH`, and `python3` is available
- **THEN** AIW uses the resolved `python3` executable

#### Scenario: No interpreter is available

- **WHEN** no explicit, bundled, or system Python interpreter is available
- **THEN** AIW returns an interpreter-not-found error without starting the
  plugin
