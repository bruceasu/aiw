# aiw-git-python-compatibility Specification

## Purpose

Define the supported Python runtime behavior for loading bundled `aiw-git`
subcommands without changing their command or model semantics.

## Requirements
### Requirement: Python 3.9 subcommand discovery

The `aiw-git` dispatcher SHALL load and discover its bundled Python
subcommands under Python 3.9 without failing because of unsupported dataclass
options.

#### Scenario: Discover the guide subcommand

- **WHEN** the dispatcher scans the bundled `plugins/aiw-git` directory under
  Python 3.9
- **THEN** discovery completes and includes the `guide` subcommand

### Requirement: Guide model behavior remains compatible

The guide subcommand MUST preserve the fields and default values of its search
match and generated-answer models after the compatibility change.

#### Scenario: Use default model values

- **WHEN** guide code creates its internal models without overriding optional
  fields
- **THEN** a search match defaults `extracted` to false and a generated answer
  defaults `save` to true
