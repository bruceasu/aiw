import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from codex_flow.workspace_manager import WorkspaceManager


class WorkspaceManagerTests(unittest.TestCase):
    def test_workspace_manager_requires_existing_directory(self):
        with TemporaryDirectory() as temp_dir:
            workspace = Path(temp_dir) / "workspace"
            workspace.mkdir()
            self.assertEqual(WorkspaceManager().ensure_existing_directory(workspace), workspace)
