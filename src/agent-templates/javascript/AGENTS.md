# AGENTS.md

## Purpose
Use this guidance for JavaScript projects and subtrees.

## Inspect First
- `package.json`, the active lockfile, and package-manager scripts
- `jsconfig.json`, if present, and the configured runtime version
- application entrypoints, module format, and nearby tests
- framework and runtime configuration used by the touched code

## Keep Stable
- ESM or CommonJS conventions and supported runtime versions
- public data shapes and validation at external boundaries
- package-manager and workspace conventions
- existing JSDoc and typing conventions

Prefer clear control flow and explicit error handling. Do not introduce a
transpiler or migrate files to TypeScript unless the task asks for it.

## Validation
Use static review by default. Follow the repository resource budget and its
existing scripts. Do not install dependencies or run tests, builds, formatters,
linters, or type checks without authorization.
