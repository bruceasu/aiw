from __future__ import annotations

import asyncio
import json
import os
import socket
import time
from contextlib import AbstractContextManager
from pathlib import Path
from typing import Dict, Optional

from codex_flow.models import LockInfo, isoformat, utc_now

if os.name == "nt":  # pragma: no cover
    import msvcrt
else:  # pragma: no cover
    import fcntl


class LockAcquisitionError(RuntimeError):
    """Unable to acquire lock."""


class FileLock(AbstractContextManager):
    def __init__(
        self,
        path: Path,
        *,
        session_id: Optional[str] = None,
        timeout: float = 10.0,
        poll_interval: float = 0.1,
    ):
        self.path = path
        self.session_id = session_id
        self.timeout = timeout
        self.poll_interval = poll_interval
        self._handle = None
        self._locked = False

    def acquire(self) -> "FileLock":
        self.path.parent.mkdir(parents=True, exist_ok=True)
        start = time.monotonic()
        self._handle = self.path.open("a+", encoding="utf-8")
        while True:
            try:
                self._lock_handle()
                self._locked = True
                self._write_metadata()
                return self
            except OSError:
                if time.monotonic() - start >= self.timeout:
                    self._safe_close_unlocked_handle()
                    raise LockAcquisitionError("Timed out acquiring lock: {}".format(self.path))
                time.sleep(self.poll_interval)

    def release(self) -> None:
        if self._handle is None:
            return
        try:
            if self._locked:
                self._unlock_handle()
        except OSError:
            pass
        finally:
            try:
                self._handle.close()
            except OSError:
                pass
            self._handle = None
            self._locked = False

    def info(self) -> Optional[LockInfo]:
        if not self.path.exists():
            return None
        try:
            data = json.loads(self.path.read_text(encoding="utf-8"))
            return LockInfo(**data)
        except Exception:
            return None

    def __enter__(self) -> "FileLock":
        return self.acquire()

    def __exit__(self, exc_type, exc, tb) -> None:
        self.release()

    def _lock_handle(self) -> None:
        assert self._handle is not None
        if os.name == "nt":  # pragma: no cover
            msvcrt.locking(self._handle.fileno(), msvcrt.LK_NBLCK, 1)
        else:  # pragma: no cover
            fcntl.flock(self._handle.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)

    def _unlock_handle(self) -> None:
        assert self._handle is not None
        if os.name == "nt":  # pragma: no cover
            msvcrt.locking(self._handle.fileno(), msvcrt.LK_UNLCK, 1)
        else:  # pragma: no cover
            fcntl.flock(self._handle.fileno(), fcntl.LOCK_UN)

    def _safe_close_unlocked_handle(self) -> None:
        if self._handle is not None:
            try:
                self._handle.close()
            except OSError:
                pass
            self._handle = None
            self._locked = False

    def _write_metadata(self) -> None:
        assert self._handle is not None
        payload = LockInfo(
            pid=os.getpid(),
            hostname=socket.gethostname(),
            acquired_at=isoformat(utc_now()),
            session_id=self.session_id,
        ).__dict__
        self._handle.seek(0)
        self._handle.truncate()
        self._handle.write(json.dumps(payload))
        self._handle.flush()
        os.fsync(self._handle.fileno())


class AsyncLockRegistry:
    def __init__(self):
        self._locks: Dict[str, asyncio.Lock] = {}

    def get(self, key: str) -> asyncio.Lock:
        lock = self._locks.get(key)
        if lock is None:
            lock = asyncio.Lock()
            self._locks[key] = lock
        return lock
