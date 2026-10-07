"""FD-029 public CLI behavior in disposable, offline Git repositories."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


GIT_DISPATCHER = Path(__file__).resolve().parents[1] / "plugins" / "aiw-git" / "aiw-git.py"
FD_ID = "FD-029"


class SquashDeliveryBlackBox(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="fd029-wt-")
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name) / "repo"
        self.root.mkdir()
        hooks = Path(self.scratch.name) / "hooks"
        hooks.mkdir()
        template = Path(self.scratch.name) / "template"
        template.mkdir()
        self.env = os.environ.copy()
        self.env.update({
            "GIT_CONFIG_NOSYSTEM": "1",
            "GIT_CONFIG_GLOBAL": os.devnull,
            "GIT_TEMPLATE_DIR": str(template),
            "GIT_CONFIG_COUNT": "1",
            "GIT_CONFIG_KEY_0": "core.hooksPath",
            "GIT_CONFIG_VALUE_0": str(hooks),
            "GIT_TERMINAL_PROMPT": "0",
        })
        self.git("init", "-b", "develop")
        self.git("config", "user.name", "FD Tester")
        self.git("config", "user.email", "fd-tester@example.invalid")
        fd = self.root / "docs/features/FD-029_BLACKBOX.md"
        fd.parent.mkdir(parents=True)
        fd.write_text("# FD-029\n\n**Status:** Open\n", encoding="utf-8")
        (self.root / ".gitignore").write_text(".ai/\n.wt/\n", encoding="utf-8")
        self.git("add", ".gitignore", "docs/features/FD-029_BLACKBOX.md")
        self.git("commit", "-m", "record FD")

    def command(self, *args, cwd=None):
        return subprocess.run(
            [str(arg) for arg in args], cwd=cwd or self.root, env=self.env,
            text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            check=False, timeout=30,
        )

    def git(self, *args, cwd=None):
        result = self.command("git", *args, cwd=cwd)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result.stdout.strip()

    def wt(self, *args, cwd=None):
        return self.command(sys.executable, GIT_DISPATCHER, "wt", *args, cwd=cwd)

    def wt_ok(self, *args, cwd=None):
        result = self.wt(*args, cwd=cwd)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result

    def wt_error(self, *args, cwd=None):
        result = self.wt(*args, cwd=cwd)
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        return result

    def add(self):
        self.wt_ok("add", FD_ID)
        tree = self.root / ".wt" / FD_ID
        self.assertTrue(tree.is_dir())
        return tree

    def commit_file(self, path, value, message, *, cwd):
        (cwd / path).write_text(value, encoding="utf-8")
        self.git("add", path, cwd=cwd)
        self.git("commit", "-m", message, cwd=cwd)
        return self.git("rev-parse", "HEAD", cwd=cwd)

    def test_multiple_work_items_become_one_single_parent_delivery(self):
        tree = self.add()
        parent_before = self.git("rev-parse", "HEAD")
        item_one = self.commit_file("item-one.txt", "one\n", "Work Item 1", cwd=tree)
        item_two = self.commit_file("item-two.txt", "two\n", "Work Item 2", cwd=tree)
        self.assertNotEqual(item_one, item_two)
        self.wt_ok("local-merge", FD_ID, cwd=tree)

        delivery = self.git("rev-parse", "HEAD")
        self.assertNotEqual(delivery, parent_before)
        self.assertEqual(self.git("rev-list", "--parents", "-n", "1", delivery).split(),
                         [delivery, parent_before])
        message = self.git("log", "-1", "--format=%B")
        self.assertIn(FD_ID, message)
        self.assertIn(f"FD-Source: {item_two}", message)
        self.assertEqual(self.git("log", "--format=%H", f"{parent_before}..{delivery}").splitlines(),
                         [delivery])
        for item in (item_one, item_two):
            ancestor = self.command("git", "merge-base", "--is-ancestor", item, delivery)
            self.assertEqual(ancestor.returncode, 1, ancestor.stdout + ancestor.stderr)
        self.assertEqual((self.root / "item-one.txt").read_text(encoding="utf-8"), "one\n")
        self.assertEqual((self.root / "item-two.txt").read_text(encoding="utf-8"), "two\n")
        self.assertEqual(self.git("rev-parse", "HEAD", cwd=tree), item_two)

    def test_delivery_uses_recorded_parent_branch(self):
        self.git("switch", "-c", "delivery")
        tree = self.add()
        record = json.loads((self.root / ".ai/fd/FD-029/workspace.json").read_text(encoding="utf-8"))
        self.assertEqual(record["parent_branch"], "delivery")
        develop_before = self.git("rev-parse", "develop")
        self.commit_file("feature.txt", "delivered\n", "Work Item", cwd=tree)
        self.wt_ok("local-merge", FD_ID)
        self.assertEqual(self.git("rev-parse", "develop"), develop_before)
        self.assertEqual((self.root / "feature.txt").read_text(encoding="utf-8"), "delivered\n")

    def test_successful_delivery_is_terminal_after_worktree_cleanup(self):
        tree = self.add()
        source = self.commit_file("feature.txt", "delivered\n", "Work Item", cwd=tree)
        self.wt_ok("local-merge", FD_ID, cwd=tree)
        delivered = self.git("rev-parse", "HEAD")
        self.assertFalse(tree.exists())
        self.assertEqual(self.git("branch", "--list", f"feature/{FD_ID}"), "")
        self.assertFalse((self.root / ".ai" / "fd" / FD_ID / "workspace.json").exists())
        self.assertTrue((self.root / ".ai" / "fd" / FD_ID / "events").is_dir())
        rejected = self.wt_error("local-merge", FD_ID, cwd=self.root)
        self.assertIn("not registered", rejected.stderr)
        self.assertEqual(self.git("rev-parse", "HEAD"), delivered)
        self.assertEqual(self.git("status", "--porcelain"), "")
        self.assertIn(f"FD-Source: {source}", self.git("log", "-1", "--format=%B"))

    def test_dirty_parent_or_fd_tree_is_rejected_before_delivery(self):
        tree = self.add()
        source = self.commit_file("feature.txt", "feature\n", "Work Item", cwd=tree)
        parent_before = self.git("rev-parse", "HEAD")
        (self.root / "uncommitted.txt").write_text("parent dirty\n", encoding="utf-8")
        self.wt_error("local-merge", FD_ID)
        self.assertEqual(self.git("rev-parse", "HEAD"), parent_before)
        (self.root / "uncommitted.txt").unlink()
        (tree / "uncommitted.txt").write_text("FD dirty\n", encoding="utf-8")
        self.wt_error("local-merge", FD_ID, cwd=tree)
        self.assertEqual(self.git("rev-parse", "HEAD"), parent_before)
        self.assertEqual(self.git("rev-parse", "HEAD", cwd=tree), source)
        self.assertFalse((self.root / "feature.txt").exists())

    def test_wrong_parent_or_fd_branch_is_rejected(self):
        tree = self.add()
        self.commit_file("feature.txt", "feature\n", "Work Item", cwd=tree)
        parent_before = self.git("rev-parse", "HEAD")
        self.git("switch", "-c", "wrong-parent")
        self.wt_error("local-merge", FD_ID)
        self.assertEqual(self.git("rev-parse", "HEAD"), parent_before)
        self.git("switch", "develop")
        self.git("switch", "-c", "wrong-feature", cwd=tree)
        self.wt_error("local-merge", FD_ID, cwd=tree)
        self.assertEqual(self.git("rev-parse", "HEAD"), parent_before)
        self.assertFalse((self.root / "feature.txt").exists())

    def test_conflict_restores_parent_and_requires_explicit_retry(self):
        self.commit_file("shared.txt", "base\n", "common base", cwd=self.root)
        tree = self.add()
        self.commit_file("shared.txt", "feature\n", "Work Item", cwd=tree)
        parent_before = self.commit_file("shared.txt", "parent\n", "parent edit", cwd=self.root)

        self.wt_error("local-merge", FD_ID, cwd=tree)
        self.assertEqual(self.git("rev-parse", "HEAD"), parent_before)
        self.assertEqual(self.git("status", "--porcelain"), "")
        self.assertEqual((self.root / "shared.txt").read_text(encoding="utf-8"), "parent\n")
        self.assertNotEqual(self.git("ls-files", "--unmerged", cwd=tree), "")
        self.assertFalse((self.root / ".git" / "MERGE_HEAD").exists())

        (tree / "shared.txt").write_text("resolved\n", encoding="utf-8")
        self.git("add", "shared.txt", cwd=tree)
        self.git("commit", "-m", "resolve against parent", cwd=tree)
        resolved_head = self.git("rev-parse", "HEAD", cwd=tree)
        self.assertEqual(self.git("rev-parse", "HEAD"), parent_before)
        self.wt_ok("local-merge", FD_ID, cwd=tree)
        self.assertEqual((self.root / "shared.txt").read_text(encoding="utf-8"), "resolved\n")
        self.assertIn(f"FD-Source: {resolved_head}", self.git("log", "-1", "--format=%B"))
        self.assertEqual(len(self.git("rev-list", "--parents", "-n", "1", "HEAD").split()), 2)


if __name__ == "__main__":
    unittest.main()
