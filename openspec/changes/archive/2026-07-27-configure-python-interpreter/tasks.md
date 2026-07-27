## 1. Runtime Configuration

- [x] 1.1 Add focused reading of `[runtime].python` from an `aiw.toml` file
  without introducing a dependency or changing unrelated configuration behavior.
- [x] 1.2 Add platform user-configuration discovery with canonical-path priority,
  home-directory compatibility fallback, first-file-only behavior, and no writes.
- [x] 1.3 Add explicit interpreter-path normalization and validation with
  source-specific errors for relative, missing, or directory values.

## 2. Interpreter Resolution

- [x] 2.1 Integrate `AIW_PYTHON`, user configuration, and program-directory
  configuration into Python interpreter resolution in the specified priority.
- [x] 2.2 Preserve bundled `python` directory and `PATH` fallback behavior for
  unconfigured installations.
- [x] 2.3 Ensure the new configuration path affects Python plugins only and does
  not change resolution for Perl, Java, Bash, JavaScript, or PowerShell.

## 3. Behavioral Tests

- [x] 3.1 Add interpreter-resolution tests covering environment, user,
  program-default, bundled, `python`, and `python3` priority.
- [x] 3.2 Add platform configuration-discovery tests covering Windows, XDG,
  home compatibility fallback, canonical-file priority, and absent files.
- [x] 3.3 Add validation and parsing tests covering empty values, unrelated TOML
  sections, comments, malformed relevant values, relative paths, missing files,
  directories, and source-specific errors.
- [x] 3.4 Verify resolution does not create user directories or configuration
  files and that non-Python interpreter behavior remains unchanged.

## 4. Documentation and Verification

- [x] 4.1 Document `[runtime].python`, `AIW_PYTHON`, platform configuration
  locations, exact precedence, compatibility fallback, validation, and the
  no-auto-create policy.
- [x] 4.2 Run Go formatting on changed Go files and inspect the scoped diff.
- [x] 4.3 Run the focused plugin package tests, `go test ./...`, `go vet ./...`,
  and `go build ./...`, recording actual results and any environment limitations.
- [x] 4.4 Update this task list with completed TODO items, verification evidence,
  and any remaining `%%` risks or questions before finishing.

## Notes

%% Do not add or upgrade a TOML dependency without separate user approval.
%% Keep the implementation scoped to Python runtime selection; a general
cross-runtime configuration framework is outside this change.
%% Verification: `go test ./internal/plugin -count=1`, `go test ./...`,
`go vet ./...`, `go build ./...`, `gofmt`, `git diff --check`, and
`openspec validate configure-python-interpreter --strict` passed.
