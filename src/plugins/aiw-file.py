#!/usr/bin/env python3
"""Read and write text files while preserving encoding and newline style."""
META = {
    "name": "aiw-file",
    "short": "read, inspect, and write workspace text files",
    "description": "Read, inspect, and write workspace text files with codec detection.",
    "commands": ["read", "info", "write"],
    "readOnly": False,
    "mutatesFiles": True,
    "requiresConfirmation": False,
    "outputFormat": "json",
}
import argparse
import json
import os
import sys
import tempfile

from aiw_codec import CodecError, detect, encode_text, newline_style, normalize_newlines


def read_file(path, requested=None):
    with open(path, "rb") as handle:
        data = handle.read()
    encoding, text, confidence = detect(data, requested)
    bom = data.startswith((b"\xef\xbb\xbf", b"\xff\xfe", b"\xfe\xff"))
    return {"path": path, "encoding": encoding, "confidence": confidence,
            "bom": bom, "newline": newline_style(text), "text": text}


def atomic_write(path, text, encoding, bom, newline):
    text = normalize_newlines(text, newline)
    parent = os.path.dirname(os.path.abspath(path))
    fd, temp = tempfile.mkstemp(prefix=".aiw-file-", dir=parent)
    os.close(fd)
    try:
        with open(temp, "wb") as handle:
            handle.write(encode_text(text, encoding, bom))
        os.replace(temp, path)
    finally:
        if os.path.exists(temp):
            os.remove(temp)


def main(argv=None):
    parser = argparse.ArgumentParser(prog="aiw file")
    sub = parser.add_subparsers(dest="command", required=True)
    for name in ("read", "info"):
        command = sub.add_parser(name)
        command.add_argument("path")
        command.add_argument("--encoding")
    write = sub.add_parser("write")
    write.add_argument("path")
    write.add_argument("--content")
    write.add_argument("--encoding", default="preserve")
    write.add_argument("--newline", default="preserve", choices=("preserve", "lf", "crlf", "cr"))
    write.add_argument("--json", action="store_true")
    args = parser.parse_args(argv)
    try:
        if args.command in ("read", "info"):
            result = read_file(args.path, args.encoding)
            if args.command == "info":
                result.pop("text")
            print(json.dumps(result, ensure_ascii=False) if args.command == "info" else result["text"], end="" if args.command == "read" else "\n")
            return 0
        existing = read_file(args.path) if os.path.exists(args.path) else None
        content = args.content
        if content is None:
            content = sys.stdin.read()
        encoding = existing["encoding"] if args.encoding == "preserve" and existing else args.encoding
        if not encoding or encoding == "preserve":
            encoding = "utf-8"
        bom = existing["bom"] if args.encoding == "preserve" and existing else False
        newline = existing["newline"] if args.newline == "preserve" and existing else args.newline
        atomic_write(args.path, content, encoding, bom, newline)
        result = {"ok": True, "path": args.path, "encoding": encoding, "bom": bom, "newline": newline}
        if args.json:
            print(json.dumps(result, ensure_ascii=False))
        return 0
    except (OSError, UnicodeError, CodecError) as exc:
        print("aiw file error: " + str(exc), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
