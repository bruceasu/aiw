"""FD-027 public CLI acceptance in disposable local Git repositories."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


GIT_DISPATCHER = Path(__file__).resolve().parents[1] / "plugins" / "aiw-git" / "aiw-git.py"
FD_PLUGIN = Path(__file__).resolve().parents[1] / "plugins" / "aiw-fd.py"


class FDWorktreeBlackBox(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="fd027-wt-")
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name) / "repo"
        self.root.mkdir()
        hooks = Path(self.scratch.name) / "hooks"
        hooks.mkdir()
        template = Path(self.scratch.name) / "template"
        template.mkdir()
        self.env = os.environ.copy()
        self.env.update(
            {
                "GIT_CONFIG_NOSYSTEM": "1",
                "GIT_CONFIG_GLOBAL": os.devnull,
                "GIT_TEMPLATE_DIR": str(template),
                "GIT_CONFIG_COUNT": "1",
                "GIT_CONFIG_KEY_0": "core.hooksPath",
                "GIT_CONFIG_VALUE_0": str(hooks),
            }
        )
        self.git("init", "-b", "develop")
        self.git("config", "user.name", "FD Tester")
        self.git("config", "user.email", "fd-tester@example.invalid")
        fd = self.root / "docs" / "features" / "FD-027_WT_BLACKBOX.md"
        fd.parent.mkdir(parents=True)
        fd.write_text("# FD-027\n\n**Status:** Open\n", encoding="utf-8")
        (self.root / ".gitignore").write_text(".ai/\n.wt/\n", encoding="utf-8")
        self.git("add", ".gitignore", "docs/features/FD-027_WT_BLACKBOX.md")
        self.git("commit", "-m", "record FD")

    def run_command(self, *args, cwd=None):
        return subprocess.run(
            [str(arg) for arg in args],
            cwd=cwd or self.root,
            env=self.env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )

    def git(self, *args, cwd=None):
        result = self.run_command("git", *args, cwd=cwd)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result.stdout.strip()

    def wt(self, *args, cwd=None):
        return self.run_command(sys.executable, GIT_DISPATCHER, "wt", *args, cwd=cwd)

    def assert_wt_ok(self, *args, cwd=None):
        result = self.wt(*args, cwd=cwd)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result

    def assert_wt_error(self, *args, cwd=None):
        result = self.wt(*args, cwd=cwd)
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        return result

    def add(self):
        self.assert_wt_ok("add", "FD-027")
        return self.root / ".wt" / "FD-027"

    def test_add_records_exact_worktree_coordinates(self):
        tree = self.add()
        record = json.loads(
            (self.root / ".ai/fd/FD-027/workspace.json").read_text(encoding="utf-8")
        )
        self.assertEqual(record["fd_id"], "FD-027")
        self.assertEqual(record["parent_branch"], "develop")
        self.assertEqual(record["branch"], "feature/FD-027")
        self.assertEqual(Path(record["worktree"]).resolve(), tree.resolve())
        self.assertEqual(self.git("branch", "--show-current", cwd=tree), "feature/FD-027")

    def test_status_and_commit_use_recorded_fd_tree(self):
        tree = self.add()
        self.assert_wt_ok("status", "FD-027", cwd=tree)
        (tree / "change.txt").write_text("fd change\n", encoding="utf-8")
        self.assert_wt_ok("commit", "FD-027", "record FD change", cwd=self.root)
        self.assertEqual(self.git("log", "-1", "--format=%s", cwd=tree), "record FD change")
        self.assertFalse((self.root / "change.txt").exists())

    def test_rejects_task_and_unknown_fd_without_creating_tree(self):
        self.assert_wt_error("add", "TASK-027")
        self.assert_wt_error("add", "FD-999")
        self.assertFalse((self.root / ".wt" / "FD-027").exists())

    def test_add_rejects_dirty_parent(self):
        (self.root / "unrelated.txt").write_text("uncommitted\n", encoding="utf-8")
        self.assert_wt_error("add", "FD-027")
        self.assertFalse((self.root / ".wt" / "FD-027").exists())

    def assert_missing_ignore_blocks_add(self, rules, missing_rules):
        (self.root / ".gitignore").write_text(rules, encoding="utf-8")
        self.git("add", ".gitignore")
        self.git("commit", "-m", "set management ignore rules")
        parent_before = self.git("rev-parse", "HEAD")
        result = self.assert_wt_error("add", "FD-027")
        diagnostic = result.stdout + result.stderr
        for rule in missing_rules:
            self.assertIn(rule, diagnostic)
        self.assertEqual(self.git("rev-parse", "HEAD"), parent_before)
        self.assertEqual(self.git("status", "--porcelain"), "")
        self.assertFalse((self.root / ".wt" / "FD-027").exists())
        self.assertFalse((self.root / ".ai" / "fd" / "FD-027" / "workspace.json").exists())
        branches = self.git("branch", "--list", "feature/FD-027")
        self.assertEqual(branches, "")

    def test_add_rejects_when_both_management_paths_are_unignored(self):
        self.assert_missing_ignore_blocks_add("", (".wt/", ".ai/"))

    def test_add_rejects_when_worktree_path_is_unignored(self):
        self.assert_missing_ignore_blocks_add(".ai/\n", (".wt/",))

    def test_add_rejects_when_metadata_path_is_unignored(self):
        self.assert_missing_ignore_blocks_add(".wt/\n", (".ai/",))

    def test_rejects_malformed_record_before_commit(self):
        tree = self.add()
        record_path = self.root / ".ai/fd/FD-027/workspace.json"
        record = json.loads(record_path.read_text(encoding="utf-8"))
        record["branch"] = "feature/OTHER"
        record_path.write_text(json.dumps(record), encoding="utf-8")
        (tree / "change.txt").write_text("uncommitted\n", encoding="utf-8")
        self.assert_wt_error("commit", "FD-027", "should not commit")
        self.assertEqual(self.git("status", "--porcelain", cwd=tree), "?? change.txt")

    def test_rejects_wrong_branch_and_dirty_worktree_before_merge(self):
        tree = self.add()
        self.git("switch", "-c", "feature/OTHER", cwd=tree)
        self.assert_wt_error("local-merge", "FD-027")
        self.assertEqual(self.git("branch", "--show-current"), "develop")
        self.git("switch", "feature/FD-027", cwd=tree)
        (tree / "uncommitted.txt").write_text("dirty\n", encoding="utf-8")
        self.assert_wt_error("local-merge", "FD-027")
        self.assertEqual(self.git("branch", "--show-current"), "develop")

    def test_successful_local_merge_targets_recorded_parent(self):
        tree = self.add()
        (tree / "feature.txt").write_text("delivered\n", encoding="utf-8")
        self.git("add", "feature.txt", cwd=tree)
        self.git("commit", "-m", "feature", cwd=tree)
        self.assert_wt_ok("local-merge", "FD-027", cwd=tree)
        self.assertEqual((self.root / "feature.txt").read_text(encoding="utf-8"), "delivered\n")

    def test_fd_plugin_does_not_dispatch_worktree(self):
        result = self.run_command(sys.executable, FD_PLUGIN, "worktree", "status", "FD-027")
        self.assertNotEqual(result.returncode, 0)


if __name__ == "__main__":
    unittest.main()
