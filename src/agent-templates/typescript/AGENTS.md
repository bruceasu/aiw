# AGENTS.md

## Purpose
Use this guidance for TypeScript projects and subtrees.

## Inspect First
- `package.json`, the active lockfile, and package-manager scripts
- `tsconfig*.json` and the TypeScript version
- application entrypoints, module format, and nearby tests
- framework and runtime configuration used by the touched code

## Keep Stable
- compiler strictness and existing type boundaries
- ESM or CommonJS conventions and supported runtime versions
- public types, serialized data shapes, and runtime validation
- package-manager and workspace conventions

Avoid `any` and unchecked type assertions when a precise type or runtime guard
fits. Type declarations do not replace validation of external data.

## Validation
Use static review by default. Follow the repository resource budget and its
existing scripts. Do not install dependencies or run tests, builds, formatters,
linters, or type checks without authorization.
