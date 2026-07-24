from __future__ import annotations

import json
from pathlib import Path
from typing import Callable, Dict, Iterable, Optional

from codex_flow.file_utils import atomic_write_text
from codex_flow.lock_manager import FileLock
from codex_flow.memory_manager import DEFAULT_MEMORY_TEMPLATE
from codex_flow.models import (
    CreateSessionRequest,
    SessionStatus,
    CodexSection,
    ExecutionSection,
    InstructionsSection,
    ResultSection,
    SessionSection,
    WorkspaceSection,
    VALID_TRANSITIONS,
    isoformat,
    utc_now,
)
from codex_flow.safety import validate_session_id


class SessionStoreError(RuntimeError):
    """Session storage error."""


class SessionStore:
    def __init__(self, root: Path):
        self.root = root
        self.sessions_dir = root / "sessions"
        self.locks_dir = root / "locks"
        self.logs_dir = root / "logs"
        self.archives_dir = root / "archive"
        self._ensure_root()

    def _ensure_root(self) -> None:
        for path in (
            self.root,
            self.sessions_dir,
            self.locks_dir,
            self.logs_dir,
            self.archives_dir,
        ):
            path.mkdir(parents=True, exist_ok=True)

    def session_dir(self, session_id: str) -> Path:
        validate_session_id(session_id)
        return self.sessions_dir / session_id

    def lock_path(self, session_id: str) -> Path:
        return self.locks_dir / f"{session_id}.lock"

    def status_path(self, session_id: str) -> Path:
        return self.session_dir(session_id) / "status.json"

    def artifact_path(self, session_id: str, filename: str) -> Path:
        if not filename or Path(filename).name != filename or filename in {".", ".."}:
            raise SessionStoreError("Artifact filename must be a single file name.")
        return self.session_dir(session_id) / "artifacts" / filename

    def write_artifact_text(
        self,
        session_id: str,
        filename: str,
        content: str,
        *,
        lock_timeout: float = 10.0,
    ) -> Path:
        path = self.artifact_path(session_id, filename)
        lock = FileLock(self.lock_path(session_id), session_id=session_id, timeout=lock_timeout)
        with lock:
            if not self.status_path(session_id).exists():
                raise SessionStoreError("Session does not exist.")
            atomic_write_text(path, content)
        return path

    def read_artifact_text(self, session_id: str, filename: str) -> str:
        path = self.artifact_path(session_id, filename)
        if not path.exists():
            raise SessionStoreError("Artifact does not exist: {}".format(filename))
        return path.read_text(encoding="utf-8")

    def create_session(self, request: CreateSessionRequest) -> SessionStatus:
        validate_session_id(request.session_id)
        session_dir = self.session_dir(request.session_id)
        if session_dir.exists():
            raise SessionStoreError("Session already exists.")
        session_dir.mkdir(parents=True)
        for child in ("prompts", "outputs", "artifacts"):
            (session_dir / child).mkdir(parents=True, exist_ok=True)
        now = isoformat(utc_now())
        status = SessionStatus(
            schema_version=1,
            session=SessionSection(
                id=request.session_id,
                title=request.title,
                backend="exec",
                state="created",
                created_at=now,
                updated_at=now,
                ephemeral=request.ephemeral,
            ),
            codex=CodexSection(
                thread_id=None,
                thread_name=request.session_id,
                codex_home=request.codex_config.codex_home,
                model=request.codex_config.model,
                profile=request.codex_config.profile,
                last_turn=0,
            ),
            workspace=WorkspaceSection(
                workspace_path=str(request.workspace_path),
                source_repo=None,
                is_git=False,
                use_worktree=False,
                base_ref=None,
                branch=None,
                base_commit=None,
                head_commit=None,
                dirty=False,
            ),
            instructions=InstructionsSection(
                system_file="instructions.md",
                memory_file="memory.md",
                instructions_hash=None,
                memory_hash=None,
            ),
            execution=ExecutionSection(
                current_phase=None,
                last_command=[],
                last_exit_code=None,
                last_started_at=None,
                last_completed_at=None,
            ),
            result=ResultSection(
                status="not_started",
                tests_passed=None,
                final_output_file=None,
                patch_file=None,
                error_message=None,
            ),
        )
        atomic_write_text(session_dir / "instructions.md", request.instructions_text)
        atomic_write_text(session_dir / "memory.md", DEFAULT_MEMORY_TEMPLATE)
        atomic_write_text(session_dir / "events.jsonl", "")
        self.save_status(status)
        return status

    def load_status(self, session_id: str) -> SessionStatus:
        path = self.status_path(session_id)
        if not path.exists():
            raise SessionStoreError("Session does not exist.")
        data = json.loads(path.read_text(encoding="utf-8"))
        return SessionStatus.from_dict(data)

    def save_status(self, status: SessionStatus) -> None:
        path = self.status_path(status.session.id)
        atomic_write_text(path, json.dumps(status.to_dict(), indent=2, ensure_ascii=False) + "\n")

    def update_status(
        self,
        session_id: str,
        update_fn: Callable[[SessionStatus], SessionStatus],
        *,
        lock_timeout: float = 10.0,
    ) -> SessionStatus:
        lock = FileLock(self.lock_path(session_id), session_id=session_id, timeout=lock_timeout)
        with lock:
            current = self.load_status(session_id)
            updated = update_fn(current)
            self.save_status(updated)
            return updated

    def transition_state(self, session_id: str, target_state: str) -> SessionStatus:
        def apply(status: SessionStatus) -> SessionStatus:
            current = status.session.state
            if current == target_state:
                return status
            if (current, target_state) not in VALID_TRANSITIONS:
                raise SessionStoreError("Invalid state transition: {} -> {}".format(current, target_state))
            status.session.state = target_state
            status.session.updated_at = isoformat(utc_now())
            return status

        return self.update_status(session_id, apply)

    def list_sessions(self) -> Iterable[SessionStatus]:
        if not self.sessions_dir.exists():
            return []
        items = []
        for path in sorted(self.sessions_dir.iterdir()):
            if path.is_dir() and (path / "status.json").exists():
                items.append(self.load_status(path.name))
        return items

    def archive_session(self, session_id: str) -> Path:
        source = self.session_dir(session_id)
        if not source.exists():
            raise SessionStoreError("Session does not exist.")
        target = self.archives_dir / session_id
        if target.exists():
            raise SessionStoreError("Archive target already exists.")
        source.rename(target)
        return target

