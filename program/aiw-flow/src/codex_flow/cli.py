from __future__ import annotations

import argparse
import asyncio
import json
import shutil
import sys
from pathlib import Path
from typing import Iterable, Optional

from codex_flow.artifact_manager import ArtifactManager
from codex_flow.backends import ExecCodexBackend
from codex_flow.config import default_session_root, load_global_config
from codex_flow.event_store import EventStore
from codex_flow.grill import GRILL_INSTRUCTIONS, build_initial_grill_prompt
from codex_flow.handoff_manager import render_handoff
from codex_flow.interactive_loop import LOOP_HELP, LoopInputKind, parse_loop_input
from codex_flow.memory_manager import MemoryManager, memory_sha256
from codex_flow.models import AppConfig, CreateSessionRequest, SessionStatus, TurnRequest, isoformat, utc_now
from codex_flow.prompt_composer import compose_prompt, load_prompt_text, save_prompt
from codex_flow.safety import validate_session_id
from codex_flow.session_store import SessionStore, SessionStoreError
from codex_flow.skill_discovery import SkillDiscovery, discover_skills
from codex_flow.workspace_context import collect_workspace_context
from codex_flow.workspace_manager import WorkspaceManager


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="aiw-flow")
    parser.add_argument("--root", type=Path, default=default_session_root(), help="State root directory.")
    subparsers = parser.add_subparsers(dest="command", required=True)

    new_parser = subparsers.add_parser("new")
    new_parser.add_argument("--id", required=True)
    new_parser.add_argument("--title", required=True)
    new_parser.add_argument("--workspace", type=Path, required=True)
    new_parser.add_argument("--instructions", type=Path, required=True)
    new_parser.add_argument("--ephemeral", action="store_true")
    new_parser.add_argument("--loop", action="store_true")
    new_parser.add_argument("--phase")
    new_parser.add_argument("--timeout", type=int)

    grill_parser = subparsers.add_parser("grill")
    grill_parser.add_argument("--id", required=True)
    grill_parser.add_argument("--title", required=True)
    grill_parser.add_argument("--workspace", type=Path, required=True)
    requirement_group = grill_parser.add_mutually_exclusive_group(required=True)
    requirement_group.add_argument("--requirement")
    requirement_group.add_argument("--requirement-file", type=Path)
    grill_parser.add_argument("--timeout", type=int)
    grill_parser.add_argument("--ephemeral", action="store_true")
    grill_parser.add_argument("--loop", action="store_true")

    loop_parser = subparsers.add_parser("loop")
    loop_parser.add_argument("session_id")
    loop_parser.add_argument("--phase")
    loop_parser.add_argument("--timeout", type=int)

    run_parser = subparsers.add_parser("run")
    run_parser.add_argument("session_id")
    run_parser.add_argument("--phase", required=True)
    run_parser.add_argument("--prompt")
    run_parser.add_argument("--prompt-file", type=Path)
    run_parser.add_argument("--timeout", type=int)
    run_parser.add_argument("--force-new-thread", action="store_true")

    continue_parser = subparsers.add_parser("continue")
    continue_parser.add_argument("session_id")
    continue_parser.add_argument("--phase", required=True)
    continue_parser.add_argument("--prompt")
    continue_parser.add_argument("--prompt-file", type=Path)
    continue_parser.add_argument("--timeout", type=int)

    status_parser = subparsers.add_parser("status")
    status_parser.add_argument("session_id")
    status_parser.add_argument("--json", action="store_true")

    list_parser = subparsers.add_parser("list")
    list_parser.add_argument("--state")

    inspect_parser = subparsers.add_parser("inspect")
    inspect_parser.add_argument("session_id")

    finish_parser = subparsers.add_parser("finish")
    finish_parser.add_argument("session_id")
    finish_parser.add_argument("--create-patch", action="store_true")

    archive_parser = subparsers.add_parser("archive")
    archive_parser.add_argument("session_id")

    delete_parser = subparsers.add_parser("delete")
    delete_parser.add_argument("session_id")
    delete_parser.add_argument("--yes", action="store_true")

    memory_parser = subparsers.add_parser("memory")
    memory_sub = memory_parser.add_subparsers(dest="memory_command", required=True)
    memory_show = memory_sub.add_parser("show")
    memory_show.add_argument("session_id")
    memory_append = memory_sub.add_parser("append")
    memory_append.add_argument("session_id")
    memory_append.add_argument("--text", required=True)
    memory_replace = memory_sub.add_parser("replace")
    memory_replace.add_argument("session_id")
    memory_replace.add_argument("--file", type=Path, required=True)

    handoff_parser = subparsers.add_parser("handoff")
    handoff_sub = handoff_parser.add_subparsers(dest="handoff_command", required=True)
    handoff_create = handoff_sub.add_parser("create")
    handoff_create.add_argument("session_id")
    handoff_create.add_argument("--focus")
    handoff_show = handoff_sub.add_parser("show")
    handoff_show.add_argument("session_id")

    daemon_parser = subparsers.add_parser("daemon")
    daemon_sub = daemon_parser.add_subparsers(dest="daemon_command", required=True)
    daemon_sub.add_parser("start")
    daemon_sub.add_parser("status")
    daemon_sub.add_parser("stop")
    return parser


def main(argv: Optional[Iterable[str]] = None) -> int:
    parser = build_parser()
    args = parser.parse_args(list(argv) if argv is not None else None)
    return asyncio.run(dispatch(args))


async def dispatch(args: argparse.Namespace) -> int:
    store = SessionStore(args.root)
    artifacts = ArtifactManager()
    config = load_global_config()

    if args.command == "new":
        result = cmd_new(args, store, config)
        if result == 0 and args.loop:
            return await cmd_loop(
                argparse.Namespace(
                    session_id=args.id,
                    phase=args.phase,
                    timeout=args.timeout,
                ),
                store,
            )
        return result
    if args.command == "grill":
        return await cmd_grill(args, store, config)
    if args.command == "loop":
        return await cmd_loop(args, store)
    if args.command == "run":
        return await cmd_run(args, store, force_new_thread=args.force_new_thread)
    if args.command == "continue":
        return await cmd_continue(args, store)
    if args.command == "status":
        return cmd_status(args, store)
    if args.command == "list":
        return cmd_list(args, store)
    if args.command == "inspect":
        return cmd_inspect(args, store)
    if args.command == "finish":
        return cmd_finish(args, store, artifacts)
    if args.command == "archive":
        return cmd_archive(args, store)
    if args.command == "delete":
        return cmd_delete(args, store)
    if args.command == "memory":
        return cmd_memory(args, store)
    if args.command == "handoff":
        return cmd_handoff(args, store)
    if args.command == "daemon":
        return cmd_daemon(args, store)
    raise RuntimeError("Unhandled command")


def cmd_new(
    args: argparse.Namespace,
    store: SessionStore,
    config: AppConfig,
) -> int:
    validate_session_id(args.id)
    instructions_text = args.instructions.read_text(encoding="utf-8")
    workspace = WorkspaceManager().ensure_existing_directory(args.workspace.resolve())
    request = CreateSessionRequest(
        session_id=args.id,
        title=args.title,
        instructions_text=instructions_text,
        workspace_path=workspace,
        codex_config=config,
        ephemeral=args.ephemeral,
    )
    status = store.create_session(request)
    print("Created session {}".format(status.session.id))
    return 0


async def cmd_grill(args: argparse.Namespace, store: SessionStore, config: AppConfig) -> int:
    validate_session_id(args.id)
    requirement = _load_grill_requirement(args.requirement, args.requirement_file)
    workspace = WorkspaceManager().ensure_existing_directory(args.workspace.resolve())
    workspace_context = collect_workspace_context(workspace)
    request = CreateSessionRequest(
        session_id=args.id,
        title=args.title,
        instructions_text=GRILL_INSTRUCTIONS,
        workspace_path=workspace,
        codex_config=config,
        ephemeral=args.ephemeral,
    )
    status = store.create_session(request)
    store.write_artifact_text(status.session.id, "workspace-context.md", workspace_context)
    turn_args = argparse.Namespace(
        session_id=status.session.id,
        phase="grill",
        prompt=build_initial_grill_prompt(requirement, workspace_context),
        prompt_file=None,
        timeout=args.timeout,
    )
    result = await _execute_turn(turn_args, store, allow_missing_thread=True, reset_thread=False)
    if result != 0 or not args.loop:
        return result
    return await cmd_loop(
        argparse.Namespace(
            session_id=status.session.id,
            phase="grill",
            timeout=args.timeout,
        ),
        store,
    )


def _load_grill_requirement(requirement: Optional[str], requirement_file: Optional[Path]) -> str:
    if requirement_file is not None:
        requirement = requirement_file.read_text(encoding="utf-8")
    normalized = (requirement or "").strip()
    if not normalized:
        raise SystemExit("Grill requirement must not be empty.")
    return normalized


async def cmd_loop(args: argparse.Namespace, store: SessionStore) -> int:
    status = store.load_status(args.session_id)
    _validate_loop_state(status)
    phase = args.phase or status.execution.current_phase or "interactive"
    print("Interactive loop for {} (phase: {}). Type /help for commands.".format(args.session_id, phase))

    while True:
        try:
            raw = input("You> ")
        except (EOFError, KeyboardInterrupt):
            print("\nExited interactive loop.")
            return 0

        loop_input = parse_loop_input(raw)
        if loop_input.kind == LoopInputKind.EMPTY:
            continue
        if loop_input.kind == LoopInputKind.HELP:
            print(LOOP_HELP, end="")
            continue
        if loop_input.kind == LoopInputKind.STATUS:
            cmd_status(argparse.Namespace(session_id=args.session_id, json=False), store)
            continue
        if loop_input.kind == LoopInputKind.MEMORY:
            cmd_memory(
                argparse.Namespace(
                    session_id=args.session_id,
                    memory_command="show",
                ),
                store,
            )
            continue
        if loop_input.kind == LoopInputKind.HANDOFF:
            cmd_handoff(
                argparse.Namespace(
                    session_id=args.session_id,
                    handoff_command="create",
                    focus=None,
                ),
                store,
            )
            continue
        if loop_input.kind == LoopInputKind.SKILLS:
            _print_skill_catalog(_discover_session_skills(store.load_status(args.session_id)))
            continue
        if loop_input.kind == LoopInputKind.EXIT:
            print("Exited interactive loop.")
            return 0
        if loop_input.kind == LoopInputKind.UNKNOWN:
            print("Unknown interactive command: {}. Type /help.".format(loop_input.text))
            continue

        exit_after_turn = loop_input.kind == LoopInputKind.DONE
        if exit_after_turn:
            if phase.lower() != "grill":
                print("/done is only available in phase grill.")
                continue
            prompt = "Grill Done"
        elif loop_input.kind == LoopInputKind.SKILL:
            prompt = _prepare_skill_prompt(
                loop_input.text,
                _discover_session_skills(store.load_status(args.session_id)),
            )
            if prompt is None:
                continue
        else:
            prompt = loop_input.text

        _validate_loop_state(store.load_status(args.session_id))
        turn_args = argparse.Namespace(
            session_id=args.session_id,
            phase=phase,
            prompt=prompt,
            prompt_file=None,
            timeout=args.timeout,
        )
        result = await _execute_turn(turn_args, store, allow_missing_thread=True, reset_thread=False)
        if result != 0 or exit_after_turn:
            return result


def _discover_session_skills(status: SessionStatus) -> SkillDiscovery:
    if not status.workspace.workspace_path:
        raise SystemExit("Session has no workspace path.")
    codex_home = Path(status.codex.codex_home) if status.codex.codex_home else None
    return discover_skills(
        Path(status.workspace.workspace_path),
        codex_home=codex_home,
    )


def _print_skill_catalog(discovery: SkillDiscovery) -> None:
    duplicates = {
        name for name, matches in discovery.by_name().items() if len(matches) > 1
    }
    for scope, title in (("project", "Project Skills"), ("user", "User Skills")):
        scoped = [skill for skill in discovery.skills if skill.scope == scope]
        if not scoped:
            continue
        print("{}:".format(title))
        for skill in scoped:
            suffix = " [ambiguous]" if skill.name in duplicates else ""
            print("  {}{} - {}".format(skill.name, suffix, skill.description))
            print("    {}".format(skill.source))
    if not discovery.skills:
        print("No discoverable Skills found.")
    for issue in discovery.issues:
        print("Warning: {}: {}".format(issue.source, issue.message))


def _prepare_skill_prompt(text: str, discovery: SkillDiscovery) -> Optional[str]:
    parts = text.split(None, 1)
    if len(parts) != 2 or not parts[1].strip():
        print("Usage: /skill NAME MESSAGE")
        return None

    name, message = parts[0], parts[1].strip()
    matches = discovery.by_name().get(name, ())
    if not matches:
        print("Skill not found: {}".format(name))
        return None
    if len(matches) > 1:
        print("Skill name is ambiguous: {}".format(name))
        for skill in matches:
            print("  {}".format(skill.source))
        return None
    return "${} {}".format(name, message)


def _validate_loop_state(status: SessionStatus) -> None:
    allowed_states = {"created", "active", "paused", "failed"}
    if status.session.state not in allowed_states:
        raise SystemExit(
            "Session state `{}` cannot enter the interactive loop.".format(status.session.state)
        )


async def cmd_run(args: argparse.Namespace, store: SessionStore, *, force_new_thread: bool) -> int:
    status = store.load_status(args.session_id)
    if status.codex.thread_id and not force_new_thread:
        raise SystemExit("Session already has thread_id. Use continue or --force-new-thread.")
    return await _execute_turn(args, store, allow_missing_thread=True, reset_thread=force_new_thread)


async def cmd_continue(args: argparse.Namespace, store: SessionStore) -> int:
    status = store.load_status(args.session_id)
    if not status.codex.thread_id:
        raise SystemExit("Session has no thread_id. Run the first turn with `run`.")
    return await _execute_turn(args, store, allow_missing_thread=False, reset_thread=False)


async def _execute_turn(
    args: argparse.Namespace,
    store: SessionStore,
    *,
    allow_missing_thread: bool,
    reset_thread: bool,
) -> int:
    status = store.load_status(args.session_id)
    session_dir = store.session_dir(args.session_id)
    memory_manager = MemoryManager(session_dir / status.instructions.memory_file)
    instructions_path = session_dir / status.instructions.system_file
    prompt_text = load_prompt_text(args.prompt, args.prompt_file, _read_stdin_if_available())
    if not prompt_text.strip():
        raise SystemExit("Prompt is required via --prompt, --prompt-file, or stdin.")
    instructions_text = instructions_path.read_text(encoding="utf-8")
    memory_text = memory_manager.read()
    next_turn = status.codex.last_turn + 1
    composed = compose_prompt(instructions_text, memory_text, args.phase, prompt_text)
    snapshot = save_prompt(session_dir / "prompts", next_turn, args.phase, composed)
    store.update_status(
        args.session_id,
        lambda current: _before_turn_status(current, snapshot.sha256, memory_sha256(memory_text), args.phase),
    )
    event_store = EventStore(session_dir / "events.jsonl")
    event_store.append("turn.started", turn=next_turn, phase=args.phase)
    backend = _build_backend(status.session.backend, status)
    await backend.start()
    try:
        result = await backend.run_turn(
            TurnRequest(
                session_id=args.session_id,
                prompt=composed,
                workspace=Path(status.workspace.workspace_path),
                thread_id=None if reset_thread else status.codex.thread_id,
                instructions=instructions_text,
                memory=memory_text,
                phase=args.phase,
                timeout_seconds=args.timeout or None,
                output_dir=session_dir / "outputs",
                turn_number=next_turn,
                ephemeral=status.session.ephemeral,
            )
        )
    except Exception as exc:
        event_store.append("turn.failed", turn=next_turn, phase=args.phase, error=str(exc))
        store.update_status(args.session_id, lambda current: _failed_turn_status(current, args.phase, str(exc)))
        await backend.close()
        raise
    finally:
        await backend.close()
    event_store.append("turn.completed", turn=next_turn, phase=args.phase, exit_code=result.exit_code)
    if result.thread_id and result.thread_id != status.codex.thread_id:
        event_store.append("thread.bound", thread_id=result.thread_id)
    store.update_status(
        args.session_id,
        lambda current: _after_turn_status(current, result, args.phase, snapshot.sha256, memory_sha256(memory_text)),
    )
    print(result.final_output)
    return 0 if result.exit_code == 0 else result.exit_code


def _build_backend(name: str, status: SessionStatus):
    config = AppConfig(
        model=status.codex.model,
        profile=status.codex.profile,
        codex_home=status.codex.codex_home,
    )
    if name == "exec":
        return ExecCodexBackend(config)
    raise SystemExit("Unsupported backend: {}".format(name))


def _read_stdin_if_available() -> Optional[str]:
    if sys.stdin.isatty():
        return None
    return sys.stdin.read()


def _before_turn_status(status: SessionStatus, prompt_hash: str, memory_hash: str, phase: str) -> SessionStatus:
    status.session.state = "running"
    status.session.updated_at = isoformat(utc_now())
    status.instructions.instructions_hash = prompt_hash
    status.instructions.memory_hash = memory_hash
    status.execution.current_phase = phase
    status.execution.last_started_at = isoformat(utc_now())
    return status


def _after_turn_status(
    status: SessionStatus,
    result,
    phase: str,
    prompt_hash: str,
    memory_hash: str,
) -> SessionStatus:
    status.session.state = "active" if result.exit_code == 0 else "failed"
    status.session.updated_at = isoformat(utc_now())
    status.codex.thread_id = result.thread_id
    status.codex.last_turn += 1
    status.instructions.instructions_hash = prompt_hash
    status.instructions.memory_hash = memory_hash
    status.execution.current_phase = phase
    status.execution.last_exit_code = result.exit_code
    status.execution.last_started_at = isoformat(result.started_at)
    status.execution.last_completed_at = isoformat(result.completed_at)
    command = result.metadata.get("command", [])
    status.execution.last_command = list(command)
    status.result.final_output_file = str(result.output_file)
    status.result.error_message = None if result.exit_code == 0 else "Turn exited with code {}".format(result.exit_code)
    return status


def _failed_turn_status(status: SessionStatus, phase: str, error: str) -> SessionStatus:
    status.session.state = "failed"
    status.session.updated_at = isoformat(utc_now())
    status.execution.current_phase = phase
    status.execution.last_completed_at = isoformat(utc_now())
    status.result.error_message = error
    return status


def cmd_status(args: argparse.Namespace, store: SessionStore) -> int:
    status = store.load_status(args.session_id)
    if args.json:
        print(json.dumps(status.to_dict(), indent=2, ensure_ascii=False))
        return 0
    print("Session ID: {}".format(status.session.id))
    print("State: {}".format(status.session.state))
    print("Thread ID: {}".format(status.codex.thread_id or "-"))
    print("Workspace: {}".format(status.workspace.workspace_path or "-"))
    print("Current Phase: {}".format(status.execution.current_phase or "-"))
    print("Last Exit Code: {}".format(status.execution.last_exit_code if status.execution.last_exit_code is not None else "-"))
    print("Last Turn: {}".format(status.codex.last_turn))
    print("Last Output: {}".format(status.result.final_output_file or "-"))
    print("Last Error: {}".format(status.result.error_message or "-"))
    return 0


def cmd_list(args: argparse.Namespace, store: SessionStore) -> int:
    sessions = list(store.list_sessions())
    for status in sessions:
        if args.state and status.session.state != args.state:
            continue
        print("{}\t{}\t{}".format(status.session.id, status.session.state, status.workspace.workspace_path))
    return 0


def cmd_inspect(args: argparse.Namespace, store: SessionStore) -> int:
    status = store.load_status(args.session_id)
    session_dir = store.session_dir(args.session_id)
    events = EventStore(session_dir / "events.jsonl").tail(10)
    memory = MemoryManager(session_dir / status.instructions.memory_file).read()
    print(json.dumps(status.to_dict(), indent=2, ensure_ascii=False))
    print("\nRecent Events:")
    for event in events:
        print(json.dumps(event, ensure_ascii=False))
    print("\nMemory Summary:")
    print(memory[:500])
    return 0


def cmd_finish(args: argparse.Namespace, store: SessionStore, artifacts: ArtifactManager) -> int:
    status = store.load_status(args.session_id)
    session_dir = store.session_dir(args.session_id)
    if args.create_patch:
        summary = artifacts.create(Path(status.workspace.workspace_path), session_dir / "artifacts")
        patch_file = summary.get("patch_file")
    else:
        patch_file = None
    store.update_status(args.session_id, lambda current: _finish_status(current, patch_file))
    print("Finished {}".format(args.session_id))
    return 0


def _finish_status(status: SessionStatus, patch_file: Optional[str]) -> SessionStatus:
    status.session.state = "completed"
    status.session.updated_at = isoformat(utc_now())
    status.result.status = "completed"
    if patch_file:
        status.result.patch_file = patch_file
    return status


def cmd_archive(args: argparse.Namespace, store: SessionStore) -> int:
    store.update_status(args.session_id, lambda current: _archive_status(current))
    target = store.archive_session(args.session_id)
    print("Archived to {}".format(target))
    return 0


def _archive_status(status: SessionStatus) -> SessionStatus:
    status.session.state = "archived"
    status.session.updated_at = isoformat(utc_now())
    return status


def cmd_delete(args: argparse.Namespace, store: SessionStore) -> int:
    session_dir = store.session_dir(args.session_id)
    if not args.yes:
        print("Delete session directory {}? Use --yes to confirm.".format(session_dir))
        return 1
    if session_dir.exists():
        shutil.rmtree(session_dir)
    print("Deleted {}".format(args.session_id))
    return 0


def cmd_memory(args: argparse.Namespace, store: SessionStore) -> int:
    status = store.load_status(args.session_id)
    manager = MemoryManager(store.session_dir(args.session_id) / status.instructions.memory_file)
    if args.memory_command == "show":
        print(manager.read())
        return 0
    if args.memory_command == "append":
        digest = manager.append_note(args.text)
        store.update_status(args.session_id, lambda current: _update_memory_hash(current, digest))
        print(digest)
        return 0
    if args.memory_command == "replace":
        digest = manager.replace(args.file.read_text(encoding="utf-8"))
        store.update_status(args.session_id, lambda current: _update_memory_hash(current, digest))
        print(digest)
        return 0
    raise SystemExit("Unsupported memory command")


def cmd_handoff(args: argparse.Namespace, store: SessionStore) -> int:
    if args.handoff_command == "create":
        status = store.load_status(args.session_id)
        session_dir = store.session_dir(args.session_id)
        memory = MemoryManager(session_dir / status.instructions.memory_file).read()
        content = render_handoff(status, session_dir, memory, focus=args.focus)
        path = store.write_artifact_text(args.session_id, "handoff.md", content)
        print("Handoff saved to {}".format(path))
        return 0
    if args.handoff_command == "show":
        try:
            content = store.read_artifact_text(args.session_id, "handoff.md")
        except SessionStoreError:
            raise SystemExit("Handoff does not exist. Run `aiw-flow handoff create {}`.".format(args.session_id))
        print(content, end="")
        return 0
    raise SystemExit("Unsupported handoff command")


def _update_memory_hash(status: SessionStatus, digest: str) -> SessionStatus:
    status.instructions.memory_hash = digest
    status.session.updated_at = isoformat(utc_now())
    return status


def cmd_daemon(args: argparse.Namespace, store: SessionStore) -> int:
    daemon_file = store.logs_dir / "daemon.json"
    if args.daemon_command == "start":
        payload = {"pid": 0, "started_at": isoformat(utc_now()), "mode": "placeholder"}
        daemon_file.write_text(json.dumps(payload, indent=2), encoding="utf-8")
        print("Recorded placeholder daemon state.")
        return 0
    if args.daemon_command == "status":
        if not daemon_file.exists():
            print("Daemon not started.")
            return 1
        print(daemon_file.read_text(encoding="utf-8"))
        return 0
    if args.daemon_command == "stop":
        if daemon_file.exists():
            daemon_file.unlink()
        print("Daemon state cleared.")
        return 0
    raise SystemExit("Unsupported daemon command")
