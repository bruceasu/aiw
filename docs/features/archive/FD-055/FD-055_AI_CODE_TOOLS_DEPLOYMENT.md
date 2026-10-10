# FD-055: 将 AI Code Tools 集成为 AIW 插件

**Status:** Complete
**Revision:** 4
**Priority:** Medium

## Problem

`src/programs/ai-code-tools` owns the implementations of `generate-ai-index`
and `ai-code-index`. AIW also needs plugin entry points for these commands.
Maintaining full copies under `src/plugins` risks code drift, and installing a
separate command package is unnecessary.

## Decision

- Keep `src/programs/ai-code-tools` as the single authoritative implementation.
- Make `src/plugins/aiw-ai-gen-index.py` and
  `src/plugins/aiw-ai-code-index.py` small AIW entry points that invoke those
  implementations.
- Have `build.py plugins` and `build.py all` copy the canonical scripts into
  `dist/plugins/`, rename them to the AIW plugin entry-point names, and install
  them directly in the installed `plugins/` directory.
- Do not install a separate `AIW_INSTALL_DIR/ai-code-tools` package or create
  global commands.

## Work Items

- [x] 1.1 Replace duplicate plugin implementations with AIW entry points.
- [x] 1.2 Stage and install the canonical scripts as plugin support files.
- [x] 1.3 Document the source-of-truth and plugin deployment model.
- [x] 1.4 Review the diff and run one compile-only check; do not run builds or
  deployment.

## Acceptance

- The two AIW plugin commands execute the implementations in
  `src/programs/ai-code-tools` in both source and installed layouts.
- `plugins` and `all` install the canonical implementations as
  `aiw-ai-code-index.py` and `aiw-ai-gen-index.py` in the AIW plugin directory;
  by default this is `C:\green\aiw\plugins\`. No separate program package is
  installed.
- `ai-code-index update --run-generator` can invoke the bundled generator.

## TODO

- [x] Work Items 1.1–1.4 implemented; independent review remains pending.

## Verification

- Compile-only command: `python -m py_compile build.py src/plugins/_ai_code_tools.py src/plugins/aiw-ai-gen-index.py src/plugins/aiw-ai-code-index.py src/programs/ai-code-tools/ai-code-index src/programs/ai-code-tools/generate-ai-index` (exit 0).
- Static review confirmed the standalone installation action is absent and the canonical scripts are staged under `dist/plugins/` with AIW plugin filenames, then copied directly into the installed plugin root.
- `build.py plugins` and `build.py all` were not run; installation has not been executed or runtime-verified.
- No tests were run.

**Completed:** 2026-10-10
**Disposition reason:** PM override requested by user: archive direct main-workspace implementation; no independent Reviewer, report, or handoff evidence.
