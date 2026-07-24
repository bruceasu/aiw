import json
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from codex_flow.models import AppConfig, CreateSessionRequest
from codex_flow.session_store import SessionStore, SessionStoreError


def make_request(tmp_path: Path) -> CreateSessionRequest:
    workspace = tmp_path / "workspace"
    workspace.mkdir()
    return CreateSessionRequest(
        session_id="ABC-123-demo",
        title="Demo",
        instructions_text="rules",
        workspace_path=workspace,
        codex_config=AppConfig(),
    )


class SessionStoreTests(unittest.TestCase):
    def test_create_session_creates_status_and_memory(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            store = SessionStore(tmp_path / ".aiw-flow")
            status = store.create_session(make_request(tmp_path))
            session_dir = store.session_dir(status.session.id)
            self.assertTrue((session_dir / "status.json").exists())
            self.assertTrue((session_dir / "memory.md").exists())

    def test_update_status_keeps_unknown_fields(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            store = SessionStore(tmp_path / ".aiw-flow")
            status = store.create_session(make_request(tmp_path))
            path = store.status_path(status.session.id)
            raw = json.loads(path.read_text(encoding="utf-8"))
            raw["unknown"] = {"value": 1}
            path.write_text(json.dumps(raw), encoding="utf-8")
            updated = store.update_status(status.session.id, lambda current: current)
            self.assertEqual(updated.extra["unknown"]["value"], 1)

    def test_invalid_transition_raises(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            store = SessionStore(tmp_path / ".aiw-flow")
            status = store.create_session(make_request(tmp_path))
            with self.assertRaises(SessionStoreError):
                store.transition_state(status.session.id, "completed")

    def test_write_and_read_artifact_text(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            store = SessionStore(tmp_path / ".aiw-flow")
            status = store.create_session(make_request(tmp_path))

            path = store.write_artifact_text(status.session.id, "handoff.md", "handoff\n")

            self.assertEqual(path, store.session_dir(status.session.id) / "artifacts" / "handoff.md")
            self.assertEqual(store.read_artifact_text(status.session.id, "handoff.md"), "handoff\n")

    def test_artifact_filename_rejects_path_traversal(self):
        with TemporaryDirectory() as temp_dir:
            store = SessionStore(Path(temp_dir) / ".aiw-flow")
            with self.assertRaises(SessionStoreError):
                store.artifact_path("ABC-123-demo", "../handoff.md")
