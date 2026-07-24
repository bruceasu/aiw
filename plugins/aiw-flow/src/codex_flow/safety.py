from __future__ import annotations

import re
from pathlib import Path


SESSION_ID_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{2,127}$")


class SafetyError(ValueError):
    """Raised when user input is unsafe."""


def validate_session_id(session_id: str) -> None:
    if not SESSION_ID_PATTERN.match(session_id):
        raise SafetyError("Invalid session id. Use 3-128 chars: letters, digits, dot, underscore, dash.")
    if ".." in session_id or "/" in session_id or "\\" in session_id:
        raise SafetyError("Session id must not contain path traversal.")


