#!/usr/bin/env python3
META = {
    "name": "aiw-patch",
    "short": "check, apply, and reverse patches",
    "description": "Check, apply, and reverse patches with Git and patch-format conversion.",
    "commands": ["check", "apply", "reverse"],
    "readOnly": False,
    "mutatesFiles": True,
    "requiresConfirmation": True,
    "outputFormat": "json",
}
import argparse
import difflib
import json
import os
import subprocess
import sys
import tempfile

from aiw_codec import CodecError, detect


class PatchError(Exception):
    pass


def decode_patch(data, encoding=None):
    try:
        return detect(data, encoding)[1]
    except Exception as exc:
        raise PatchError(str(exc)) from exc


def apply_update(original, body, path):
    lines = original.splitlines(keepends=True)
    result, cursor = [], 0
    for raw in body:
        if raw.startswith("@@"):
            continue
        if raw.startswith(" "):
            expected = raw[1:]
            try:
                index = next(i for i in range(cursor, len(lines)) if lines[i].rstrip("\r\n") == expected.rstrip("\r\n"))
            except StopIteration as exc:
                raise PatchError("Update File context not found: " + path) from exc
            result.extend(lines[cursor:index])
            result.append(expected)
            cursor = index + 1
        elif raw.startswith("-"):
            expected = raw[1:]
            if cursor >= len(lines) or lines[cursor].rstrip("\r\n") != expected.rstrip("\r\n"):
                raise PatchError("Update File removal does not match: " + path)
            cursor += 1
        elif raw.startswith("+"):
            result.append(raw[1:])
        elif raw:
            raise PatchError("unsupported Update File line in: " + path)
    result.extend(lines[cursor:])
    return "".join(result)


def unified(path, old, new):
    return "".join(difflib.unified_diff(
        old.splitlines(keepends=True), new.splitlines(keepends=True),
        fromfile="a/" + path, tofile="b/" + path, lineterm="\n"))


def convert_begin_patch(text):
    lines = text.splitlines(keepends=True)
    if not any(x.rstrip("\r\n") == "*** Begin Patch" for x in lines):
        return text
    if not any(x.rstrip("\r\n") == "*** End Patch" for x in lines):
        raise PatchError("missing *** End Patch")
    operations, current = [], None
    for line in lines:
        value = line.rstrip("\r\n")
        if value in ("*** Begin Patch", "*** End Patch"):
            continue
        if value.startswith("*** Update File: "):
            current = ["update", value[17:], []]
            operations.append(current)
        elif value.startswith("*** Add File: "):
            current = ["add", value[15:], []]
            operations.append(current)
        elif value.startswith("*** Delete File: "):
            current = ["delete", value[18:], []]
            operations.append(current)
        elif value.startswith("*** Move to File: "):
            if current is None:
                raise PatchError("Move to File has no source operation")
            current[0] = "move"
            current.append(value[19:])
        elif current is not None:
            current[2].append(line)
        elif value:
            raise PatchError("unexpected content outside patch operation")
    rendered = []
    for operation in operations:
        kind, path, body = operation[:3]
        if kind == "add":
            new = "".join(x[1:] for x in body if x.startswith("+"))
            rendered.append(unified(path, "", new))
        elif kind == "delete":
            with open(path, encoding="utf-8", newline="") as handle:
                old = handle.read()
            rendered.append(unified(path, old, ""))
        elif kind == "update":
            with open(path, encoding="utf-8", newline="") as handle:
                old = handle.read()
            rendered.append(unified(path, old, apply_update(old, body, path)))
        elif kind == "move":
            target = operation[3]
            with open(path, encoding="utf-8", newline="") as handle:
                old = handle.read()
            rendered.extend((unified(path, old, ""), unified(target, "", old)))
    return "".join(rendered)


def git_apply(patch_text, operation, threeway=False, index=False):
    fd, path = tempfile.mkstemp(prefix="aiw-patch-", suffix=".diff")
    os.close(fd)
    try:
        with open(path, "w", encoding="utf-8", newline="") as f:
            f.write(patch_text)
        check = ["git", "apply", "--check"]
        if threeway:
            check.append("--3way")
        if index:
            check.append("--index")
        check.append(path)
        checked = subprocess.run(check, text=True, capture_output=True)
        if checked.returncode:
            return checked.returncode, checked.stdout, checked.stderr
        command = ["git", "apply"]
        if operation == "reverse":
            command.append("-R")
        if threeway:
            command.append("--3way")
        if index:
            command.append("--index")
        command.append(path)
        applied = subprocess.run(command, text=True, capture_output=True)
        return applied.returncode, applied.stdout, applied.stderr
    finally:
        try:
            os.remove(path)
        except OSError:
            pass


def main(argv=None):
    parser = argparse.ArgumentParser(prog="aiw patch")
    parser.add_argument("operation", choices=("check", "apply", "reverse"))
    parser.add_argument("patch")
    parser.add_argument("--encoding")
    parser.add_argument("--3way", dest="threeway", action="store_true")
    parser.add_argument("--index", action="store_true")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args(argv)
    try:
        data = sys.stdin.buffer.read() if args.patch == "-" else open(args.patch, "rb").read()
        text = convert_begin_patch(decode_patch(data, args.encoding))
        code, out, err = git_apply(text, args.operation, args.threeway, args.index)
        result = {"ok": code == 0, "operation": args.operation, "exitCode": code, "stdout": out, "stderr": err}
        if args.json:
            print(json.dumps(result, ensure_ascii=False))
        else:
            if out:
                print(out, end="")
            if err:
                print(err, end="", file=sys.stderr)
        return code
    except (OSError, PatchError, UnicodeError, CodecError) as exc:
        result = {"ok": False, "operation": args.operation, "exitCode": 2, "stdout": "", "stderr": str(exc)}
        if args.json:
            print(json.dumps(result, ensure_ascii=False))
        else:
            print("aiw patch error: " + str(exc), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
