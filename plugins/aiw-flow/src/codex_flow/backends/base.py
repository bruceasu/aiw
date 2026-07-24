from __future__ import annotations

import json
from dataclasses import dataclass, field
from typing import Any, Dict, Iterable, List, Optional


@dataclass
class ParsedExecOutput:
    thread_id: Optional[str] = None
    final_output: str = ""
    raw_lines: List[str] = field(default_factory=list)
    invalid_lines: List[str] = field(default_factory=list)
    events: List[Dict[str, Any]] = field(default_factory=list)


def parse_exec_jsonl(lines: Iterable[str]) -> ParsedExecOutput:
    parsed = ParsedExecOutput()
    messages: List[str] = []
    for line in lines:
        normalized = line.rstrip("\n")
        parsed.raw_lines.append(normalized)
        if not normalized.strip():
            continue
        try:
            event = json.loads(normalized)
        except json.JSONDecodeError:
            parsed.invalid_lines.append(normalized)
            continue
        parsed.events.append(event)
        thread_id = event.get("thread_id") or event.get("session_id")
        if event.get("type") == "thread.started" and thread_id:
            parsed.thread_id = str(thread_id)
        item = event.get("item")
        if isinstance(item, dict):
            text = item.get("text")
            if item.get("type") == "agent_message" and isinstance(text, str):
                messages.append(text)
        if event.get("type") == "agent_message" and isinstance(event.get("text"), str):
            messages.append(event["text"])
        if event.get("type") == "response.completed":
            message = event.get("message")
            if isinstance(message, str):
                messages.append(message)
    if messages:
        parsed.final_output = "\n".join(messages).strip()
    return parsed

