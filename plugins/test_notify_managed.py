"""Offline contract tests: no real transport or sender subprocess is permitted."""

import contextlib
import errno
import importlib.util
import io
import json
import os
from pathlib import Path
import socket
import ssl
import unittest
from unittest.mock import Mock, patch
import urllib.error


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(filename))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


teams = load("managed_teams_fixture", "send_teams_msg.py")
notify = load("managed_notify_fixture", "aiw-notify.py")
SECRET = "fixture-only-not-a-real-credential"


class ManagedNotificationTests(unittest.TestCase):
    def setUp(self):
        self.stack = contextlib.ExitStack()
        self.addCleanup(self.stack.close)
        for target in ("socket.socket", "socket.create_connection", "urllib.request.urlopen", "subprocess.run"):
            self.stack.enter_context(patch(target, side_effect=AssertionError("real transport is forbidden")))
        self.stack.enter_context(patch.dict(os.environ, {"FIXTURE_NOTIFY_SECRET": SECRET, "UNRELATED_SECRET": "do-not-forward"}, clear=True))

    def request(self):
        return {"schema_version": 1, "notification_id": "n-1", "attempt_id": "a-1", "text": "Fixture message.",
                "endpoint": "https://example.invalid/message/sendMessage/fixture/channel", "credential_env": "FIXTURE_NOTIFY_SECRET",
                "ca_file": "", "sender": "fixture", "timeout_seconds": 3}

    def invoke(self, error=None):
        request = self.request()
        tls = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
        opener = Mock()
        response = Mock(status=200)
        response.read.side_effect = AssertionError("response body must not be read")
        opener.open.return_value.__enter__ = Mock(return_value=response)
        opener.open.return_value.__exit__ = Mock(return_value=False)
        if error is not None:
            opener.open.side_effect = error
        stdout, stderr = io.StringIO(), io.StringIO()
        stdin = io.TextIOWrapper(io.BytesIO(json.dumps(request).encode()), encoding="utf-8")
        with patch.object(teams.sys, "stdin", stdin), contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr), \
                patch.object(teams.ssl, "create_default_context", return_value=tls) as make_tls, \
                patch.object(teams.urllib.request, "build_opener", return_value=opener) as build:
            self.assertEqual(teams.managed_main(), 0)
        raw = stdout.getvalue()
        self.assertNotIn(SECRET, raw + stderr.getvalue())
        self.assertEqual(stderr.getvalue(), "")
        self.assertEqual(opener.open.call_count, 1, "adapter must not retry internally")
        result = json.loads(raw)
        self.assertEqual(result["notification_id"], "n-1")
        self.assertEqual(result["attempt_id"], "a-1")
        self.assertTrue(result["attempted"])
        return result, opener, tls, make_tls, build, response

    def test_typed_errors_and_http_classification(self):
        cases = [
            (urllib.error.URLError(ssl.SSLCertVerificationError(1, SECRET)), "configuration_failure", "tls"),
            (PermissionError(SECRET), "permission_failure", "permission"),
            (urllib.error.URLError(ConnectionRefusedError(SECRET)), "network_failure", "network"),
            (socket.gaierror(-2, SECRET), "network_failure", "network"),
            (ConnectionResetError(SECRET), "network_failure", "network"),
            (OSError(errno.ENETUNREACH, SECRET), "network_failure", "network"),
            (TimeoutError(SECRET), "unknown", "timeout"),
            (RuntimeError("network " + SECRET), "unknown", "protocol"),
        ]
        for error, outcome, code in cases:
            with self.subTest(error=type(error).__name__, outcome=outcome):
                result, *_ = self.invoke(error)
                self.assertEqual((result["outcome"], result["error_code"]), (outcome, code))
        for status in (302, 401, 403, 429, 503):
            with self.subTest(http_status=status):
                error = urllib.error.HTTPError("https://example.invalid", status, SECRET, {}, io.BytesIO(SECRET.encode()))
                result, *_ = self.invoke(error)
                self.assertEqual(result["outcome"], "permission_failure" if status in (401, 403) else "service_failure")
                self.assertEqual(result["http_status"], status)

    def test_tls_redirect_and_attempt_only_response(self):
        result, opener, tls, make_tls, build, response = self.invoke()
        self.assertEqual(result["outcome"], "attempted")
        self.assertEqual(result["http_status"], 200)
        self.assertEqual(tls.verify_mode, ssl.CERT_REQUIRED)
        self.assertTrue(tls.check_hostname)
        make_tls.assert_called_once_with(cafile=None)
        handlers = build.call_args.args
        self.assertEqual(handlers[0].proxies, {})
        self.assertIsNone(handlers[1].redirect_request(None, None, 302, "redirect", {}, "https://other.invalid"))
        response.read.assert_not_called()
        outgoing = opener.open.call_args.args[0]
        self.assertEqual(outgoing.full_url, self.request()["endpoint"])
        self.assertEqual(json.loads(outgoing.data)["ePassword"], SECRET)
        self.assertEqual(opener.open.call_args.kwargs["timeout"], 3)

    def plugin_request(self):
        config = {"teams_url": self.request()["endpoint"], "credential_env": "FIXTURE_NOTIFY_SECRET", "ca_file": "", "sender": "fixture", "timeout_seconds": 3}
        reference = {"enabled": True, "ready": True, "channel": "teams", "target_reference": "fixed-target", "config_reference": "fixed-config"}
        request = {"schema_version": 2, "notification_id": "n-1", "attempt_id": "a-1", "text": "Fixture message.",
                   "content_digest": notify.digest(b"Fixture message."), "channel": "teams", "target_reference": "fixed-target", "config_reference": "fixed-config"}
        return config, reference, request

    def test_frozen_target_and_protocol_reject_before_dispatch(self):
        config, reference, request = self.plugin_request()
        for key in ("target_reference", "config_reference", "content_digest"):
            with self.subTest(changed=key), patch.object(notify, "project_config", return_value=(config, reference)), \
                    patch.object(notify.subprocess, "run") as run:
                changed = {**request, key: "changed"}
                result = notify.managed_dispatch(changed)
                self.assertEqual(result["outcome"], "configuration_failure")
                self.assertFalse(result["attempted"])
                run.assert_not_called()
        for raw in (b'{"schema_version":2,"schema_version":2}', b'{} {}', b'\xff'):
            with self.subTest(raw=repr(raw)), self.assertRaises((ValueError, UnicodeError)):
                notify.decode(raw)

    def test_child_protocol_and_environment_only_credential(self):
        config, reference, request = self.plugin_request()
        child = {"schema_version": 1, "notification_id": "n-1", "attempt_id": "a-1", "attempted": True,
                 "outcome": "attempted", "error_code": "", "http_status": 200}
        for mismatch in (False, True):
            with self.subTest(mismatched_attempt=mismatch), patch.object(notify, "project_config", return_value=(config, reference)), \
                    patch.object(notify.subprocess, "run", return_value=Mock(returncode=0, stdout=json.dumps({**child, "attempt_id": "other" if mismatch else "a-1"}).encode())) as run:
                result = notify.managed_dispatch(request)
                self.assertEqual(result["outcome"], "unknown" if mismatch else "attempted")
                self.assertEqual(result["content_digest"], request["content_digest"])
                run.assert_called_once()
                argv = run.call_args.args[0]
                kwargs = run.call_args.kwargs
                self.assertEqual(argv[-1], "--managed-json")
                self.assertIn("-I", argv)
                self.assertNotIn(SECRET, str(argv) + kwargs["input"].decode() + json.dumps(result))
                self.assertEqual(kwargs["env"]["FIXTURE_NOTIFY_SECRET"], SECRET)
                self.assertNotIn("UNRELATED_SECRET", kwargs["env"])
                self.assertEqual(kwargs["timeout"], 35)


if __name__ == "__main__":
    unittest.main()
