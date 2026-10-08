"""FD-034 public patch command behavior in disposable, offline Git repositories.

Run only after a Planner authorizes this exact command and FD revision:
python -B -m unittest tests.test_fd034_git_patch_blackbox -v
"""

import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


PROJECT = Path(__file__).resolve().parents[1]
PLUGIN = PROJECT / "plugins" / "aiw-git" / "aiw-git.py"


class FD034GitPatchBlackBox(unittest.TestCase):
    def setUp(self):
        scratch = tempfile.TemporaryDirectory(prefix="fd034-git-patch-")
        self.addCleanup(scratch.cleanup)
        self.scratch = Path(scratch.name)
        template = self.scratch / "git-template"
        template.mkdir()
        hooks = self.scratch / "hooks"
        hooks.mkdir()
        self.env = os.environ.copy()
        for name in tuple(self.env):
            if name.startswith("GIT_"):
                self.env.pop(name)
        self.env.update({
            "GIT_CONFIG_NOSYSTEM": "1",
            "GIT_CONFIG_GLOBAL": os.devnull,
            "GIT_CONFIG_COUNT": "1",
            "GIT_CONFIG_KEY_0": "core.hooksPath",
            "GIT_CONFIG_VALUE_0": str(hooks),
            "GIT_TEMPLATE_DIR": str(template),
            "GIT_TERMINAL_PROMPT": "0",
            "PYTHONDONTWRITEBYTECODE": "1",
        })
        self.repo = self.make_repo("source")

    def run_command(self, args, cwd, *, check=True):
        result = subprocess.run(
            [str(arg) for arg in args], cwd=cwd, env=self.env,
            capture_output=True, text=True, encoding="utf-8", errors="replace",
            timeout=20, check=False,
        )
        if check:
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result

    def git(self, *args, cwd=None, check=True):
        return self.run_command(["git", *args], cwd or self.repo, check=check)

    def patch(self, *args, cwd=None, check=True):
        return self.run_command(
            [sys.executable, "-B", PLUGIN, "patch", *args],
            cwd or self.repo, check=check,
        )

    def make_repo(self, name):
        repo = self.scratch / name
        repo.mkdir()
        self.git("init", "-q", cwd=repo)
        self.git("config", "user.name", "FD Tester", cwd=repo)
        self.git("config", "user.email", "fd-tester@example.invalid", cwd=repo)
        (repo / "alpha.txt").write_text("base alpha\n", encoding="utf-8")
        (repo / "beta.txt").write_text("base beta\n", encoding="utf-8")
        (repo / "image.bin").write_bytes(b"\x00base-binary\x00")
        self.git("add", "alpha.txt", "beta.txt", "image.bin", cwd=repo)
        self.git("commit", "-q", "-m", "base", cwd=repo)
        return repo

    def output(self, name):
        return self.scratch / name

    def assert_no_output(self, path):
        self.assertFalse(path.exists(), str(path))
        self.assertFalse(Path(str(path) + ".md").exists(), str(path) + ".md")

    def test_default_exports_tracked_staged_unstaged_and_binary_with_guide(self):
        (self.repo / "alpha.txt").write_text("staged alpha\n", encoding="utf-8")
        self.git("add", "alpha.txt")
        (self.repo / "beta.txt").write_text("unstaged beta\n", encoding="utf-8")
        (self.repo / "image.bin").write_bytes(b"\x00changed-binary\x00")
        (self.repo / "untracked.txt").write_text("SECRET_UNTRACKED\n", encoding="utf-8")
        destination = self.output("default.patch")

        self.patch("create", destination)

        content = destination.read_text(encoding="utf-8")
        guide = Path(str(destination) + ".md").read_text(encoding="utf-8")
        self.assertEqual(content, self.git("diff", "--binary", "--full-index", "HEAD").stdout)
        self.assertIn("GIT binary patch", content)
        self.assertNotIn("untracked.txt", content)
        self.assertIn("alpha.txt", guide)
        self.assertIn("git apply", guide)
        self.assertIn("未跟踪", guide)

    def test_staged_and_worktree_select_independent_diffs(self):
        (self.repo / "alpha.txt").write_text("staged alpha\n", encoding="utf-8")
        self.git("add", "alpha.txt")
        (self.repo / "beta.txt").write_text("unstaged beta\n", encoding="utf-8")
        staged = self.output("staged.patch")
        worktree = self.output("worktree.patch")

        self.patch("create", staged, "--staged")
        self.patch("create", worktree, "--worktree")

        self.assertEqual(staged.read_text(encoding="utf-8"),
                         self.git("diff", "--binary", "--full-index", "--cached").stdout)
        self.assertEqual(worktree.read_text(encoding="utf-8"),
                         self.git("diff", "--binary", "--full-index").stdout)
        self.assertTrue(Path(str(staged) + ".md").exists())
        self.assertTrue(Path(str(worktree) + ".md").exists())

    def test_two_refs_use_direct_tree_diff_and_record_fixed_ids(self):
        base = self.git("rev-parse", "HEAD").stdout.strip()
        self.git("switch", "-q", "-c", "left")
        (self.repo / "alpha.txt").write_text("left alpha\n", encoding="utf-8")
        self.git("commit", "-q", "-am", "left")
        left_id = self.git("rev-parse", "HEAD").stdout.strip()
        self.git("switch", "-q", "-c", "right", base)
        (self.repo / "beta.txt").write_text("right beta\n", encoding="utf-8")
        self.git("commit", "-q", "-am", "right")
        right_id = self.git("rev-parse", "HEAD").stdout.strip()
        (self.repo / "beta.txt").write_text("unrelated worktree edit\n", encoding="utf-8")
        expected = self.git("diff", "--binary", "--full-index", left_id, right_id).stdout

        branch_patch = self.output("branches.patch")
        self.patch("create", branch_patch, "--from", "left", "--to", "right")
        guide = Path(str(branch_patch) + ".md").read_text(encoding="utf-8")
        self.assertEqual(branch_patch.read_text(encoding="utf-8"), expected)
        for value in ("left", "right", left_id, right_id):
            self.assertIn(value, guide)
        self.assertNotIn("unrelated worktree edit", branch_patch.read_text(encoding="utf-8"))

        ids_patch = self.output("ids.patch")
        self.patch("create", ids_patch, "--from", left_id, "--to", right_id)
        self.assertEqual(ids_patch.read_text(encoding="utf-8"), expected)

    def test_invalid_ref_options_fail_without_output(self):
        cases = [
            ("--from", "HEAD"),
            ("--to", "HEAD"),
            ("--from", "HEAD", "--to", "HEAD", "--staged"),
            ("--from", "HEAD", "--to", "HEAD", "--worktree"),
            ("--from", "missing-fd034-ref", "--to", "HEAD"),
        ]
        for number, options in enumerate(cases):
            with self.subTest(options=options):
                destination = self.output(f"invalid-{number}.patch")
                result = self.patch("create", destination, *options, check=False)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assert_no_output(destination)

    def test_empty_diff_git_failure_and_existing_outputs_do_not_overwrite(self):
        empty = self.output("empty.patch")
        self.assertNotEqual(self.patch("create", empty, check=False).returncode, 0)
        self.assert_no_output(empty)

        (self.repo / "alpha.txt").write_text("changed alpha\n", encoding="utf-8")
        existing_patch = self.output("existing.patch")
        existing_patch.write_text("keep patch", encoding="utf-8")
        self.assertNotEqual(self.patch("create", existing_patch, check=False).returncode, 0)
        self.assertEqual(existing_patch.read_text(encoding="utf-8"), "keep patch")
        self.assertFalse(Path(str(existing_patch) + ".md").exists())

        existing_guide = self.output("existing-guide.patch")
        guide_path = Path(str(existing_guide) + ".md")
        guide_path.write_text("keep guide", encoding="utf-8")
        self.assertNotEqual(self.patch("create", existing_guide, check=False).returncode, 0)
        self.assertFalse(existing_guide.exists())
        self.assertEqual(guide_path.read_text(encoding="utf-8"), "keep guide")

        nonrepo = self.scratch / "nonrepo"
        nonrepo.mkdir()
        git_failure = self.output("git-failure.patch")
        self.assertNotEqual(self.patch("create", git_failure, cwd=nonrepo, check=False).returncode, 0)
        self.assert_no_output(git_failure)

    def make_alpha_patch(self):
        (self.repo / "alpha.txt").write_text("patched alpha\n", encoding="utf-8")
        destination = self.output("apply.patch")
        self.patch("create", destination)
        return destination

    def test_apply_changes_only_worktree_and_recognizes_already_applied(self):
        destination = self.make_alpha_patch()
        target = self.make_repo("target")
        original_head = self.git("rev-parse", "HEAD", cwd=target).stdout

        self.patch("apply", destination, cwd=target)

        self.assertEqual((target / "alpha.txt").read_text(encoding="utf-8"), "patched alpha\n")
        self.assertEqual(self.git("diff", "--cached", cwd=target).stdout, "")
        self.assertEqual(self.git("rev-parse", "HEAD", cwd=target).stdout, original_head)
        before = self.git("status", "--porcelain", cwd=target).stdout
        repeated = self.patch("apply", destination, cwd=target, check=False)
        self.assertNotEqual(repeated.returncode, 0)
        self.assertIn("already be applied", (repeated.stdout + repeated.stderr).lower())
        self.assertEqual(self.git("status", "--porcelain", cwd=target).stdout, before)

    def test_failed_precheck_preserves_target_and_reports_recovery(self):
        destination = self.make_alpha_patch()
        target = self.make_repo("conflict-target")
        (target / "alpha.txt").write_text("different target version\n", encoding="utf-8")
        before_status = self.git("status", "--porcelain", cwd=target).stdout
        before_head = self.git("rev-parse", "HEAD", cwd=target).stdout

        result = self.patch("apply", destination, cwd=target, check=False)

        self.assertNotEqual(result.returncode, 0)
        self.assertEqual((target / "alpha.txt").read_text(encoding="utf-8"),
                         "different target version\n")
        self.assertEqual(self.git("status", "--porcelain", cwd=target).stdout, before_status)
        self.assertEqual(self.git("diff", "--cached", cwd=target).stdout, "")
        self.assertEqual(self.git("rev-parse", "HEAD", cwd=target).stdout, before_head)
        feedback = result.stdout + result.stderr
        self.assertIn("patch", feedback.lower())
        self.assertTrue("git diff" in feedback or "版本" in feedback, feedback)

    def test_empty_and_missing_patch_are_rejected(self):
        empty = self.output("empty-input.patch")
        empty.write_bytes(b"")
        before = self.git("status", "--porcelain").stdout
        for path in (empty, self.output("missing-input.patch")):
            with self.subTest(path=path):
                result = self.patch("apply", path, check=False)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.git("status", "--porcelain").stdout, before)

    def test_help_exposes_patch_creation_and_application(self):
        help_text = self.patch("--help").stdout
        create_help = self.patch("create", "--help").stdout
        apply_help = self.patch("apply", "--help").stdout
        for value in ("patch", "create", "apply"):
            self.assertIn(value, help_text + create_help + apply_help)
        for value in ("--staged", "--worktree", "--from", "--to"):
            self.assertIn(value, help_text + create_help)


if __name__ == "__main__":
    unittest.main()
