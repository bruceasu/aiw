#!/usr/bin/env python3
"""Persisted-notification dispatch Plugin protocol for AIW."""

import argparse
import json
import sys


META = {
    "name": "aiw-notify",
    "short": "dispatch a persisted workflow notification",
    "description": "Accept one persisted notification from Workflow Core and return an idempotency receipt.",
    "commands": ["dispatch"],
    "readOnly": True,
    "mutatesFiles": False,
    "requiresConfirmation": False,
    "outputFormat": "json",
}


def dispatch(as_json: bool) -> int:
    if not as_json:
        raise SystemExit("use: aiw notify dispatch --json")
    try:
        notification = json.load(sys.stdin)
    except json.JSONDecodeError as exc:
        print(json.dumps({"status": "rejected", "error": f"invalid JSON: {exc}"}))
        return 2
    notification_id = notification.get("id")
    topic = notification.get("topic")
    payload = notification.get("payload")
    if not isinstance(notification_id, str) or not notification_id or not isinstance(topic, str) or not topic or payload is None:
        print(json.dumps({"status": "rejected", "error": "id, topic, and payload are required"}))
        return 2

    # Delivery providers can replace this Plugin while retaining the stable ID
    # as their idempotency key. The bundled implementation is intentionally a
    # deterministic local acknowledgement and performs no network activity.
    print(json.dumps({
        "notification_id": notification_id,
        "status": "accepted",
        "receipt": f"aiw-notify:{notification_id}",
    }, ensure_ascii=False))
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(prog="aiw-notify")
    sub = parser.add_subparsers(dest="command")
    dispatch_parser = sub.add_parser("dispatch")
    dispatch_parser.add_argument("--json", action="store_true")
    args = parser.parse_args()
    if args.command == "dispatch":
        return dispatch(args.json)
    parser.error("use: aiw-notify dispatch --json")
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
