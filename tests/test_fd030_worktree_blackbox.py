"""FD-030 public CLI behavior in disposable, offline Git repositories."""

import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


FD_ID = "FD-030"
REPO = Path(__file__).resolve().parents[1]
GO_COMMAND = "go"


class FD030WorktreeBlackBox(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.runtime = tempfile.TemporaryDirectory(
            prefix=".fd030-aiw-runtime-", dir=REPO
        )
        cls.addClassCleanup(cls.runtime.cleanup)
        cls.runtime_root = Path(cls.runtime.name)
        cls.aiw_root = cls.runtime_root / "aiw-root"
        cls.aiw_root.mkdir()
        cls.aiw = cls.aiw_root / ("aiw.exe" if os.name == "nt" else "aiw")
        plugin_dir = cls.aiw_root / "plugins" / "aiw-git"
        plugin_dir.parent.mkdir(parents=True)
        shutil.copytree(REPO / "plugins" / "aiw-git", plugin_dir)

        cls.build_env = os.environ.copy()
        for name in tuple(cls.build_env):
            if name.startswith("GO"):
                cls.build_env.pop(name)
        cls.build_env.update({
            "GOCACHE": str(cls.runtime_root / "go-cache"),
            "GOWORK": "off",
            "GOENV": "off",
            "GOPROXY": "off",
            "GOSUMDB": "off",
            "GOTOOLCHAIN": "local",
        })
        built = subprocess.run(
            [GO_COMMAND, "build", "-mod=readonly", "-o", str(cls.aiw), "./cmd/aiw"],
            cwd=REPO / "src",
            env=cls.build_env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
            timeout=180,
        )
        if built.returncode != 0:
            raise RuntimeError("offline CLI build failed: " + built.stdout + built.stderr)

    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(
            prefix=".fd030-wt-", dir=REPO
        )
        self.addCleanup(self.scratch.cleanup)
        scratch = Path(self.scratch.name)
        self.root = scratch / "repo"
        self.root.mkdir()
        hooks = scratch / "hooks"
        hooks.mkdir()
        template = scratch / "template"
        template.mkdir()
        self.env = os.environ.copy()
        for name in tuple(self.env):
            if name.startswith("GIT_"):
                self.env.pop(name)
        self.env.update({
            "GIT_CONFIG_NOSYSTEM": "1",
            "GIT_CONFIG_GLOBAL": os.devnull,
            "GIT_TEMPLATE_DIR": str(template),
            "GIT_CONFIG_COUNT": "1",
            "GIT_CONFIG_KEY_0": "core.hooksPath",
            "GIT_CONFIG_VALUE_0": str(hooks),
            "GIT_TERMINAL_PROMPT": "0",
            "AIW_ROOT": str(self.aiw_root),
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

    def command(self, *args):
        return subprocess.run(
            [str(self.aiw), "git", "wt", *map(str, args)],
            cwd=self.root,
            env=self.env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
            timeout=60,
        )

    def test_standalone_aiw_wt_entrypoint_is_not_available(self):
        result = subprocess.run(
            [str(self.aiw), "wt", "list"],
            cwd=self.root,
            env=self.env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
            timeout=30,
        )
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)

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

    def test_failed_delivery_keeps_worktree_and_branch(self):
        result = self.command("add", FD_ID)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        worktree = self.root / ".wt" / FD_ID
        self.assertTrue(worktree.is_dir())

        (worktree / "uncommitted.txt").write_text("pending\n", encoding="utf-8")
        result = self.command("local-merge", FD_ID)

        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertTrue(worktree.is_dir())
        self.assertEqual(self.git("branch", "--list", f"feature/{FD_ID}"), f"feature/{FD_ID}")

    def test_successful_delivery_cleans_worktree_and_branch_but_keeps_receipts(self):
        result = self.command("add", FD_ID)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        worktree = self.root / ".wt" / FD_ID
        self.assertTrue(worktree.is_dir())

        result = self.command("status", FD_ID)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        (worktree / "delivered.txt").write_text("delivered\n", encoding="utf-8")
        result = self.command("commit", FD_ID, "record test change")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        source_sha = self.git("rev-parse", "HEAD", cwd=worktree)

        result = self.command("local-merge", FD_ID)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertFalse(worktree.exists())
        self.assertEqual(self.git("branch", "--list", f"feature/{FD_ID}"), "")
        self.assertTrue((self.root / ".ai" / "fd" / FD_ID).exists())
        subject = self.git("show", "-s", "--format=%B", "HEAD")
        self.assertIn(f"FD-Source: {source_sha}", subject)
        self.assertEqual(len(self.git("rev-list", "--parents", "-n", "1", "HEAD").split()), 2)


if __name__ == "__main__":
    unittest.main()
