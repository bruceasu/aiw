"""Unit tests for validate_metrics_review.py."""
from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import validate_metrics_review as v  # noqa: E402


def _full_valid_markdown() -> str:
    return "\n".join(
        [
            "# Metrics Spec",
            "",
            "## Status",
            "READY",
            "",
            "## 1. Context",
            "some context",
            "",
            "## 2. Metric Registry",
            "",
            "## 3. Source Mapping",
            "",
            "## 4. Financial Correctness",
            "",
            "## 5. Consistency Review",
            "",
            "## 6. Ownership",
            "",
            "## 7. Known Limitations",
            "",
            "## 8. Open Issues",
            "",
        ]
    )


class ParseHeadingsTests(unittest.TestCase):
    def test_extracts_atx_headings_and_ignores_code_blocks(self):
        md = "\n".join(
            [
                "# Metrics Spec",
                "text",
                "## Status",
                "```",
                "## Not A Heading",
                "```",
                "## 1. Context",
            ]
        )
        headings = v.parse_headings(md)
        titles = [(h.level, h.title) for h in headings]
        self.assertEqual(
            titles,
            [(1, "metrics spec"), (2, "status"), (2, "1. context")],
        )


class ValidateTextTests(unittest.TestCase):
    def test_full_valid_document_passes(self):
        errors = v.validate_text(_full_valid_markdown(), v.REQUIRED, strict=False)
        self.assertEqual(errors, [])

    def test_full_valid_document_passes_strict(self):
        errors = v.validate_text(_full_valid_markdown(), v.REQUIRED, strict=True)
        self.assertEqual(errors, [])

    def test_missing_heading_is_reported(self):
        md = _full_valid_markdown().replace("## 4. Financial Correctness\n", "")
        errors = v.validate_text(md, v.REQUIRED, strict=False)
        self.assertTrue(
            any("financial correctness" in e for e in errors),
            msg=f"expected financial correctness error, got: {errors}",
        )

    def test_body_text_alone_does_not_satisfy_heading_requirement(self):
        md = _full_valid_markdown().replace(
            "## 5. Consistency Review\n",
            "This paragraph mentions consistency review in prose only.\n",
        )
        errors = v.validate_text(md, v.REQUIRED, strict=False)
        self.assertTrue(any("consistency review" in e for e in errors))

    def test_strict_flags_out_of_order_sections(self):
        md = _full_valid_markdown().replace(
            "## 2. Metric Registry\n\n## 3. Source Mapping\n",
            "## 3. Source Mapping\n\n## 2. Metric Registry\n",
        )
        errors = v.validate_text(md, v.REQUIRED, strict=True)
        self.assertTrue(
            any("out of order" in e for e in errors),
            msg=f"expected out-of-order error, got: {errors}",
        )

    def test_non_strict_accepts_out_of_order(self):
        md = _full_valid_markdown().replace(
            "## 2. Metric Registry\n\n## 3. Source Mapping\n",
            "## 3. Source Mapping\n\n## 2. Metric Registry\n",
        )
        errors = v.validate_text(md, v.REQUIRED, strict=False)
        self.assertEqual(errors, [])

    def test_fenced_code_block_headings_are_ignored(self):
        md = _full_valid_markdown().replace(
            "## 2. Metric Registry\n",
            "```\n## 2. Metric Registry\n```\n",
        )
        errors = v.validate_text(md, v.REQUIRED, strict=False)
        self.assertTrue(any("metric registry" in e for e in errors))


class FileAndCliTests(unittest.TestCase):
    def _write(self, name: str, content: str) -> Path:
        # Write with a BOM to prove utf-8-sig tolerance.
        p = Path(self.tmp) / name
        p.write_bytes(b"\xef\xbb\xbf" + content.encode("utf-8"))
        return p

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = self._tmp.name

    def tearDown(self):
        self._tmp.cleanup()

    def test_validate_file_with_bom(self):
        p = self._write("METRICS_SPEC.md", _full_valid_markdown())
        errors = v.validate_file(p, strict=True)
        self.assertEqual(errors, [])

    def test_validate_file_missing_returns_error(self):
        p = Path(self.tmp) / "nope.md"
        errors = v.validate_file(p)
        self.assertTrue(any("file not found" in e for e in errors))

    def test_cli_json_reports_ok(self):
        p = self._write("METRICS_SPEC.md", _full_valid_markdown())
        result = subprocess.run(
            [sys.executable, str(HERE / "validate_metrics_review.py"), str(p), "--json"],
            capture_output=True,
            text=True,
            check=False,
        )
        self.assertEqual(result.returncode, 0, msg=result.stdout + result.stderr)
        payload = json.loads(result.stdout.strip())
        self.assertTrue(payload["ok"])
        self.assertEqual(payload["errors"], [])


if __name__ == "__main__":
    unittest.main(verbosity=2)