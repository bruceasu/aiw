import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from codex_flow.handoff_manager import render_handoff
from codex_flow.models import AppConfig, CreateSessionRequest
from codex_flow.session_store import SessionStore


class HandoffManagerTests(unittest.TestCase):
    def test_renders_session_facts_and_focus(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            workspace = tmp_path / "workspace"
            workspace.mkdir()
            store = SessionStore(tmp_path / ".ai")
            status = store.create_session(
                CreateSessionRequest(
                    session_id="ABC-123-demo",
                    title="Demo requirement",
                    instructions_text="rules",
                    workspace_path=workspace,
                    codex_config=AppConfig(),
                )
            )
            session_dir = store.session_dir(status.session.id)
            (session_dir / "outputs" / "0001-final.txt").write_text("Latest answer", encoding="utf-8")
            (session_dir / "artifacts" / "workspace-context.md").write_text("context", encoding="utf-8")
            memory = """# Session Memory

## Goal

Ship the feature.

## Confirmed Findings

The API is stable.

## Decisions

Use CSV.

## Modified Files

None.

## Validation State

Not run.

## Open Issues

Choose encoding.
"""

            handoff = render_handoff(status, session_dir, memory, focus="Continue validation.")

            self.assertIn("Ship the feature.", handoff)
            self.assertIn("Use CSV.", handoff)
            self.assertIn("Continue validation.", handoff)
            self.assertIn("Latest answer", handoff)
            self.assertIn("artifacts/workspace-context.md", handoff)

    def test_truncates_latest_output(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            workspace = tmp_path / "workspace"
            workspace.mkdir()
            store = SessionStore(tmp_path / ".ai")
            status = store.create_session(
                CreateSessionRequest(
                    session_id="ABC-123-demo",
                    title="Demo",
                    instructions_text="rules",
                    workspace_path=workspace,
                    codex_config=AppConfig(),
                )
            )
            session_dir = store.session_dir(status.session.id)
            (session_dir / "outputs" / "0001-final.txt").write_text("x" * 100, encoding="utf-8")

            handoff = render_handoff(status, session_dir, "# Session Memory\n", max_output_chars=10)

            self.assertIn("x" * 10, handoff)
            self.assertNotIn("x" * 11, handoff)
            self.assertIn("[TRUNCATED: see complete output artifact]", handoff)
