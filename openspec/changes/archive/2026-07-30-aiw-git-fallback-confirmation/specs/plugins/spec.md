## ADDED Requirements

### Requirement: Confirm native Git fallback

The `aiw-git` dispatcher SHALL require explicit user confirmation before
delegating a subcommand that is not defined by `aiw-git` to native Git.

#### Scenario: Known local command takes precedence

- **WHEN** the requested subcommand is defined by `aiw-git`
- **THEN** the dispatcher uses the local command implementation and does not
  offer native fallback

#### Scenario: Unknown command asks before delegation

- **WHEN** the requested subcommand is not defined by `aiw-git` and an
  interactive terminal is available
- **THEN** the dispatcher displays the exact candidate `git <subcommand> ...`
  invocation and asks for confirmation before starting Git

#### Scenario: Fallback is refused by default

- **WHEN** the user enters an empty answer, EOF, or any answer other than `y`
  or `yes`
- **THEN** the dispatcher does not start Git and returns a non-zero refusal
  result

#### Scenario: Non-interactive fallback is refused

- **WHEN** the requested subcommand is unknown and no interactive terminal is
  available
- **THEN** the dispatcher reports that confirmation is required, does not
  block for input, and does not start Git

#### Scenario: Git arguments do not approve fallback

- **WHEN** an unknown command's arguments contain `--force`, `--yes`, or another
  Git option
- **THEN** those arguments are forwarded only after separate explicit fallback
  confirmation

#### Scenario: Approved fallback preserves native execution

- **WHEN** the user explicitly approves fallback
- **THEN** the dispatcher invokes `git` with the original arguments as distinct
  argv values, preserves native output, and returns Git's exit status

#### Scenario: Native Git is unavailable

- **WHEN** fallback is requested but no native Git executable is available
- **THEN** the dispatcher reports the missing executable and does not prompt or
  start a subprocess

#### Scenario: Unknown help also requires confirmation

- **WHEN** the user requests help for a subcommand that is not defined by
  `aiw-git`
- **THEN** the dispatcher treats `git help <subcommand>` as a native fallback
  candidate and applies the same confirmation policy
