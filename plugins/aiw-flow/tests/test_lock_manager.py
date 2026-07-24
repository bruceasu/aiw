import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from codex_flow.lock_manager import FileLock, LockAcquisitionError


class LockManagerTests(unittest.TestCase):
    def test_file_lock_conflict(self):
        with TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "lock.json"
            first = FileLock(path, timeout=0.2)
            second = FileLock(path, timeout=0.2, poll_interval=0.05)
            with first:
                with self.assertRaises(LockAcquisitionError):
                    second.acquire()
