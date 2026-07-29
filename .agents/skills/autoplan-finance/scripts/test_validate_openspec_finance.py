"""Unit tests for validate_openspec_finance.py.

Run with:
  python scripts/test_validate_openspec_finance.py
"""
from __future__ import annotations

import contextlib
import io
import json
import shutil
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
import validate_openspec_finance as v  # type: ignore  # noqa: E402


GOOD_FILES = {
    "requirements.md": "# Requirements\n\n## Decision Flow\n\n| A | B |\n",
    "design.md": "# Design\n",
    "tasks.md": "# Tasks\n",
    "metrics.md": (
        "# Metrics\n\n"
        "## Metric Registry\n\n"
        "## Source Mapping\n\n"
        "## Financial Correctness\n\n"
        "## Consistency Review\n"
    ),
    "permissions.md": (
        "# Permissions\n\n"
        "## Roles\n\n"
        "## Permission Matrix\n\n"
        "## Field-level Restrictions\n\n"
        "## Data Scope Rules\n"
    ),
    "audit.md": (
        "# Audit\n\n"
        "## Audited Actions\n\n"
        "## Retention Policy\n\n"
        "## Audit Query Requirements\n"
    ),
    "release.md": (
        "# Release\n\n"
        "## Decision\n\n"
        "## Release Checklist\n\n"
        "## Rollback Plan\n\n"
        "## Open Release Risks\n"
    ),
}


class ValidatorTests(unittest.TestCase):
    def setUp(self) -> None:
        self.dir = Path(tempfile.mkdtemp(prefix="finval-"))

    def tearDown(self) -> None:
        shutil.rmtree(self.dir, ignore_errors=True)

    def write(self, files):  # type: ignore[no-untyped-def]
        for name, body in files.items():
            (self.dir / name).write_text(body, encoding="utf-8")

    def test_all_good_passes(self) -> None:
        self.write(GOOD_FILES)
        errors = v.validate(self.dir, strict=False)
        self.assertEqual(errors, [])

    def test_missing_file_reported(self) -> None:
        files = dict(GOOD_FILES)
        del files["audit.md"]
        self.write(files)
        errors = v.validate(self.dir, strict=False)
        self.assertTrue(any("missing required file: audit.md" in e for e in errors))

    def test_missing_heading_reported(self) -> None:
        files = dict(GOOD_FILES)
        files["metrics.md"] = "# Metrics\n\n## Metric Registry\n"
        self.write(files)
        errors = v.validate(self.dir, strict=False)
        self.assertTrue(any("source mapping" in e for e in errors))
        self.assertTrue(any("financial correctness" in e for e in errors))
        self.assertTrue(any("consistency review" in e for e in errors))

    def test_substring_only_body_does_not_pass(self) -> None:
        # The old validator would pass on the word appearing in body text.
        # The new validator must require an actual heading, not any occurrence.
        files = dict(GOOD_FILES)
        files["release.md"] = "# Release\n\nWe made a decision here.\n"
        self.write(files)
        errors = v.validate(self.dir, strict=False)
        self.assertTrue(any("(h2) 'decision'" in e for e in errors))

    def test_strict_order_enforced(self) -> None:
        files = dict(GOOD_FILES)
        files["metrics.md"] = (
            "# Metrics\n\n"
            "## Source Mapping\n\n"
            "## Metric Registry\n\n"
            "## Financial Correctness\n\n"
            "## Consistency Review\n"
        )
        self.write(files)
        strict_errors = v.validate(self.dir, strict=True)
        loose_errors = v.validate(self.dir, strict=False)
        self.assertEqual(loose_errors, [])
        self.assertTrue(any("out of order" in e for e in strict_errors))

    def test_code_block_headings_ignored(self) -> None:
        files = dict(GOOD_FILES)
        files["metrics.md"] = (
            "# Metrics\n\n"
            "```markdown\n"
            "## Metric Registry\n"
            "## Source Mapping\n"
            "## Financial Correctness\n"
            "## Consistency Review\n"
            "```\n"
        )
        self.write(files)
        errors = v.validate(self.dir, strict=False)
        self.assertTrue(any("metric registry" in e for e in errors))

    def test_qualifier_suffix_accepted(self) -> None:
        files = dict(GOOD_FILES)
        files["metrics.md"] = (
            "# Metrics\n\n"
            "## Metric Registry (Golden Definitions)\n\n"
            "## Source Mapping — Upstream Systems\n\n"
            "## Financial Correctness\n\n"
            "## Consistency Review\n"
        )
        self.write(files)
        errors = v.validate(self.dir, strict=False)
        self.assertEqual(errors, [])

    def test_utf8_bom_is_tolerated(self) -> None:
        # Windows-authored Markdown often starts with a UTF-8 BOM (e.g., via
        # PowerShell Set-Content -Encoding utf8). The validator must still
        # recognize the h1 on the first line.
        self.write(GOOD_FILES)
        bom_path = self.dir / "requirements.md"
        bom_path.write_bytes(b"\xef\xbb\xbf" + bom_path.read_bytes())
        errors = v.validate(self.dir, strict=False)
        self.assertEqual(errors, [])

    def test_json_output_via_main(self) -> None:
        self.write(GOOD_FILES)
        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            rc = v.main([str(self.dir), "--json"])
        self.assertEqual(rc, 0)
        payload = json.loads(buf.getvalue().strip())
        self.assertTrue(payload["ok"])
        self.assertEqual(payload["errors"], [])

    def test_missing_specs_dir_returns_2(self) -> None:
        missing = self.dir / "does-not-exist"
        buf = io.StringIO()
        err = io.StringIO()
        with contextlib.redirect_stdout(buf), contextlib.redirect_stderr(err):
            rc = v.main([str(missing)])
        self.assertEqual(rc, 2)


if __name__ == "__main__":
    unittest.main(verbosity=2)