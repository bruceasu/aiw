import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from codex_flow.prompt_composer import compose_prompt, prompt_sha256, save_prompt


class PromptComposerTests(unittest.TestCase):
    def test_compose_prompt_order(self):
        result = compose_prompt("INST", "MEM", "implement", "Fix bug")
        self.assertLess(result.index("INST"), result.index("MEM"))
        self.assertLess(result.index("MEM"), result.index("implement"))
        self.assertLess(result.index("implement"), result.index("Fix bug"))

    def test_prompt_hash_stable(self):
        text = "hello"
        self.assertEqual(prompt_sha256(text), prompt_sha256(text))

    def test_save_prompt_writes_utf8(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            snapshot = save_prompt(tmp_path, 1, "analyze", "utf8 prompt")
            self.assertTrue(snapshot.path.exists())
            self.assertEqual(snapshot.path.read_text(encoding="utf-8"), "utf8 prompt")
