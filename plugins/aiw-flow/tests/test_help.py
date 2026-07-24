import argparse
import contextlib
import io
import unittest
from pathlib import Path

from codex_flow.cli import build_parser


class HelpTests(unittest.TestCase):
    def render_help(self, *argv: str) -> str:
        stdout = io.StringIO()
        with contextlib.redirect_stdout(stdout):
            with self.assertRaises(SystemExit) as raised:
                build_parser().parse_args([*argv, "--help"])
        self.assertEqual(raised.exception.code, 0)
        return stdout.getvalue()

    def test_top_level_help_is_a_task_oriented_command_map(self):
        output = self.render_help()

        self.assertIn("Manage long-running Codex Sessions", output)
        self.assertIn("new -> run -> continue (repeat) -> finish -> archive", output)
        self.assertIn("Quick starts:", output)
        self.assertIn("aiw-flow loop TASK-123 --phase analyze", output)
        self.assertIn("Put --root before COMMAND", output)
        for command in (
            "new",
            "grill",
            "loop",
            "run",
            "continue",
            "status",
            "list",
            "inspect",
            "finish",
            "archive",
            "delete",
            "memory",
            "handoff",
            "daemon",
        ):
            self.assertRegex(output, r"(?m)^\s+{}\s+\S".format(command))

    def test_top_level_commands_have_actionable_help_and_examples(self):
        expected_text = {
            "new": ("--workspace PATH", "does not create a worktree"),
            "grill": ("--requirement TEXT", "but not both"),
            "loop": ("/help inside the loop", "aiw-flow loop TASK-123"),
            "run": ("--force-new-thread", "Supply a prompt"),
            "continue": ("already has a Thread ID", "--prompt-file FILE"),
            "status": ("--json", "scripts or CI"),
            "list": ("--state STATE", "one Session per line"),
            "inspect": ("last ten events", "aiw-flow inspect TASK-123"),
            "finish": ("does not commit, push, or clean", "--create-patch"),
            "archive": ("archive directory", "aiw-flow archive TASK-123"),
            "delete": ("unless --yes is present", "workspace, branch, and worktree are not removed"),
            "memory": ("show", "aiw-flow memory ACTION --help"),
            "handoff": ("without calling Codex", "aiw-flow handoff ACTION --help"),
            "daemon": ("does not start a background worker", "aiw-flow daemon status"),
        }

        for command, fragments in expected_text.items():
            with self.subTest(command=command):
                output = self.render_help(command)
                normalized = " ".join(output.split())
                self.assertIn("Examples:", output)
                for fragment in fragments:
                    self.assertIn(" ".join(fragment.split()), normalized)

    def test_nested_actions_have_summaries_arguments_and_examples(self):
        cases = {
            ("memory", "show"): ("SESSION_ID", "complete saved Memory"),
            ("memory", "append"): ("--text TEXT", "Note to append"),
            ("memory", "replace"): ("--file FILE", "Replace all current Memory"),
            ("handoff", "create"): ("--focus TEXT", "Resolve the encoding decision"),
            ("handoff", "show"): ("SESSION_ID", "Create it first"),
            ("daemon", "start"): ("No background process is started", "aiw-flow daemon start"),
            ("daemon", "status"): ("placeholder state", "aiw-flow daemon status"),
            ("daemon", "stop"): ("No process signal is sent", "aiw-flow daemon stop"),
        }

        for argv, fragments in cases.items():
            with self.subTest(argv=argv):
                output = self.render_help(*argv)
                normalized = " ".join(output.split())
                self.assertIn("Examples:", output)
                for fragment in fragments:
                    self.assertIn(" ".join(fragment.split()), normalized)

    def test_every_parser_argument_has_help_text(self):
        def assert_parser(parser: argparse.ArgumentParser) -> None:
            self.assertTrue(parser.description)
            self.assertTrue(parser.epilog)
            for action in parser._actions:
                if isinstance(action, argparse._SubParsersAction):
                    for child in action.choices.values():
                        assert_parser(child)
                elif action.dest != "help":
                    self.assertIsNotNone(action.help, action.dest)

        assert_parser(build_parser())

    def test_parser_values_remain_compatible(self):
        parser = build_parser()

        new_args = parser.parse_args(
            [
                "new",
                "--id",
                "TASK-123",
                "--title",
                "Fix login",
                "--workspace",
                "./worktree",
                "--instructions",
                "./instructions.md",
                "--loop",
                "--phase",
                "analyze",
                "--timeout",
                "900",
            ]
        )
        self.assertEqual(new_args.command, "new")
        self.assertEqual(new_args.id, "TASK-123")
        self.assertEqual(new_args.workspace, Path("./worktree"))
        self.assertEqual(new_args.instructions, Path("./instructions.md"))
        self.assertTrue(new_args.loop)
        self.assertFalse(new_args.ephemeral)
        self.assertEqual(new_args.phase, "analyze")
        self.assertEqual(new_args.timeout, 900)

        run_args = parser.parse_args(
            [
                "--root",
                "./state",
                "run",
                "TASK-123",
                "--phase",
                "implement",
                "--prompt-file",
                "./prompt.md",
                "--force-new-thread",
            ]
        )
        self.assertEqual(run_args.root, Path("./state"))
        self.assertEqual(run_args.command, "run")
        self.assertEqual(run_args.session_id, "TASK-123")
        self.assertEqual(run_args.prompt_file, Path("./prompt.md"))
        self.assertTrue(run_args.force_new_thread)

        handoff_args = parser.parse_args(
            ["handoff", "create", "TASK-123", "--focus", "Continue tests."]
        )
        self.assertEqual(handoff_args.command, "handoff")
        self.assertEqual(handoff_args.handoff_command, "create")
        self.assertEqual(handoff_args.session_id, "TASK-123")
        self.assertEqual(handoff_args.focus, "Continue tests.")

    def test_required_arguments_are_still_required(self):
        stderr = io.StringIO()
        with contextlib.redirect_stderr(stderr):
            with self.assertRaises(SystemExit) as raised:
                build_parser().parse_args(["new"])

        self.assertEqual(raised.exception.code, 2)
        self.assertIn("required", stderr.getvalue())


if __name__ == "__main__":
    unittest.main()
