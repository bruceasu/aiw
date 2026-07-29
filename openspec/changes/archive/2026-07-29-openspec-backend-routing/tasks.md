## 1. Backend detection

- [x] 1.1 Add backend mode parsing with `auto`, `openspec`, and `native` values.
- [x] 1.2 Implement cross-platform OpenSpec discovery using `AIW_OPENSPEC_BIN`, PATH, and `--version` verification.
- [x] 1.3 Add unit tests for missing, invalid, configured, and Windows `.cmd` executables.

## 2. Delegation seam

- [x] 2.1 Define the supported operation-to-OpenSpec command mapping without changing native behavior.
- [x] 2.2 Delegate verified operations with validated identifiers, working directory, and exit-code propagation.
- [x] 2.3 Fail before writes when explicit OpenSpec delegation is unavailable or the subprocess fails.

## 3. Compatibility and documentation

- [x] 3.1 Preserve native default behavior for existing scripts and add backend diagnostics.
- [x] 3.2 Update CLI help and README with backend selection and Skill/OpenSpec guidance.
- [x] 3.3 Run Go formatting, tests, vet, build, and strict OpenSpec validation.
