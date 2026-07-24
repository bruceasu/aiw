from __future__ import annotations

import hashlib
from pathlib import Path
from typing import Optional

from codex_flow.models import PromptSnapshot
from codex_flow.file_utils import atomic_write_text


def compose_prompt(instructions: str, memory: str, phase: str, prompt: str) -> str:
    return (
        "[Persistent Execution Instructions]\n\n"
        f"{instructions.strip()}\n\n"
        "[Session Memory]\n\n"
        f"{memory.strip()}\n\n"
        "[Current Phase]\n\n"
        f"{phase.strip()}\n\n"
        "[Current Task]\n\n"
        f"{prompt.rstrip()}\n"
    )


def prompt_sha256(text: str) -> str:
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def save_prompt(prompt_dir: Path, turn_number: int, phase: str, content: str) -> PromptSnapshot:
    prompt_dir.mkdir(parents=True, exist_ok=True)
    path = prompt_dir / "{:04d}-{}.md".format(turn_number, phase)
    atomic_write_text(path, content)
    return PromptSnapshot(content=content, sha256=prompt_sha256(content), path=path)


def load_prompt_text(prompt: Optional[str], prompt_file: Optional[Path], stdin_text: Optional[str]) -> str:
    values = []
    if prompt:
        values.append(prompt)
    if prompt_file:
        values.append(prompt_file.read_text(encoding="utf-8"))
    if stdin_text:
        values.append(stdin_text)
    return "\n\n".join(item.rstrip() for item in values if item and item.strip())

