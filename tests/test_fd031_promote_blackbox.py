"""FD-031 promote behavior through public commands in disposable projects.

Run only with a revision-bound FD Tester authorization. All binaries, Go caches,
Git repositories, and generated FD records live under one system temp directory.
"""

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SOURCE = Path(__file__).resolve().parents[1]
ISSUE_ID = "REQ00001-promotion-fixture"
TITLE = '批准的 Issue "alpha beta"'
MODULES = (
    "github.com/openai/openai-go@v1.12.0",
    "github.com/manifoldco/promptui@v0.9.0",
    "github.com/chzyer/readline@v0.0.0-20180603132655-2972be24d48e",
    "github.com/tidwall/gjson@v1.14.4",
    "github.com/tidwall/match@v1.1.1",
    "github.com/tidwall/pretty@v1.2.1",
    "github.com/tidwall/sjson@v1.2.5",
    "golang.org/x/sys@v0.29.0",
)


class PromoteBlackBox(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory(prefix="fd031-promote-")
        cls.addClassCleanup(cls.temp.cleanup)
        cls.runtime = Path(cls.temp.name)
        cls.aiw_root = cls.runtime / "aiw"
        cls.aiw_root.mkdir()
        cls.aiw = cls.aiw_root / ("aiw.exe" if os.name == "nt" else "aiw")
        plugins = cls.aiw_root / "plugins"
        plugins.mkdir()
        shutil.copy2(SOURCE / "plugins" / "aiw-fd.py", plugins / "aiw-fd.py")
        shutil.copytree(SOURCE / "plugins" / "aiw-req", plugins / "aiw-req")
        cls.aiw_req = plugins / "aiw-req" / ("aiw-req.exe" if os.name == "nt" else "aiw-req")

        existing_cache = subprocess.run(
            ["go", "env", "GOMODCACHE"], cwd=SOURCE,
            capture_output=True, text=True, timeout=15, check=True,
        ).stdout.strip()
        for module in MODULES:
            source = Path(existing_cache, module)
            if not source.is_dir():
                raise RuntimeError(f"Offline module unavailable: {module}")
            shutil.copytree(source, cls.runtime / "mod-cache" / module)
            name, version = module.split("@", 1)
            downloads = Path(existing_cache, "cache", "download", name, "@v")
            copied_downloads = cls.runtime / "mod-cache" / "cache" / "download" / name / "@v"
            copied_downloads.mkdir(parents=True, exist_ok=True)
            for cached_file in downloads.glob(version + ".*"):
                shutil.copy2(cached_file, copied_downloads / cached_file.name)

        env = os.environ.copy()
        env.update({
            "GOCACHE": str(cls.runtime / "go-cache"),
            "GOMODCACHE": str(cls.runtime / "mod-cache"),
            "GOPATH": str(cls.runtime / "go-path"),
            "GOENV": "off",
            "GOWORK": "off",
            "GOPROXY": "off",
            "GOSUMDB": "off",
            "GOTOOLCHAIN": "local",
        })
        for package, target in (("./cmd/aiw", cls.aiw), ("./cmd/aiw-req", cls.aiw_req)):
            result = subprocess.run(
                ["go", "build", "-mod=readonly", "-o", str(target), package],
                cwd=SOURCE, env=env, capture_output=True, text=True, timeout=180,
            )
            if result.returncode:
                raise RuntimeError(f"Offline CLI build failed for {package}: {result.stdout}{result.stderr}")

    def setUp(self):
        self.root = self.runtime / self.id().rsplit(".", 1)[-1]
        self.root.mkdir()
        (self.root / "docs" / "features").mkdir(parents=True)
        (self.root / "docs" / "requirements").mkdir(parents=True)
        (self.root / "docs" / "templates").mkdir(parents=True)
        shutil.copy2(SOURCE / "docs" / "templates" / "TEMPLATE.md", self.root / "docs" / "templates" / "TEMPLATE.md")
        self.env = os.environ.copy()
        self.env.pop("AIW_FD_ROLE_RUNNER", None)
        self.env.update({
            "AIW_ROOT": str(self.aiw_root),
            "PATH": str(self.aiw_root) + os.pathsep + self.env.get("PATH", ""),
            "PYTHONDONTWRITEBYTECODE": "1",
            "PYTHONIOENCODING": "utf-8",
            "GIT_CONFIG_NOSYSTEM": "1",
            "GIT_CONFIG_GLOBAL": os.devnull,
            "GIT_TERMINAL_PROMPT": "0",
        })
        result = subprocess.run(
            ["git", "init", "--quiet"], cwd=self.root, env=self.env,
            capture_output=True, text=True, timeout=15,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def issue(self, *, status="APPROVED", approval="APPROVED", promotion="NOT_STARTED"):
        folder = self.root / "docs" / "requirements" / ISSUE_ID
        folder.mkdir(parents=True)
        record = folder / "requirement.toml"
        record.write_text(
            f'id = "{ISSUE_ID}"\n'
            f'title = {json.dumps(TITLE, ensure_ascii=False)}\n'
            'parent_id = ""\n'
            f'status = "{status}"\n'
            'created = "2026-10-08"\nupdated = "2026-10-08"\nrevision = 3\n\n'
            f'[approval]\nstatus = "{approval}"\nby = "tester"\n'
            'at = "2026-10-08T00:00:00+00:00"\nreason = "fixture"\n'
            'source_digest = "fixture"\n\n'
            f'[promotion]\nstatus = "{promotion}"\ntask_id = "legacy-task"\n',
            encoding="utf-8",
        )
        return record

    def cli(self, alias="issue"):
        return subprocess.run(
            [str(self.aiw), alias, "promote", ISSUE_ID],
            cwd=self.root, env=self.env, capture_output=True, text=True, timeout=30,
        )

    def generated(self):
        return list((self.root / "docs" / "features").glob("FD-*_*.md"))

    def assert_no_task(self):
        self.assertFalse((self.root / "docs" / "tasks").exists())
        self.assertFalse((self.root / ".ai" / "task").exists())

    def test_issue_alias_creates_fd_and_planner_handoff(self):
        record = self.issue()
        before = record.read_bytes()
        result = self.cli("issue")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        (fd_path,) = self.generated()
        self.assertIn(TITLE, fd_path.read_text(encoding="utf-8"))
        self.assertIn(ISSUE_ID, fd_path.read_text(encoding="utf-8"))
        fd_id = fd_path.name.split("_", 1)[0]
        receipts = list((self.root / ".ai" / "fd" / fd_id / "events").glob("*.json"))
        self.assertEqual(len(receipts), 1)
        handoff = json.loads(receipts[0].read_text(encoding="utf-8"))
        self.assertEqual((handoff["event_type"], handoff["target_role"]), ("design-requested", "planner"))
        self.assertEqual(record.read_bytes(), before)
        self.assert_no_task()

    def test_req_alias_creates_fd(self):
        record = self.issue(promotion="SPEC_DRAFTED")
        before = record.read_bytes()
        result = self.cli("req")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(len(self.generated()), 1)
        self.assertEqual(record.read_bytes(), before)
        self.assert_no_task()

    def test_unapproved_issue_is_rejected(self):
        record = self.issue(status="DECIDED", approval="PENDING")
        before = record.read_bytes()
        result = self.cli()
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(self.generated(), [])
        self.assertEqual(record.read_bytes(), before)
        self.assert_no_task()

    def test_issue_and_approval_must_both_be_approved(self):
        self.issue(status="APPROVED", approval="PENDING")
        result = self.cli()
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(self.generated(), [])
        self.assert_no_task()

    def test_missing_issue_is_rejected(self):
        result = self.cli()
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(self.generated(), [])
        self.assert_no_task()

    def test_duplicate_link_is_rejected(self):
        record = self.issue()
        first = self.cli()
        self.assertEqual(first.returncode, 0, first.stdout + first.stderr)
        before = record.read_bytes()
        again = self.cli()
        self.assertNotEqual(again.returncode, 0, again.stdout + again.stderr)
        self.assertEqual(len(self.generated()), 1)
        self.assertEqual(record.read_bytes(), before)
        self.assert_no_task()

    def test_fd_creation_failure_is_reported(self):
        record = self.issue()
        before = record.read_bytes()
        template = self.root / "docs" / "templates" / "TEMPLATE.md"
        template.unlink()
        template.mkdir()
        result = self.cli()
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("TEMPLATE.md", result.stdout + result.stderr)
        self.assertEqual(self.generated(), [])
        self.assertFalse((self.root / ".ai" / "fd" / "FD-001").exists())
        self.assertEqual(record.read_bytes(), before)
        self.assert_no_task()

    def test_both_help_entries_name_promote(self):
        for alias in ("issue", "req"):
            with self.subTest(alias=alias):
                result = subprocess.run(
                    [str(self.aiw), alias, "--help"], cwd=self.root,
                    env=self.env, capture_output=True, text=True, timeout=15,
                )
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertIn("promote <id>", result.stdout)


if __name__ == "__main__":
    unittest.main()
