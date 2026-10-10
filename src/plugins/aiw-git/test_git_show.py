#!/usr/bin/env python3
import contextlib
import importlib.util
import io
import os
import unittest
from unittest import mock


def load_git_show():
    here = os.path.dirname(__file__)
    path = os.path.join(here, "git-show.py")
    spec = importlib.util.spec_from_file_location("aiw_git_show_test", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class GitShowFileHistoryTests(unittest.TestCase):
    def setUp(self):
        self.git_show = load_git_show()

    def assert_command(self, argv, expected):
        with mock.patch.object(self.git_show.core, "run_cmd", return_value=0) as run_cmd:
            self.assertEqual(self.git_show.main(argv), 0)
        run_cmd.assert_called_once_with(expected)

    def test_file_history_modes(self):
        cases = [
            (["file", "README.md"], ["git", "log", "--follow", "--", "README.md"]),
            (
                ["file", "--oneline", "src/App.java"],
                ["git", "log", "--follow", "--oneline", "--", "src/App.java"],
            ),
            (
                ["file", "--patch", "README.md"],
                ["git", "log", "--follow", "-p", "--", "README.md"],
            ),
            (
                ["file", "--stat", "README.md"],
                ["git", "log", "--follow", "--stat", "--", "README.md"],
            ),
            (
                ["file", "--graph", "README.md"],
                [
                    "git",
                    "log",
                    "--follow",
                    "--graph",
                    "--decorate",
                    "--oneline",
                    "--",
                    "README.md",
                ],
            ),
            (
                ["file", "--full", "README.md"],
                ["git", "log", "--follow", "-p", "--stat", "--", "README.md"],
            ),
            (
                ["file", "-1", "README.md"],
                ["git", "log", "--follow", "--oneline", "--", "README.md"],
            ),
            (
                ["file", "-p", "README.md"],
                ["git", "log", "--follow", "-p", "--", "README.md"],
            ),
            (
                ["file", "-s", "README.md"],
                ["git", "log", "--follow", "--stat", "--", "README.md"],
            ),
            (
                ["file", "-g", "README.md"],
                [
                    "git",
                    "log",
                    "--follow",
                    "--graph",
                    "--decorate",
                    "--oneline",
                    "--",
                    "README.md",
                ],
            ),
            (
                ["file", "-f", "README.md"],
                ["git", "log", "--follow", "-p", "--stat", "--", "README.md"],
            ),
            (
                ["file-hist", "README.md"],
                ["git", "log", "--follow", "-p", "--stat", "--", "README.md"],
            ),
        ]

        for argv, expected in cases:
            with self.subTest(argv=argv):
                self.assert_command(argv, expected)

    def test_file_history_can_disable_follow_and_preserve_unusual_paths(self):
        self.assert_command(
            ["file", "--stat", "--no-follow", "--", "-draft notes.md"],
            ["git", "log", "--stat", "--", "-draft notes.md"],
        )
        self.assert_command(
            ["file-hist", "--no-follow", "--", "-draft notes.md"],
            ["git", "log", "-p", "--stat", "--", "-draft notes.md"],
        )

    def test_blame_and_historical_file_content(self):
        self.assert_command(
            ["blame", "src/App.java"],
            ["git", "blame", "--", "src/App.java"],
        )
        self.assert_command(
            ["file-at", "HEAD~3", "docs/Release Notes.md"],
            ["git", "show", "HEAD~3:docs/Release Notes.md"],
        )

    def test_line_range_and_function_history(self):
        self.assert_command(
            ["lines", "10,30", "src/App.java"],
            ["git", "log", "-L", "10,30:src/App.java"],
        )
        self.assert_command(
            ["lines", ":main", "src/main.py"],
            ["git", "log", "-L", ":main:src/main.py"],
        )

    def test_commit_files_lists_status_and_paths(self):
        self.assert_command(["commit-files", "HEAD~1"], ["git", "diff-tree", "--no-commit-id", "--name-status", "-r", "HEAD~1"])
        self.assert_command(["commit-files", "--names", "HEAD"], ["git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD"])
        self.assert_command(["commit-files", "--root", "HEAD"], ["git", "diff-tree", "--no-commit-id", "--name-status", "-r", "--root", "HEAD"])
    def test_usage_errors_do_not_run_git(self):
        invalid_commands = [
            ["file"],
            ["file-hist"],
            ["fh"],
            ["fh", "--no-follow", "README.md"],
            ["file", "--patch", "--stat", "README.md"],
            ["file", "--unknown", "README.md"],
            ["file-hist", "--unknown", "README.md"],
            ["in", "HEAD"],
            ["out", "HEAD"],
            ["c", "--check"],
            ["ck", "--check"],
            ["conflicts-diff", "--diff"],
            ["conflicts-staged", "--staged"],
            ["cf", "--names", "HEAD"],
            ["commit-files-names", "--names", "HEAD"],
            ["commit-files-root", "--root", "HEAD"],
            ["whatchanged-names", "--names", "HEAD"],
            ["blame"],
            ["file-at", "HEAD"],
            ["lines", "10,30"],
            ["commit-files", "HEAD", "HEAD~1"],
            ["commit-files", "--unknown"],
        ]

        for argv in invalid_commands:
            with self.subTest(argv=argv):
                stdout, stderr = io.StringIO(), io.StringIO()
                with mock.patch.object(self.git_show.core, "run_cmd") as run_cmd:
                    with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
                        self.assertEqual(self.git_show.main(argv), 2)
                run_cmd.assert_not_called()
                self.assertIn("error:", stderr.getvalue())
                self.assertIn("Usage:", stdout.getvalue())

    def test_help_lists_file_history_views_and_examples(self):
        stdout = io.StringIO()
        with contextlib.redirect_stdout(stdout):
            self.assertEqual(self.git_show.main(["--help"]), 0)

        output = stdout.getvalue()
        self.assertIn("Daily Quick Start:", output)
        self.assertIn("st", output)
        self.assertIn("lg", output)
        self.assertIn("ck", output)
        for view in ("file", "file-hist", "blame", "file-at", "lines"):
            self.assertIn(view, output)
        self.assertIn("aiw git show file --full src/App.java", output)

    def test_existing_status_view_is_unchanged(self):
        self.assert_command(["status"], ["git", "status", "-sb"])

    def test_subcommand_aliases(self):
        cases = [
            (["s"], ["git", "status", "-sb"]),
            (["st"], ["git", "status", "-sb"]),
            (
                ["l"],
                [
                    "git",
                    "log",
                    "--all",
                    "--color",
                    "--graph",
                    "--pretty=format:%Cred%h%Creset - %C(yellow)%d%Creset %s %Cgreen[%cr] %C(bold blue)<%an>%Creset",
                    "--abbrev-commit",
                    "--date=relative",
                ],
            ),
            (["f", "README.md"], ["git", "log", "--follow", "-p", "--stat", "--", "README.md"]),
            (["fh", "README.md"], ["git", "log", "--follow", "-p", "--stat", "--", "README.md"]),
            (["in"], ["git", "log", "--oneline", "HEAD..origin/main"]),
            (["out"], ["git", "log", "--oneline", "origin/main..HEAD"]),
        ]

        for argv, expected in cases:
            with self.subTest(argv=argv):
                if argv[0] in {"in", "out"}:
                    with mock.patch.object(self.git_show.core, "git_output", return_value="origin/main\n"):
                        self.assert_command(argv, expected)
                else:
                    self.assert_command(argv, expected)

    def test_short_daily_subcommands(self):
        self.assert_command(["st"], ["git", "status", "-sb"])
        self.assert_command(
            ["lg"],
            [
                "git",
                "log",
                "--all",
                "--color",
                "--graph",
                "--pretty=format:%Cred%h%Creset - %C(yellow)%d%Creset %s %Cgreen[%cr] %C(bold blue)<%an>%Creset",
                "--abbrev-commit",
                "--date=relative",
            ],
        )
        self.assert_command(
            ["lg", "-n", "20"],
            [
                "git",
                "log",
                "--all",
                "--color",
                "--graph",
                "--pretty=format:%Cred%h%Creset - %C(yellow)%d%Creset %s %Cgreen[%cr] %C(bold blue)<%an>%Creset",
                "--abbrev-commit",
                "--date=relative",
                "-n",
                "20",
            ],
        )
        self.assert_command(
            ["l1"],
            ["git", "log", "--pretty=oneline"],
        )
        self.assert_command(
            ["l1", "-n", "50"],
            ["git", "log", "--pretty=oneline", "-n", "50"],
        )
        self.assert_command(
            ["log-date"],
            [
                "git",
                "log",
                "--color",
                "--graph",
                "--pretty=format:%Cred%h%Creset %Cgreen%ad%Creset | %s %C(yellow)%d%Creset %C(bold blue)<%an>%Creset",
                "--date=short",
            ],
        )
        self.assert_command(
            ["log-date", "-n", "40"],
            [
                "git",
                "log",
                "--color",
                "--graph",
                "--pretty=format:%Cred%h%Creset %Cgreen%ad%Creset | %s %C(yellow)%d%Creset %C(bold blue)<%an>%Creset",
                "--date=short",
                "-n",
                "40",
            ],
        )
        self.assert_command(
            ["fh", "README.md"],
            ["git", "log", "--follow", "-p", "--stat", "--", "README.md"],
        )
        self.assert_command(
            ["file-oneline", "README.md"],
            ["git", "log", "--follow", "--oneline", "--", "README.md"],
        )
        self.assert_command(
            ["file-patch", "README.md"],
            ["git", "log", "--follow", "-p", "--", "README.md"],
        )
        self.assert_command(
            ["file-stat", "README.md"],
            ["git", "log", "--follow", "--stat", "--", "README.md"],
        )
        self.assert_command(
            ["file-graph", "README.md"],
            ["git", "log", "--follow", "--graph", "--decorate", "--oneline", "--", "README.md"],
        )
        self.assert_command(
            ["cf", "HEAD~1"],
            ["git", "diff-tree", "--no-commit-id", "--name-status", "-r", "HEAD~1"],
        )
        self.assert_command(
            ["commit-files-names", "HEAD~1"],
            ["git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD~1"],
        )
        self.assert_command(
            ["commit-files-root", "HEAD~1"],
            ["git", "diff-tree", "--no-commit-id", "--name-status", "-r", "--root", "HEAD~1"],
        )
        self.assert_command(
            ["ck"],
            ["git", "diff", "--check"],
        )
        self.assert_command(
            ["conflicts-diff"],
            ["git", "diff", "--diff-filter=U"],
        )
        self.assert_command(
            ["conflicts-staged"],
            ["git", "diff", "--staged"],
        )
        self.assert_command(
            ["whatchanged-names", "HEAD~1"],
            ["git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD~1"],
        )
        with mock.patch.object(self.git_show.core, "git_output", return_value="origin/main\n"):
            self.assert_command(["in"], ["git", "log", "--oneline", "HEAD..origin/main"])
            self.assert_command(["out"], ["git", "log", "--oneline", "origin/main..HEAD"])

    def test_short_conflict_list_with_files_avoids_run_cmd(self):
        output = io.StringIO()
        with mock.patch.object(self.git_show.core, "git_output", return_value="a.txt\nb.txt\n"), mock.patch.object(
            self.git_show.core, "run_cmd"
        ) as run_cmd, contextlib.redirect_stdout(output):
            self.assertEqual(self.git_show.main(["c"]), 0)

        run_cmd.assert_not_called()
        text = output.getvalue()
        self.assertIn("2 conflicted file(s)", text)
        self.assertIn("a.txt", text)
        self.assertIn("b.txt", text)

    def test_short_conflict_list_no_files(self):
        output = io.StringIO()
        with mock.patch.object(self.git_show.core, "git_output", return_value=""), contextlib.redirect_stdout(output):
            self.assertEqual(self.git_show.main(["c"]), 0)

        self.assertIn("No unresolved conflicts found.", output.getvalue())

    def test_short_commands_use_builtin_defaults(self):
        self.assert_command(
            ["l"],
            [
                "git",
                "log",
                "--all",
                "--color",
                "--graph",
                "--pretty=format:%Cred%h%Creset - %C(yellow)%d%Creset %s %Cgreen[%cr] %C(bold blue)<%an>%Creset",
                "--abbrev-commit",
                "--date=relative",
            ],
        )
        self.assert_command(["s"], ["git", "status", "-sb"])
        self.assert_command(["f", "README.md"], ["git", "log", "--follow", "-p", "--stat", "--", "README.md"])

    def test_log_style_aliases(self):
        self.assert_command(
            ["log", "g", "-n", "2"],
            [
                "git",
                "log",
                "--all",
                "--color",
                "--graph",
                "--pretty=format:%Cred%h%Creset - %C(yellow)%d%Creset %s %Cgreen[%cr] %C(bold blue)<%an>%Creset",
                "--abbrev-commit",
                "--date=relative",
                "-n",
                "2",
            ],
        )
        self.assert_command(
            ["log", "1", "-n", "5"],
            ["git", "log", "--pretty=oneline", "-n", "5"],
        )
        self.assert_command(
            ["log", "d", "-n", "4"],
            [
                "git",
                "log",
                "--color",
                "--graph",
                "--pretty=format:%Cred%h%Creset %Cgreen%ad%Creset | %s %C(yellow)%d%Creset %C(bold blue)<%an>%Creset",
                "--date=short",
                "-n",
                "4",
            ],
        )


if __name__ == "__main__":
    unittest.main()
