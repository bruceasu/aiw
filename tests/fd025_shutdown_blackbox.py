"""FD-025: temporary loopback lifecycle; no /v1/responses or model execution."""
import hashlib
import http.client
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
INSTALL = Path("C:/green/aiw")
SOURCE_CONFIG = INSTALL / "plugins/aiw-gw/gateway.json"
GATEWAY = INSTALL / "plugins/aiw-gw/agent-gateway.exe"
ENTRY = INSTALL / "plugins/aiw-gw/aiw-gw.py"
SOURCE_EVENT = "FD-025-000010-implementation-ready"
SESSION = "fd025-tester-20261004-a173fa"
FD = ROOT / "docs/features/FD-025_GATEWAY_GRACEFUL_SHUTDOWN_AND_STOP_SCRIPT.md"
AUTH = ROOT / "docs/features/reports/FD-025-test-authorization-r4.json"
RAW = ROOT / "docs/features/reports/FD-025-test-results-r4.json"
COMMAND = "python tests/fd025_shutdown_blackbox.py"
digest = lambda data: hashlib.sha256(data).hexdigest()
fd_bytes = FD.read_bytes()
test_digest = digest(Path(__file__).read_bytes())
binary_digest = digest(GATEWAY.read_bytes())
entry_digest = digest(ENTRY.read_bytes())
authorization = json.loads(AUTH.read_text(encoding="utf-8"))
ad = authorization["data"]
if not (ad["decision"] == "approved" and ad["implementation_event"] == SOURCE_EVENT
        and ad["fd_digest"] == digest(fd_bytes) and ad["tester_session"] == SESSION
        and ad["fd_revision"] == 10 and authorization["source_event"] == SOURCE_EVENT
        and ad["test_source_digest"] == test_digest
        and ad["installed_binary_digest"] == binary_digest
        and ad["installed_entry_digest"] == entry_digest
        and ad["command"] == COMMAND):
    raise SystemExit("FD-025 revision-bound authorization mismatch; no process started")

results = []
process = None
stop_attempted = False
port = None
temporary = tempfile.TemporaryDirectory(prefix="fd025-shutdown-")
# Preserve the temporary workspace if graceful shutdown fails; never delete a live lock.
temporary._finalizer.detach()
temp_root = Path(temporary.name).resolve()
state = temp_root / "state"
work = temp_root / "work"
config_path = temp_root / "gateway temp.json"
foreign_cwd = temp_root / "foreign workdir"
foreign_cwd.mkdir()
error_log = temp_root / "gateway-stderr.log"
error_handle = error_log.open("wb")
key = secrets.token_urlsafe(32)
env = dict(os.environ)
env["PATH"] = str(INSTALL) + os.pathsep + env.get("PATH", "")
flags = subprocess.CREATE_NO_WINDOW if os.name == "nt" else 0

def command(script):
    comspec = os.environ.get("COMSPEC", "cmd.exe")
    script_path = ROOT / "scripts" / script
    return f'"{comspec}" /d /s /c ""{script_path}" --config "{config_path}""'

def safe_stderr_class():
    data = error_log.read_bytes().lower()
    for marker, category in [(b"flag provided but not defined", "unknown-cli-flag"),
                             (b"no such file", "missing-file"),
                             (b"cannot find", "missing-file"),
                             (b"address already in use", "listen-port-conflict"),
                             (b"config", "configuration-error")]:
        if marker in data:
            return category
    return "stderr-present-unclassified" if data else "no-stderr"

def add(name, passed, detail=None):
    result = {"name": name, "status": "passed" if passed else "failed"}
    if detail is not None:
        result["safe_detail"] = detail
    results.append(result)
    if not passed:
        raise AssertionError(name)

def listener():
    try:
        with socket.create_connection(("127.0.0.1", port), timeout=0.2):
            return True
    except OSError:
        return False

def control(method="POST", query="", body=b"", credential=None):
    conn = http.client.HTTPConnection("127.0.0.1", port, timeout=2)
    headers = {} if credential is None else {"Authorization": "Bearer " + credential}
    try:
        conn.request(method, "/internal/shutdown" + query, body=body, headers=headers)
        response = conn.getresponse()
        status = response.status
        response.read(4096)  # Discard; never record body or raw error.
        return status
    finally:
        conn.close()

def invoke_stop():
    return subprocess.run(command("stop-gateway.bat"), cwd=foreign_cwd, env=env,
                          stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                          stderr=error_handle, creationflags=flags, timeout=18).returncode

try:
    if os.name != "nt" or not (INSTALL / "aiw.exe").is_file():
        raise RuntimeError("installed-windows-entry-unavailable")
    source = json.loads(SOURCE_CONFIG.read_text(encoding="utf-8"))
    config = {name: source[name] for name in ("codex_path", "codex_script", "models", "timezone") if name in source}
    del source
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as reserved:
        reserved.bind(("127.0.0.1", 0))
        port = reserved.getsockname()[1]
    if port == 43127:
        raise RuntimeError("protected-user-port-selected")
    models = config["models"]
    if not models:
        raise RuntimeError("no-model-map")
    config.update({"mode": "development", "listen": "127.0.0.1:" + str(port),
                   "state_dir": str(state), "workspace_dir": str(work),
                   "timeout_seconds": 60, "global_concurrency": 1,
                   "principals": [{"id": "fd025-test", "enabled": True, "keys": [key],
                                   "allowed_models": list(models), "rpm": 10, "daily_limit": 10,
                                   "concurrency": 1}]})
    config_path.write_text(json.dumps(config), encoding="utf-8")
    del config
    process = subprocess.Popen(command("start-gateway.bat"), cwd=foreign_cwd, env=env,
                               stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                               stderr=error_handle, creationflags=flags)
    deadline = time.monotonic() + 5
    while time.monotonic() < deadline and process.poll() is None:
        if listener() and (state / "gateway.lock").exists():
            break
        time.sleep(0.05)
    add("从任意工作目录启动脚本且临时loopback监听就绪",
        process.poll() is None and listener() and (state / "gateway.lock").exists(),
        {"process_exit_code": process.poll(), "stderr_class": safe_stderr_class()})
    for name, method, query, body, credential, expected in [
        ("未认证POST拒绝且服务继续运行", "POST", "", b"", None, 401),
        ("无效合成Key拒绝且服务继续运行", "POST", "", b"", "wrong-" + key, 401),
        ("有效认证GET拒绝且服务继续运行", "GET", "", b"", key, 405),
        ("有效认证query拒绝且服务继续运行", "POST", "?test=1", b"", key, 400),
        ("有效认证非空body拒绝且服务继续运行", "POST", "", b"x", key, 400),
    ]:
        actual = control(method, query, body, credential)
        add(name, actual == expected and process.poll() is None and listener(), {"http_status": actual})
    stop_attempted = True
    stop_code = invoke_stop()
    add("stop脚本正常返回成功", stop_code == 0, {"exit_code": stop_code})
    process_code = process.wait(timeout=2)
    add("Gateway及同步启动脚本正常退出0", process_code == 0, {"exit_code": process_code})
    add("临时监听关闭且gateway.lock释放", not listener() and not (state / "gateway.lock").exists())
    repeat_code = invoke_stop()
    add("重复stop返回成功", repeat_code == 0, {"exit_code": repeat_code})
    metadata_count = len(list((state / "metadata").glob("*.json")))
    content_count = len(list((state / "content").glob("*.json")))
    add("不提交模型请求且没有执行或正文记录", metadata_count == 0 and content_count == 0,
        {"execution_records": metadata_count, "content_records": content_count})
    default_dir = temp_root / "isolated plugin default"
    default_dir.mkdir()
    default_entry = default_dir / ENTRY.name
    shutil.copy2(ENTRY, default_entry)
    shutil.copy2(GATEWAY, default_dir / GATEWAY.name)
    shutil.copy2(config_path, default_dir / "gateway.json")
    default_code = subprocess.run([sys.executable, str(default_entry), "stop"],
                                  cwd=foreign_cwd, env=env, stdin=subprocess.DEVNULL,
                                  stdout=subprocess.DEVNULL, stderr=error_handle,
                                  creationflags=flags, timeout=18).returncode
    add("隔离插件默认配置stop不带config返回成功", default_code == 0,
        {"exit_code": default_code})
except Exception as error:
    # Only classify the exception; never output command/environment/config/raw response.
    if not results or results[-1]["status"] != "failed":
        results.append({"name": "黑盒测试基础设施或关停异常", "status": "failed",
                        "error_class": type(error).__name__})
finally:
    if process is not None and process.poll() is None and not stop_attempted:
        try:
            invoke_stop()  # One cleanup stop only if normal stop was never attempted.
            process.wait(timeout=2)
        except Exception:
            pass
    stopped = process is None or process.poll() is not None
    safe_cleanup = stopped and not (state / "gateway.lock").exists()
    error_handle.close()
    if safe_cleanup:
        if not temp_root.is_relative_to(Path(tempfile.gettempdir()).resolve()):
            raise RuntimeError("temporary cleanup boundary mismatch")
        temporary.cleanup()
    key = None

output = {"fd_id": "FD-025", "source_event": SOURCE_EVENT, "tester_session": SESSION,
          "fd_revision": 10, "fd_digest": digest(fd_bytes), "test_source_digest": test_digest,
          "installed_binary_digest": binary_digest,
          "installed_entry_digest": entry_digest,
          "authorization_record": "docs/features/reports/FD-025-test-authorization-r4.json",
          "transport": "isolated loopback shutdown only; no model requests",
          "protected_port_43127_used": False, "results": results,
          "passed": sum(row["status"] == "passed" for row in results),
          "failed": sum(row["status"] == "failed" for row in results),
          "temporary_workspace_removed": safe_cleanup,
          "temporary_workspace_preserved": None if safe_cleanup else str(temp_root)}
with RAW.open("x", encoding="utf-8") as handle:
    json.dump(output, handle, ensure_ascii=False, indent=2)
    handle.write("\n")
print(json.dumps(output, ensure_ascii=True, indent=2))
raise SystemExit(1 if output["failed"] else 0)
