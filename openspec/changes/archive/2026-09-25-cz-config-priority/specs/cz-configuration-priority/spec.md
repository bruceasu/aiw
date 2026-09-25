## ADDED Requirements

### Requirement: cz-specific provider and model configuration takes precedence

The `cz` command MUST resolve its provider and model independently from the global AI workflow when explicit `cz` configuration exists. The effective precedence MUST be CLI overrides, runtime environment overrides, `cz` configuration, global `ai` configuration, and provider defaults, in that order.

#### Scenario: cz provider overrides global provider

- **WHEN** `[cz].provider` and `[ai].provider` are both configured and no higher-priority override is present
- **THEN** `cz` MUST invoke the provider configured by `[cz].provider`

#### Scenario: cz model overrides global model

- **WHEN** `[cz].model` and `[ai].model` are both configured and no higher-priority override is present
- **THEN** `cz` MUST use the model configured by `[cz].model`

#### Scenario: global configuration remains the fallback

- **WHEN** `cz` does not configure a provider or model and `[ai]` provides one
- **THEN** `cz` MUST use the applicable `[ai]` provider/model before applying provider defaults

### Requirement: explicit runtime overrides remain authoritative

The `cz` command MUST preserve explicit CLI and runtime environment overrides above file-based `cz` and `ai` configuration.

#### Scenario: CLI provider and model override files

- **WHEN** the user supplies `--provider` and/or `--model`
- **THEN** the supplied values MUST be used regardless of `[cz]` or `[ai]` file values

#### Scenario: environment overrides file configuration

- **WHEN** an applicable provider/model environment override is present and no CLI override is supplied
- **THEN** the environment value MUST take precedence over `[cz]` and `[ai]` file values

### Requirement: provider-specific model resolution is consistent

When the selected provider has provider-specific model settings, `cz` MUST resolve the setting for that selected provider without mixing credentials or endpoints from a different provider selection.

#### Scenario: selected provider uses its cz-specific model

- **WHEN** `[cz].provider` selects a provider and its provider-specific `[cz]` model is configured
- **THEN** `cz` MUST use that provider-specific `cz` model before generic or global model fallbacks

#### Scenario: missing cz model falls back safely

- **WHEN** `[cz]` selects a provider but provides no usable model
- **THEN** `cz` MUST fall back to the applicable global model and then the selected provider's default model without changing the selected provider

### Requirement: global fast profile is preferred over top-level global values

When `cz` has no higher-priority provider/model override, it MUST prefer the
global `ai.profiles.fast` profile over top-level `ai.provider` and `ai.model`.
If the fast profile is not configured, `cz` MUST use the top-level global
values before applying provider defaults.

#### Scenario: configured fast profile overrides global top-level values

- **WHEN** `ai.profiles.fast` and top-level `ai.provider` or `ai.model` are both configured and no `cz`, CLI, or environment override is present
- **THEN** `cz` MUST use the provider and model resolved from `ai.profiles.fast`

#### Scenario: absent fast profile falls back to top-level global values

- **WHEN** `ai.profiles.fast` is not configured and top-level `ai.provider` and/or `ai.model` is configured
- **THEN** `cz` MUST use the applicable top-level global value before applying provider defaults

#### Scenario: fast profile resolution does not mix global connection settings

- **WHEN** `ai.profiles.fast` is selected as the global fallback
- **THEN** `cz` MUST resolve the profile's provider, model, and profile-scoped connection settings as one selection and MUST NOT mix them with unrelated top-level provider settings
