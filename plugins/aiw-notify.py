#!/usr/bin/env python3
"""Versioned, bounded text attempts. Core alone owns retry policy."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
from urllib.parse import urlsplit

META = {
    "name": "aiw-notify",
    "short": "attempt a persisted text notification",
    "description": "Attempt console or explicit HTTPS Teams text; no delivery receipt or internal retry.",
    "commands": ["dispatch", "preflight"],
    "readOnly": False,
    "mutatesFiles": False,
    "requiresConfirmation": False,
    "outputFormat": "json",
}

LIMIT = 64 * 1024
FIELDS = {"schema_version", "notification_id", "attempt_id", "content_digest", "text", "channel", "target_reference", "config_reference"}
CONFIG_FIELDS = {"schema_version", "enabled", "channel", "target_id", "teams_url", "credential_env", "ca_file", "sender", "timeout_seconds"}


def unique_object(pairs):
    value = {}
    for key, item in pairs:
        if key in value:
            raise ValueError("protocol")
        value[key] = item
    return value


def decode(raw):
    if len(raw) > LIMIT:
        raise ValueError("protocol")
    return json.loads(raw.decode("utf-8"), object_pairs_hook=unique_object)


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def project_config():
    import tomllib
    root = Path.cwd().resolve()
    path = root / "aiw.toml"
    empty = {"enabled": False, "ready": False, "channel": "", "target_reference": "", "config_reference": ""}
    if not path.is_file():
        return {}, empty
    with path.open("rb") as stream:
        raw = stream.read(LIMIT + 1)
    if len(raw) > LIMIT:
        raise ValueError("configuration")
    config = tomllib.loads(raw.decode("utf-8")).get("notifications")
    if config is None:
        return {}, empty
    if not isinstance(config, dict) or set(config) - CONFIG_FIELDS or type(config.get("schema_version")) is not int or config["schema_version"] != 1 or type(config.get("enabled", False)) is not bool:
        raise ValueError("configuration")
    if not config.get("enabled", False):
        return config, empty
    channel = config.get("channel")
    if channel not in ("console", "teams"):
        raise ValueError("configuration")
    normalized = {"schema_version": 1, "channel": channel, "target_id": config.get("target_id", "console" if channel == "console" else ""), "teams_url": config.get("teams_url", ""), "credential_env": config.get("credential_env", ""), "ca_file": config.get("ca_file", ""), "sender": config.get("sender", ""), "timeout_seconds": config.get("timeout_seconds", 30)}
    if any(not isinstance(normalized[key], str) for key in normalized if key not in ("schema_version", "timeout_seconds")):
        raise ValueError("configuration")
    timeout = normalized["timeout_seconds"]
    if type(timeout) is not int or not 1 <= timeout <= 30 or not normalized["target_id"]:
        raise ValueError("configuration")
    ready = True
    ca_digest = ""
    if channel == "teams":
        endpoint = urlsplit(normalized["teams_url"])
        if endpoint.scheme != "https" or not endpoint.hostname or endpoint.username or endpoint.password or endpoint.query or endpoint.fragment or any(ch.isspace() for ch in normalized["teams_url"]):
            raise ValueError("configuration")
        if not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*", normalized["credential_env"]):
            raise ValueError("configuration")
        ready = bool(os.environ.get(normalized["credential_env"]))
        if normalized["ca_file"]:
            ca = (root / normalized["ca_file"]).resolve(strict=True)
            with ca.open("rb") as stream:
                ca_bytes = stream.read(1024 * 1024 + 1)
            if len(ca_bytes) > 1024 * 1024:
                raise ValueError("configuration")
            normalized["ca_file"] = str(ca)
            ca_digest = digest(ca_bytes)
    canonical = json.dumps({**normalized, "ca_digest": ca_digest}, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
    reference = digest(canonical)
    return normalized, {"enabled": True, "ready": ready, "channel": channel, "target_reference": digest((channel + ":" + normalized["target_id"] + ":" + normalized["teams_url"] + ":" + ca_digest).encode("utf-8")), "config_reference": reference}


def result(request, outcome, attempted, error_code="", status=None):
    return {"schema_version": 2, "notification_id": request.get("notification_id", ""), "attempt_id": request.get("attempt_id", ""), "content_digest": request.get("content_digest", ""), "attempted": attempted, "outcome": outcome, "error_code": error_code, "http_status": status}


def valid_request(request):
    if not isinstance(request, dict) or set(request) != FIELDS or type(request["schema_version"]) is not int or request["schema_version"] != 2:
        return False
    if any(not isinstance(request[key], str) for key in FIELDS - {"schema_version"}):
        return False
    raw = request["text"].encode("utf-8")
    return bool(request["notification_id"] and request["attempt_id"] and raw and len(raw) <= 16 * 1024 and digest(raw) == request["content_digest"])


def managed_dispatch(request):
    if not valid_request(request):
        return result(request if isinstance(request, dict) else {}, "configuration_failure", False, "protocol")
    try:
        config, reference = project_config()
        if not reference["enabled"] or not reference["ready"] or any(request[key] != reference[key] for key in ("channel", "config_reference", "target_reference")):
            return result(request, "configuration_failure", False, "configuration")
    except Exception:
        return result(request, "configuration_failure", False, "configuration")
    if reference["channel"] == "console":
        try:
            print(request["text"], file=sys.stderr, flush=True)
            return result(request, "attempted", True)
        except Exception:
            return result(request, "unknown", True, "protocol")
    child = {"schema_version": 1, "notification_id": request["notification_id"], "attempt_id": request["attempt_id"], "text": request["text"], "endpoint": config["teams_url"], "credential_env": config["credential_env"], "ca_file": config["ca_file"], "sender": config["sender"], "timeout_seconds": config["timeout_seconds"]}
    allowed = {"SYSTEMROOT", "WINDIR", "PATH", "TEMP", "TMP", "LANG", "LC_ALL"}
    environment = {key: value for key, value in os.environ.items() if key.upper() in allowed}
    environment[config["credential_env"]] = os.environ[config["credential_env"]]
    script = Path.cwd().resolve() / "plugins" / "send_teams_msg.py"
    try:
        response = subprocess.run([sys.executable, "-I", str(script), "--managed-json"], input=json.dumps(child, ensure_ascii=False).encode("utf-8"), stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, cwd=Path.cwd(), env=environment, timeout=35, check=False)
        data = decode(response.stdout)
        expected = {"schema_version", "notification_id", "attempt_id", "attempted", "outcome", "error_code", "http_status"}
        if response.returncode != 0 or not isinstance(data, dict) or set(data) != expected or data["schema_version"] != 1 or data["notification_id"] != request["notification_id"] or data["attempt_id"] != request["attempt_id"] or type(data["attempted"]) is not bool:
            raise ValueError("protocol")
        if data["outcome"] not in ("attempted", "network_failure", "configuration_failure", "permission_failure", "service_failure", "unknown") or data["error_code"] not in ("", "configuration", "permission", "network", "service", "protocol", "timeout", "tls"):
            raise ValueError("protocol")
        status = data["http_status"]
        if status is not None and (type(status) is not int or not 100 <= status <= 599):
            raise ValueError("protocol")
        return result(request, data["outcome"], data["attempted"], data["error_code"], status)
    except PermissionError:
        return result(request, "permission_failure", False, "permission")
    except Exception:
        return result(request, "unknown", True, "protocol")


def main():
    parser = argparse.ArgumentParser(prog="aiw-notify")
    parser.add_argument("command", choices=("dispatch", "preflight"))
    parser.add_argument("--json", action="store_true", required=True)
    args = parser.parse_args()
    try:
        request = decode(sys.stdin.buffer.read(LIMIT + 1))
        if args.command == "preflight":
            if request != {"schema_version": 2}:
                raise ValueError("protocol")
            _, value = project_config()
        elif isinstance(request, dict) and "schema_version" not in request:
            # v1 is a historical local acknowledgement only. It never sends.
            if not isinstance(request.get("id"), str) or not request["id"] or not request.get("topic") or "payload" not in request:
                raise ValueError("protocol")
            value = {"notification_id": request["id"], "status": "accepted", "receipt": "aiw-notify:" + request["id"]}
        else:
            value = managed_dispatch(request)
        print(json.dumps(value, ensure_ascii=False))
        return 0
    except Exception:
        print('{"error_code":"configuration"}')
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
