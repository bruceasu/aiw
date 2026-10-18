"""AIW ZIP package format. Python 3.11+, standard library only."""
from __future__ import annotations

import hashlib
import io
import json
import os
from pathlib import Path, PurePosixPath
import re
import stat
import tempfile
import tomllib
import zipfile


class PackageError(Exception):
    pass


MAX_BYTES = 512 * 1024 * 1024
MAX_FILE = 64 * 1024 * 1024
MAX_META = 4 * 1024 * 1024
MAX_ENTRIES = 20000
IDENT = re.compile(r"[A-Za-z0-9][A-Za-z0-9._+-]{0,119}\Z")
HASH = re.compile(r"[0-9a-f]{64}\Z")
RESERVED = {"con", "prn", "aux", "nul", *(f"com{i}" for i in range(1, 10)),
            *(f"lpt{i}" for i in range(1, 10))}


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def identifier(value: str) -> str:
    if not isinstance(value, str) or not IDENT.fullmatch(value) or value in {".", ".."}:
        raise PackageError(f"Invalid package name/version: {value!r}")
    return value


def relative(value: str) -> str:
    if not isinstance(value, str) or not value or len(value) > 1024:
        raise PackageError(f"Invalid package path: {value!r}")
    parts = value.split("/")
    if (PurePosixPath(value).is_absolute() or any(
        not p or p in {".", ".."} or p.endswith((".", " "))
        or p.split(".")[0].casefold() in RESERVED
        or any(ord(c) < 32 or c in '\\:*?"<>|' for c in p) for p in parts
    )):
        raise PackageError(f"Unsafe package path: {value!r}")
    return value


def safe_path(root: Path, value: str) -> Path:
    relative(value)
    path = root
    # Inspect existing ancestors as well as the leaf, including Windows junctions.
    for ancestor in reversed((root, *root.parents)):
        reject_link(ancestor)
    for part in value.split("/"):
        path = path / part
        reject_link(path)
    return path


def reject_link(path: Path) -> None:
    try:
        info = path.lstat()
    except FileNotFoundError:
        return
    if stat.S_ISLNK(info.st_mode) or getattr(info, "st_file_attributes", 0) & 0x400:
        raise PackageError(f"Links/reparse points are not supported: {path}")


def zip_members(source, *, outer: bool = False) -> dict[str, bytes]:
    with zipfile.ZipFile(source) as archive:
        entries = archive.infolist()
        if len(entries) > MAX_ENTRIES:
            raise PackageError("Too many ZIP entries")
        result, folded, total = {}, set(), 0
        for entry in entries:
            if entry.orig_filename != entry.filename:
                raise PackageError("Truncated ZIP filename")
            name = relative(entry.filename.rstrip("/") if entry.is_dir() else entry.filename)
            key = name.casefold()
            mode = entry.external_attr >> 16
            if key in folded or stat.S_IFMT(mode) not in (0, stat.S_IFREG, stat.S_IFDIR):
                raise PackageError(f"Duplicate or special ZIP entry: {name}")
            folded.add(key)
            if entry.flag_bits & 1:
                raise PackageError("Encrypted ZIP entries are not supported")
            limit = MAX_BYTES if outer and name == "data.zip" else MAX_FILE
            total += entry.file_size
            if entry.file_size > limit or total > MAX_BYTES:
                raise PackageError("ZIP payload exceeds package limits")
            content = archive.read(entry)  # validates CRC, including directory entries
            if entry.is_dir():
                if content:
                    raise PackageError(f"Nonempty directory entry: {name}")
            else:
                result[name] = content
        return result


def validate(info: dict, payload: dict[str, bytes], scripts: dict[str, bytes]) -> str:
    allowed = {"format_version", "name", "version", "kind", "data_sha256", "files"}
    if set(info) != allowed or type(info.get("format_version")) is not int or info["format_version"] != 1:
        raise PackageError("Unsupported info schema; expected format_version=1")
    identifier(info["name"])
    identifier(info["version"])
    if info["kind"] not in {"program", "plugin", "skill", "resource"}:
        raise PackageError("Invalid package kind")
    if not isinstance(info["data_sha256"], str) or not HASH.fullmatch(info["data_sha256"]):
        raise PackageError("Invalid data_sha256")
    files = info["files"]
    if not isinstance(files, list) or not files or len(files) > MAX_ENTRIES:
        raise PackageError("info must declare a nonempty files array")
    sources, targets = set(), {"windows": {}, "linux": {}}
    for item in files:
        if not isinstance(item, dict) or set(item) != {
            "source", "target", "platform", "sha256", "config", "executable"
        }:
            raise PackageError("Invalid files entry")
        src, dst = relative(item["source"]), relative(item["target"])
        if dst.casefold() == ".aiw" or dst.casefold().startswith(".aiw/"):
            raise PackageError("Package cannot write engine state")
        if src in sources or src not in payload:
            raise PackageError(f"Missing/duplicate declared payload: {src}")
        sources.add(src)
        if item["sha256"] != digest(payload[src]):
            raise PackageError(f"File checksum mismatch: {src}")
        if type(item["config"]) is not bool or type(item["executable"]) is not bool:
            raise PackageError("config/executable must be booleans")
        platform = item["platform"]
        if platform not in {"all", "windows", "linux"}:
            raise PackageError("Unsupported file platform")
        for system in targets if platform == "all" else (platform,):
            paths = targets[system]
            folded = dst.casefold()
            if any(folded == p or folded.startswith(p + "/") or p.startswith(folded + "/") for p in paths):
                raise PackageError(f"Conflicting target path: {dst}")
            paths[folded] = dst
    if sources != set(payload):
        raise PackageError("Undeclared files in data.zip")
    for name, data in scripts.items():
        if len(data) > MAX_META:
            raise PackageError(f"Script too large: {name}")
        try:
            compile(data, name, "exec")
        except (SyntaxError, ValueError) as error:
            raise PackageError(f"Invalid Python script {name}: {error}") from error
    canonical = {key: value for key, value in info.items() if key != "data_sha256"}
    canonical["files"] = sorted(files, key=lambda item: item["source"])
    return digest(json.dumps({"info": canonical, "scripts": {
        name: digest(data) for name, data in scripts.items()
    }}, sort_keys=True, separators=(",", ":")).encode())


def read_package(path: Path) -> tuple[dict, dict[str, bytes], dict[str, bytes], str]:
    if path.stat().st_size > MAX_BYTES:
        raise PackageError("Package exceeds size limit")
    outer = zip_members(path, outer=True)
    if not {"info", "data.zip"} <= outer.keys() or outer.keys() - {"info", "data.zip", "preinst", "postinst"}:
        raise PackageError("Expected info, data.zip and optional preinst/postinst at ZIP root")
    if len(outer["info"]) > MAX_META:
        raise PackageError("info is too large")
    try:
        info = tomllib.loads(outer["info"].decode("utf-8"))
    except (UnicodeError, tomllib.TOMLDecodeError) as error:
        raise PackageError(f"Invalid UTF-8 TOML info: {error}") from error
    if info.get("data_sha256") != digest(outer["data.zip"]):
        raise PackageError("data.zip checksum mismatch")
    payload = zip_members(io.BytesIO(outer["data.zip"]))
    scripts = {name: outer[name] for name in ("preinst", "postinst") if name in outer}
    fingerprint = validate(info, payload, scripts)
    if path.name != f"{info['name']}-{info['version']}.zip":
        raise PackageError("Package filename does not match info name/version")
    return info, payload, scripts, fingerprint


def atomic_bytes(path: Path, content: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    descriptor, temporary = tempfile.mkstemp(prefix=".aiw-", dir=path.parent)
    try:
        with os.fdopen(descriptor, "wb") as stream:
            stream.write(content)
        os.replace(temporary, path)
    finally:
        Path(temporary).unlink(missing_ok=True)


def write_package(path: Path, name: str, version: str, kind: str,
                  entries: list[tuple[Path, str, str, str, bool, bool]],
                  scripts: dict[str, bytes] | None = None) -> None:
    """entries: input path, archive source, install target, platform, config, executable."""
    payload, files = {}, []
    total = 0
    for original, source, target, platform, config, executable in entries:
        reject_link(original)
        if not original.is_file() or original.stat().st_size > MAX_FILE:
            raise PackageError(f"Missing/oversized input: {original}")
        if source in payload:
            raise PackageError(f"Duplicate source: {source}")
        payload[source] = original.read_bytes()
        total += len(payload[source])
        if total > MAX_BYTES or len(payload) > MAX_ENTRIES:
            raise PackageError("Package payload exceeds limits")
        files.append(dict(source=source, target=target, platform=platform,
                          config=config, executable=executable, sha256=digest(payload[source])))
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for source, data in sorted(payload.items()):
            archive.writestr(source, data)
    data = buffer.getvalue()
    info = dict(format_version=1, name=name, version=version, kind=kind,
                data_sha256=digest(data), files=files)
    scripts = scripts or {}
    validate(info, payload, scripts)
    lines = [f"{key} = {json.dumps(value)}" for key, value in info.items() if key != "files"]
    for item in files:
        lines.append("\n[[files]]")
        for key, value in item.items():
            lines.append(f"{key} = {str(value).lower() if type(value) is bool else json.dumps(value)}")
    result = io.BytesIO()
    with zipfile.ZipFile(result, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("info", "\n".join(lines) + "\n")
        archive.writestr("data.zip", data)
        for script, content in scripts.items():
            if script not in {"preinst", "postinst"}:
                raise PackageError(f"Unsupported script: {script}")
            archive.writestr(script, content)
    if len(result.getvalue()) > MAX_BYTES or sum(map(len, payload.values())) > MAX_BYTES:
        raise PackageError("Package exceeds size limit")
    atomic_bytes(path, result.getvalue())


def pack_workspace(workspace: Path, output: Path) -> None:
    """Create a package from data/, info and optional Python preinst/postinst."""
    raw = tomllib.loads((workspace / "info").read_text(encoding="utf-8"))
    entries = [(safe_path(workspace / "data", item["source"]), item["source"], item["target"],
                item["platform"], item["config"], item["executable"]) for item in raw["files"]]
    scripts = {name: (workspace / name).read_bytes() for name in ("preinst", "postinst")
               if (workspace / name).is_file()}
    write_package(output, raw["name"], raw["version"], raw["kind"], entries, scripts)
