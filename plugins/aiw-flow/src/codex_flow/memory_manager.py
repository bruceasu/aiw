from __future__ import annotations

import hashlib
from pathlib import Path

from codex_flow.file_utils import atomic_write_text


DEFAULT_MEMORY_TEMPLATE = """# Session Memory

## Goal

## Confirmed Findings

## Decisions

## Modified Files

## Validation State

## Open Issues
"""


def memory_sha256(text: str) -> str:
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


class MemoryManager:
    def __init__(self, path: Path):
        self.path = path

    def ensure_exists(self) -> None:
        if not self.path.exists():
            atomic_write_text(self.path, DEFAULT_MEMORY_TEMPLATE)

    def read(self) -> str:
        self.ensure_exists()
        return self.path.read_text(encoding="utf-8")

    def replace(self, text: str) -> str:
        atomic_write_text(self.path, text.rstrip() + "\n")
        return memory_sha256(self.read())

    def append_note(self, text: str) -> str:
        current = self.read().rstrip() + "\n\n## Memory Note\n\n" + text.rstrip() + "\n"
        atomic_write_text(self.path, current)
        return memory_sha256(current)

