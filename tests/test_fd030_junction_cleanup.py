"""FD-030 public behavior for safe cleanup of a linked .ai directory."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


REPO = Path(__file__).resolve().parents[1]
GIT_DISPATCHER = REPO / "plugins" / "aiw-git" / "aiw-git.py"
FD_ID = "FD-030"


class JunctionCleanupBlackBox(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="fd030-junction-")
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

    @staticmethod
    def is_directory_link(path):
        is_junction = getattr(path, "is_junction", None)
        return bool(is_junction and is_junction()) or path.is_symlink()

    def test_successful_delivery_removes_link_body_and_preserves_shared_evidence(self):
        added = self.command("add", FD_ID)
        self.assertEqual(added.returncode, 0, added.stdout + added.stderr)
        tree = self.root / ".wt" / FD_ID
        linked_ai = tree / ".ai"
        shared_ai = self.root / ".ai"
        evidence = shared_ai / "fd" / FD_ID / "workspace.json"
        self.assertTrue(evidence.is_file())
        if not self.is_directory_link(linked_ai):
            self.skipTest("this platform did not create a linked .ai directory")
        record = json.loads(evidence.read_text(encoding="utf-8"))
        self.assertEqual(Path(record["worktree"]).resolve(), tree.resolve())
        self.assertEqual(linked_ai.resolve(), shared_ai.resolve())

        (tree / "delivered.txt").write_text("delivered\n", encoding="utf-8")
        self.git("add", "delivered.txt", cwd=tree)
        self.git("commit", "-m", "FD work", cwd=tree)
        delivered = self.command("local-merge", FD_ID, cwd=tree)

        self.assertEqual(delivered.returncode, 0, delivered.stdout + delivered.stderr)
        self.assertFalse(linked_ai.exists())
        self.assertFalse(tree.exists())
        self.assertTrue(evidence.is_file(), "cleanup removed shared FD evidence")
        self.assertEqual(self.git("branch", "--list", f"feature/{FD_ID}"), "")


if __name__ == "__main__":
    unittest.main()
