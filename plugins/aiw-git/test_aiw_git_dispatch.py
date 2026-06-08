#!/usr/bin/env python3
import contextlib
import importlib.util
import io
import os
import tempfile
import textwrap
import unittest


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


if __name__ == "__main__":
    unittest.main()
