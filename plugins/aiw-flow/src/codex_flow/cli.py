from __future__ import annotations

import argparse
import asyncio
import json
import shutil
import sys
from pathlib import Path
from typing import Any, Iterable, Optional

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


HELP_FORMATTER = argparse.RawDescriptionHelpFormatter


def _examples(*commands: str) -> str:
    return "Examples:\n{}".format("\n".join("  {}".format(command) for command in commands))


def _add_command(
    subparsers: Any,
    name: str,
    *,
    summary: str,
    description: str,
    examples: Iterable[str],
) -> argparse.ArgumentParser:
    return subparsers.add_parser(
        name,
        help=summary,
        description=description,
        epilog=_examples(*examples),
        formatter_class=HELP_FORMATTER,
    )


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="aiw-flow",
        description=(
            "Manage long-running Codex Sessions with saved prompts, Memory, "
            "Thread state, outputs, and handoffs."
        ),
        epilog=(
            "Common workflow:\n"
            "  new -> run -> continue (repeat) -> finish -> archive\n"
            "    +-> loop for an interactive Session\n"
            "  grill -> loop to clarify a requirement before implementation\n"
            "\n"
            "Quick starts:\n"
            "  aiw-flow new --id TASK-123 --title \"Fix login\" --workspace ./worktree "
            "--instructions ./instructions.md\n"
            "  aiw-flow loop TASK-123 --phase analyze\n"
            "  aiw-flow grill --id TASK-124 --title \"Clarify export\" --workspace ./worktree "
            "--requirement \"Add CSV export\" --loop\n"
            "\n"
            "Put --root before COMMAND when using a custom state directory.\n"
            "Run `aiw-flow COMMAND --help` for command details and examples."
        ),
        formatter_class=HELP_FORMATTER,
    )
    parser.add_argument(
        "--root",
        type=Path,
        default=default_session_root(),
        metavar="PATH",
        help="Directory that stores Session state. Put this option before COMMAND.",
    )
    subparsers = parser.add_subparsers(
        dest="command",
        required=True,
        title="commands",
        metavar="COMMAND",
        description="Choose the next action for a Session.",
    )

    new_parser = _add_command(
        subparsers,
        "new",
        summary="Create a Session without running Codex yet.",
        description=(
            "Create a Session and save its workspace and persistent instructions.\n"
            "Add --loop to wait for the first message immediately after creation."
        ),
        examples=(
            "aiw-flow new --id TASK-123 --title \"Fix login\" "
            "--workspace ./worktree --instructions ./instructions.md",
            "aiw-flow new --id TASK-123 --title \"Fix login\" "
            "--workspace ./worktree --instructions ./instructions.md "
            "--loop --phase analyze",
        ),
    )
    new_parser.add_argument("--id", required=True, metavar="SESSION_ID", help="Unique Session ID, such as TASK-123.")
    new_parser.add_argument("--title", required=True, metavar="TEXT", help="Short human-readable task title.")
    new_parser.add_argument(
        "--workspace",
        type=Path,
        required=True,
        metavar="PATH",
        help="Existing directory where Codex will work. aiw-flow does not create a worktree.",
    )
    new_parser.add_argument(
        "--instructions",
        type=Path,
        required=True,
        metavar="FILE",
        help="UTF-8 file with persistent rules for every turn.",
    )
    new_parser.add_argument("--ephemeral", action="store_true", help="Ask Codex to use an ephemeral execution mode.")
    new_parser.add_argument("--loop", action="store_true", help="Enter the interactive loop after creating the Session.")
    new_parser.add_argument(
        "--phase",
        metavar="NAME",
        help="Phase used by --loop. The default is interactive.",
    )
    new_parser.add_argument(
        "--timeout",
        type=int,
        metavar="SECONDS",
        help="Maximum time for each Codex turn in --loop.",
    )

    grill_parser = _add_command(
        subparsers,
        "grill",
        summary="Clarify a requirement with a guided Codex interview.",
        description=(
            "Create a requirement-discovery Session and run its first Codex turn.\n"
            "Provide the requirement as text or a UTF-8 file, but not both.\n"
            "Add --loop to answer follow-up questions in the same terminal."
        ),
        examples=(
            "aiw-flow grill --id TASK-124 --title \"Clarify export\" "
            "--workspace ./worktree --requirement \"Add CSV export\"",
            "aiw-flow grill --id TASK-124 --title \"Clarify export\" "
            "--workspace ./worktree --requirement-file ./requirement.md --loop",
        ),
    )
    grill_parser.add_argument("--id", required=True, metavar="SESSION_ID", help="Unique Session ID for the interview.")
    grill_parser.add_argument("--title", required=True, metavar="TEXT", help="Short title for the requirement.")
    grill_parser.add_argument(
        "--workspace",
        type=Path,
        required=True,
        metavar="PATH",
        help="Existing workspace to inspect for project context.",
    )
    requirement_group = grill_parser.add_mutually_exclusive_group(required=True)
    requirement_group.add_argument("--requirement", metavar="TEXT", help="Requirement text written on the command line.")
    requirement_group.add_argument(
        "--requirement-file",
        type=Path,
        metavar="FILE",
        help="UTF-8 file that contains the requirement.",
    )
    grill_parser.add_argument("--timeout", type=int, metavar="SECONDS", help="Maximum time for each Codex turn.")
    grill_parser.add_argument("--ephemeral", action="store_true", help="Ask Codex to use an ephemeral execution mode.")
    grill_parser.add_argument("--loop", action="store_true", help="Enter the interactive loop after the first answer.")

    loop_parser = _add_command(
        subparsers,
        "loop",
        summary="Talk to an existing Session in one terminal process.",
        description=(
            "Read messages until /exit, EOF, or Ctrl+C. Each message is saved as a normal turn.\n"
            "Use /help inside the loop to see local commands. Completed or archived Sessions cannot enter."
        ),
        examples=(
            "aiw-flow loop TASK-123",
            "aiw-flow loop TASK-124 --phase grill --timeout 900",
        ),
    )
    loop_parser.add_argument("session_id", metavar="SESSION_ID", help="Session to open.")
    loop_parser.add_argument(
        "--phase",
        metavar="NAME",
        help="Phase for new turns. Defaults to the current Session phase, then interactive.",
    )
    loop_parser.add_argument("--timeout", type=int, metavar="SECONDS", help="Maximum time for each Codex turn.")

    run_parser = _add_command(
        subparsers,
        "run",
        summary="Run the first Codex turn for a Session.",
        description=(
            "Start a Codex Thread for a Session created by `new`.\n"
            "Supply a prompt with --prompt, --prompt-file, or stdin.\n"
            "--force-new-thread discards the saved Thread link and should be used only for recovery."
        ),
        examples=(
            "aiw-flow run TASK-123 --phase analyze --prompt \"Find the root cause.\"",
            "aiw-flow run TASK-123 --phase implement --prompt-file ./implement.md --timeout 900",
        ),
    )
    run_parser.add_argument("session_id", metavar="SESSION_ID", help="Session to run.")
    run_parser.add_argument("--phase", required=True, metavar="NAME", help="Turn phase, such as analyze or implement.")
    run_parser.add_argument("--prompt", metavar="TEXT", help="Prompt text for this turn.")
    run_parser.add_argument("--prompt-file", type=Path, metavar="FILE", help="UTF-8 file with the prompt.")
    run_parser.add_argument("--timeout", type=int, metavar="SECONDS", help="Maximum time for this Codex turn.")
    run_parser.add_argument(
        "--force-new-thread",
        action="store_true",
        help="Start a new Thread even if the Session already has one.",
    )

    continue_parser = _add_command(
        subparsers,
        "continue",
        summary="Run another turn on the saved Codex Thread.",
        description=(
            "Continue a Session that already has a Thread ID.\n"
            "Supply a prompt with --prompt, --prompt-file, or stdin."
        ),
        examples=(
            "aiw-flow continue TASK-123 --phase implement --prompt \"Apply the approved fix.\"",
            "aiw-flow continue TASK-123 --phase fix-tests --prompt-file ./fix-tests.md",
        ),
    )
    continue_parser.add_argument("session_id", metavar="SESSION_ID", help="Session whose Thread will continue.")
    continue_parser.add_argument("--phase", required=True, metavar="NAME", help="Turn phase, such as implement or fix-tests.")
    continue_parser.add_argument("--prompt", metavar="TEXT", help="Prompt text for this turn.")
    continue_parser.add_argument("--prompt-file", type=Path, metavar="FILE", help="UTF-8 file with the prompt.")
    continue_parser.add_argument("--timeout", type=int, metavar="SECONDS", help="Maximum time for this Codex turn.")

    status_parser = _add_command(
        subparsers,
        "status",
        summary="Show the current state of one Session.",
        description="Show state, Thread, workspace, phase, last turn, output, and error information.",
        examples=("aiw-flow status TASK-123", "aiw-flow status TASK-123 --json"),
    )
    status_parser.add_argument("session_id", metavar="SESSION_ID", help="Session to report.")
    status_parser.add_argument("--json", action="store_true", help="Print the complete status as JSON for scripts or CI.")

    list_parser = _add_command(
        subparsers,
        "list",
        summary="List saved Sessions, optionally filtered by state.",
        description="Print one Session per line with its state and workspace.",
        examples=("aiw-flow list", "aiw-flow list --state active"),
    )
    list_parser.add_argument("--state", metavar="STATE", help="Only show Sessions in this exact state.")

    inspect_parser = _add_command(
        subparsers,
        "inspect",
        summary="Inspect full Session status, recent events, and Memory.",
        description="Print detailed JSON status, the last ten events, and a short Memory preview for diagnosis.",
        examples=("aiw-flow inspect TASK-123",),
    )
    inspect_parser.add_argument("session_id", metavar="SESSION_ID", help="Session to inspect.")

    finish_parser = _add_command(
        subparsers,
        "finish",
        summary="Mark a Session completed and optionally save a patch artifact.",
        description=(
            "Mark the Session completed. --create-patch records review artifacts from the workspace.\n"
            "This command does not commit, push, or clean a Git worktree."
        ),
        examples=("aiw-flow finish TASK-123", "aiw-flow finish TASK-123 --create-patch"),
    )
    finish_parser.add_argument("session_id", metavar="SESSION_ID", help="Session to mark completed.")
    finish_parser.add_argument(
        "--create-patch",
        action="store_true",
        help="Save workspace diff artifacts before completion.",
    )

    archive_parser = _add_command(
        subparsers,
        "archive",
        summary="Move a Session from active storage into the archive.",
        description="Mark the Session archived and move its saved state into the archive directory.",
        examples=("aiw-flow archive TASK-123",),
    )
    archive_parser.add_argument("session_id", metavar="SESSION_ID", help="Session to archive.")

    delete_parser = _add_command(
        subparsers,
        "delete",
        summary="Delete aiw-flow's saved state for one Session.",
        description=(
            "Delete only the Session state directory. The workspace, branch, and worktree are not removed.\n"
            "The command does nothing unless --yes is present."
        ),
        examples=("aiw-flow delete TASK-123 --yes",),
    )
    delete_parser.add_argument("session_id", metavar="SESSION_ID", help="Session state to delete.")
    delete_parser.add_argument("--yes", action="store_true", help="Confirm permanent deletion of the Session state.")

    memory_parser = _add_command(
        subparsers,
        "memory",
        summary="Show or update the saved context used in future turns.",
        description=(
            "Manage the Session Memory that is included in future Codex prompts.\n"
            "Choose an action below, then run `aiw-flow memory ACTION --help` for details."
        ),
        examples=(
            "aiw-flow memory show TASK-123",
            "aiw-flow memory append TASK-123 --text \"The timeout is confirmed.\"",
        ),
    )
    memory_sub = memory_parser.add_subparsers(
        dest="memory_command",
        required=True,
        title="memory actions",
        metavar="ACTION",
        description="Choose how to read or update Memory.",
    )
    memory_show = _add_command(
        memory_sub,
        "show",
        summary="Print the current Session Memory.",
        description="Print the complete saved Memory text for one Session.",
        examples=("aiw-flow memory show TASK-123",),
    )
    memory_show.add_argument("session_id", metavar="SESSION_ID", help="Session whose Memory will be shown.")
    memory_append = _add_command(
        memory_sub,
        "append",
        summary="Append one confirmed note to Session Memory.",
        description="Add text to the end of Memory and update its saved hash.",
        examples=("aiw-flow memory append TASK-123 --text \"CSV is enough for release one.\"",),
    )
    memory_append.add_argument("session_id", metavar="SESSION_ID", help="Session whose Memory will change.")
    memory_append.add_argument("--text", required=True, metavar="TEXT", help="Note to append.")
    memory_replace = _add_command(
        memory_sub,
        "replace",
        summary="Replace Session Memory with a UTF-8 file.",
        description="Replace all current Memory with the contents of one UTF-8 file.",
        examples=("aiw-flow memory replace TASK-123 --file ./confirmed-findings.md",),
    )
    memory_replace.add_argument("session_id", metavar="SESSION_ID", help="Session whose Memory will be replaced.")
    memory_replace.add_argument("--file", type=Path, required=True, metavar="FILE", help="UTF-8 file used as the new Memory.")

    handoff_parser = _add_command(
        subparsers,
        "handoff",
        summary="Create or show a deterministic Session handoff.",
        description=(
            "Build a handoff from saved status, Memory, recent output, and artifacts without calling Codex.\n"
            "Choose an action below, then run `aiw-flow handoff ACTION --help` for details."
        ),
        examples=(
            "aiw-flow handoff create TASK-123 --focus \"Continue validation.\"",
            "aiw-flow handoff show TASK-123",
        ),
    )
    handoff_sub = handoff_parser.add_subparsers(
        dest="handoff_command",
        required=True,
        title="handoff actions",
        metavar="ACTION",
        description="Choose whether to create or read the handoff artifact.",
    )
    handoff_create = _add_command(
        handoff_sub,
        "create",
        summary="Create or refresh artifacts/handoff.md.",
        description="Create a deterministic handoff file. --focus records the recommended next area of work.",
        examples=(
            "aiw-flow handoff create TASK-123",
            "aiw-flow handoff create TASK-123 --focus \"Resolve the encoding decision.\"",
        ),
    )
    handoff_create.add_argument("session_id", metavar="SESSION_ID", help="Session used to build the handoff.")
    handoff_create.add_argument("--focus", metavar="TEXT", help="Optional next-step focus included in the handoff.")
    handoff_show = _add_command(
        handoff_sub,
        "show",
        summary="Print the saved handoff artifact.",
        description="Print artifacts/handoff.md. Create it first if it does not exist.",
        examples=("aiw-flow handoff show TASK-123",),
    )
    handoff_show.add_argument("session_id", metavar="SESSION_ID", help="Session whose handoff will be shown.")

    daemon_parser = _add_command(
        subparsers,
        "daemon",
        summary="Manage placeholder daemon state for development.",
        description=(
            "Record, inspect, or clear placeholder daemon state.\n"
            "This does not start a background worker or execute Codex."
        ),
        examples=("aiw-flow daemon start", "aiw-flow daemon status", "aiw-flow daemon stop"),
    )
    daemon_sub = daemon_parser.add_subparsers(
        dest="daemon_command",
        required=True,
        title="daemon actions",
        metavar="ACTION",
        description="Choose how to manage the placeholder state file.",
    )
    _add_command(
        daemon_sub,
        "start",
        summary="Record placeholder daemon state.",
        description="Write a placeholder daemon state file. No background process is started.",
        examples=("aiw-flow daemon start",),
    )
    _add_command(
        daemon_sub,
        "status",
        summary="Print placeholder daemon state.",
        description="Print the saved placeholder state, or report that it has not been started.",
        examples=("aiw-flow daemon status",),
    )
    _add_command(
        daemon_sub,
        "stop",
        summary="Clear placeholder daemon state.",
        description="Remove the placeholder state file. No process signal is sent.",
        examples=("aiw-flow daemon stop",),
    )
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
        if loop_input.kind == LoopInputKind.FORK:
            cmd_handoff(
                argparse.Namespace(
                    session_id=args.session_id,
                    handoff_command="create",
                    focus=None,
                ),
                store,
            )
            handoff = store.read_artifact_text(args.session_id, "handoff.md")
            turn_args = argparse.Namespace(
                session_id=args.session_id,
                phase=phase,
                prompt=handoff,
                prompt_file=None,
                timeout=args.timeout,
            )
            result = await _execute_turn(turn_args, store, allow_missing_thread=True, reset_thread=True)
            print("Forked a fresh Thread from the handoff and exited the loop.")
            return result
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
