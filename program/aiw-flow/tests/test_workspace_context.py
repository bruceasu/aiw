import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import patch

from codex_flow.workspace_context import ContextLimits, collect_workspace_context, redact_sensitive_text


class WorkspaceContextTests(unittest.TestCase):
    def test_collects_allow_list_and_skips_sensitive_files(self):
        with TemporaryDirectory() as temp_dir:
            workspace = Path(temp_dir)
            (workspace / "README.md").write_text("Project notes", encoding="utf-8")
            (workspace / ".env").write_text("OPENAI_API_KEY=do-not-read", encoding="utf-8")
            (workspace / "src").mkdir()
            (workspace / "src" / "package.json").write_text('{"name": "demo"}', encoding="utf-8")

            with patch("codex_flow.workspace_context.run_command", side_effect=FileNotFoundError()):
                context = collect_workspace_context(workspace)

            self.assertIn("README.md", context)
            self.assertIn("src/package.json", context)
            self.assertNotIn("do-not-read", context)
            self.assertIn("Git metadata unavailable", context)

    def test_enforces_entry_and_byte_limits(self):
        with TemporaryDirectory() as temp_dir:
            workspace = Path(temp_dir)
            for index in range(5):
                (workspace / "file-{}.txt".format(index)).write_text("ignored", encoding="utf-8")
            (workspace / "README.md").write_text("x" * 100, encoding="utf-8")

            limits = ContextLimits(max_depth=1, max_entries=3, max_file_bytes=10, max_total_bytes=10)
            with patch("codex_flow.workspace_context.run_command", side_effect=FileNotFoundError()):
                context = collect_workspace_context(workspace, limits)

            self.assertIn("[TRUNCATED: entry limit reached]", context)
            self.assertNotIn("x" * 11, context)

    def test_redacts_assignments_bearer_tokens_and_private_keys(self):
        source = (
            'OPENAI_API_KEY="secret-value"\n'
            '"password": "hunter2"\n'
            "Authorization: Bearer abc.def.ghi\n"
            "-----BEGIN PRIVATE KEY-----\nprivate\n-----END PRIVATE KEY-----"
        )

        redacted = redact_sensitive_text(source)

        self.assertNotIn("secret-value", redacted)
        self.assertNotIn("hunter2", redacted)
        self.assertNotIn("abc.def.ghi", redacted)
        self.assertNotIn("\nprivate\n", redacted)
        self.assertGreaterEqual(redacted.count("[REDACTED]"), 3)

    def test_rejects_missing_workspace(self):
        with TemporaryDirectory() as temp_dir:
            with self.assertRaises(ValueError):
                collect_workspace_context(Path(temp_dir) / "missing")
