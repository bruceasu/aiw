# AIW Git Subcommands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate git plugins into `plugins/aiw-git/`, rename subcommand scripts to `git-*`, and make `plugins/aiw-git/aiw-git.py` the single `aiw git` dispatcher with layered help.

**Architecture:** Keep Go plugin discovery unchanged and concentrate the change inside the git plugin bundle. Add a Python dispatcher that scans sibling `git-*` scripts, loads Python metadata for help, and dispatches execution to the chosen subcommand. Update affected docs to describe `aiw git` instead of old top-level `aiw-git-*` entrypoints.

**Tech Stack:** Python 3 stdlib, existing plugin metadata conventions, Go help/plugin integration, PowerShell verification commands

---

### Task 1: Add dispatcher behavior tests

**Files:**
- Create: `plugins/aiw-git/test_aiw_git_dispatch.py`
- Test: `plugins/aiw-git/aiw-git.py`

- [ ] **Step 1: Write the failing test**

```python
import importlib.util
import os
import tempfile
import textwrap
import unittest


def load_dispatcher():
    path = os.path.join(os.path.dirname(__file__), "aiw-git.py")
    spec = importlib.util.spec_from_file_location("aiw_git_dispatcher", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class DispatcherTest(unittest.TestCase):
    def test_collects_git_star_python_subcommands(self):
        dispatcher = load_dispatcher()
        with tempfile.TemporaryDirectory() as td:
            with open(os.path.join(td, "git-show.py"), "w", encoding="utf-8") as f:
                f.write("META={'name':'show','short':'show short','usage':'show','examples':['show']}\n")
            with open(os.path.join(td, "git-add-remote.py"), "w", encoding="utf-8") as f:
                f.write("META={'name':'add-remote','short':'add short','usage':'add-remote','examples':['add-remote']}\n")
            cmds = dispatcher.discover_subcommands(td)
            self.assertEqual(sorted(cmds.keys()), ["add-remote", "show"])
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python -m unittest plugins.aiw-git.test_aiw_git_dispatch`
Expected: FAIL because `discover_subcommands` does not exist yet or current dispatcher shape does not match.

- [ ] **Step 3: Write minimal implementation**

```python
def discover_subcommands(base_dir):
    raise NotImplementedError
```

- [ ] **Step 4: Run test to verify it passes**

Run: `python -m unittest plugins.aiw-git.test_aiw_git_dispatch`
Expected: dispatcher discovery test passes after real implementation is added in Task 2.

- [ ] **Step 5: Commit**

```bash
git add plugins/aiw-git/test_aiw_git_dispatch.py plugins/aiw-git/aiw-git.py
git commit -m "test: cover aiw git dispatcher discovery"
```

### Task 2: Rebuild the aiw git dispatcher

**Files:**
- Modify: `plugins/aiw-git/aiw-git.py`
- Modify: `plugins/aiw-git/aiw-git-core.py`

- [ ] **Step 1: Expand failing tests for help and dispatch**

```python
    def test_help_show_renders_detailed_command_help(self):
        dispatcher = load_dispatcher()
        with tempfile.TemporaryDirectory() as td:
            with open(os.path.join(td, "git-show.py"), "w", encoding="utf-8") as f:
                f.write(textwrap.dedent(\"\"\"\
                    META = {
                        'name': 'show',
                        'short': 'show short',
                        'long': 'show long',
                        'usage': 'show [args]',
                        'args': [{'flag': '--x', 'description': 'x'}],
                        'examples': ['show demo'],
                    }
                    def main(argv):
                        return 0
                \"\"\"))
            text = dispatcher.render_detailed_help(dispatcher.discover_subcommands(td)['show'])
            self.assertIn('show long', text)
            self.assertIn('show demo', text)
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `python -m unittest plugins.aiw-git.test_aiw_git_dispatch -v`
Expected: FAIL because help rendering and dispatch helpers are missing.

- [ ] **Step 3: Implement minimal dispatcher**

```python
def discover_subcommands(base_dir):
    ...

def render_overview_help(commands):
    ...

def render_detailed_help(command):
    ...

def main(argv):
    ...
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `python -m unittest plugins.aiw-git.test_aiw_git_dispatch -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add plugins/aiw-git/aiw-git.py plugins/aiw-git/aiw-git-core.py plugins/aiw-git/test_aiw_git_dispatch.py
git commit -m "feat: add aiw git dispatcher"
```

### Task 3: Move and rename git subcommand scripts

**Files:**
- Modify/Create/Delete: `plugins/aiw-git/*.py`
- Delete: root-level `plugins/aiw-git-*.py`

- [ ] **Step 1: Write the failing filesystem-oriented test**

```python
    def test_repo_git_show_module_is_discoverable(self):
        dispatcher = load_dispatcher()
        cmds = dispatcher.discover_subcommands(os.path.dirname(__file__))
        self.assertIn('show', cmds)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python -m unittest plugins.aiw-git.test_aiw_git_dispatch -v`
Expected: FAIL until repository files are renamed to `git-show.py` and friends.

- [ ] **Step 3: Perform the move and metadata normalization**

```text
Rename:
- plugins/aiw-git-show.py -> plugins/aiw-git/git-show.py
- plugins/aiw-git-add-mirror.py -> plugins/aiw-git/git-add-mirror.py
- ...
Update per-file META:
- name: plain subcommand name
- usage/examples: aiw git ...
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `python -m unittest plugins.aiw-git.test_aiw_git_dispatch -v`
Expected: PASS and `show` is discoverable from repository layout.

- [ ] **Step 5: Commit**

```bash
git add plugins/aiw-git
git commit -m "refactor: move git subcommands under aiw-git"
```

### Task 4: Update usage docs and run verification

**Files:**
- Modify: `docs/usage/aiw-git-*.md`
- Modify: `internal/commands/help/super-help.go` only if required by the new layout

- [ ] **Step 1: Update affected documentation**

```text
Replace examples like:
- aiw git-show ...
With:
- aiw git show ...
```

- [ ] **Step 2: Run targeted verification**

Run: `python -m unittest plugins.aiw-git.test_aiw_git_dispatch -v`
Expected: PASS

Run: `go test ./internal/commands/help ./internal/plugin`
Expected: PASS

Run: `python plugins/aiw-git/aiw-git.py -h`
Expected: concise overview help

Run: `python plugins/aiw-git/aiw-git.py help show`
Expected: detailed help for `show`

- [ ] **Step 3: Commit**

```bash
git add docs/usage plugins/aiw-git internal/commands/help internal/plugin
git commit -m "docs: update aiw git command usage"
```

## Self-Review

- Spec coverage: discovery naming, help layering, file relocation, and removal of old top-level git plugin entrypoints are all covered by Tasks 1-4.
- Placeholder scan: every task points to exact files and commands; no `TBD` markers remain.
- Type consistency: dispatcher helpers use one naming scheme: `discover_subcommands`, `render_overview_help`, and `render_detailed_help`.
