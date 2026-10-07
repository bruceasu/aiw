"""FD merge conflict recovery through the supported aiw-git dispatcher."""

import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


REPO = Path(__file__).resolve().parents[1]
GIT_DISPATCHER = REPO / "plugins" / "aiw-git" / "aiw-git.py"
FD_ID = "FD-030"


class FDConflictRecovery(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="fd030-conflict-")
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name) / "repo"
        self.root.mkdir()
        hooks = Path(self.scratch.name) / "hooks"
        hooks.mkdir()
        template = Path(self.scratch.name) / "template"
        template.mkdir()
        self.env = os.environ.copy()
        for key in tuple(self.env):
            if key.startswith("GIT_"):
                self.env.pop(key)
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
        fd = self.root / "docs" / "features" / f"{FD_ID}_BLACKBOX.md"
        fd.parent.mkdir(parents=True)
        fd.write_text(f"# {FD_ID}\n\n**Status:** Open\n", encoding="utf-8")
        (self.root / ".gitignore").write_text(".ai/\n.wt/\n", encoding="utf-8")
        self.git("add", ".gitignore", str(fd.relative_to(self.root)))
        self.git("commit", "-m", "record FD fixture")

    def command(self, *args, cwd=None):
        return subprocess.run(
            [sys.executable, str(GIT_DISPATCHER), "wt", *map(str, args)],
            cwd=cwd or self.root,
            env=self.env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
            timeout=30,
        )

    def git(self, *args, cwd=None):
        result = subprocess.run(
            ["git", *map(str, args)],
            cwd=cwd or self.root,
            env=self.env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
            timeout=30,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result.stdout.strip()

    def test_conflict_recovers_in_fd_tree_and_requires_explicit_retry(self):
        (self.root / "shared.txt").write_text("base\n", encoding="utf-8")
        self.git("add", "shared.txt")
        self.git("commit", "-m", "common base")
        self.assertEqual(self.command("add", FD_ID).returncode, 0)
        tree = self.root / ".wt" / FD_ID

        (tree / "shared.txt").write_text("feature\n", encoding="utf-8")
        self.git("add", "shared.txt", cwd=tree)
        self.git("commit", "-m", "feature edit", cwd=tree)
        (self.root / "shared.txt").write_text("parent\n", encoding="utf-8")
        self.git("add", "shared.txt")
        self.git("commit", "-m", "parent edit")
        parent_before = self.git("rev-parse", "HEAD")

        failed = self.command("local-merge", FD_ID, cwd=tree)
        self.assertNotEqual(failed.returncode, 0, failed.stdout + failed.stderr)
        self.assertTrue(tree.is_dir())
        self.assertEqual(self.git("rev-parse", "HEAD"), parent_before)
        self.assertEqual(self.git("status", "--porcelain"), "")
        self.assertNotEqual(self.git("ls-files", "--unmerged", cwd=tree), "")
        self.assertFalse((self.root / ".git" / "MERGE_HEAD").exists())

        (tree / "shared.txt").write_text("resolved\n", encoding="utf-8")
        self.git("add", "shared.txt", cwd=tree)
        self.git("commit", "-m", "resolve conflict", cwd=tree)
        source = self.git("rev-parse", "HEAD", cwd=tree)
        delivered = self.command("local-merge", FD_ID, cwd=tree)
        self.assertEqual(delivered.returncode, 0, delivered.stdout + delivered.stderr)
        self.assertEqual((self.root / "shared.txt").read_text(encoding="utf-8"), "resolved\n")
        self.assertFalse(tree.exists())
        self.assertEqual(self.git("branch", "--list", f"feature/{FD_ID}"), "")
        self.assertIn(f"FD-Source: {source}", self.git("log", "-1", "--format=%B"))
        self.assertEqual(len(self.git("rev-list", "--parents", "-n", "1", "HEAD").split()), 2)


if __name__ == "__main__":
    unittest.main()
