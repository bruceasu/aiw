from __future__ import annotations

from pathlib import Path
class WorkspaceError(RuntimeError):
    """Workspace validation failed."""


class WorkspaceManager:
    def ensure_existing_directory(self, workspace: Path) -> Path:
        resolved = workspace.resolve()
        if not resolved.exists() or not resolved.is_dir():
            raise WorkspaceError("Workspace does not exist: {}".format(resolved))
        return resolved
