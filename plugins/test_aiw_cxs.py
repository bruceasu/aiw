import importlib.util
import json
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock


MODULE_PATH = Path(__file__).with_name("aiw-cxs.py")
SPEC = importlib.util.spec_from_file_location("aiw_cxs", MODULE_PATH)
aiw_cxs = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = aiw_cxs
SPEC.loader.exec_module(aiw_cxs)


class CxsPluginTests(unittest.TestCase):
    def write_session(self, root: Path, session_id: str, cwd: Path) -> Path:
        path = root / f"rollout-{session_id}.jsonl"
        records = [
            {
                "type": "session_meta",
                "payload": {"id": session_id, "cwd": str(cwd)},
            },
            {
                "type": "response_item",
                "payload": {
                    "role": "user",
                    "content": [{"type": "input_text", "text": "Fix aliases"}],
                },
            },
            {
                "type": "response_item",
                "payload": {
                    "role": "assistant",
                    "content": [{"type": "output_text", "text": "Working on it"}],
                },
            },
        ]
        path.write_text(
            "".join(json.dumps(record) + "\n" for record in records),
            encoding="utf-8",
        )
        return path

    def test_session_metadata_extracts_original_cwd(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            workspace = root / "repo"
            workspace.mkdir()
            session_id = "123e4567-e89b-12d3-a456-426614174000"
            path = self.write_session(root, session_id, workspace)

            session = aiw_cxs.inspect_session_file(path, scan_events=20)

            self.assertEqual(session.session_id, session_id)
            self.assertEqual(session.original_cwd, workspace.resolve())

    def test_non_metadata_cwd_is_not_used_for_workspace_identity(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            unrelated = root / "wrong"
            path = root / "session.jsonl"
            records = [
                {"type": "session_meta", "payload": {"id": "session-1"}},
                {
                    "type": "response_item",
                    "payload": {
                        "role": "user",
                        "cwd": str(unrelated),
                        "content": "This is ordinary event data.",
                    },
                },
            ]
            path.write_text(
                "".join(json.dumps(record) + "\n" for record in records),
                encoding="utf-8",
            )

            session = aiw_cxs.inspect_session_file(path, scan_events=20)

            self.assertIsNone(session.original_cwd)

    def test_legacy_cache_without_cwd_is_rescanned(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            sessions_dir = root / "sessions"
            workspace_index = root / "repo" / ".ai"
            original_cwd = root / "repo"
            sessions_dir.mkdir()
            original_cwd.mkdir()
            session_id = "123e4567-e89b-12d3-a456-426614174001"
            path = self.write_session(sessions_dir, session_id, original_cwd)
            stat = path.stat()
            cache_path = aiw_cxs.workspace_cache_path(workspace_index)
            cache_path.parent.mkdir(parents=True)
            cache_path.write_text(
                json.dumps(
                    {
                        "schema_version": 1,
                        "files": {
                            str(path.resolve()): {
                                "session_id": session_id,
                                "path": str(path),
                                "mtime_ns": stat.st_mtime_ns,
                                "size": stat.st_size,
                                "title": "legacy",
                                "first_user": "",
                                "turns": 0,
                            }
                        },
                    }
                ),
                encoding="utf-8",
            )

            sessions = aiw_cxs.scan_sessions(sessions_dir, workspace_index)

            self.assertEqual(sessions[0].original_cwd, original_cwd.resolve())
            refreshed = json.loads(cache_path.read_text(encoding="utf-8"))
            cached = next(iter(refreshed["files"].values()))
            self.assertEqual(cached["original_cwd"], str(original_cwd.resolve()))

    def test_workspace_filter_includes_descendants_and_excludes_unknown(self):
        workspace = Path("C:/work/repo").resolve()

        def session(session_id, cwd):
            return aiw_cxs.SessionMeta(
                session_id=session_id,
                path=Path(f"{session_id}.jsonl"),
                mtime_ns=1,
                size=1,
                title=session_id,
                first_user="",
                turns=0,
                original_cwd=cwd,
            )

        inside = session("inside", workspace / "module")
        outside = session("outside", workspace.parent / "other")
        unknown = session("unknown", None)

        filtered = aiw_cxs.filter_sessions_for_workspace(
            [inside, outside, unknown], workspace
        )

        self.assertEqual(filtered, [inside])
        self.assertEqual(
            aiw_cxs.filter_sessions_for_workspace(
                [inside, outside, unknown], workspace, include_all=True
            ),
            [inside, outside, unknown],
        )

    def test_alias_rename_preserves_binding_and_rejects_conflict(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            workspace = Path(temp_dir) / ".ai"
            aiw_cxs.save_index(
                workspace,
                {
                    "aliases": {
                        "old": {"session_id": "one", "note": "keep"},
                        "taken": {"session_id": "two"},
                    }
                },
            )

            aiw_cxs.rename_alias(workspace, "old", "new")

            aliases = aiw_cxs.load_index(workspace, create=False)["aliases"]
            self.assertNotIn("old", aliases)
            self.assertEqual(aliases["new"]["session_id"], "one")
            self.assertEqual(aliases["new"]["note"], "keep")
            with self.assertRaises(aiw_cxs.AliasConflictError):
                aiw_cxs.rename_alias(workspace, "new", "taken")

    def test_preview_omits_system_and_tool_events(self):
        objects = [
            {"role": "system", "content": "secret"},
            {"role": "user", "content": "question"},
            {"role": "assistant", "content": "answer"},
            {"role": "tool", "content": "raw output"},
        ]

        preview = aiw_cxs.render_conversation(objects)

        self.assertIn("question", preview)
        self.assertIn("answer", preview)
        self.assertNotIn("secret", preview)
        self.assertNotIn("raw output", preview)

    def test_interactive_resume_plan_uses_original_directory(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            cwd = Path(temp_dir)
            session = aiw_cxs.SessionMeta(
                session_id="session-1",
                path=cwd / "session.jsonl",
                mtime_ns=1,
                size=1,
                title="Session",
                first_user="",
                turns=0,
                original_cwd=cwd,
            )

            plan = aiw_cxs.plan_interactive_resume(
                session, terminal_available=True
            )

            self.assertTrue(plan.can_launch)
            self.assertEqual(plan.command, ("codex", "resume", "session-1"))
            self.assertEqual(plan.cwd, cwd.resolve())

    def test_session_table_values_include_session_identity(self):
        session = aiw_cxs.SessionMeta(
            session_id="session-identity",
            path=Path("session.jsonl"),
            mtime_ns=1,
            size=1,
            title="Title",
            first_user="",
            turns=2,
            original_cwd=Path("C:/repo"),
        )

        values = aiw_cxs.session_table_values(session, ["alias"])

        self.assertEqual(values[0], "session-identity")
        self.assertIn("alias", values)

    def test_cli_resume_capability_requires_resume_help(self):
        completed = mock.Mock(
            returncode=0,
            stdout="Usage: codex resume [OPTIONS] [SESSION_ID]",
            stderr="",
        )
        with mock.patch.object(
            aiw_cxs.shutil, "which", return_value="C:/bin/codex.cmd"
        ), mock.patch.object(
            aiw_cxs.subprocess, "run", return_value=completed
        ) as run:
            executable = aiw_cxs.validate_interactive_resume_capability()

        self.assertEqual(executable, "C:/bin/codex.cmd")
        run.assert_called_once_with(
            ["C:/bin/codex.cmd", "resume", "--help"],
            check=False,
            capture_output=True,
            text=True,
        )

    def test_gui_command_and_workspace_filter_are_discoverable(self):
        parser = aiw_cxs.make_parser()

        gui_args = parser.parse_args(["gui"])
        list_args = parser.parse_args(["list", "--current-workspace"])

        self.assertEqual(gui_args.cmd, "gui")
        self.assertTrue(list_args.current_workspace)


if __name__ == "__main__":
    unittest.main()
