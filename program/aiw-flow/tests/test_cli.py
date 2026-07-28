import io
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import AsyncMock, patch

from codex_flow.cli import main
from codex_flow.session_store import SessionStore


class CliTests(unittest.TestCase):
    def test_cli_new_and_status(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            workspace = tmp_path / "workspace"
            workspace.mkdir()
            instructions = tmp_path / "AGENTS.md"
            instructions.write_text("rules", encoding="utf-8")
            root = tmp_path / ".aiw-flow"
            stdout = io.StringIO()
            with redirect_stdout(stdout):
                self.assertEqual(
                    main(
                        [
                            "--root",
                            str(root),
                            "new",
                            "--id",
                            "ABC-123-demo",
                            "--workspace",
                            str(workspace),
                        ]
                    ),
                    0,
                )
                self.assertEqual(main(["--root", str(root), "status", "ABC-123-demo"]), 0)
            self.assertIn("Session ID: ABC-123-demo", stdout.getvalue())

    def test_cli_delete_requires_confirmation(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            workspace = tmp_path / "workspace"
            workspace.mkdir()
            instructions = tmp_path / "AGENTS.md"
            instructions.write_text("rules", encoding="utf-8")
            root = tmp_path / ".aiw-flow"
            main(
                [
                    "--root",
                    str(root),
                    "new",
                    "--id",
                    "ABC-123-demo",
                    "--workspace",
                    str(workspace),
                ]
            )
            stdout = io.StringIO()
            with redirect_stdout(stdout):
                self.assertEqual(main(["--root", str(root), "delete", "ABC-123-demo"]), 1)
            self.assertIn("Use --yes to confirm", stdout.getvalue())

    def test_cli_grill_creates_context_and_starts_first_turn(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            workspace = tmp_path / "workspace"
            workspace.mkdir()
            (workspace / "README.md").write_text("Demo project", encoding="utf-8")
            root = tmp_path / ".ai"
            execute_turn = AsyncMock(return_value=0)

            with patch("codex_flow.cli._execute_turn", execute_turn):
                self.assertEqual(
                    main(
                        [
                            "--root",
                            str(root),
                            "grill",
                            "--id",
                            "ABC-123-grill",
                            "--workspace",
                            str(workspace),
                            "--requirement",
                            "Add export support.",
                        ]
                    ),
                    0,
                )

            store = SessionStore(root)
            session_dir = store.session_dir("ABC-123-grill")
            instructions = (session_dir / "instructions.md").read_text(encoding="utf-8")
            context = (session_dir / "artifacts" / "workspace-context.md").read_text(encoding="utf-8")
            self.assertIn("Ask at most one user decision question", instructions)
            self.assertIn("Demo project", context)
            execute_turn.assert_awaited_once()
            turn_args = execute_turn.await_args.args[0]
            self.assertIn("Add export support.", turn_args.prompt)
            self.assertEqual(turn_args.phase, "grill")

    def test_cli_grill_rejects_empty_requirement_file(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            workspace = tmp_path / "workspace"
            workspace.mkdir()
            requirement_file = tmp_path / "requirement.md"
            requirement_file.write_text(" \n", encoding="utf-8")
            root = tmp_path / ".ai"

            with self.assertRaisesRegex(SystemExit, "must not be empty"):
                main(
                    [
                        "--root",
                        str(root),
                        "grill",
                        "--id",
                        "ABC-123-grill",
                        "--workspace",
                        str(workspace),
                        "--requirement-file",
                        str(requirement_file),
                    ]
                )

            self.assertFalse((root / "sessions" / "ABC-123-grill").exists())

    def test_cli_handoff_create_and_show(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            workspace = tmp_path / "workspace"
            workspace.mkdir()
            instructions = tmp_path / "AGENTS.md"
            instructions.write_text("rules", encoding="utf-8")
            root = tmp_path / ".ai"
            main(
                [
                    "--root",
                    str(root),
                    "new",
                    "--id",
                    "ABC-123-demo",
                    "--workspace",
                    str(workspace),
                ]
            )
            stdout = io.StringIO()

            with redirect_stdout(stdout):
                self.assertEqual(
                    main(
                        [
                            "--root",
                            str(root),
                            "handoff",
                            "create",
                            "ABC-123-demo",
                            "--focus",
                            "Continue tests.",
                        ]
                    ),
                    0,
                )
                self.assertEqual(main(["--root", str(root), "handoff", "show", "ABC-123-demo"]), 0)

            output = stdout.getvalue()
            self.assertIn("Handoff saved to", output)
            self.assertIn("# Agent Handoff: ABC-123-demo", output)
            self.assertIn("Continue tests.", output)

    def test_cli_handoff_show_reports_missing_artifact(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            workspace = tmp_path / "workspace"
            workspace.mkdir()
            instructions = tmp_path / "AGENTS.md"
            instructions.write_text("rules", encoding="utf-8")
            root = tmp_path / ".ai"
            main(
                [
                    "--root",
                    str(root),
                    "new",
                    "--id",
                    "ABC-123-demo",
                    "--workspace",
                    str(workspace),
                ]
            )

            with self.assertRaisesRegex(SystemExit, "handoff create"):
                main(["--root", str(root), "handoff", "show", "ABC-123-demo"])
