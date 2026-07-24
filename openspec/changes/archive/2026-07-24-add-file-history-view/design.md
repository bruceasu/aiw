## Context

`plugins/aiw-git/git-show.py` dispatches a fixed set of read-only repository views and delegates external execution to `aiw-git-core.py`. The new capability must fit that structure, remain compatible with Python 3.9, preserve existing view behavior, and pass file paths as distinct process arguments without shell interpolation.

## Goals / Non-Goals

**Goals:**

- Expose common file-history workflows through discoverable `aiw git show` views.
- Build deterministic Git argument lists that are easy to unit test.
- Preserve filenames containing spaces and separate pathspecs with `--` where Git supports it.
- Return usage errors before invoking Git when required arguments are missing or conflicting.

**Non-Goals:**

- Installing global Git aliases or changing user Git configuration.
- Reimplementing Git history, rename detection, blame, or diff rendering.
- Adding interactive selection, pagination, or a graphical user interface.
- Changing the existing `log`, `status`, conflict, upstream, or `whatchanged` views.

## Decisions

1. Add four top-level views: `file`, `blame`, `file-at`, and `lines`.
   - `file` covers the related `git log` variants with mutually exclusive flags: `--oneline`, `--patch`, `--stat`, `--graph`, and `--full`.
   - `blame` maps to `git blame -- <path>`.
   - `file-at` maps to `git show <revision>:<path>`.
   - `lines` maps to `git log -L <selector>:<path>`.
   - Separate views keep required positional arguments explicit and avoid expanding the existing `log` parser.

2. `file` follows renames by default and accepts `--no-follow`.
   - Rename-aware history is the useful default for file evolution.
   - An opt-out preserves direct access to Git's path-limited behavior.

3. Forward commands as argument arrays through `core.run_cmd`.
   - This matches existing code, preserves spaces in paths, and avoids shell quoting or command injection.

4. Validate the wrapper's own syntax, but let Git validate revisions, paths, and `-L` selectors.
   - Duplicated Git validation would become incomplete and brittle.
   - Wrapper errors return status 2 and show the relevant help.

5. Test command construction by replacing `core.run_cmd`.
   - Unit tests remain deterministic and do not depend on repository history.
   - One help/dispatch test confirms the views remain discoverable.

## Risks / Trade-offs

- [Risk] `git log --follow` works for a single path and has Git-defined rename heuristics. → Require exactly one path and document the opt-out.
- [Risk] A `file-at` path is embedded in Git's `<revision>:<path>` object expression. → Keep revision and path as separate CLI inputs, combine them without a shell, and leave object-expression validation to Git.
- [Risk] `git log -L` has selector syntax that varies between line ranges and function names. → Forward one selector string unchanged and document examples for both forms.
- [Trade-off] The wrapper exposes curated modes rather than every Git option. → Users can continue using raw Git for uncommon combinations; `--full` covers the recommended combined view.
