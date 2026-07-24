from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Mapping, Optional, Protocol


SESSION_STATES = {
    "created",
    "active",
    "running",
    "paused",
    "failed",
    "completed",
    "archived",
    "deleted",
}

VALID_TRANSITIONS = {
    ("created", "running"),
    ("running", "active"),
    ("running", "failed"),
    ("active", "running"),
    ("active", "paused"),
    ("paused", "running"),
    ("failed", "running"),
    ("active", "completed"),
    ("completed", "archived"),
    ("archived", "deleted"),
}


def utc_now() -> datetime:
    return datetime.now(timezone.utc)


def isoformat(value: datetime) -> str:
    return value.astimezone(timezone.utc).isoformat()


@dataclass(frozen=True)
class TurnRequest:
    session_id: str
    prompt: str
    workspace: Path
    thread_id: Optional[str]
    instructions: str
    memory: str
    phase: str
    timeout_seconds: Optional[int] = None
    output_dir: Optional[Path] = None
    turn_number: int = 0
    ephemeral: bool = False


@dataclass(frozen=True)
class TurnResult:
    thread_id: Optional[str]
    final_output: str
    exit_code: int
    events_file: Path
    output_file: Path
    started_at: datetime
    completed_at: datetime
    interrupted: bool = False
    stderr_file: Optional[Path] = None
    metadata: Dict[str, Any] = field(default_factory=dict)


class CodexBackend(Protocol):
    async def start(self) -> None:
        ...

    async def run_turn(self, request: TurnRequest) -> TurnResult:
        ...

    async def close(self) -> None:
        ...


@dataclass
class SessionSection:
    id: str
    title: str
    backend: str
    state: str
    created_at: str
    updated_at: str
    ephemeral: bool = False


@dataclass
class CodexSection:
    thread_id: Optional[str]
    thread_name: str
    codex_home: Optional[str]
    model: Optional[str]
    profile: Optional[str]
    last_turn: int


@dataclass
class WorkspaceSection:
    source_repo: Optional[str]
    workspace_path: Optional[str]
    is_git: bool
    use_worktree: bool
    base_ref: Optional[str]
    branch: Optional[str]
    base_commit: Optional[str]
    head_commit: Optional[str]
    dirty: bool


@dataclass
class InstructionsSection:
    system_file: str
    memory_file: str
    instructions_hash: Optional[str]
    memory_hash: Optional[str]


@dataclass
class ExecutionSection:
    current_phase: Optional[str]
    last_command: List[str]
    last_exit_code: Optional[int]
    last_started_at: Optional[str]
    last_completed_at: Optional[str]


@dataclass
class ResultSection:
    status: str
    tests_passed: Optional[bool]
    final_output_file: Optional[str]
    patch_file: Optional[str]
    error_message: Optional[str]


@dataclass
class SessionStatus:
    schema_version: int
    session: SessionSection
    codex: CodexSection
    workspace: WorkspaceSection
    instructions: InstructionsSection
    execution: ExecutionSection
    result: ResultSection
    extra: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        result = {
            "schema_version": self.schema_version,
            "session": self.session.__dict__.copy(),
            "codex": self.codex.__dict__.copy(),
            "workspace": self.workspace.__dict__.copy(),
            "instructions": self.instructions.__dict__.copy(),
            "execution": self.execution.__dict__.copy(),
            "result": self.result.__dict__.copy(),
        }
        result.update(self.extra)
        return result

    @classmethod
    def from_dict(cls, raw: Mapping[str, Any]) -> "SessionStatus":
        known = {
            "schema_version",
            "session",
            "codex",
            "workspace",
            "instructions",
            "execution",
            "result",
        }
        extra = {key: value for key, value in raw.items() if key not in known}
        return cls(
            schema_version=int(raw.get("schema_version", 1)),
            session=SessionSection(**dict(raw.get("session", {}))),
            codex=CodexSection(**dict(raw.get("codex", {}))),
            workspace=WorkspaceSection(**dict(raw.get("workspace", {}))),
            instructions=InstructionsSection(**dict(raw.get("instructions", {}))),
            execution=ExecutionSection(**dict(raw.get("execution", {}))),
            result=ResultSection(**dict(raw.get("result", {}))),
            extra=extra,
        )


@dataclass(frozen=True)
class PromptSnapshot:
    content: str
    sha256: str
    path: Path


@dataclass(frozen=True)
class LockInfo:
    pid: int
    hostname: str
    acquired_at: str
    session_id: Optional[str]


@dataclass
class AppConfig:
    model: Optional[str] = None
    profile: Optional[str] = None
    sandbox: Optional[str] = None
    approval_policy: Optional[str] = None
    codex_home: Optional[str] = None
    timeout: Optional[int] = None
    additional_codex_args: List[str] = field(default_factory=list)


@dataclass
class CreateSessionRequest:
    session_id: str
    title: str
    instructions_text: str
    workspace_path: Path
    codex_config: AppConfig
    ephemeral: bool = False

