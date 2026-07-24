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
        ]

        for argv, expected in cases:
            with self.subTest(argv=argv):
                self.assert_command(argv, expected)

    def test_file_history_can_disable_follow_and_preserve_unusual_paths(self):
        self.assert_command(
            ["file", "--stat", "--no-follow", "--", "-draft notes.md"],
            ["git", "log", "--stat", "--", "-draft notes.md"],
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

    def test_usage_errors_do_not_run_git(self):
        invalid_commands = [
            ["file"],
            ["file", "--patch", "--stat", "README.md"],
            ["file", "--unknown", "README.md"],
            ["blame"],
            ["file-at", "HEAD"],
            ["lines", "10,30"],
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
        for view in ("file", "blame", "file-at", "lines"):
            self.assertIn(view, output)
        self.assertIn("aiw git show file --full src/App.java", output)

    def test_existing_status_view_is_unchanged(self):
        self.assert_command(["status"], ["git", "status", "-sb"])


if __name__ == "__main__":
    unittest.main()
