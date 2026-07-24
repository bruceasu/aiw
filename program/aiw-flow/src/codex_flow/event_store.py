from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Dict, Iterable, List

from codex_flow.models import isoformat, utc_now
from codex_flow.file_utils import atomic_write_text


class EventStore:
    def __init__(self, path: Path):
        self.path = path
        self.path.parent.mkdir(parents=True, exist_ok=True)
        if not self.path.exists():
            atomic_write_text(self.path, "")

    def append(self, event_type: str, **payload: Any) -> None:
        data = {"time": isoformat(utc_now()), "type": event_type}
        data.update(payload)
        self._append_line(data)

    def append_raw(self, raw_line: str, source: str = "codex.raw") -> None:
        self._append_line(
            {
                "time": isoformat(utc_now()),
                "type": "codex.raw",
                "source": source,
                "raw": raw_line.rstrip("\n"),
            }
        )

    def tail(self, limit: int = 20) -> List[Dict[str, Any]]:
        if not self.path.exists():
            return []
        lines = self.path.read_text(encoding="utf-8").splitlines()
        result = []
        for line in lines[-limit:]:
            if not line.strip():
                continue
            result.append(json.loads(line))
        return result

    def _append_line(self, payload: Dict[str, Any]) -> None:
        with self.path.open("a", encoding="utf-8", newline="\n") as handle:
            handle.write(json.dumps(payload, ensure_ascii=False) + "\n")
            handle.flush()

