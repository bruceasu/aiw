"""Git-fixture coverage for the opt-in merge-resolution handoff."""
import importlib.util
import os
import subprocess
import tempfile
import unittest
from pathlib import Path


PLUGIN = Path(__file__).with_name("aiw-wt.py")


class MergeResolutionFixtureTest(unittest.TestCase):
    def setUp(self):
        self.tempdir = tempfile.TemporaryDirectory()
        self.root = Path(self.tempdir.name)
        self.previous_cwd = Path.cwd()
        self.git("init", "--initial-branch=main")
        self.git("config", "user.email", "fixture@example.test")
        self.git("config", "user.name", "Fixture")
        self.write("safe.txt", "base\n")
        self.write(".ai/tasks/task-1/task.toml", self.metadata())
        self.write("openspec/changes/task-1/tasks.md", "# Tasks\n")
        self.git("add", ".")
        self.git("commit", "-m", "base")
        self.git("worktree", "add", "-b", "feature/task-1", ".wt/task-1", "main")
        self.git("-C", ".wt/task-1", "config", "user.email", "fixture@example.test")
        self.git("-C", ".wt/task-1", "config", "user.name", "Fixture")
        self.write(".wt/task-1/safe.txt", "task version\n")
        self.write(".wt/task-1/.ai/tasks/task-1/task.toml", self.metadata("task metadata"))
        self.git("-C", ".wt/task-1", "add", ".")
        self.git("-C", ".wt/task-1", "commit", "-m", "task changes")
        self.write("safe.txt", "parent version\n")
        self.write(".ai/tasks/task-1/task.toml", self.metadata("parent metadata"))
        self.git("add", ".")
        self.git("commit", "-m", "parent changes")
        os.chdir(self.root)
        spec = importlib.util.spec_from_file_location("aiw_wt_fixture", PLUGIN)
        self.plugin = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.plugin)

    def tearDown(self):
        os.chdir(self.previous_cwd)
        self.tempdir.cleanup()

    def test_opt_in_proposal_rejection_and_confirmed_application(self):
        self.assertEqual(2, self.plugin.pull("task-1", resolve_agent=True))
        handoff = self.root / ".ai/tasks/task-1/merge-resolution"
        request = (handoff / "proposal-request.md").read_text(encoding="utf-8")
        self.assertIn("### safe.txt", request)
        self.assertIn(".ai/tasks/task-1/task.toml", request)
        self.assertIn("Task metadata is protected", request)
        self.assertEqual(2, self.plugin.apply_merge_resolution("task-1", False))
        self.assertIn("UU safe.txt", self.status())

        (handoff / "proposal.patch").write_text(
            "diff --git a/safe.txt b/safe.txt\n--- a/safe.txt\n+++ b/safe.txt\n@@ -1,5 +1 @@\n"
            "-<<<<<<< HEAD\n-parent version\n-=======\n-task version\n->>>>>>> feature/task-1\n+resolved version\n",
            encoding="utf-8",
        )
        self.assertEqual(0, self.plugin.apply_merge_resolution("task-1", True))
        status = self.status()
        self.assertNotIn("UU safe.txt", status)
        self.assertIn("UU .ai/tasks/task-1/task.toml", status)
        self.assertEqual("resolved version\n", (self.root / "safe.txt").read_text(encoding="utf-8"))

    def metadata(self, note="base metadata"):
        return ('id = "task-1"\nbranch = "feature/task-1"\nparent_branch = "main"\n'
                'worktree = ".wt/task-1"\nworkspace_kind = "isolated"\n' f'# {note}\n')

    def write(self, relative, content):
        path = self.root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")

    def git(self, *args):
        subprocess.run(["git", *args], cwd=self.root, check=True, capture_output=True, text=True)

    def status(self):
        return subprocess.run(["git", "status", "--porcelain"], cwd=self.root, check=True, capture_output=True, text=True).stdout


if __name__ == "__main__":
    unittest.main()
