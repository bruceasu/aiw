# Tasks

- [x] 1. Update the unknown-subcommand path to offer explicit native Git
      fallback confirmation with refusal as the default.
- [x] 2. Keep known local command dispatch and local help precedence unchanged.
- [x] 3. Add safe candidate-command rendering, non-interactive detection, and
      clear refusal/unavailable-Git diagnostics.
- [x] 4. Preserve native Git argv forwarding, stdout/stderr behavior, and exit
      status after approval.
- [x] 5. Update `aiw-git` overview and unknown-help diagnostics to document the
      fallback policy.
- [x] 6. Add focused dispatcher tests for approval, refusal, non-interactive
      invocation, help fallback, argv preservation, and exit-code propagation.
- [x] 7. Static-review the final diff and record verification results; do not
      run tests unless explicitly authorized.

## Verification

- Static review completed for the dispatcher, focused tests, and checklist.
- Tests were intentionally not run because the user did not explicitly
  authorize runtime validation.

%% Focused test command, if authorized later: `python -m unittest
%% plugins/aiw-git/test_aiw_git_dispatch.py` from the repository root.
