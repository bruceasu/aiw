import unittest

from codex_flow.interactive_loop import LOOP_HELP, LoopInput, LoopInputKind, parse_loop_input


class InteractiveLoopParserTests(unittest.TestCase):
    def test_parses_message_and_empty_input(self):
        self.assertEqual(parse_loop_input("  hello  "), LoopInput(LoopInputKind.MESSAGE, "hello"))
        self.assertEqual(parse_loop_input(" \n"), LoopInput(LoopInputKind.EMPTY))

    def test_parses_known_commands(self):
        expectations = {
            "/help": LoopInputKind.HELP,
            "/status": LoopInputKind.STATUS,
            "/memory": LoopInputKind.MEMORY,
            "/handoff": LoopInputKind.HANDOFF,
            "/skills": LoopInputKind.SKILLS,
            "/done": LoopInputKind.DONE,
            "/exit": LoopInputKind.EXIT,
        }
        for raw, kind in expectations.items():
            with self.subTest(raw=raw):
                self.assertEqual(parse_loop_input(raw), LoopInput(kind))

    def test_reports_unknown_command(self):
        self.assertEqual(
            parse_loop_input("/missing"),
            LoopInput(LoopInputKind.UNKNOWN, "/missing"),
        )
        self.assertEqual(
            parse_loop_input("/status now"),
            LoopInput(LoopInputKind.UNKNOWN, "/status now"),
        )

    def test_parses_skill_invocation_and_missing_arguments(self):
        self.assertEqual(
            parse_loop_input("/skill metrics-review Review the metrics"),
            LoopInput(
                LoopInputKind.SKILL,
                "metrics-review Review the metrics",
            ),
        )
        self.assertEqual(parse_loop_input("/skill"), LoopInput(LoopInputKind.SKILL))
        self.assertEqual(
            parse_loop_input("/skills now"),
            LoopInput(LoopInputKind.UNKNOWN, "/skills now"),
        )

    def test_double_slash_escapes_message(self):
        self.assertEqual(
            parse_loop_input("//review"),
            LoopInput(LoopInputKind.MESSAGE, "/review"),
        )
        self.assertEqual(
            parse_loop_input("//skill metrics-review Review this text"),
            LoopInput(
                LoopInputKind.MESSAGE,
                "/skill metrics-review Review this text",
            ),
        )
        self.assertIn("//text", LOOP_HELP)
