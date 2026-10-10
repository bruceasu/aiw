# JavaScript Domain

## Inspect First
- Read `package.json`, the active lockfile, runtime configuration, and package scripts.
- Trace the touched code through its callers, module format, and nearby tests.
- Check the existing framework and conventions for JSDoc or runtime schemas.

## Keep Stable
- Preserve ESM or CommonJS and the supported runtime version.
- Validate untrusted data at runtime and handle promise failures explicitly.
- Follow existing JSDoc and package-manager conventions.
- Do not introduce TypeScript, a transpiler, or dependencies unless requested.

## Validate
Use static review by default. Run only checks authorized by the repository
resource budget, using the narrowest existing package script or runtime check.
