#!/usr/bin/env python3
"""
aiw-cxs.py

Lightweight Codex CLI session viewer and helper.

Features:
  - Scan Codex session JSONL files
  - Cache session metadata to speed up repeated listing
  - List, show, and tail sessions in readable form
  - Bind human-friendly aliases to Codex session IDs
  - Browse workspace sessions in a desktop GUI
  - Resume interactive Codex in a session's original directory
  - Run `codex exec` or `codex exec resume`
  - Attach UTF-8 text files to a prompt
  - Atomically update local index/cache files

This tool intentionally does not implement Agent orchestration, MCP lifecycle,
Git worktrees, or workflow state machines.

No third-party dependencies.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
from collections import deque
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Iterable, Iterator, Mapping, Optional, Sequence


DEFAULT_CODEX_HOME = Path(os.environ.get("CODEX_HOME", Path.home() / ".codex"))
DEFAULT_CODEX_SESSIONS = DEFAULT_CODEX_HOME / "sessions"
DEFAULT_WORKSPACE = Path.cwd() / ".ai"
INDEX_RELATIVE = Path("sessions") / "index.json"
CACHE_RELATIVE = Path("sessions") / "cache.json"
INDEX_SCHEMA_VERSION = 2
CACHE_SCHEMA_VERSION = 2
DEFAULT_SCAN_EVENTS = 300
MAX_ATTACH_BYTES = 8 * 1024 * 1024

UUID_RE = re.compile(
    r"[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-"
    r"[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"
)
ALIAS_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$")

ROLE_KEYS = ("role", "type", "kind")
TEXT_KEYS = ("content", "text", "message", "input", "output")
TIME_KEYS = ("timestamp", "time", "created_at", "createdAt", "ts")
SESSION_ID_KEYS = (
    "thread_id",
    "threadId",
    "session_id",
    "sessionId",
    "conversation_id",
    "conversationId",
    "id",
)
NESTED_KEYS = ("message", "item", "event", "delta", "payload", "data")
WORKING_DIRECTORY_KEYS = (
    "cwd",
    "working_directory",
    "workingDirectory",
    "workdir",
)


class AiwCxsError(RuntimeError):
    """Expected user-facing failure."""


class AliasConflictError(AiwCxsError):
    """An alias mutation would overwrite an existing binding."""


@dataclass(frozen=True)
class SessionMeta:
    session_id: str
    path: Path
    mtime_ns: int
    size: int
    title: str
    first_user: str
    turns: int
    original_cwd: Optional[Path] = None

    @property
    def mtime(self) -> float:
        return self.mtime_ns / 1_000_000_000

    @property
    def mtime_text(self) -> str:
        return dt.datetime.fromtimestamp(self.mtime).strftime("%Y-%m-%d %H:%M:%S")

    def to_cache(self) -> dict[str, Any]:
        data = asdict(self)
        data["path"] = str(self.path)
        data["original_cwd"] = (
            str(self.original_cwd) if self.original_cwd is not None else None
        )
        return data

    @classmethod
    def from_cache(cls, data: Mapping[str, Any]) -> "SessionMeta":
        if "original_cwd" not in data:
            raise KeyError("original_cwd")
        original_cwd = data.get("original_cwd")
        return cls(
            session_id=str(data["session_id"]),
            path=Path(str(data["path"])),
            mtime_ns=int(data["mtime_ns"]),
            size=int(data["size"]),
            title=str(data.get("title", "")),
            first_user=str(data.get("first_user", "")),
            turns=int(data.get("turns", 0)),
            original_cwd=(
                Path(str(original_cwd)).expanduser().resolve()
                if original_cwd
                else None
            ),
        )


@dataclass(frozen=True)
class InteractiveResumePlan:
    command: tuple[str, ...]
    cwd: Optional[Path]
    can_launch: bool
    reason: str = ""


def eprint(*args: object) -> None:
    print(*args, file=sys.stderr)


def now_iso() -> str:
    return dt.datetime.now().astimezone().isoformat(timespec="seconds")


def workspace_index_path(workspace: Path) -> Path:
    return workspace / INDEX_RELATIVE


def workspace_cache_path(workspace: Path) -> Path:
    return workspace / CACHE_RELATIVE


def atomic_write_text(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, temp_name = tempfile.mkstemp(
        prefix=f".{path.name}.", suffix=".tmp", dir=str(path.parent)
    )
    temp_path = Path(temp_name)
    try:
        with os.fdopen(fd, "w", encoding="utf-8", newline="\n") as handle:
            handle.write(text)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temp_path, path)
    except BaseException:
        try:
            temp_path.unlink(missing_ok=True)
        except OSError:
            pass
        raise


def write_json_atomic(path: Path, data: Mapping[str, Any]) -> None:
    text = json.dumps(data, indent=2, ensure_ascii=False, sort_keys=True) + "\n"
    atomic_write_text(path, text)


def default_index() -> dict[str, Any]:
    return {"schema_version": INDEX_SCHEMA_VERSION, "aliases": {}}


def load_index(workspace: Path, *, create: bool = True) -> dict[str, Any]:
    path = workspace_index_path(workspace)
    if not path.exists():
        data = default_index()
        if create:
            write_json_atomic(path, data)
        return data
    try:
        raw = json.loads(path.read_text(encoding="utf-8"))
    except OSError as exc:
        raise AiwCxsError(f"Cannot read index: {path}: {exc}") from exc
    except json.JSONDecodeError as exc:
        raise AiwCxsError(f"Invalid index JSON: {path}: {exc}") from exc
    if not isinstance(raw, dict):
        raise AiwCxsError(f"Index root must be a JSON object: {path}")
    aliases = raw.get("aliases")
    if not isinstance(aliases, dict):
        raw["aliases"] = {}
    raw.setdefault("schema_version", 1)
    return raw


def save_index(workspace: Path, data: Mapping[str, Any]) -> None:
    normalized = dict(data)
    normalized["schema_version"] = INDEX_SCHEMA_VERSION
    normalized.setdefault("aliases", {})
    write_json_atomic(workspace_index_path(workspace), normalized)


def load_cache(workspace: Path) -> dict[str, Any]:
    path = workspace_cache_path(workspace)
    if not path.exists():
        return {"schema_version": CACHE_SCHEMA_VERSION, "files": {}}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return {"schema_version": CACHE_SCHEMA_VERSION, "files": {}}
    if not isinstance(data, dict) or not isinstance(data.get("files"), dict):
        return {"schema_version": CACHE_SCHEMA_VERSION, "files": {}}
    return data


def save_cache(workspace: Path, files: Mapping[str, Any]) -> None:
    write_json_atomic(
        workspace_cache_path(workspace),
        {
            "schema_version": CACHE_SCHEMA_VERSION,
            "updated_at": now_iso(),
            "files": dict(files),
        },
    )


def iter_jsonl(path: Path, limit: Optional[int] = None) -> Iterator[dict[str, Any]]:
    count = 0
    try:
        with path.open("r", encoding="utf-8", errors="replace") as handle:
            for raw_line in handle:
                line = raw_line.strip()
                if not line:
                    continue
                try:
                    obj = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if not isinstance(obj, dict):
                    continue
                yield obj
                count += 1
                if limit is not None and count >= limit:
                    return
    except OSError:
        return


def tail_jsonl(path: Path, limit: int) -> Iterable[dict[str, Any]]:
    if limit <= 0:
        return ()
    items: deque[dict[str, Any]] = deque(maxlen=limit)
    for obj in iter_jsonl(path):
        items.append(obj)
    return tuple(items)


def flatten_text(value: Any, depth: int = 0) -> str:
    if value is None or depth > 6:
        return ""
    if isinstance(value, str):
        return value
    if isinstance(value, (int, float, bool)):
        return str(value)
    if isinstance(value, list):
        return "\n".join(
            part for item in value if (part := flatten_text(item, depth + 1))
        )
    if isinstance(value, dict):
        for key in TEXT_KEYS:
            if key in value:
                text = flatten_text(value[key], depth + 1)
                if text:
                    return text
        parts: list[str] = []
        for item in value.values():
            if isinstance(item, (str, int, float, bool)):
                parts.append(str(item))
        return " ".join(parts)
    return ""


def get_role(obj: Mapping[str, Any], depth: int = 0) -> str:
    if depth > 6:
        return "event"
    for key in ROLE_KEYS:
        value = obj.get(key)
        if not isinstance(value, str):
            continue
        low = value.lower()
        if low in {"user", "assistant", "system", "tool", "developer"}:
            return low
        if "user" in low:
            return "user"
        if "assistant" in low or "agent" in low:
            return "assistant"
        if "tool" in low or "command" in low:
            return "tool"
    for key in NESTED_KEYS:
        nested = obj.get(key)
        if isinstance(nested, dict):
            role = get_role(nested, depth + 1)
            if role != "event":
                return role
    return "event"


def get_text(obj: Mapping[str, Any], depth: int = 0) -> str:
    if depth > 6:
        return ""
    for key in TEXT_KEYS:
        if key in obj:
            text = flatten_text(obj[key], depth + 1)
            if text:
                return text
    for key in NESTED_KEYS:
        nested = obj.get(key)
        if isinstance(nested, dict):
            text = get_text(nested, depth + 1)
            if text:
                return text
    return ""


def get_time(obj: Mapping[str, Any]) -> str:
    for key in TIME_KEYS:
        value = obj.get(key)
        if value is not None:
            return str(value)
    for key in NESTED_KEYS:
        nested = obj.get(key)
        if isinstance(nested, dict):
            value = get_time(nested)
            if value:
                return value
    return ""


def normalize_original_cwd(value: Any) -> Optional[Path]:
    if not isinstance(value, str) or not value.strip():
        return None
    try:
        path = Path(value).expanduser()
    except (OSError, ValueError):
        return None
    if not path.is_absolute():
        return None
    try:
        return path.resolve()
    except OSError:
        return path.absolute()


def extract_original_cwd(
    obj: Mapping[str, Any], depth: int = 0
) -> Optional[Path]:
    if depth > 6:
        return None
    for key in WORKING_DIRECTORY_KEYS:
        path = normalize_original_cwd(obj.get(key))
        if path is not None:
            return path
    for key in NESTED_KEYS:
        value = obj.get(key)
        if isinstance(value, dict):
            path = extract_original_cwd(value, depth + 1)
            if path is not None:
                return path
    return None


def is_session_metadata_record(obj: Mapping[str, Any]) -> bool:
    record_type = str(obj.get("type", "")).strip().lower()
    return record_type in {"session_meta", "session_metadata"}


def extract_session_id_from_obj(
    obj: Mapping[str, Any], depth: int = 0
) -> Optional[str]:
    if depth > 6:
        return None
    for key in SESSION_ID_KEYS:
        value = obj.get(key)
        if not isinstance(value, str):
            continue
        match = UUID_RE.search(value)
        if match:
            return match.group(0)
        if key not in {"id"} and len(value) >= 8:
            return value
    for value in obj.values():
        if isinstance(value, str):
            match = UUID_RE.search(value)
            if match:
                return match.group(0)
        elif isinstance(value, dict):
            nested = extract_session_id_from_obj(value, depth + 1)
            if nested:
                return nested
    return None


def extract_session_id_from_path(path: Path) -> str:
    match = UUID_RE.search(path.name)
    if match:
        return match.group(0)
    for obj in iter_jsonl(path, limit=30):
        session_id = extract_session_id_from_obj(obj)
        if session_id:
            return session_id
    return path.stem


def truncate(text: str, width: int) -> str:
    normalized = " ".join(text.split())
    if len(normalized) <= width:
        return normalized
    return normalized[: max(0, width - 1)] + "…"


def inspect_session_file(path: Path, scan_events: int) -> SessionMeta:
    stat = path.stat()
    session_id = extract_session_id_from_path(path)
    first_user = ""
    title = ""
    turns = 0
    original_cwd: Optional[Path] = None
    for obj in iter_jsonl(path, limit=scan_events):
        if original_cwd is None and is_session_metadata_record(obj):
            original_cwd = extract_original_cwd(obj)
        role = get_role(obj)
        text = get_text(obj)
        if role in {"user", "assistant"} and text:
            turns += 1
        if role == "user" and text and not first_user:
            first_user = truncate(text, 96)
        if not title:
            for key in ("title", "summary", "name"):
                value = obj.get(key)
                if isinstance(value, str) and value.strip():
                    title = truncate(value, 96)
                    break
    if not title:
        title = first_user or path.stem
    return SessionMeta(
        session_id=session_id,
        path=path,
        mtime_ns=stat.st_mtime_ns,
        size=stat.st_size,
        title=title,
        first_user=first_user,
        turns=turns,
        original_cwd=original_cwd,
    )


def scan_sessions(
    root: Path,
    workspace: Path,
    *,
    refresh: bool = False,
    scan_events: int = DEFAULT_SCAN_EVENTS,
) -> list[SessionMeta]:
    if not root.exists():
        return []

    cache = load_cache(workspace)
    cached_files = cache.get("files", {}) if not refresh else {}
    if not isinstance(cached_files, dict):
        cached_files = {}

    files: list[Path] = []
    try:
        files = [path for path in root.rglob("*.jsonl") if path.is_file()]
    except OSError as exc:
        raise AiwCxsError(f"Cannot scan sessions directory {root}: {exc}") from exc

    metas: list[SessionMeta] = []
    new_cache: dict[str, Any] = {}
    for path in files:
        try:
            stat = path.stat()
        except OSError:
            continue
        key = str(path.resolve())
        cached = cached_files.get(key)
        meta: Optional[SessionMeta] = None
        if isinstance(cached, dict):
            try:
                candidate = SessionMeta.from_cache(cached)
                if (
                    candidate.mtime_ns == stat.st_mtime_ns
                    and candidate.size == stat.st_size
                ):
                    meta = candidate
            except (KeyError, TypeError, ValueError):
                meta = None
        if meta is None:
            try:
                meta = inspect_session_file(path, scan_events)
            except OSError:
                continue
        metas.append(meta)
        new_cache[key] = meta.to_cache()

    try:
        save_cache(workspace, new_cache)
    except OSError as exc:
        eprint(f"Warning: cannot update session cache: {exc}")

    metas.sort(key=lambda item: item.mtime_ns, reverse=True)
    return metas


def normalized_path_key(path: Path) -> str:
    return os.path.normcase(os.path.abspath(str(path.expanduser())))


def session_belongs_to_workspace(session: SessionMeta, workspace: Path) -> bool:
    if session.original_cwd is None:
        return False
    session_key = normalized_path_key(session.original_cwd)
    workspace_key = normalized_path_key(workspace)
    try:
        return os.path.commonpath([session_key, workspace_key]) == workspace_key
    except ValueError:
        return False


def filter_sessions_for_workspace(
    sessions: Iterable[SessionMeta],
    workspace: Path,
    *,
    include_all: bool = False,
) -> list[SessionMeta]:
    items = list(sessions)
    if include_all:
        return items
    return [
        session
        for session in items
        if session_belongs_to_workspace(session, workspace)
    ]


def alias_session_id(value: Any) -> Optional[str]:
    if isinstance(value, dict):
        session_id = value.get("session_id")
        return str(session_id) if session_id else None
    return str(value) if value else None


def resolve_alias(ref: str, workspace: Path) -> str:
    aliases = load_index(workspace, create=False).get("aliases", {})
    if isinstance(aliases, dict) and ref in aliases:
        session_id = alias_session_id(aliases[ref])
        if session_id:
            return session_id
    return ref


def resolve_session(
    ref: str,
    root: Path,
    workspace: Path,
    *,
    refresh: bool = False,
) -> SessionMeta:
    target = resolve_alias(ref, workspace)
    sessions = scan_sessions(root, workspace, refresh=refresh)

    exact = [item for item in sessions if item.session_id == target]
    if exact:
        return exact[0]

    prefix = [item for item in sessions if item.session_id.startswith(target)]
    if len(prefix) == 1:
        return prefix[0]
    if len(prefix) > 1:
        candidates = "\n".join(
            f"  {item.session_id}  {item.mtime_text}  {item.path}" for item in prefix[:10]
        )
        raise AiwCxsError(f"Ambiguous session reference {ref!r}:\n{candidates}")

    path_matches = [item for item in sessions if target in item.path.name]
    if len(path_matches) == 1:
        return path_matches[0]
    if len(path_matches) > 1:
        candidates = "\n".join(
            f"  {item.session_id}  {item.mtime_text}  {item.path}"
            for item in path_matches[:10]
        )
        raise AiwCxsError(f"Ambiguous session path match {ref!r}:\n{candidates}")

    raise AiwCxsError(f"Session not found: {ref}")


def resolve_session_id_or_ref(ref: str, root: Path, workspace: Path) -> str:
    target = resolve_alias(ref, workspace)
    try:
        return resolve_session(target, root, workspace).session_id
    except AiwCxsError:
        if UUID_RE.fullmatch(target):
            return target
        raise


def reverse_aliases(workspace: Path) -> dict[str, list[str]]:
    aliases = load_index(workspace, create=False).get("aliases", {})
    result: dict[str, list[str]] = {}
    if not isinstance(aliases, dict):
        return result
    for alias, value in aliases.items():
        session_id = alias_session_id(value)
        if session_id:
            result.setdefault(session_id, []).append(str(alias))
    for names in result.values():
        names.sort()
    return result


def list_cmd(args: argparse.Namespace) -> int:
    sessions = scan_sessions(
        args.sessions_dir,
        args.workspace,
        refresh=args.refresh,
        scan_events=args.scan_events,
    )
    if args.current_workspace:
        sessions = filter_sessions_for_workspace(sessions, args.workspace.parent)
    sessions = sessions[: args.limit] if args.limit is not None else sessions
    aliases = reverse_aliases(args.workspace)

    if args.json:
        payload = []
        for session in sessions:
            item = session.to_cache()
            item["updated"] = session.mtime_text
            item["aliases"] = aliases.get(session.session_id, [])
            payload.append(item)
        print(json.dumps(payload, indent=2, ensure_ascii=False))
        return 0

    if not sessions:
        print(f"No Codex sessions found under {args.sessions_dir}")
        return 0

    print(f"{'ALIAS':20} {'SESSION_ID':36} {'UPDATED':19} {'TURNS':5} TITLE")
    print("-" * 116)
    for session in sessions:
        names = ",".join(aliases.get(session.session_id, []))
        print(
            f"{truncate(names, 20):20} "
            f"{session.session_id[:36]:36} "
            f"{session.mtime_text:19} "
            f"{session.turns:<5} "
            f"{truncate(session.title, 76)}"
        )
    return 0


def render_objects(objects: Iterable[Mapping[str, Any]]) -> str:
    blocks: list[str] = []
    for obj in objects:
        text = get_text(obj)
        if not text:
            continue
        role = get_role(obj).upper()
        timestamp = get_time(obj)
        header = f"## {role}"
        if timestamp:
            header += f" [{timestamp}]"
        blocks.append(f"{header}\n\n{text.strip()}")
    return "\n\n".join(blocks)


def render_conversation(objects: Iterable[Mapping[str, Any]]) -> str:
    return render_objects(
        obj
        for obj in objects
        if get_role(obj).lower() in {"user", "assistant"}
    )


def show_cmd(args: argparse.Namespace) -> int:
    session = resolve_session(
        args.ref, args.sessions_dir, args.workspace, refresh=args.refresh
    )
    print(f"# {session.session_id}")
    print(f"Path: {session.path}")
    print(f"Updated: {session.mtime_text}")
    print("-" * 80)
    print(render_objects(iter_jsonl(session.path, limit=args.events)))
    return 0


def tail_cmd(args: argparse.Namespace) -> int:
    session = resolve_session(
        args.ref, args.sessions_dir, args.workspace, refresh=args.refresh
    )
    print(render_objects(tail_jsonl(session.path, args.events)))
    return 0


def validate_alias(name: str) -> str:
    if not ALIAS_RE.fullmatch(name):
        raise AiwCxsError(
            "Invalid alias. Use 1-80 characters: letters, digits, '.', '_' or '-'; "
            "the first character must be alphanumeric."
        )
    return name


def bind_alias(
    workspace: Path,
    name: str,
    session: SessionMeta,
    *,
    note: str = "",
    replace: bool = False,
) -> None:
    name = validate_alias(name)
    index = load_index(workspace, create=False)
    aliases = index.setdefault("aliases", {})
    assert isinstance(aliases, dict)
    if name in aliases and not replace:
        old_id = alias_session_id(aliases[name]) or "unknown"
        raise AliasConflictError(
            f"Alias already exists: {name} -> {old_id}. Confirm replacement first."
        )
    aliases[name] = {
        "session_id": session.session_id,
        "path": str(session.path),
        "updated_at": now_iso(),
        "note": note,
    }
    save_index(workspace, index)


def rename_alias(
    workspace: Path,
    old_name: str,
    new_name: str,
    *,
    replace: bool = False,
) -> None:
    new_name = validate_alias(new_name)
    index = load_index(workspace, create=False)
    aliases = index.get("aliases", {})
    if not isinstance(aliases, dict) or old_name not in aliases:
        raise AiwCxsError(f"Alias not found: {old_name}")
    if new_name != old_name and new_name in aliases and not replace:
        raise AliasConflictError(f"Alias already exists: {new_name}")
    value = aliases[old_name]
    aliases[new_name] = value
    if new_name != old_name:
        del aliases[old_name]
    save_index(workspace, index)


def remove_alias(workspace: Path, name: str) -> str:
    index = load_index(workspace, create=False)
    aliases = index.get("aliases", {})
    if not isinstance(aliases, dict) or name not in aliases:
        raise AiwCxsError(f"Alias not found: {name}")
    old_id = alias_session_id(aliases.pop(name)) or "unknown"
    save_index(workspace, index)
    return old_id


def bind_cmd(args: argparse.Namespace) -> int:
    name = validate_alias(args.name)
    if args.ref:
        session = resolve_session(args.ref, args.sessions_dir, args.workspace)
    else:
        sessions = scan_sessions(args.sessions_dir, args.workspace)
        if not sessions:
            raise AiwCxsError("No sessions found to bind")
        session = sessions[0]

    bind_alias(
        args.workspace,
        name,
        session,
        note=args.note or "",
        replace=args.force,
    )
    print(f"Bound {name} -> {session.session_id}")
    return 0


def unbind_cmd(args: argparse.Namespace) -> int:
    old_id = remove_alias(args.workspace, args.name)
    print(f"Removed {args.name} -> {old_id}")
    return 0


def aliases_cmd(args: argparse.Namespace) -> int:
    aliases = load_index(args.workspace, create=False).get("aliases", {})
    if not isinstance(aliases, dict) or not aliases:
        print("No aliases found")
        return 0

    if args.json:
        print(json.dumps(aliases, indent=2, ensure_ascii=False, sort_keys=True))
        return 0

    print(f"{'ALIAS':24} {'SESSION_ID':36} NOTE")
    print("-" * 96)
    for alias, value in sorted(aliases.items()):
        if isinstance(value, dict):
            session_id = str(value.get("session_id", ""))[:36]
            note = str(value.get("note", ""))
        else:
            session_id = str(value)[:36]
            note = ""
        print(f"{alias:24} {session_id:36} {note}")
    return 0


def read_attach(path_text: str) -> tuple[Path, str]:
    path = Path(path_text).expanduser().resolve()
    if not path.is_file():
        raise AiwCxsError(f"Attach file not found: {path}")
    try:
        size = path.stat().st_size
    except OSError as exc:
        raise AiwCxsError(f"Cannot stat attach file {path}: {exc}") from exc
    if size > MAX_ATTACH_BYTES:
        raise AiwCxsError(
            f"Attach file is too large ({size} bytes; limit {MAX_ATTACH_BYTES}): {path}"
        )
    try:
        return path, path.read_text(encoding="utf-8", errors="replace")
    except OSError as exc:
        raise AiwCxsError(f"Cannot read attach file {path}: {exc}") from exc


def build_prompt(args: argparse.Namespace) -> str:
    parts: list[str] = []
    if args.message:
        parts.append(args.message)
    for item in args.attach or []:
        path, content = read_attach(item)
        parts.append(f"Additional context from {path}:\n\n{content}")
    if not parts:
        parts.append("Continue the previous task.")
    return "\n\n---\n\n".join(parts)


def build_codex_exec_command(
    *,
    prompt: str,
    sessions_dir: Path,
    workspace: Path,
    session_ref: Optional[str] = None,
    use_last: bool = False,
    output_last_message: Optional[Path] = None,
    codex_args: Sequence[str] = (),
) -> list[str]:
    if session_ref and use_last:
        raise AiwCxsError("--session and --last cannot be used together")

    if session_ref or use_last:
        command = ["codex", "exec", "resume"]
        if use_last:
            command.append("--last")
        else:
            assert session_ref is not None
            command.append(resolve_session_id_or_ref(session_ref, sessions_dir, workspace))
    else:
        command = ["codex", "exec"]

    command.extend(codex_args)
    if output_last_message:
        output_last_message.parent.mkdir(parents=True, exist_ok=True)
        command.extend(["-o", str(output_last_message)])
    command.append(prompt)
    return command


def display_command(command: Sequence[str]) -> str:
    if os.name == "nt":
        return subprocess.list2cmdline(list(command))
    return shlex.join(command)


def build_interactive_resume_command(session_id: str) -> tuple[str, ...]:
    return ("codex", "resume", session_id)


def validate_interactive_resume_capability() -> str:
    executable = shutil.which("codex")
    if executable is None:
        raise AiwCxsError("codex command not found in PATH")
    try:
        completed = subprocess.run(
            [executable, "resume", "--help"],
            check=False,
            capture_output=True,
            text=True,
        )
    except OSError as exc:
        raise AiwCxsError(f"Cannot inspect the codex command: {exc}") from exc
    output = f"{completed.stdout}\n{completed.stderr}".lower()
    if completed.returncode != 0 or not re.search(
        r"usage:\s*codex(?:\.exe|\.cmd)?\s+resume\b", output
    ):
        raise AiwCxsError(
            "The installed codex command does not advertise interactive "
            "`codex resume` support."
        )
    return executable


def plan_interactive_resume(
    session: SessionMeta,
    *,
    terminal_available: Optional[bool] = None,
) -> InteractiveResumePlan:
    command = build_interactive_resume_command(session.session_id)
    if session.original_cwd is None:
        return InteractiveResumePlan(
            command=command,
            cwd=None,
            can_launch=False,
            reason="The session working directory is unknown.",
        )
    cwd = session.original_cwd.expanduser().resolve()
    if not cwd.is_dir():
        return InteractiveResumePlan(
            command=command,
            cwd=cwd,
            can_launch=False,
            reason=f"The session working directory does not exist: {cwd}",
        )
    if terminal_available is None:
        terminal_available = sys.stdin.isatty() and sys.stdout.isatty()
    if not terminal_available:
        return InteractiveResumePlan(
            command=command,
            cwd=cwd,
            can_launch=False,
            reason="No interactive terminal is available.",
        )
    return InteractiveResumePlan(command=command, cwd=cwd, can_launch=True)


def run_interactive_resume(
    plan: InteractiveResumePlan, *, executable: Optional[str] = None
) -> int:
    if not plan.can_launch or plan.cwd is None:
        raise AiwCxsError(plan.reason or "Interactive resume is unavailable")
    executable = executable or validate_interactive_resume_capability()
    command = [executable, *plan.command[1:]]
    try:
        completed = subprocess.run(command, cwd=plan.cwd, check=False)
    except OSError as exc:
        raise AiwCxsError(f"Cannot start interactive Codex: {exc}") from exc
    return completed.returncode


def session_table_values(
    session: SessionMeta, aliases: Sequence[str]
) -> tuple[object, ...]:
    return (
        session.session_id,
        ", ".join(aliases),
        session.title,
        session.mtime_text,
        session.turns,
        str(session.original_cwd or "unknown"),
    )


def run_or_print_codex_command(
    command: Sequence[str], *, dry_run: bool, cwd: Optional[Path]
) -> int:
    if dry_run:
        if cwd:
            print(f"cd {shlex.quote(str(cwd))}")
        print(display_command(command))
        return 0
    try:
        completed = subprocess.run(list(command), cwd=cwd, check=False)
    except FileNotFoundError as exc:
        raise AiwCxsError("codex command not found in PATH") from exc
    except OSError as exc:
        raise AiwCxsError(f"Cannot start codex: {exc}") from exc
    return completed.returncode


def normalize_output_path(value: Optional[str]) -> Optional[Path]:
    return Path(value).expanduser().resolve() if value else None


def normalize_cwd(value: Optional[Path]) -> Optional[Path]:
    if value is None:
        return None
    path = value.expanduser().resolve()
    if not path.is_dir():
        raise AiwCxsError(f"Working directory does not exist: {path}")
    return path


def resume_cmd(args: argparse.Namespace) -> int:
    command = build_codex_exec_command(
        prompt=build_prompt(args),
        sessions_dir=args.sessions_dir,
        workspace=args.workspace,
        session_ref=args.ref,
        output_last_message=normalize_output_path(args.output_last_message),
        codex_args=args.codex_arg or (),
    )
    return run_or_print_codex_command(
        command, dry_run=args.dry_run, cwd=normalize_cwd(args.cwd)
    )


def exec_cmd(args: argparse.Namespace) -> int:
    command = build_codex_exec_command(
        prompt=build_prompt(args),
        sessions_dir=args.sessions_dir,
        workspace=args.workspace,
        session_ref=args.session,
        use_last=args.last,
        output_last_message=normalize_output_path(args.output_last_message),
        codex_args=args.codex_arg or (),
    )
    return run_or_print_codex_command(
        command, dry_run=args.dry_run, cwd=normalize_cwd(args.cwd)
    )


def path_cmd(args: argparse.Namespace) -> int:
    session = resolve_session(args.ref, args.sessions_dir, args.workspace)
    print(session.path)
    return 0


def gui_cmd(args: argparse.Namespace) -> int:
    try:
        import tkinter as tk
        from tkinter import messagebox, simpledialog, ttk
    except ImportError as exc:
        raise AiwCxsError(
            "Tkinter is unavailable in the configured Python runtime. "
            "Use `aiw cxs list`, `show`, and `bind` instead."
        ) from exc

    try:
        root = tk.Tk()
    except tk.TclError as exc:
        raise AiwCxsError(
            "Cannot open the session GUI. Start it from a desktop environment "
            "with an available display."
        ) from exc

    root.title("AIW Codex Sessions")
    root.geometry("1180x720")
    root.minsize(820, 500)

    session_by_id: dict[str, SessionMeta] = {}
    alias_by_id: dict[str, list[str]] = {}
    resume_holder: dict[str, Any] = {}
    show_all = tk.BooleanVar(value=False)

    toolbar = ttk.Frame(root, padding=(8, 8, 8, 4))
    toolbar.pack(fill=tk.X)
    ttk.Label(toolbar, text=f"Workspace: {args.workspace.parent}").pack(side=tk.LEFT)

    body = ttk.Panedwindow(root, orient=tk.HORIZONTAL)
    body.pack(fill=tk.BOTH, expand=True, padx=8, pady=(4, 8))

    left = ttk.Frame(body)
    right = ttk.Frame(body)
    body.add(left, weight=3)
    body.add(right, weight=4)

    columns = ("session_id", "alias", "title", "updated", "turns", "cwd")
    tree = ttk.Treeview(left, columns=columns, show="headings", selectmode="browse")
    headings = {
        "session_id": ("Session ID", 250),
        "alias": ("Alias", 130),
        "title": ("Title", 260),
        "updated": ("Updated", 145),
        "turns": ("Turns", 55),
        "cwd": ("Original directory", 280),
    }
    for column, (label, width) in headings.items():
        tree.heading(column, text=label)
        tree.column(column, width=width, minwidth=50, stretch=column in {"title", "cwd"})
    tree_scroll = ttk.Scrollbar(left, orient=tk.VERTICAL, command=tree.yview)
    tree.configure(yscrollcommand=tree_scroll.set)
    tree.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
    tree_scroll.pack(side=tk.RIGHT, fill=tk.Y)

    preview = tk.Text(right, wrap=tk.WORD, state=tk.DISABLED, padx=10, pady=10)
    preview_scroll = ttk.Scrollbar(right, orient=tk.VERTICAL, command=preview.yview)
    preview.configure(yscrollcommand=preview_scroll.set)
    preview.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
    preview_scroll.pack(side=tk.RIGHT, fill=tk.Y)

    def selected_session() -> Optional[SessionMeta]:
        selected = tree.selection()
        if not selected:
            return None
        return session_by_id.get(selected[0])

    def set_preview(text: str) -> None:
        preview.configure(state=tk.NORMAL)
        preview.delete("1.0", tk.END)
        preview.insert("1.0", text)
        preview.configure(state=tk.DISABLED)

    def refresh_sessions() -> None:
        sessions = scan_sessions(
            args.sessions_dir,
            args.workspace,
            refresh=True,
            scan_events=DEFAULT_SCAN_EVENTS,
        )
        visible = filter_sessions_for_workspace(
            sessions,
            args.workspace.parent,
            include_all=bool(show_all.get()),
        )
        aliases = reverse_aliases(args.workspace)
        session_by_id.clear()
        alias_by_id.clear()
        for item in tree.get_children():
            tree.delete(item)
        for session in visible:
            session_by_id[session.session_id] = session
            names = aliases.get(session.session_id, [])
            alias_by_id[session.session_id] = names
            tree.insert(
                "",
                tk.END,
                iid=session.session_id,
                values=session_table_values(session, names),
            )
        set_preview(
            "Select a session to preview it."
            if visible
            else "No sessions found for this workspace."
        )

    def preview_selected(_event: object = None) -> None:
        session = selected_session()
        if session is None:
            return
        text = render_conversation(iter_jsonl(session.path, limit=args.events))
        set_preview(text or "No user or assistant messages found.")

    def edit_alias() -> None:
        session = selected_session()
        if session is None:
            messagebox.showinfo("Alias", "Select a session first.", parent=root)
            return
        names = alias_by_id.get(session.session_id, [])
        old_name = names[0] if names else ""
        new_name = simpledialog.askstring(
            "Set alias",
            "Enter an alias:",
            initialvalue=old_name,
            parent=root,
        )
        if new_name is None or new_name == old_name:
            return
        def apply_change(change: Any) -> bool:
            try:
                change(False)
                return True
            except AliasConflictError as exc:
                if not messagebox.askyesno(
                    "Replace alias",
                    f"{exc}\n\nReplace the existing alias?",
                    parent=root,
                ):
                    return False
                change(True)
                return True

        try:
            if old_name:
                changed = apply_change(
                    lambda replace: rename_alias(
                        args.workspace,
                        old_name,
                        new_name,
                        replace=replace,
                    )
                )
            else:
                changed = apply_change(
                    lambda replace: bind_alias(
                        args.workspace,
                        new_name,
                        session,
                        replace=replace,
                    )
                )
        except AiwCxsError as exc:
            messagebox.showerror("Alias error", str(exc), parent=root)
            return
        if not changed:
            return
        refresh_sessions()

    def delete_alias() -> None:
        session = selected_session()
        if session is None:
            messagebox.showinfo("Alias", "Select a session first.", parent=root)
            return
        names = alias_by_id.get(session.session_id, [])
        if not names:
            messagebox.showinfo("Alias", "This session has no alias.", parent=root)
            return
        name = names[0]
        if len(names) > 1:
            selected_name = simpledialog.askstring(
                "Remove alias",
                "Enter the alias to remove:",
                initialvalue=name,
                parent=root,
            )
            if selected_name is None:
                return
            name = selected_name
        if not messagebox.askyesno(
            "Remove alias",
            f"Remove alias {name!r}?",
            parent=root,
        ):
            return
        try:
            remove_alias(args.workspace, name)
        except AiwCxsError as exc:
            messagebox.showerror("Alias error", str(exc), parent=root)
            return
        refresh_sessions()

    def resume_selected() -> None:
        session = selected_session()
        if session is None:
            messagebox.showinfo("Resume", "Select a session first.", parent=root)
            return
        plan = plan_interactive_resume(session)
        if not plan.can_launch:
            command = display_command(plan.command)
            cwd_text = str(plan.cwd) if plan.cwd is not None else "unknown"
            copied = False
            try:
                root.clipboard_clear()
                root.clipboard_append(command)
                root.update()
                copied = True
            except tk.TclError:
                copied = False
            copy_note = "\nThe command was copied to the clipboard." if copied else ""
            messagebox.showwarning(
                "Cannot resume here",
                (
                    f"{plan.reason}\n\nDirectory: {cwd_text}\n"
                    f"Command: {command}{copy_note}"
                ),
                parent=root,
            )
            return
        try:
            executable = validate_interactive_resume_capability()
        except AiwCxsError as exc:
            messagebox.showerror("Cannot resume", str(exc), parent=root)
            return
        resume_holder["plan"] = plan
        resume_holder["executable"] = executable
        root.destroy()

    ttk.Checkbutton(
        toolbar,
        text="Show all workspaces",
        variable=show_all,
        command=refresh_sessions,
    ).pack(side=tk.RIGHT, padx=(8, 0))
    ttk.Button(toolbar, text="Refresh", command=refresh_sessions).pack(side=tk.RIGHT)

    actions = ttk.Frame(root, padding=(8, 0, 8, 8))
    actions.pack(fill=tk.X)
    ttk.Button(actions, text="Set / Rename Alias", command=edit_alias).pack(
        side=tk.LEFT
    )
    ttk.Button(actions, text="Remove Alias", command=delete_alias).pack(
        side=tk.LEFT, padx=(8, 0)
    )
    ttk.Button(actions, text="Resume Interactive Codex", command=resume_selected).pack(
        side=tk.RIGHT
    )

    tree.bind("<<TreeviewSelect>>", preview_selected)
    refresh_sessions()
    root.mainloop()

    plan = resume_holder.get("plan")
    return (
        run_interactive_resume(
            plan,
            executable=resume_holder.get("executable"),
        )
        if isinstance(plan, InteractiveResumePlan)
        else 0
    )


def add_execution_options(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("message", nargs="?", help="prompt/context")
    parser.add_argument(
        "-a", "--attach", action="append", help="attach a UTF-8 text file; repeatable"
    )
    parser.add_argument("-o", "--output-last-message", help="write final model message")
    parser.add_argument("--cwd", type=Path, help="working directory for codex")
    parser.add_argument(
        "--codex-arg",
        action="append",
        help="pass one additional argument to codex; repeatable",
    )
    parser.add_argument("--dry-run", action="store_true", help="print command only")


def make_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="aiw cxs",
        description="View and manage Codex CLI session records and resume Codex work.",
        epilog=(
            "Examples:\n"
            "  aiw cxs list -n 20\n"
            "  aiw cxs exec --last \"continue the latest task\"\n"
            "  aiw cxs show payment-retry\n"
            "\nRun `aiw cxs COMMAND --help` for command details."
        ),
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        "--sessions-dir", type=Path, default=DEFAULT_CODEX_SESSIONS
    )
    parser.add_argument("--workspace", type=Path, default=DEFAULT_WORKSPACE)

    sub = parser.add_subparsers(dest="cmd", required=True)

    command = sub.add_parser("list", help="list recent Codex sessions")
    command.add_argument("-n", "--limit", type=int, default=20)
    command.add_argument("--json", action="store_true", help="output JSON")
    command.add_argument("--refresh", action="store_true", help="ignore metadata cache")
    command.add_argument(
        "--current-workspace",
        action="store_true",
        help="show only sessions from the current workspace",
    )
    command.add_argument(
        "--scan-events", type=int, default=DEFAULT_SCAN_EVENTS, help=argparse.SUPPRESS
    )
    command.set_defaults(func=list_cmd)

    command = sub.add_parser("show", help="show a readable session preview")
    command.add_argument("ref", help="session ID/prefix or alias")
    command.add_argument("-e", "--events", type=int, default=80)
    command.add_argument("--refresh", action="store_true")
    command.set_defaults(func=show_cmd)

    command = sub.add_parser("tail", help="show the last session events")
    command.add_argument("ref", help="session ID/prefix or alias")
    command.add_argument("-e", "--events", type=int, default=30)
    command.add_argument("--refresh", action="store_true")
    command.set_defaults(func=tail_cmd)

    command = sub.add_parser("bind", help="bind an alias to a session")
    command.add_argument("name", help="alias, e.g. payment-retry")
    command.add_argument("ref", nargs="?", help="session ID/prefix; newest if omitted")
    command.add_argument("--note", default="")
    command.add_argument("--force", action="store_true", help="replace existing alias")
    command.set_defaults(func=bind_cmd)

    command = sub.add_parser("unbind", help="remove an alias")
    command.add_argument("name")
    command.set_defaults(func=unbind_cmd)

    command = sub.add_parser("aliases", help="list aliases")
    command.add_argument("--json", action="store_true")
    command.set_defaults(func=aliases_cmd)

    command = sub.add_parser("resume", help="run codex exec resume by alias or ID")
    command.add_argument("ref", help="session ID/prefix or alias")
    add_execution_options(command)
    command.set_defaults(func=resume_cmd)

    command = sub.add_parser("exec", help="run codex exec, optionally resuming a session")
    command.add_argument("--session", help="session ID/prefix or alias")
    command.add_argument("--last", action="store_true", help="resume newest Codex session")
    add_execution_options(command)
    command.set_defaults(func=exec_cmd)

    command = sub.add_parser("path", help="print the JSONL path for a session")
    command.add_argument("ref", help="session ID/prefix or alias")
    command.set_defaults(func=path_cmd)

    command = sub.add_parser(
        "gui",
        help="browse workspace sessions and resume interactive Codex",
        description=(
            "Open a desktop browser for current-workspace Codex sessions. "
            "Preview conversations, maintain aliases, and hand the selected "
            "session to interactive `codex resume`."
        ),
        epilog=(
            "The GUI uses the session's original working directory. If the "
            "directory or an interactive terminal is unavailable, it shows a "
            "copyable command instead.\n\nExample:\n  aiw cxs gui"
        ),
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    command.add_argument(
        "-e",
        "--events",
        type=int,
        default=80,
        help="maximum JSONL events to inspect in the preview",
    )
    command.set_defaults(func=gui_cmd)

    return parser


def normalize_args(args: argparse.Namespace) -> None:
    args.sessions_dir = args.sessions_dir.expanduser().resolve()
    args.workspace = args.workspace.expanduser().resolve()
    if hasattr(args, "limit") and args.limit is not None and args.limit < 0:
        raise AiwCxsError("--limit must be zero or greater")
    if hasattr(args, "events") and args.events < 0:
        raise AiwCxsError("--events must be zero or greater")


def main(argv: Optional[Sequence[str]] = None) -> int:
    parser = make_parser()
    try:
        args = parser.parse_args(argv)
        normalize_args(args)
        return int(args.func(args))
    except AiwCxsError as exc:
        eprint(f"error: {exc}")
        return 2
    except KeyboardInterrupt:
        eprint("Interrupted")
        return 130


if __name__ == "__main__":
    raise SystemExit(main())
