import importlib.util
import io
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock


MODULE_PATH = Path(__file__).with_name("aiw-github.py")
SPEC = importlib.util.spec_from_file_location("aiw_github", MODULE_PATH)
aiw_github = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(aiw_github)


class GithubPluginTests(unittest.TestCase):
    def test_parser_uses_actual_commands_and_global_json_option(self):
        parser = aiw_github.build_parser()

        args = parser.parse_args(["--json", "list-issue", "owner/repo"])
        self.assertTrue(args.json)
        self.assertEqual(args.cmd, "list-issue")
        self.assertEqual(args.repo, "owner/repo")

        update = parser.parse_args(["update-issue", "owner/repo", "12", "--title", "New"])
        self.assertEqual(update.cmd, "update-issue")
        self.assertEqual(update.number, 12)
        self.assertEqual(update.title, "New")

    def test_read_body_from_file_and_stdin(self):
        with tempfile.NamedTemporaryFile("w", encoding="utf-8", delete=False) as handle:
            handle.write("# From file\n")
            path = handle.name
        try:
            args = type("Args", (), {"body_file": path, "body": None})()
            self.assertEqual(aiw_github.read_body(args), "# From file\n")
        finally:
            Path(path).unlink()

        args = type("Args", (), {"body_file": "-", "body": None})()
        with mock.patch.object(sys, "stdin", io.StringIO("# From stdin\n")):
            self.assertEqual(aiw_github.read_body(args), "# From stdin\n")

    def test_create_issue_uses_body_file_and_returns_identity_fields(self):
        parser = aiw_github.build_parser()
        args = parser.parse_args(["create-issue", "owner/repo", "--title", "Title", "--body", "Body"])
        captured = {}

        def fake_request(method, path, token, params=None, json_body=None):
            captured.update(method=method, path=path, token=token, json_body=json_body)
            return {"number": 12, "html_url": "https://github.com/owner/repo/issues/12", "state": "open"}

        with mock.patch.object(aiw_github, "request", side_effect=fake_request), \
                mock.patch.object(aiw_github, "emit_issue_panel") as emit:
            aiw_github.create_issue(args, "token")

        self.assertEqual(captured["method"], "POST")
        self.assertEqual(captured["path"], "/repos/owner/repo/issues")
        self.assertEqual(captured["json_body"], {"title": "Title", "body": "Body"})
        emitted = emit.call_args.args[0]
        self.assertEqual(emitted["repository"], "owner/repo")
        self.assertEqual(emitted["number"], 12)
        self.assertEqual(emitted["url"], "https://github.com/owner/repo/issues/12")
        self.assertEqual(emitted["state"], "open")

    def test_update_issue_patches_title_and_body(self):
        parser = aiw_github.build_parser()
        args = parser.parse_args(["update-issue", "owner/repo", "12", "--title", "New", "--body", "Body"])
        captured = {}

        def fake_request(method, path, token, params=None, json_body=None):
            captured.update(method=method, path=path, json_body=json_body)
            return {"number": 12, "html_url": "url", "state": "open"}

        with mock.patch.object(aiw_github, "request", side_effect=fake_request), \
                mock.patch.object(aiw_github, "emit_issue_panel"):
            aiw_github.update_issue(args, "token")

        self.assertEqual(captured, {
            "method": "PATCH",
            "path": "/repos/owner/repo/issues/12",
            "json_body": {"title": "New", "body": "Body"},
        })

    def test_update_issue_rejects_empty_payload(self):
        parser = aiw_github.build_parser()
        args = parser.parse_args(["update-issue", "owner/repo", "12"])
        with self.assertRaises(SystemExit):
            aiw_github.update_issue(args, "token")

    def test_discover_repo_supports_origin_remote(self):
        outputs = iter(["C:/repo\n", "git@github.com:owner/repo.git\n"])
        with mock.patch.object(aiw_github, "run_git", side_effect=lambda args: type("Result", (), {"stdout": next(outputs)})()):
            self.assertEqual(aiw_github.discover_repo(), "owner/repo")


if __name__ == "__main__":
    unittest.main()
