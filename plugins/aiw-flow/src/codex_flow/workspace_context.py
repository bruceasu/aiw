from __future__ import annotations

import os
import re
from dataclasses import dataclass
from pathlib import Path
from typing import List, Sequence, Tuple

from codex_flow.process_utils import run_command


METADATA_FILENAMES = {
    "agents.md",
    "cargo.toml",
    "go.mod",
    "makefile",
    "package.json",
    "pyproject.toml",
    "readme",
    "readme.md",
    "requirements.txt",
}

SKIPPED_DIRECTORY_NAMES = {
    "__pycache__",
    "build",
    "dist",
    "node_modules",
    "target",
    "vendor",
}

_ASSIGNMENT_PATTERN = re.compile(
    r"""(?im)^(\s*(?:export\s+)?["']?[A-Za-z0-9_.-]*"""
    r"""(?:api[_-]?key|password|passwd|secret|token)"""
    r"""[A-Za-z0-9_.-]*["']?\s*[:=]\s*)(.+)$"""
)
_BEARER_PATTERN = re.compile(r"(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+")
_PRIVATE_KEY_PATTERN = re.compile(
    r"-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----",
    re.DOTALL,
)


@dataclass(frozen=True)
class ContextLimits:
    max_depth: int = 3
    max_entries: int = 80
    max_file_bytes: int = 2_000
    max_total_bytes: int = 12_000


def redact_sensitive_text(text: str) -> str:
    redacted = _ASSIGNMENT_PATTERN.sub(lambda match: match.group(1) + "[REDACTED]", text)
    redacted = _BEARER_PATTERN.sub("Bearer [REDACTED]", redacted)
    return _PRIVATE_KEY_PATTERN.sub("[REDACTED PRIVATE KEY]", redacted)


def collect_workspace_context(workspace: Path, limits: ContextLimits = ContextLimits()) -> str:
    resolved = workspace.resolve()
    if not resolved.is_dir():
        raise ValueError("Workspace does not exist: {}".format(resolved))

    entries, metadata_files, tree_truncated = _discover_workspace(resolved, limits)
    sections = [
        "# Workspace Context",
        "",
        "Workspace: `{}`".format(resolved),
        "",
        "## Filesystem Structure",
        "",
        "~~~~text",
        ".",
    ]
    sections.extend(entries)
    if tree_truncated:
        sections.append("[TRUNCATED: entry limit reached]")
    sections.extend(["~~~~", "", "## Project Metadata"])

    remaining_bytes = limits.max_total_bytes
    content_truncated = False
    if not metadata_files:
        sections.extend(["", "No allow-listed metadata files were found."])
    for path in metadata_files:
        if remaining_bytes <= 0:
            content_truncated = True
            break
        relative = path.relative_to(resolved).as_posix()
        try:
            raw = path.read_bytes()
        except (OSError, PermissionError) as exc:
            sections.extend(["", "### `{}`".format(relative), "", "Unavailable: `{}`".format(type(exc).__name__)])
            continue
        allowed = min(len(raw), limits.max_file_bytes, remaining_bytes)
        snippet = raw[:allowed].decode("utf-8", errors="replace")
        snippet = redact_sensitive_text(snippet)
        sections.extend(["", "### `{}`".format(relative), "", "~~~~text", snippet.rstrip(), "~~~~"])
        remaining_bytes -= allowed
        if allowed < len(raw):
            sections.append("[TRUNCATED: file or total byte limit reached]")
            content_truncated = True

    sections.extend(["", "## Git Context", ""])
    sections.extend(_collect_git_context(resolved))
    if content_truncated:
        sections.extend(["", "Context content was truncated to the configured byte limits."])
    return "\n".join(sections).rstrip() + "\n"


def _discover_workspace(
    workspace: Path,
    limits: ContextLimits,
) -> Tuple[List[str], List[Path], bool]:
    entries: List[str] = []
    metadata_files: List[Path] = []
    truncated = False

    for current_root, directory_names, file_names in os.walk(workspace, topdown=True, followlinks=False):
        current = Path(current_root)
        depth = len(current.relative_to(workspace).parts)
        directory_names[:] = sorted(
            name
            for name in directory_names
            if _include_directory(name) and depth < limits.max_depth
        )
        file_names = sorted(file_names)

        children: Sequence[Tuple[str, bool]] = [
            *((name, True) for name in directory_names),
            *((name, False) for name in file_names),
        ]
        for name, is_directory in children:
            if len(entries) >= limits.max_entries:
                truncated = True
                directory_names[:] = []
                break
            path = current / name
            relative = path.relative_to(workspace)
            entry_depth = len(relative.parts)
            if entry_depth > limits.max_depth:
                continue
            indent = "  " * max(0, entry_depth - 1)
            entries.append("{}{}{}".format(indent, name, "/" if is_directory else ""))
            if not is_directory and name.lower() in METADATA_FILENAMES:
                metadata_files.append(path)
        if truncated:
            break

    return entries, metadata_files, truncated


def _include_directory(name: str) -> bool:
    lowered = name.lower()
    return not name.startswith(".") and lowered not in SKIPPED_DIRECTORY_NAMES


def _collect_git_context(workspace: Path) -> List[str]:
    try:
        branch = run_command(["git", "branch", "--show-current"], cwd=workspace).strip() or "(detached)"
        commit = run_command(["git", "log", "-1", "--oneline"], cwd=workspace).strip()
        status = run_command(["git", "status", "--short"], cwd=workspace).strip()
    except Exception as exc:
        return ["Git metadata unavailable: `{}`".format(type(exc).__name__)]

    return [
        "- Branch: `{}`".format(redact_sensitive_text(branch)),
        "- Last commit: `{}`".format(redact_sensitive_text(commit or "(none)")),
        "- Working tree: {}".format("dirty" if status else "clean"),
    ]
