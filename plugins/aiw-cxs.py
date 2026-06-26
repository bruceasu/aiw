#!/usr/bin/env python3
"""
aiw_codex_session.py

Small Codex session viewer/helper for Personal AI Workspace style workflows.

Features:
  - Scan ~/.codex/sessions for Codex session jsonl files
  - List recent sessions with inferred title/id/time
  - Preview/show/tail a session in readable text
  - Bind a business task name to a Codex session id
  - Resume a session by id or alias through `codex exec resume`
    - Run `codex exec` directly, or target a session via `--session`
  - Attach notes/context files to a session resume prompt

No third-party dependencies.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import re
import shlex
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional, Tuple


DEFAULT_CODEX_SESSIONS = Path.home() / ".codex" / "sessions"
DEFAULT_WORKSPACE = Path.cwd() / ".ai"
INDEX_RELATIVE = Path("sessions") / "index.json"

UUID_RE = re.compile(
    r"[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"
)

ROLE_KEYS = ("role", "type", "kind")
TEXT_KEYS = ("content", "text", "message", "input", "output")
TIME_KEYS = ("timestamp", "time", "created_at", "createdAt", "ts")
SESSION_ID_KEYS = ("session_id", "sessionId", "conversation_id", "conversationId", "id")


@dataclass
class SessionMeta:
    session_id: str
    path: Path
    mtime: float
    title: str
    first_user: str
    turns: int

    @property
    def mtime_text(self) -> str:
        return dt.datetime.fromtimestamp(self.mtime).strftime("%Y-%m-%d %H:%M:%S")


def eprint(*args: object) -> None:
    print(*args, file=sys.stderr)


def workspace_index_path(workspace: Path) -> Path:
    return workspace / INDEX_RELATIVE


def ensure_index(workspace: Path) -> Path:
    p = workspace_index_path(workspace)
    p.parent.mkdir(parents=True, exist_ok=True)
    if not p.exists():
        p.write_text(json.dumps({"aliases": {}}, indent=2, ensure_ascii=False), encoding="utf-8")
    return p


def load_index(workspace: Path) -> Dict[str, Any]:
    p = ensure_index(workspace)
    try:
        data = json.loads(p.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise SystemExit(f"Invalid index JSON: {p}: {exc}")
    if "aliases" not in data or not isinstance(data["aliases"], dict):
        data["aliases"] = {}
    return data


def save_index(workspace: Path, data: Dict[str, Any]) -> None:
    p = ensure_index(workspace)
    p.write_text(json.dumps(data, indent=2, ensure_ascii=False, sort_keys=True), encoding="utf-8")


def iter_jsonl(path: Path, limit: Optional[int] = None) -> Iterable[Dict[str, Any]]:
    count = 0
    try:
        with path.open("r", encoding="utf-8", errors="replace") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    obj = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if isinstance(obj, dict):
                    yield obj
                    count += 1
                    if limit is not None and count >= limit:
                        return
    except OSError:
        return


def flatten_text(value: Any, depth: int = 0) -> str:
    if value is None:
        return ""
    if isinstance(value, str):
        return value
    if depth > 4:
        return ""
    if isinstance(value, list):
        parts = [flatten_text(v, depth + 1) for v in value]
        return "\n".join(p for p in parts if p)
    if isinstance(value, dict):
        # Common OpenAI-like shapes: {type: text, text: ...}, {content: [...]}
        for k in TEXT_KEYS:
            if k in value:
                t = flatten_text(value[k], depth + 1)
                if t:
                    return t
        # Last resort: collect short scalar values
        parts: List[str] = []
        for _, v in value.items():
            if isinstance(v, (str, int, float, bool)):
                parts.append(str(v))
        return " ".join(parts)
    return str(value)


def get_role(obj: Dict[str, Any]) -> str:
    for k in ROLE_KEYS:
        v = obj.get(k)
        if isinstance(v, str):
            low = v.lower()
            if low in {"user", "assistant", "system", "tool", "developer"}:
                return low
            if "user" in low:
                return "user"
            if "assistant" in low or "agent" in low:
                return "assistant"
            if "tool" in low:
                return "tool"
    # Some Codex events may wrap role under message
    msg = obj.get("message")
    if isinstance(msg, dict):
        return get_role(msg)
    return "event"


def get_text(obj: Dict[str, Any]) -> str:
    # Direct fields
    for k in TEXT_KEYS:
        if k in obj:
            t = flatten_text(obj[k])
            if t:
                return t
    # Nested message or item payload
    for k in ("message", "item", "event", "delta", "payload"):
        v = obj.get(k)
        if isinstance(v, dict):
            t = get_text(v)
            if t:
                return t
    return ""


def get_time(obj: Dict[str, Any]) -> str:
    for k in TIME_KEYS:
        v = obj.get(k)
        if v is not None:
            return str(v)
    return ""


def extract_session_id_from_obj(obj: Dict[str, Any]) -> Optional[str]:
    for k in SESSION_ID_KEYS:
        v = obj.get(k)
        if isinstance(v, str):
            m = UUID_RE.search(v)
            if m:
                return m.group(0)
            if k in {"session_id", "sessionId", "conversation_id", "conversationId"} and len(v) >= 8:
                return v
    for v in obj.values():
        if isinstance(v, str):
            m = UUID_RE.search(v)
            if m:
                return m.group(0)
        elif isinstance(v, dict):
            nested = extract_session_id_from_obj(v)
            if nested:
                return nested
    return None


def extract_session_id_from_path(path: Path) -> str:
    m = UUID_RE.search(path.name)
    if m:
        return m.group(0)
    for obj in iter_jsonl(path, limit=20):
        sid = extract_session_id_from_obj(obj)
        if sid:
            return sid
    return path.stem


def truncate(s: str, width: int) -> str:
    s = " ".join(s.split())
    if len(s) <= width:
        return s
    return s[: max(0, width - 1)] + "…"


def scan_sessions(root: Path) -> List[SessionMeta]:
    if not root.exists():
        return []
    files = sorted(root.rglob("*.jsonl"), key=lambda p: p.stat().st_mtime, reverse=True)
    metas: List[SessionMeta] = []
    for p in files:
        sid = extract_session_id_from_path(p)
        first_user = ""
        title = ""
        turns = 0
        for obj in iter_jsonl(p, limit=200):
            role = get_role(obj)
            text = get_text(obj)
            if role in {"user", "assistant"} and text:
                turns += 1
            if role == "user" and text and not first_user:
                first_user = truncate(text, 96)
            if not title:
                # Some event logs have title-ish fields.
                for key in ("title", "summary", "name"):
                    val = obj.get(key)
                    if isinstance(val, str) and val.strip():
                        title = truncate(val, 96)
                        break
        if not title:
            title = first_user or p.stem
        metas.append(SessionMeta(sid, p, p.stat().st_mtime, title, first_user, turns))
    return metas


def resolve_session(ref: str, root: Path, workspace: Path) -> SessionMeta:
    index = load_index(workspace)
    aliases = index.get("aliases", {})
    if ref in aliases:
        sid = aliases[ref].get("session_id") if isinstance(aliases[ref], dict) else aliases[ref]
    else:
        sid = ref
    sessions = scan_sessions(root)
    # Prefix match, then path/name match.
    matches = [s for s in sessions if s.session_id == sid or s.session_id.startswith(sid)]
    if not matches:
        matches = [s for s in sessions if sid in s.path.name]
    if not matches:
        raise SystemExit(f"Session not found: {ref}")
    if len(matches) > 1:
        eprint("Multiple sessions matched; using newest:")
        for s in matches[:5]:
            eprint(f"  {s.session_id}  {s.mtime_text}  {s.path}")
    return sorted(matches, key=lambda s: s.mtime, reverse=True)[0]


def resolve_session_id_or_ref(ref: str, root: Path, workspace: Path) -> str:
    try:
        return resolve_session(ref, root, workspace).session_id
    except SystemExit:
        # Allow explicit UUID (or UUID prefix) passthrough even if local session logs are missing.
        if UUID_RE.fullmatch(ref) or UUID_RE.match(ref):
            return ref
        raise


def list_cmd(args: argparse.Namespace) -> None:
    sessions = scan_sessions(args.sessions_dir)
    if not sessions:
        print(f"No Codex sessions found under {args.sessions_dir}")
        return
    index = load_index(args.workspace)
    reverse_alias: Dict[str, List[str]] = {}
    for alias, val in index.get("aliases", {}).items():
        sid = val.get("session_id") if isinstance(val, dict) else val
        if sid:
            reverse_alias.setdefault(sid, []).append(alias)
    print(f"{'ALIAS':20} {'SESSION_ID':36} {'UPDATED':19} {'TURNS':5} TITLE")
    print("-" * 110)
    for s in sessions[: args.limit]:
        aliases = ",".join(reverse_alias.get(s.session_id, []))
        print(f"{truncate(aliases, 20):20} {s.session_id[:36]:36} {s.mtime_text:19} {s.turns:<5} {truncate(s.title, 70)}")


def render_session(path: Path, max_events: Optional[int], tail: bool = False) -> List[str]:
    objs = list(iter_jsonl(path))
    if tail and max_events is not None:
        objs = objs[-max_events:]
    elif max_events is not None:
        objs = objs[:max_events]
    lines: List[str] = []
    for obj in objs:
        role = get_role(obj)
        text = get_text(obj)
        if not text:
            continue
        time = get_time(obj)
        header = role.upper()
        if time:
            header += f" [{time}]"
        lines.append(f"\n## {header}\n")
        lines.append(text.strip())
        lines.append("\n")
    return lines


def show_cmd(args: argparse.Namespace) -> None:
    s = resolve_session(args.ref, args.sessions_dir, args.workspace)
    print(f"# {s.session_id}")
    print(f"Path: {s.path}")
    print(f"Updated: {s.mtime_text}")
    print("-" * 80)
    lines = render_session(s.path, args.events, tail=False)
    print("\n".join(lines).strip())


def tail_cmd(args: argparse.Namespace) -> None:
    s = resolve_session(args.ref, args.sessions_dir, args.workspace)
    lines = render_session(s.path, args.events, tail=True)
    print("\n".join(lines).strip())


def bind_cmd(args: argparse.Namespace) -> None:
    if args.ref:
        s = resolve_session(args.ref, args.sessions_dir, args.workspace)
    else:
        sessions = scan_sessions(args.sessions_dir)
        if not sessions:
            raise SystemExit("No sessions found to bind")
        s = sessions[0]
    index = load_index(args.workspace)
    index.setdefault("aliases", {})[args.name] = {
        "session_id": s.session_id,
        "path": str(s.path),
        "updated_at": dt.datetime.now().isoformat(timespec="seconds"),
        "note": args.note or "",
    }
    save_index(args.workspace, index)
    print(f"Bound {args.name} -> {s.session_id}")


def aliases_cmd(args: argparse.Namespace) -> None:
    index = load_index(args.workspace)
    aliases = index.get("aliases", {})
    if not aliases:
        print("No aliases found")
        return
    print(f"{'ALIAS':24} {'SESSION_ID':36} NOTE")
    print("-" * 90)
    for alias, val in sorted(aliases.items()):
        if isinstance(val, dict):
            print(f"{alias:24} {str(val.get('session_id',''))[:36]:36} {val.get('note','')}")
        else:
            print(f"{alias:24} {str(val)[:36]:36}")


def build_resume_prompt(args: argparse.Namespace) -> str:
    parts: List[str] = []
    if args.message:
        parts.append(args.message)
    for file in args.attach or []:
        p = Path(file)
        if not p.exists():
            raise SystemExit(f"Attach file not found: {p}")
        content = p.read_text(encoding="utf-8", errors="replace")
        parts.append(f"Additional context from {p}:\n\n{content}")
    if not parts:
        parts.append("Continue the previous task.")
    return "\n\n---\n\n".join(parts)


def build_codex_exec_command(
    prompt: str,
    sessions_dir: Path,
    workspace: Path,
    session_ref: Optional[str] = None,
    use_last: bool = False,
    output_last_message: Optional[str] = None,
) -> List[str]:
    if session_ref and use_last:
        raise SystemExit("--session and --last cannot be used together")

    if session_ref or use_last:
        cmd = ["codex", "exec", "resume"]
        if use_last:
            cmd.append("--last")
        else:
            cmd.append(resolve_session_id_or_ref(session_ref, sessions_dir, workspace))
        if output_last_message:
            cmd.extend(["-o", output_last_message])
        cmd.append(prompt)
        return cmd
    cmd = ["codex", "exec"]
    if output_last_message:
        cmd.extend(["-o", output_last_message])
    cmd.append(prompt)
    return cmd


def run_or_print_codex_command(cmd: List[str], dry_run: bool) -> None:
    if dry_run:
        print(" ".join(shlex.quote(x) for x in cmd))
        return
    try:
        raise SystemExit(subprocess.call(cmd))
    except FileNotFoundError:
        raise SystemExit("codex command not found in PATH")


def resume_cmd(args: argparse.Namespace) -> None:
    prompt = build_resume_prompt(args)
    cmd = build_codex_exec_command(
        prompt=prompt,
        sessions_dir=args.sessions_dir,
        workspace=args.workspace,
        session_ref=args.ref,
        output_last_message=args.output_last_message,
    )
    run_or_print_codex_command(cmd, args.dry_run)


def open_cmd(args: argparse.Namespace) -> None:
    s = resolve_session(args.ref, args.sessions_dir, args.workspace)
    print(str(s.path))


def exec_cmd(args: argparse.Namespace) -> None:
    prompt = build_resume_prompt(args)
    cmd = build_codex_exec_command(
        prompt=prompt,
        sessions_dir=args.sessions_dir,
        workspace=args.workspace,
        session_ref=args.session,
        use_last=args.last,
        output_last_message=args.output_last_message,
    )
    run_or_print_codex_command(cmd, args.dry_run)


def make_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="aiw-codex-session",
        description="View and manage Codex CLI session records.",
    )
    p.add_argument("--sessions-dir", type=Path, default=DEFAULT_CODEX_SESSIONS)
    p.add_argument("--workspace", type=Path, default=DEFAULT_WORKSPACE)

    sub = p.add_subparsers(dest="cmd", required=True)

    sp = sub.add_parser("list", help="List recent Codex sessions")
    sp.add_argument("-n", "--limit", type=int, default=20)
    sp.set_defaults(func=list_cmd)

    sp = sub.add_parser("show", help="Show a readable session preview")
    sp.add_argument("ref", help="session id/prefix or alias")
    sp.add_argument("-e", "--events", type=int, default=80)
    sp.set_defaults(func=show_cmd)

    sp = sub.add_parser("tail", help="Show the last events of a session")
    sp.add_argument("ref", help="session id/prefix or alias")
    sp.add_argument("-e", "--events", type=int, default=30)
    sp.set_defaults(func=tail_cmd)

    sp = sub.add_parser("bind", help="Bind an alias to a session")
    sp.add_argument("name", help="business task alias, e.g. payment-retry")
    sp.add_argument("ref", nargs="?", help="session id/prefix; defaults to newest")
    sp.add_argument("--note", default="")
    sp.set_defaults(func=bind_cmd)

    sp = sub.add_parser("aliases", help="List aliases")
    sp.set_defaults(func=aliases_cmd)

    sp = sub.add_parser(
        "resume",
        help="Run codex exec resume by alias or id",
        description=(
            "Resume a Codex session and send a new prompt.\n\n"
            "Examples:\n"
            "  aiw cxs resume payment-retry \"continue with tests\"\n"
            "  aiw cxs resume 123e4567-e89b-12d3-a456-426614174000 --dry-run"
        ),
        formatter_class=argparse.RawTextHelpFormatter,
    )
    sp.add_argument("ref", help="session id/prefix or alias")
    sp.add_argument("message", nargs="?", help="new prompt/context to append")
    sp.add_argument("-a", "--attach", action="append", help="attach context file; can repeat")
    sp.add_argument("-o", "--output-last-message", help="write final model message to file")
    sp.add_argument("--dry-run", action="store_true", help="print the codex command only")
    sp.set_defaults(func=resume_cmd)

    sp = sub.add_parser(
        "exec",
        help="Run codex exec; optionally target a session",
        description=(
            "Run codex exec directly, or target a specific session.\n\n"
            "Examples:\n"
            "  aiw cxs exec \"summarize current diff\"\n"
            "  aiw cxs exec --session payment-retry \"continue implementation\"\n"
            "  aiw cxs exec --last \"continue the latest session\"\n"
            "  aiw cxs exec --session 123e4567-e89b-12d3-a456-426614174000 --dry-run"
        ),
        formatter_class=argparse.RawTextHelpFormatter,
    )
    sp.add_argument("message", nargs="?", help="initial prompt/context")
    sp.add_argument("-a", "--attach", action="append", help="attach context file; can repeat")
    sp.add_argument("--session", help="session id/prefix or alias; uses codex exec resume")
    sp.add_argument("--last", action="store_true", help="resume the newest recorded session")
    sp.add_argument("-o", "--output-last-message", help="write final model message to file")
    sp.add_argument("--dry-run", action="store_true", help="print the codex command only")
    sp.set_defaults(func=exec_cmd)

    sp = sub.add_parser("path", help="Print the jsonl path for a session")
    sp.add_argument("ref", help="session id/prefix or alias")
    sp.set_defaults(func=open_cmd)

    return p


def main(argv: Optional[List[str]] = None) -> int:
    parser = make_parser()
    args = parser.parse_args(argv)
    args.sessions_dir = args.sessions_dir.expanduser().resolve()
    args.workspace = args.workspace.expanduser().resolve()
    args.func(args)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
