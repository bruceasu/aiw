# TypeScript Project Guidance

Read `AGENTS.md` first, then `typescript/AGENTS.md`.

- Inspect `package.json`, the active lockfile, `tsconfig*.json`, and nearby tests.
- Preserve compiler strictness, module format, package manager, and runtime support.
- Prefer precise types and runtime guards over `any` or unchecked assertions.
- Do not add dependencies or run restricted checks without resource-budget authorization.
