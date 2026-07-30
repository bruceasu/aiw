#!/usr/bin/env python3
import contextlib
import importlib.util
import io
import os
import tempfile
import textwrap
import unittest
from unittest import mock


def load_dispatcher():
    here = os.path.dirname(__file__)
    path = os.path.join(here, "aiw-git.py")
    spec = importlib.util.spec_from_file_location("aiw_git_dispatcher", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def write_python_command(path, name, short, long_text=None):
    content = textwrap.dedent(
        f"""\
        META = {{
            "name": "{name}",
            "short": "{short}",
            "long": "{long_text or short}",
            "usage": "{name} [args]",
            "args": [{{"flag": "--demo", "description": "demo flag"}}],
            "examples": ["{name} demo"],
        }}

        def main(argv):
            return 0
        """
    )
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)


class DispatcherTests(unittest.TestCase):
    def test_repo_layout_discovers_show_subcommand(self):
        dispatcher = load_dispatcher()
        commands = dispatcher.discover_subcommands(os.path.dirname(__file__))
        self.assertIn("show", commands)
        self.assertIn("guide", commands)

        guide = commands["guide"]["module"]
        match = guide.SearchMatch(
            path=guide.Path("guide.md"),
            score=1,
            filename_score=1,
            content_score=0,
            excerpt="example",
        )
        answer = guide.CodexAnswer(
            title="Example",
            slug="example",
            content="Example content",
        )

        self.assertFalse(match.extracted)
        self.assertTrue(answer.save)

    def test_discover_subcommands_scans_git_star_python_files(self):
        dispatcher = load_dispatcher()
        with tempfile.TemporaryDirectory() as td:
            write_python_command(os.path.join(td, "git-show.py"), "show", "show short")
            write_python_command(os.path.join(td, "git-add-remote.py"), "add-remote", "add short")
            with open(os.path.join(td, "aiw-git-core.py"), "w", encoding="utf-8") as f:
                f.write("# helper")

            commands = dispatcher.discover_subcommands(td)

            self.assertEqual(sorted(commands.keys()), ["add-remote", "show"])

    def test_render_detailed_help_includes_long_text_and_examples(self):
        dispatcher = load_dispatcher()
        with tempfile.TemporaryDirectory() as td:
            write_python_command(
                os.path.join(td, "git-show.py"),
                "show",
                "show short",
                long_text="show long help",
            )

            commands = dispatcher.discover_subcommands(td)
            rendered = dispatcher.render_detailed_help(commands["show"])

            self.assertIn("show long help", rendered)
            self.assertIn("show demo", rendered)

    def test_main_help_subcommand_renders_detailed_help(self):
        dispatcher = load_dispatcher()
        with tempfile.TemporaryDirectory() as td:
            write_python_command(os.path.join(td, "git-show.py"), "show", "show short")
            old_here = dispatcher.HERE
            dispatcher.HERE = td
            try:
                stdout = io.StringIO()
                with contextlib.redirect_stdout(stdout):
                    rc = dispatcher.main(["help", "show"])
            finally:
                dispatcher.HERE = old_here

            self.assertEqual(rc, 0)
            self.assertIn("show short", stdout.getvalue())

    def test_unknown_command_refuses_by_default(self):
        dispatcher = load_dispatcher()
        stdin = mock.Mock()
        stdin.isatty.return_value = True
        stdin.readline.return_value = "\n"
        stderr = io.StringIO()
        with mock.patch.object(dispatcher.sys, "stdin", stdin), mock.patch.object(
            dispatcher.shutil, "which", return_value="/usr/bin/git"
        ), mock.patch.object(dispatcher.subprocess, "run") as run_cmd, contextlib.redirect_stderr(
            stderr
        ):
            rc = dispatcher.main(["unknown-command", "--force"])

        self.assertEqual(rc, 2)
        run_cmd.assert_not_called()
        self.assertIn("Candidate native command", stderr.getvalue())
        self.assertIn("Native Git fallback refused", stderr.getvalue())

    def test_unknown_command_requires_tty_without_reading_input(self):
        dispatcher = load_dispatcher()
        stdin = mock.Mock()
        stdin.isatty.return_value = False
        stderr = io.StringIO()
        with mock.patch.object(dispatcher.sys, "stdin", stdin), mock.patch.object(
            dispatcher.shutil, "which", return_value="/usr/bin/git"
        ), mock.patch.object(dispatcher.subprocess, "run") as run_cmd, contextlib.redirect_stderr(
            stderr
        ):
            rc = dispatcher.main(["unknown-command"])

        self.assertEqual(rc, 2)
        stdin.readline.assert_not_called()
        run_cmd.assert_not_called()
        self.assertIn("requires interactive confirmation", stderr.getvalue())

    def test_approved_unknown_command_preserves_argv_and_exit_code(self):
        dispatcher = load_dispatcher()
        stdin = mock.Mock()
        stdin.isatty.return_value = True
        stdin.readline.return_value = "yes\n"
        proc = mock.Mock(returncode=17)
        stderr = io.StringIO()
        with mock.patch.object(dispatcher.sys, "stdin", stdin), mock.patch.object(
            dispatcher.shutil, "which", return_value="/usr/bin/git"
        ), mock.patch.object(dispatcher.subprocess, "run", return_value=proc) as run_cmd, contextlib.redirect_stderr(
            stderr
        ):
            rc = dispatcher.main(["custom", "--message", "value with spaces"])

        self.assertEqual(rc, 17)
        run_cmd.assert_called_once_with(["git", "custom", "--message", "value with spaces"])
        self.assertIn("fallback: delegating", stderr.getvalue())

    def test_unknown_help_uses_native_git_help_after_confirmation(self):
        dispatcher = load_dispatcher()
        stdin = mock.Mock()
        stdin.isatty.return_value = True
        stdin.readline.return_value = "y\n"
        proc = mock.Mock(returncode=0)
        with mock.patch.object(dispatcher.sys, "stdin", stdin), mock.patch.object(
            dispatcher.shutil, "which", return_value="/usr/bin/git"
        ), mock.patch.object(dispatcher.subprocess, "run", return_value=proc) as run_cmd:
            rc = dispatcher.main(["help", "custom"])

        self.assertEqual(rc, 0)
        run_cmd.assert_called_once_with(["git", "help", "custom"])


if __name__ == "__main__":
    unittest.main()
