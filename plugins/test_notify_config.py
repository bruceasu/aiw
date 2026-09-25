"""Real TOML configuration tests; requires Python 3.11+, with no transport."""

import contextlib
import importlib.util
import io
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location("notify_config_fixture", Path(__file__).with_name("aiw-notify.py"))
notify = importlib.util.module_from_spec(spec)
spec.loader.exec_module(notify)


class NotificationConfigTests(unittest.TestCase):
    def setUp(self):
        self.stack = contextlib.ExitStack()
        self.addCleanup(self.stack.close)
        self.root = Path(self.stack.enter_context(tempfile.TemporaryDirectory()))
        previous = Path.cwd()
        os.chdir(self.root)
        self.stack.callback(os.chdir, previous)
        self.stack.enter_context(patch.dict(os.environ, {}, clear=True))
        for target in ("socket.socket", "socket.create_connection", "urllib.request.urlopen", "subprocess.run"):
            self.stack.enter_context(patch(target, side_effect=AssertionError("transport forbidden in config tests")))

    def write(self, text):
        (self.root / "aiw.toml").write_text(text, encoding="utf-8")

    def teams_config(self, target="fixture", extra=""):
        return ('[notifications]\nschema_version=1\nenabled=true\nchannel="teams"\n'
                'target_id="' + target + '"\nteams_url="https://example.invalid/message/sendMessage/p/c"\n'
                'credential_env="FIXTURE_SECRET"\n' + extra)

    def test_missing_and_disabled_configuration_never_ready(self):
        for content in (None, '[other]\nvalue="unrelated"\n', '[notifications]\nschema_version=1\nenabled=false\n'):
            with self.subTest(content=content):
                if content is not None:
                    self.write(content)
                _, reference = notify.project_config()
                self.assertFalse(reference["enabled"])
                self.assertFalse(reference["ready"])
                self.assertEqual(reference["target_reference"], "")

    def test_real_config_identity_credentials_and_ca(self):
        self.write(self.teams_config())
        (self.root / ".env").write_text("FIXTURE_SECRET=must-not-load\n", encoding="utf-8")
        config, missing = notify.project_config()
        self.assertFalse(missing["ready"])
        self.assertNotIn("FIXTURE_SECRET", os.environ)
        with patch.dict(os.environ, {"FIXTURE_SECRET": "first-fixture-value"}):
            _, first = notify.project_config()
        with patch.dict(os.environ, {"FIXTURE_SECRET": "rotated-fixture-value"}):
            _, rotated = notify.project_config()
        self.assertTrue(first["ready"])
        self.assertEqual(first, rotated)
        self.assertEqual(first["config_reference"], missing["config_reference"])
        self.assertNotIn("first-fixture-value", str(config) + str(first))
        self.write(self.teams_config(target="another-target"))
        _, changed = notify.project_config()
        self.assertNotEqual(first["target_reference"], changed["target_reference"])
        self.assertNotEqual(first["config_reference"], changed["config_reference"])
        # Only CA identity is inspected here, not certificate validity or TLS.
        self.write(self.teams_config(extra='ca_file="fixture-ca.pem"\n'))
        ca = self.root / "fixture-ca.pem"
        ca.write_text("first fixture CA bytes", encoding="utf-8")
        normalized, old_ca = notify.project_config()
        self.assertEqual(normalized["ca_file"], str(ca.resolve()))
        ca.write_text("changed fixture CA bytes", encoding="utf-8")
        _, new_ca = notify.project_config()
        self.assertNotEqual(old_ca["target_reference"], new_ca["target_reference"])
        self.assertNotEqual(old_ca["config_reference"], new_ca["config_reference"])

    def test_invalid_real_toml_configuration_is_rejected(self):
        base = self.teams_config()
        cases = {
            "unknown-field": base + 'surprise=true\n',
            "boolean-timeout": base + 'timeout_seconds=true\n',
            "excess-timeout": base + 'timeout_seconds=31\n',
            "http": base.replace('https://', 'http://'),
            "userinfo": base.replace('https://example.invalid', 'https://user@example.invalid'),
            "query": base.replace('/p/c"', '/p/c?token=fixture"'),
            "wrong-version": base.replace('schema_version=1', 'schema_version=2'),
            "duplicate-key": base + 'enabled=true\n',
            "invalid-env-name": base.replace('FIXTURE_SECRET', 'BAD-NAME'),
        }
        for name, text in cases.items():
            with self.subTest(case=name):
                self.write(text)
                with self.assertRaises(ValueError):
                    notify.project_config()
        self.write(base + 'ca_file="missing-ca.pem"\n')
        with self.assertRaises(FileNotFoundError):
            notify.project_config()

    def test_console_dispatch_rechecks_actual_configuration(self):
        self.write('[notifications]\nschema_version=1\nenabled=true\nchannel="console"\ntarget_id="local"\n')
        _, reference = notify.project_config()
        request = {"schema_version": 2, "notification_id": "fixture-notice", "attempt_id": "fixture-attempt", "text": "Local fixture only.",
                   "content_digest": notify.digest(b"Local fixture only."), "channel": "console", "config_reference": reference["config_reference"], "target_reference": reference["target_reference"]}
        display = io.StringIO()
        with contextlib.redirect_stderr(display):
            result = notify.managed_dispatch(request)
        self.assertEqual(result["outcome"], "attempted")
        self.assertEqual(display.getvalue(), "Local fixture only.\n")
        self.write('[notifications]\nschema_version=1\nenabled=true\nchannel="console"\ntarget_id="changed"\n')
        display = io.StringIO()
        with contextlib.redirect_stderr(display):
            result = notify.managed_dispatch(request)
        self.assertEqual(result["outcome"], "configuration_failure")
        self.assertFalse(result["attempted"])
        self.assertEqual(display.getvalue(), "")


if __name__ == "__main__":
    unittest.main()
