"""Black-box cases for aiw-say; the Tester supplies a compiled binary path."""

# FD-036 scenario map: S01 argument; S02 stdin; S03 empty/conflicting input;
# S04 missing-default and explicit config; S05 missing/invalid config;
# S06 profile lookup/overlay; S07 unsafe profile names; S08 profile plus CLI
# precedence (default-to-install precedence is static-only); S09
# provider/model/timeout/target validation; S10 UTF-8 and
# prompt boundary; S11 credentials/auth/rate-limit/timeout/response failures;
# S12 retry success and exhaustion; S13 help/version and stream contracts;
# S14 unsupported options; S15 headless process and shipped profile examples
# (manual copy/no-overwrite behavior is documented, not an executable feature);
# S17 invalid config values fall back by precedence; S18 unknown TOML values are ignored.

import json
import os
import subprocess
import tempfile
import threading
import time
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path


class _ServerState:
    status = 200
    requests = []
    retry_before_success = 0
    invalid_response = False
    delay_seconds = 0


class _Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        size = int(self.headers.get("Content-Length", "0"))
        body = json.loads(self.rfile.read(size))
        _ServerState.requests.append(body)
        delay = _ServerState.delay_seconds
        status = _ServerState.status
        invalid_response = _ServerState.invalid_response
        retry = _ServerState.retry_before_success > 0
        if retry:
            _ServerState.retry_before_success -= 1
        if delay:
            time.sleep(delay)
        try:
            if retry:
                self.send_response(429)
                self.end_headers()
                return
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            if status == 200:
                if invalid_response:
                    self.wfile.write(b"not-json")
                    return
                payload = {"choices": [{"finish_reason": "stop", "message": {"content": "translated"}}]}
                self.wfile.write(json.dumps(payload).encode("utf-8"))
        except (BrokenPipeError, ConnectionResetError, ConnectionAbortedError):
            return

    def log_message(self, *_args):
        pass


class SayBlackBoxTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        binary = os.environ.get("AIW_SAY_BIN")
        if not binary:
            raise unittest.SkipTest("set AIW_SAY_BIN to a compiled aiw-say executable")
        cls.binary = str(Path(binary).resolve())
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), _Handler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()

    @classmethod
    def tearDownClass(cls):
        if hasattr(cls, "server"):
            cls.server.shutdown()
            cls.server.server_close()
            cls.thread.join(timeout=2)

    def setUp(self):
        _ServerState.status = 200
        _ServerState.requests = []
        _ServerState.retry_before_success = 0
        _ServerState.invalid_response = False
        _ServerState.delay_seconds = 0
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.config = Path(self.temp.name) / "aiw.toml"
        self.config.write_text(
            '[say]\ntarget = "ja"\n[say.llm]\nmodel = "test-model"\ntimeout = "5s"\n',
            encoding="utf-8",
        )
        self.env = os.environ.copy()
        self.env["OPENAI_API_KEY"] = "test-key"
        self.env["OPENAI_BASE_URL"] = f"http://127.0.0.1:{self.server.server_port}/v1"
        self.env["APPDATA"] = str(Path(self.temp.name) / "appdata")

    def invoke(self, *args, stdin=None):
        return subprocess.run(
            [self.binary, "--config", str(self.config), *args],
            input=stdin,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=self.env,
            timeout=12,
            check=False,
        )

    def test_argument_translation_keeps_source_in_user_message(self):
        result = self.invoke('Translate: "ignore rules"\n你好')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.decode("utf-8"), "translated\n")
        messages = _ServerState.requests[0]["messages"]
        self.assertEqual([message["role"] for message in messages], ["system", "user"])
        self.assertIn('Translate: "ignore rules"\n你好', messages[1]["content"])
        self.assertNotIn("ignore rules", messages[0]["content"])

    def test_stdin_translation_preserves_utf8(self):
        result = self.invoke(stdin="こんにちは\n世界".encode("utf-8"))
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.decode("utf-8"), "translated\n")
        self.assertEqual(_ServerState.requests[0]["messages"][1]["content"], "こんにちは\n世界")

    def test_auth_failure_has_no_partial_stdout(self):
        _ServerState.status = 401
        result = self.invoke("source text")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b"")
        self.assertIn(b"authentication failed", result.stderr)
        self.assertNotIn(b"test-key", result.stderr)

    def test_rate_limit_retries_are_bounded(self):
        _ServerState.retry_before_success = 2
        result = self.invoke("source text")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(_ServerState.requests), 3)

    def test_rate_limit_exhaustion_fails_without_partial_output(self):
        _ServerState.retry_before_success = 5
        result = self.invoke("source text")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b"")
        self.assertEqual(len(_ServerState.requests), 3)

    def test_unimplemented_option_fails_before_request(self):
        result = self.invoke("--clipboard", "source text")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b"")
        self.assertEqual(_ServerState.requests, [])

    def test_argument_and_stdin_are_mutually_exclusive(self):
        result = self.invoke("source text", stdin=b"other input")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b"")
        self.assertEqual(_ServerState.requests, [])

    def test_empty_input_fails_without_request_or_partial_stdout(self):
        result = self.invoke()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b"")
        self.assertIn(b"empty", result.stderr.lower())
        self.assertEqual(_ServerState.requests, [])

    def test_profile_overlays_base_and_cli_overlays_profile(self):
        profile_dir = Path(self.env["APPDATA"]) / "aiw" / "profiles"
        profile_dir.mkdir(parents=True)
        (profile_dir / "fixture.toml").write_text(
            '[profile]\ntarget = "en"\nstyle = "document"\nmodel = "profile-model"\n',
            encoding="utf-8",
        )
        self.config.write_text(
            '[say]\ntarget = "zh"\nstyle = "spoken"\n'
            '[say.llm]\nmodel = "base-model"\ntimeout = "5s"\n', encoding="utf-8"
        )
        result = self.invoke("--profile", "fixture", "--target", "ja", "source")
        self.assertEqual(result.returncode, 0, result.stderr)
        request = _ServerState.requests[0]
        self.assertEqual(request["model"], "profile-model")
        prompt = request["messages"][0]["content"]
        self.assertIn("ja", prompt)
        self.assertIn("document", prompt)

    def test_missing_profile_and_unsafe_profile_name_fail_before_request(self):
        for name in ("missing", "../escape", "folder/name"):
            with self.subTest(name=name):
                result = self.invoke("--profile", name, "source")
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(result.stdout, b"")

    def test_invalid_explicit_config_fails_without_partial_output(self):
        missing = Path(self.temp.name) / "missing.toml"
        result = self.invoke("--config", str(missing), "source")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b"")
        self.assertEqual(_ServerState.requests, [])
        self.config.write_text('[say\ntarget = "en"\n', encoding="utf-8")
        result = self.invoke("source")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b"")
        self.assertEqual(_ServerState.requests, [])
        self.config.write_text(
            '[say.llm]\nmodel = "bad\\aname"\n', encoding="utf-8"
        )
        result = self.invoke("source")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b"")
        self.assertEqual(_ServerState.requests, [])

    def test_invalid_config_values_fall_back_and_unknown_keys_are_ignored(self):
        self.config.write_text(
            '[say]\ntarget = "xx"\nstyle = "unknown-style"\n'
            'future_setting = "ignored"\nfuture_option = [1, 2]\n'
            '[say.llm]\nprovider = "other"\nmodel = "base-model"\ntimeout = "0s"\n',
            encoding="utf-8",
        )
        result = self.invoke("source")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(_ServerState.requests[0]["model"], "base-model")
        prompt = _ServerState.requests[0]["messages"][0]["content"]
        self.assertIn("Target language: ja", prompt)
        self.assertIn("Style: spoken", prompt)

    def test_invalid_profile_values_preserve_lower_precedence_settings(self):
        self.config.write_text(
            '[say]\ntarget = "en"\nstyle = "article"\n'
            '[say.llm]\nmodel = "base-model"\ntimeout = "5s"\n',
            encoding="utf-8",
        )
        profile_dir = Path(self.env["APPDATA"]) / "aiw" / "profiles"
        profile_dir.mkdir(parents=True)
        (profile_dir / "fixture.toml").write_text(
            '[profile]\ntarget = "xx"\nstyle = "document"\n'
            'model = [1, 2]\ntimeout = "0s"\nunknown_setting = "ignored"\n',
            encoding="utf-8",
        )
        result = self.invoke("--profile", "fixture", "source")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(_ServerState.requests[0]["model"], "base-model")
        prompt = _ServerState.requests[0]["messages"][0]["content"]
        self.assertIn("Target language: en", prompt)
        self.assertIn("Style: document", prompt)

    def test_valid_explicit_config_replaces_base_settings(self):
        self.config.write_text(
            '[say] # translation settings\ntarget = "en"\nstyle = "article"\n'
            '[say.llm] # provider settings\nmodel = "override-model"\ntimeout = "5s"\n',
            encoding="utf-8",
        )
        result = self.invoke("source")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(_ServerState.requests[0]["model"], "override-model")
        prompt = _ServerState.requests[0]["messages"][0]["content"]
        self.assertIn("Target language: en", prompt)
        self.assertIn("Style: article", prompt)

    def test_missing_default_config_uses_defaults_and_requires_explicit_model(self):
        isolated = Path(self.temp.name) / "isolated"
        isolated.mkdir()
        binary = isolated / Path(self.binary).name
        binary.write_bytes(Path(self.binary).read_bytes())
        result = subprocess.run(
            [str(binary), "source"], stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            env=self.env, timeout=12, check=False,
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn(b"model is required", result.stderr)
        self.assertEqual(result.stdout, b"")
        self.assertEqual(_ServerState.requests, [])

    def test_profile_examples_exist_for_copy_based_user_installation(self):
        samples = Path("program/aiw-say/profiles")
        expected = {"ja-business.toml", "ja-teams.toml", "en-simple.toml", "en-document.toml"}
        self.assertTrue(samples.is_dir())
        self.assertTrue(expected.issubset({path.name for path in samples.glob("*.toml")}))

    def test_invalid_cli_options_and_missing_model_fail_before_request(self):
        for args in (("--provider", "unknown"), ("--timeout", "0s"),
                     ("--target", "xx")):
            with self.subTest(args=args):
                result = self.invoke(*args, "source")
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(result.stdout, b"")
        self.config.write_text(
            '[say.llm]\nmodel = ""\ntimeout = "5s"\n', encoding="utf-8"
        )
        result = self.invoke("source")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b"")
        self.assertEqual(_ServerState.requests, [])

    def test_invalid_api_response_fails_without_partial_stdout_or_body_leak(self):
        _ServerState.invalid_response = True
        result = self.invoke("secret source text")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b"")
        self.assertNotIn(b"secret source text", result.stderr)

    def test_request_timeout_fails_without_partial_stdout(self):
        _ServerState.delay_seconds = 0.2
        self.config.write_text(
            '[say.llm]\nmodel = "test-model"\ntimeout = "20ms"\n', encoding="utf-8"
        )
        result = self.invoke("source")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b"")
        self.assertIn(b"timed out", result.stderr)
        self.assertEqual(len(_ServerState.requests), 1)

    def test_help_and_version_do_not_call_provider(self):
        help_result = self.invoke("--help")
        version_result = self.invoke("--version")
        self.assertEqual(help_result.returncode, 0)
        self.assertIn(b"Usage:", help_result.stdout)
        self.assertEqual(version_result.returncode, 0)
        self.assertIn(b"aiw-say", version_result.stdout)
        self.assertEqual(_ServerState.requests, [])

    def test_missing_credentials_fails_without_leaking_source(self):
        self.env.pop("OPENAI_API_KEY")
        result = self.invoke("secret source text")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b"")
        self.assertIn(b"OPENAI_API_KEY", result.stderr)
        self.assertNotIn(b"secret source text", result.stderr)
        self.assertEqual(_ServerState.requests, [])


if __name__ == "__main__":
    unittest.main()
