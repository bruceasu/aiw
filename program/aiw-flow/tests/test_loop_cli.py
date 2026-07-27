import io
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import AsyncMock, Mock, patch

from codex_flow.cli import main
from codex_flow.models import TurnResult, utc_now
from codex_flow.session_store import SessionStore
from codex_flow.skill_discovery import SkillDiscovery, SkillInfo, SkillIssue


class LoopCliTests(unittest.TestCase):
    def _create_session(self, tmp_path: Path, session_id: str = "ABC-123-loop"):
        workspace = tmp_path / "workspace"
        workspace.mkdir()
        instructions = tmp_path / "instructions.md"
        instructions.write_text("rules", encoding="utf-8")
        root = tmp_path / ".ai"
        main(
            [
                "--root",
                str(root),
                "new",
                "--id",
                session_id,
                "--title",
                "Loop demo",
                "--workspace",
                str(workspace),
                "--instructions",
                str(instructions),
            ]
        )
        return root, workspace

    def _skill(
        self,
        workspace: Path,
        *,
        name: str = "metrics-review",
        scope: str = "project",
        folder: str = "metrics-review",
    ) -> SkillInfo:
        return SkillInfo(
            name=name,
            description="Review metric definitions.",
            scope=scope,
            source=workspace / ".agents" / "skills" / folder,
        )

    def test_loop_runs_sequential_messages_with_saved_phase(self):
        with TemporaryDirectory() as temp_dir:
            root, _ = self._create_session(Path(temp_dir))
            store = SessionStore(root)
            status = store.load_status("ABC-123-loop")
            status.codex.thread_id = "thread-123"
            status.execution.current_phase = "analyze"
            store.save_status(status)
            execute_turn = AsyncMock(return_value=0)

            with (
                patch("builtins.input", side_effect=["first", "second", "/exit"]),
                patch("codex_flow.cli._execute_turn", execute_turn),
                redirect_stdout(io.StringIO()),
            ):
                result = main(["--root", str(root), "loop", "ABC-123-loop"])

            self.assertEqual(result, 0)
            self.assertEqual(execute_turn.await_count, 2)
            self.assertEqual(execute_turn.await_args_list[0].args[0].prompt, "first")
            self.assertEqual(execute_turn.await_args_list[1].args[0].prompt, "second")
            self.assertEqual(execute_turn.await_args_list[0].args[0].phase, "analyze")

    def test_new_loop_accepts_first_message_without_run_command(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            workspace = tmp_path / "workspace"
            workspace.mkdir()
            instructions = tmp_path / "instructions.md"
            instructions.write_text("rules", encoding="utf-8")
            root = tmp_path / ".ai"
            execute_turn = AsyncMock(return_value=0)

            with (
                patch("builtins.input", side_effect=["first message", "/exit"]),
                patch("codex_flow.cli._execute_turn", execute_turn),
                redirect_stdout(io.StringIO()),
            ):
                result = main(
                    [
                        "--root",
                        str(root),
                        "new",
                        "--id",
                        "ABC-123-loop",
                        "--title",
                        "Loop demo",
                        "--workspace",
                        str(workspace),
                        "--instructions",
                        str(instructions),
                        "--loop",
                    ]
                )

            self.assertEqual(result, 0)
            self.assertTrue((root / "sessions" / "ABC-123-loop" / "status.json").exists())
            execute_turn.assert_awaited_once()
            self.assertEqual(execute_turn.await_args.args[0].prompt, "first message")
            self.assertEqual(execute_turn.await_args.args[0].phase, "interactive")

    def test_local_commands_do_not_execute_model_turn(self):
        with TemporaryDirectory() as temp_dir:
            root, _ = self._create_session(Path(temp_dir))
            execute_turn = AsyncMock(return_value=0)
            stdout = io.StringIO()

            with (
                patch(
                    "builtins.input",
                    side_effect=["/help", "/status", "/memory", "/handoff", "/done", "/exit"],
                ),
                patch("codex_flow.cli._execute_turn", execute_turn),
                redirect_stdout(stdout),
            ):
                result = main(["--root", str(root), "loop", "ABC-123-loop"])

            self.assertEqual(result, 0)
            execute_turn.assert_not_awaited()
            self.assertTrue((root / "sessions" / "ABC-123-loop" / "artifacts" / "handoff.md").exists())
            self.assertIn("/done is only available in phase grill.", stdout.getvalue())

    def test_fork_uses_handoff_as_new_thread_prompt_and_exits(self):
        with TemporaryDirectory() as temp_dir:
            root, _ = self._create_session(Path(temp_dir))
            execute_turn = AsyncMock(return_value=0)
            with (
                patch("builtins.input", side_effect=["/fork"]),
                patch("codex_flow.cli._execute_turn", execute_turn),
            ):
                result = main(["--root", str(root), "loop", "ABC-123-loop"])

            self.assertEqual(result, 0)
            execute_turn.assert_awaited_once()
            request = execute_turn.await_args.args[0]
            self.assertIn("# Agent Handoff", request.prompt)
            self.assertTrue(execute_turn.await_args.kwargs["reset_thread"])

    def test_skills_lists_scopes_warnings_and_ambiguity_without_turn(self):
        with TemporaryDirectory() as temp_dir:
            root, workspace = self._create_session(Path(temp_dir))
            project_skill = self._skill(workspace)
            user_skill = self._skill(
                workspace,
                scope="user",
                folder="user-metrics-review",
            )
            issue_path = workspace / ".codex" / "skills" / "broken"
            discovery = SkillDiscovery(
                skills=(project_skill, user_skill),
                issues=(SkillIssue(issue_path, "Missing YAML frontmatter"),),
            )
            execute_turn = AsyncMock(return_value=0)
            stdout = io.StringIO()

            with (
                patch("builtins.input", side_effect=["/skills", "/exit"]),
                patch("codex_flow.cli.discover_skills", return_value=discovery),
                patch("codex_flow.cli._execute_turn", execute_turn),
                redirect_stdout(stdout),
            ):
                result = main(["--root", str(root), "loop", "ABC-123-loop"])

            self.assertEqual(result, 0)
            execute_turn.assert_not_awaited()
            output = stdout.getvalue()
            self.assertIn("Project Skills:", output)
            self.assertIn("User Skills:", output)
            self.assertIn("metrics-review [ambiguous]", output)
            self.assertIn(str(project_skill.source), output)
            self.assertIn("Warning: {}".format(issue_path), output)

    def test_skill_invocation_uses_normal_turn_persistence(self):
        with TemporaryDirectory() as temp_dir:
            root, workspace = self._create_session(Path(temp_dir))
            discovery = SkillDiscovery(
                skills=(self._skill(workspace),),
                issues=(),
            )
            output_dir = Path(temp_dir) / "backend"
            output_dir.mkdir()
            backend = AsyncMock()
            backend.run_turn.return_value = TurnResult(
                thread_id="thread-skill",
                final_output="reviewed",
                exit_code=0,
                events_file=output_dir / "events.jsonl",
                output_file=output_dir / "final.txt",
                started_at=utc_now(),
                completed_at=utc_now(),
                metadata={"command": ["codex", "exec"]},
            )

            with (
                patch(
                    "builtins.input",
                    side_effect=[
                        "/skill metrics-review Review the revenue metrics",
                        "/exit",
                    ],
                ),
                patch("codex_flow.cli.discover_skills", return_value=discovery),
                patch("codex_flow.cli._build_backend", return_value=backend),
                patch("codex_flow.cli._read_stdin_if_available", return_value=None),
                redirect_stdout(io.StringIO()),
            ):
                result = main(["--root", str(root), "loop", "ABC-123-loop"])

            self.assertEqual(result, 0)
            status = SessionStore(root).load_status("ABC-123-loop")
            self.assertEqual(status.codex.last_turn, 1)
            self.assertEqual(status.codex.thread_id, "thread-skill")
            prompt = (
                root
                / "sessions"
                / "ABC-123-loop"
                / "prompts"
                / "0001-interactive.md"
            ).read_text(encoding="utf-8")
            self.assertIn(
                "$metrics-review Review the revenue metrics",
                prompt,
            )
            backend.run_turn.assert_awaited_once()

    def test_skill_command_rejects_missing_unknown_and_duplicate_names(self):
        with TemporaryDirectory() as temp_dir:
            root, workspace = self._create_session(Path(temp_dir))
            duplicate = SkillDiscovery(
                skills=(
                    self._skill(workspace),
                    self._skill(workspace, scope="user", folder="duplicate"),
                ),
                issues=(),
            )
            execute_turn = AsyncMock(return_value=0)
            stdout = io.StringIO()

            with (
                patch(
                    "builtins.input",
                    side_effect=[
                        "/skill",
                        "/skill missing Run this",
                        "/skill metrics-review Run this",
                        "/exit",
                    ],
                ),
                patch("codex_flow.cli.discover_skills", return_value=duplicate),
                patch("codex_flow.cli._execute_turn", execute_turn),
                redirect_stdout(stdout),
            ):
                result = main(["--root", str(root), "loop", "ABC-123-loop"])

            self.assertEqual(result, 0)
            execute_turn.assert_not_awaited()
            output = stdout.getvalue()
            self.assertIn("Usage: /skill NAME MESSAGE", output)
            self.assertIn("Skill not found: missing", output)
            self.assertIn("Skill name is ambiguous: metrics-review", output)
            self.assertEqual(output.count(".agents"), 2)

    def test_skill_commands_refresh_discovery_and_preserve_direct_invocation(self):
        with TemporaryDirectory() as temp_dir:
            root, workspace = self._create_session(Path(temp_dir))
            empty = SkillDiscovery(skills=(), issues=())
            available = SkillDiscovery(
                skills=(self._skill(workspace),),
                issues=(),
            )
            execute_turn = AsyncMock(return_value=0)
            discover = Mock(side_effect=[empty, available])
            stdout = io.StringIO()

            with (
                patch(
                    "builtins.input",
                    side_effect=[
                        "/skills",
                        "/skill metrics-review Review this",
                        "$metrics-review Review directly",
                        "/exit",
                    ],
                ),
                patch("codex_flow.cli.discover_skills", discover),
                patch("codex_flow.cli._execute_turn", execute_turn),
                redirect_stdout(stdout),
            ):
                result = main(["--root", str(root), "loop", "ABC-123-loop"])

            self.assertEqual(result, 0)
            self.assertEqual(discover.call_count, 2)
            self.assertEqual(execute_turn.await_count, 2)
            self.assertEqual(
                execute_turn.await_args_list[0].args[0].prompt,
                "$metrics-review Review this",
            )
            self.assertEqual(
                execute_turn.await_args_list[1].args[0].prompt,
                "$metrics-review Review directly",
            )
            self.assertIn("No discoverable Skills found.", stdout.getvalue())

    def test_done_sends_final_grill_turn_and_exits(self):
        with TemporaryDirectory() as temp_dir:
            root, _ = self._create_session(Path(temp_dir))
            execute_turn = AsyncMock(return_value=0)

            with (
                patch("builtins.input", return_value="/done"),
                patch("codex_flow.cli._execute_turn", execute_turn),
                redirect_stdout(io.StringIO()),
            ):
                result = main(
                    [
                        "--root",
                        str(root),
                        "loop",
                        "ABC-123-loop",
                        "--phase",
                        "grill",
                    ]
                )

            self.assertEqual(result, 0)
            execute_turn.assert_awaited_once()
            self.assertEqual(execute_turn.await_args.args[0].prompt, "Grill Done")

    def test_grill_loop_enters_interaction_after_first_success(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            workspace = tmp_path / "workspace"
            workspace.mkdir()
            root = tmp_path / ".ai"
            execute_turn = AsyncMock(return_value=0)

            with (
                patch("builtins.input", side_effect=["CSV only", "/exit"]),
                patch("codex_flow.cli._execute_turn", execute_turn),
                redirect_stdout(io.StringIO()),
            ):
                result = main(
                    [
                        "--root",
                        str(root),
                        "grill",
                        "--id",
                        "ABC-123-grill",
                        "--title",
                        "Clarify export",
                        "--workspace",
                        str(workspace),
                        "--requirement",
                        "Add export.",
                        "--loop",
                    ]
                )

            self.assertEqual(result, 0)
            self.assertEqual(execute_turn.await_count, 2)
            self.assertIn("Start Grill requirement discovery", execute_turn.await_args_list[0].args[0].prompt)
            self.assertEqual(execute_turn.await_args_list[1].args[0].prompt, "CSV only")
            self.assertEqual(execute_turn.await_args_list[1].args[0].phase, "grill")

    def test_grill_loop_does_not_read_input_after_first_failure(self):
        with TemporaryDirectory() as temp_dir:
            tmp_path = Path(temp_dir)
            workspace = tmp_path / "workspace"
            workspace.mkdir()
            root = tmp_path / ".ai"
            execute_turn = AsyncMock(return_value=7)

            with (
                patch("builtins.input") as input_mock,
                patch("codex_flow.cli._execute_turn", execute_turn),
                redirect_stdout(io.StringIO()),
            ):
                result = main(
                    [
                        "--root",
                        str(root),
                        "grill",
                        "--id",
                        "ABC-123-grill",
                        "--title",
                        "Clarify export",
                        "--workspace",
                        str(workspace),
                        "--requirement",
                        "Add export.",
                        "--loop",
                    ]
                )

            self.assertEqual(result, 7)
            input_mock.assert_not_called()

    def test_loop_exits_cleanly_on_eof_and_idle_interrupt(self):
        for terminal_error in (EOFError(), KeyboardInterrupt()):
            with self.subTest(error=type(terminal_error).__name__), TemporaryDirectory() as temp_dir:
                root, _ = self._create_session(Path(temp_dir))
                stdout = io.StringIO()
                with patch("builtins.input", side_effect=terminal_error), redirect_stdout(stdout):
                    result = main(["--root", str(root), "loop", "ABC-123-loop"])

                self.assertEqual(result, 0)
                self.assertEqual(SessionStore(root).load_status("ABC-123-loop").session.state, "created")
                self.assertIn("Exited interactive loop.", stdout.getvalue())

    def test_loop_rejects_completed_session_before_input(self):
        with TemporaryDirectory() as temp_dir:
            root, _ = self._create_session(Path(temp_dir))
            store = SessionStore(root)
            status = store.load_status("ABC-123-loop")
            status.session.state = "completed"
            store.save_status(status)

            with patch("builtins.input") as input_mock:
                with self.assertRaisesRegex(SystemExit, "completed"):
                    main(["--root", str(root), "loop", "ABC-123-loop"])

            input_mock.assert_not_called()

    def test_loop_returns_failed_turn_exit_code(self):
        with TemporaryDirectory() as temp_dir:
            root, _ = self._create_session(Path(temp_dir))
            execute_turn = AsyncMock(return_value=9)

            with (
                patch("builtins.input", return_value="run this"),
                patch("codex_flow.cli._execute_turn", execute_turn),
                redirect_stdout(io.StringIO()),
            ):
                result = main(["--root", str(root), "loop", "ABC-123-loop"])

            self.assertEqual(result, 9)
