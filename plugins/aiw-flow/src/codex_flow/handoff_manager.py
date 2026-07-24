from __future__ import annotations

from pathlib import Path
from typing import Dict, List, Optional, Tuple

from codex_flow.models import SessionStatus


DEFAULT_OUTPUT_EXCERPT_CHARS = 4_000


def render_handoff(
    status: SessionStatus,
    session_dir: Path,
    memory: str,
    *,
    focus: Optional[str] = None,
    max_output_chars: int = DEFAULT_OUTPUT_EXCERPT_CHARS,
) -> str:
    memory_sections = _parse_memory_sections(memory)
    latest_output_path, latest_output = _latest_output(session_dir, max_output_chars)
    artifact_references = _artifact_references(session_dir, latest_output_path)
    next_action = focus.strip() if focus and focus.strip() else _default_next_action(memory_sections)

    lines = [
        "# Agent Handoff: {}".format(status.session.title),
        "",
        "Session ID: `{}`".format(status.session.id),
        "",
        "## Continuation Focus",
        "",
        next_action,
        "",
        "## Goal",
        "",
        _section(memory_sections, "Goal", status.session.title),
        "",
        "## Current State",
        "",
        "- Session state: `{}`".format(status.session.state),
        "- Current phase: `{}`".format(status.execution.current_phase or "not started"),
        "- Last turn: `{}`".format(status.codex.last_turn),
        "- Last exit code: `{}`".format(
            status.execution.last_exit_code if status.execution.last_exit_code is not None else "not available"
        ),
        "- Workspace: `{}`".format(status.workspace.workspace_path or "not available"),
        "- Codex thread: `{}`".format(status.codex.thread_id or "not bound"),
        "",
        "## Confirmed Findings",
        "",
        _section(memory_sections, "Confirmed Findings"),
        "",
        "## Decisions",
        "",
        _section(memory_sections, "Decisions"),
        "",
        "## Modified Files",
        "",
        _section(memory_sections, "Modified Files"),
        "",
        "## Validation State",
        "",
        _section(memory_sections, "Validation State"),
        "",
        "## Open Issues",
        "",
        _section(memory_sections, "Open Issues"),
        "",
        "## Latest Agent Output",
        "",
    ]
    if latest_output_path is None:
        lines.append("No completed agent output is stored.")
    else:
        lines.extend(
            [
                "Excerpt from `{}`:".format(latest_output_path),
                "",
                "~~~~text",
                latest_output,
                "~~~~",
            ]
        )
    lines.extend(
        [
            "",
            "## Recommended Next Action",
            "",
            next_action,
            "",
            "## Suggested Skills",
            "",
            "- Read the referenced session artifacts before changing code.",
            "- Inspect the workspace Git diff and status.",
            "- Run the repository-standard focused and full validation commands.",
            "",
            "## Artifact References",
            "",
        ]
    )
    lines.extend("- `{}`".format(path) for path in artifact_references)
    return "\n".join(lines).rstrip() + "\n"


def _parse_memory_sections(memory: str) -> Dict[str, str]:
    sections: Dict[str, List[str]] = {}
    current: Optional[str] = None
    for line in memory.splitlines():
        if line.startswith("## "):
            current = line[3:].strip()
            sections.setdefault(current, [])
        elif current is not None:
            sections[current].append(line)
    return {name: "\n".join(lines).strip() for name, lines in sections.items()}


def _section(sections: Dict[str, str], name: str, fallback: str = "Not recorded.") -> str:
    return sections.get(name, "").strip() or fallback


def _default_next_action(sections: Dict[str, str]) -> str:
    open_issues = sections.get("Open Issues", "").strip()
    if open_issues:
        return "Resolve the first recorded open issue, then update session memory."
    return "Review the latest output and session memory, then continue the current phase."


def _latest_output(session_dir: Path, max_chars: int) -> Tuple[Optional[str], str]:
    outputs_dir = session_dir / "outputs"
    candidates = sorted(outputs_dir.glob("*-final.txt")) if outputs_dir.exists() else []
    if not candidates:
        return None, ""
    latest = candidates[-1]
    text = latest.read_text(encoding="utf-8", errors="replace").strip()
    relative = latest.relative_to(session_dir).as_posix()
    if len(text) > max_chars:
        text = text[:max_chars].rstrip() + "\n[TRUNCATED: see complete output artifact]"
    return relative, text


def _artifact_references(session_dir: Path, latest_output_path: Optional[str]) -> List[str]:
    references = ["status.json", "instructions.md", "memory.md", "events.jsonl"]
    prompt_candidates = sorted((session_dir / "prompts").glob("*.md"))
    if prompt_candidates:
        references.append(prompt_candidates[-1].relative_to(session_dir).as_posix())
    if latest_output_path:
        references.append(latest_output_path)
    for name in ("workspace-context.md", "git-status.txt", "git-diff.txt", "final.patch", "summary.json"):
        path = session_dir / "artifacts" / name
        if path.exists():
            references.append(path.relative_to(session_dir).as_posix())
    return references
