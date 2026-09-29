# CZ configuration specification

## Requirements

### Requirement: CZ owns its LLM configuration

`cz` MUST resolve its LLM settings from `[cz]`, provider-specific `[cz.copilot]`,
`[cz.codex]`, and `[cz.openai]`, plus explicit CZ CLI and environment overrides.
It MUST NOT inherit the global `[ai]` provider, model, or fast profile.

#### Scenario: independent provider models

- **WHEN** each CZ provider has its own model configured
- **THEN** each SDK invocation uses only the selected provider's model and credentials

#### Scenario: CLI override

- **WHEN** `--provider` or `--model` is given
- **THEN** it overrides the corresponding CZ configuration for that invocation

### Requirement: ordered LLM fallback

Without a selected provider, `--llm` MUST attempt Copilot SDK, Codex SDK, then
OpenAI SDK. A failed provider or invalid candidate MUST advance to the next.
When all attempts fail, CZ MUST explain the failure briefly and enter the same
manual wizard as `--no-llm`. No fallback MUST commit without user review.

#### Scenario: all providers fail

- **WHEN** no SDK returns a valid candidate
- **THEN** CZ opens its manual commit message wizard

#### Scenario: explicit provider fails

- **WHEN** `--provider` selects one SDK and that SDK fails
- **THEN** CZ opens the manual wizard without calling another SDK

### Requirement: CZ remains an independent plugin

`aiw cz` MUST run the TypeScript implementation through the plugin entry point.
Its runtime requires Node.js 22.12.0 or newer. The Go AI provider workflow
MUST NOT be changed as a side effect of CZ provider selection.
