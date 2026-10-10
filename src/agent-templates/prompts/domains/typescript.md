# TypeScript Domain

## Inspect First
- Read `package.json`, the active lockfile, `tsconfig*.json`, and package scripts.
- Trace the touched types through callers, serialized data, and runtime validation.
- Check the existing framework, module format, and test conventions.

## Keep Stable
- Preserve strict compiler options and avoid widening types to silence errors.
- Prefer discriminated unions, generics, and type guards over `any` or unsafe casts.
- Validate untrusted data at runtime; TypeScript types disappear after compilation.
- Do not change package managers, module formats, dependencies, or runtime support without need.

## Validate
Use static review by default. Run only checks authorized by the repository
resource budget, using the narrowest existing package script or compiler command.
