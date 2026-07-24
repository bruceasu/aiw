import unittest

from codex_flow.backends.base import parse_exec_jsonl


class ExecEventParserTests(unittest.TestCase):
    def test_parse_exec_jsonl_extracts_thread_and_output(self):
        parsed = parse_exec_jsonl(
            [
                '{"type":"thread.started","thread_id":"abc"}\n',
                '{"type":"item.completed","item":{"type":"agent_message","text":"done"}}\n',
            ]
        )
        self.assertEqual(parsed.thread_id, "abc")
        self.assertEqual(parsed.final_output, "done")

    def test_parse_exec_jsonl_keeps_invalid_lines(self):
        parsed = parse_exec_jsonl(['not-json\n'])
        self.assertEqual(parsed.invalid_lines, ["not-json"])
        self.assertEqual(parsed.final_output, "")
