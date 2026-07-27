from __future__ import annotations

from dataclasses import dataclass
from enum import Enum


LOOP_HELP = """Interactive commands:
  /help                  Show this help.
  /status                Show the current Session status.
  /memory                Show the current Session Memory.
  /handoff               Create artifacts/handoff.md.
  /fork                  Create a handoff, start a fresh Thread, and exit.
  /skills                List discoverable Codex Skills.
  /skill NAME MESSAGE    Invoke a discovered Skill for one turn.
  /done                  Finish Grill discovery and exit after the final response.
  /exit                  Exit without sending another turn.
  //text                 Send a message that starts with '/'.
"""


class LoopInputKind(str, Enum):
    MESSAGE = "message"
    EMPTY = "empty"
    HELP = "help"
    STATUS = "status"
    MEMORY = "memory"
    HANDOFF = "handoff"
    FORK = "fork"
    SKILLS = "skills"
    SKILL = "skill"
    DONE = "done"
    EXIT = "exit"
    UNKNOWN = "unknown"


@dataclass(frozen=True)
class LoopInput:
    kind: LoopInputKind
    text: str = ""


_COMMANDS = {
    "/help": LoopInputKind.HELP,
    "/status": LoopInputKind.STATUS,
    "/memory": LoopInputKind.MEMORY,
    "/handoff": LoopInputKind.HANDOFF,
    "/fork": LoopInputKind.FORK,
    "/skills": LoopInputKind.SKILLS,
    "/done": LoopInputKind.DONE,
    "/exit": LoopInputKind.EXIT,
}


def parse_loop_input(raw: str) -> LoopInput:
    normalized = raw.strip()
    if not normalized:
        return LoopInput(LoopInputKind.EMPTY)
    if normalized.startswith("//"):
        return LoopInput(LoopInputKind.MESSAGE, normalized[1:])
    if not normalized.startswith("/"):
        return LoopInput(LoopInputKind.MESSAGE, normalized)
    if normalized == "/skill":
        return LoopInput(LoopInputKind.SKILL)
    if normalized.startswith("/skill "):
        return LoopInput(LoopInputKind.SKILL, normalized[len("/skill ") :].strip())
    kind = _COMMANDS.get(normalized)
    if kind is None:
        return LoopInput(LoopInputKind.UNKNOWN, normalized)
    return LoopInput(kind)
