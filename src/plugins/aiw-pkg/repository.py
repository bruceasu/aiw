"""Package repository adapters. Remote operations occur only on explicit calls."""
from __future__ import annotations

from contextlib import contextmanager
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import subprocess
import tempfile
import tomllib
from urllib.parse import urlsplit, quote
from urllib.request import urlopen
import uuid

from archive import (HASH, MAX_BYTES, MAX_ENTRIES, PackageError, atomic_bytes,
                     digest, identifier, read_package, references, safe_path)

INDEX_LIMIT = 16 * 1024 * 1024
KINDS = {"program", "plugin", "skill", "resource", "collection"}
SEMVER = re.compile(r"(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?\Z")


def version_key(version):
    if version == "dev":
        return (-1, -1, -1, 0, ())
    match = SEMVER.fullmatch(version)
    if not match:
        return None
    pre = match[4]
    parts = pre.split(".") if pre else []
    if any(not p or (p.isdigit() and len(p) > 1 and p.startswith("0")) for p in parts):
        return None
    return (*map(int, match.group(1, 2, 3)), int(pre is None),
            tuple((0, int(p)) if p.isdigit() else (1, p) for p in parts))


def package_path(name, version):
    identifier(name)
    identifier(version)
    return f"{name}/{version}/{name}-{version}.zip"


def entry_for(package):
    info, _, _, fingerprint = read_package(package)
    return dict(name=info["name"], version=info["version"], kind=info["kind"],
                path=package_path(info["name"], info["version"]), size=package.stat().st_size,
                sha256=digest(package.read_bytes()), fingerprint=fingerprint,
                dependencies=info.get("dependencies", []), members=info.get("members", []),
                platforms=sorted({f["platform"] for f in info["files"]} or {"all"}))


def decode_index(content):
    if len(content) > INDEX_LIMIT:
        raise PackageError("Repository index exceeds size limit")
    raw = json.loads(content)
    if not isinstance(raw, dict) or set(raw) != {"schema", "packages"} or type(raw["schema"]) is not int or raw["schema"] != 1:
        raise PackageError("Unsupported repository index")
    entries = raw["packages"]
    if not isinstance(entries, list) or len(entries) > MAX_ENTRIES:
        raise PackageError("Invalid repository package list")
    seen = set()
    for item in entries:
        required = {"name", "version", "kind", "path", "size", "sha256", "fingerprint", "dependencies", "members", "platforms"}
        if not isinstance(item, dict) or set(item) != required:
            raise PackageError("Invalid index entry")
        identity = (identifier(item["name"]), identifier(item["version"]))
        if identity in seen or item["path"] != package_path(*identity):
            raise PackageError("Duplicate identity or unsafe index path")
        seen.add(identity)
        if (item["kind"] not in KINDS or type(item["size"]) is not int
                or not 0 < item["size"] <= MAX_BYTES
                or not isinstance(item["sha256"], str) or not HASH.fullmatch(item["sha256"])
                or not isinstance(item["fingerprint"], str) or not HASH.fullmatch(item["fingerprint"])):
            raise PackageError("Invalid index identity, size or checksum")
        references(item["dependencies"])
        references(item["members"])
        if ((item["kind"] != "collection" and item["members"])
                or (item["kind"] == "collection" and not item["members"])
                or not isinstance(item["platforms"], list) or not item["platforms"]
                or any(p not in {"all", "windows", "linux"} for p in item["platforms"])):
            raise PackageError("Invalid index members/platforms")
    return entries


def encode_index(entries):
    content = json.dumps(dict(schema=1, packages=sorted(entries, key=lambda e: (e["name"], e["version"]))),
                         ensure_ascii=False, indent=2).encode("utf-8")
    decode_index(content)
    return content


def select(entries, name, version=None):
    identifier(name)
    candidates = [item for item in entries if item["name"] == name and (version is None or item["version"] == version)]
    if version is None:
        candidates = [item for item in candidates if version_key(item["version"]) is not None]
        if candidates:
            best = max(version_key(item["version"]) for item in candidates)
            candidates = [item for item in candidates if version_key(item["version"]) == best]
    if len(candidates) != 1:
        raise PackageError(f"Package missing or ambiguous: {name} {version or 'latest'}")
    return candidates[0]


def command(args):
    executable = shutil.which(args[0])
    if not executable:
        raise PackageError(f"Required tool unavailable: {args[0]}; install it explicitly")
    result = subprocess.run([executable, *args[1:]], capture_output=True, check=False)
    if result.returncode:
        raise PackageError(f"{args[0]} failed ({result.returncode}): {result.stderr.decode('utf-8', 'replace')[-2000:]}")
    return result.stdout


class Repo:
    def __init__(self, address):
        if not isinstance(address, str) or not address.strip():
            raise PackageError("Configure [repo].url or pass --repo")
        self.address = address.rstrip("/")
        if re.match(r"^[A-Za-z]:[\\/]", address) or address.startswith(("\\\\", "//")):
            self.kind = "local"
        else:
            scheme = urlsplit(address).scheme.lower()
            if scheme and scheme not in {"http", "https", "s3", "ssh"}:
                raise PackageError(f"Unsupported repository scheme: {scheme}")
            self.kind = scheme or "local"
        self.root = Path(address).expanduser().resolve() if self.kind == "local" else None
        self.url = urlsplit(address) if self.kind != "local" else None
        if self.url and (not self.url.hostname or self.url.query or self.url.fragment or self.url.password):
            raise PackageError("Invalid repository URL")
        if self.kind == "ssh":
            host = self.url.hostname
            user = self.url.username
            if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]*", host) or (user and not re.fullmatch(r"[A-Za-z0-9_][A-Za-z0-9._-]*", user)):
                raise PackageError("Unsafe SSH host/user")
            if not self.url.path.startswith("/") or self.url.path == "/":
                raise PackageError("SSH repository requires an absolute directory path")
            self.host = (user + "@" if user else "") + host

    def ssh(self, arguments):
        options = ["-p", str(self.url.port)] if self.url.port else []
        return command(["ssh", *options, self.host, shlex.join(arguments)])

    def read(self, path, limit):
        # Only callers supply canonical package paths or the fixed index name.
        if self.kind == "local":
            target = safe_path(self.root, path)
            with target.open("rb") as stream:
                content = stream.read(limit + 1)
        elif self.kind in {"http", "https"}:
            with urlopen(self.address + "/" + quote(path, safe="/"), timeout=30) as stream:
                content = stream.read(limit + 1)
        elif self.kind == "s3":
            with tempfile.TemporaryDirectory(prefix="aiw-s3-read-") as temporary:
                target = Path(temporary) / "object"
                key = self.url.path.strip("/") + "/" + path if self.url.path.strip("/") else path
                command(["aws", "s3api", "get-object", "--bucket", self.url.netloc, "--key", key,
                         "--range", f"bytes=0-{limit}", str(target)])
                content = target.read_bytes()
        else:
            remote = self.url.path.rstrip("/") + "/" + path
            script = "import sys; f=open(sys.argv[1], 'rb'); sys.stdout.buffer.write(f.read(int(sys.argv[2])+1))"
            content = self.ssh(["python3", "-c", script, remote, str(limit)])
        if len(content) > limit:
            raise PackageError(f"Repository object exceeds size limit: {path}")
        return content

    def index(self):
        return decode_index(self.read("index.json", INDEX_LIMIT))

    def fetch(self, entry, folder):
        content = self.read(entry["path"], entry["size"])
        if len(content) != entry["size"] or digest(content) != entry["sha256"]:
            raise PackageError(f"Repository size/SHA-256 mismatch: {entry['path']}")
        target = safe_path(folder, entry["path"])
        atomic_bytes(target, content)
        if entry_for(target) != entry:
            raise PackageError(f"Package identity/metadata mismatch: {entry['path']}")
        return target

    def publish(self, package):
        if self.kind == "local":
            return local_publish(self.root, package)
        if self.kind in {"http", "https"}:
            raise PackageError("HTTP publish is not implemented; configure a local/S3/SSH publish_url")
        if self.kind == "s3":
            return s3_write(self, package=package)
        return ssh_write(self, package=package)

    def unpublish(self, name, version):
        package_path(name, version)
        if self.kind == "local":
            return local_unpublish(self.root, name, version)
        if self.kind in {"http", "https"}:
            raise PackageError("HTTP unpublish is not implemented; configure a local/S3/SSH publish_url")
        if self.kind == "s3":
            return s3_write(self, name=name, version=version)
        return ssh_write(self, name=name, version=version)


@contextmanager
def repository_lock(root):
    root.mkdir(parents=True, exist_ok=True)
    lock = safe_path(root, ".aiw-repository-lock")
    try:
        lock.mkdir()
    except FileExistsError as error:
        raise PackageError(f"Repository locked; inspect {lock}; no automatic takeover") from error
    try:
        yield
    finally:
        lock.rmdir()


def local_entries(root):
    index = safe_path(root, "index.json")
    return decode_index(index.read_bytes()) if index.exists() else []


def local_publish(root, package):
    entry = entry_for(package)
    with repository_lock(root):
        entries = local_entries(root)
        existing = [e for e in entries if (e["name"], e["version"]) == (entry["name"], entry["version"])]
        target = safe_path(root, entry["path"])
        if existing:
            if not target.is_file() or entry_for(target) != existing[0]:
                raise PackageError(f"Existing repository object is damaged: {target}")
            if existing[0]["fingerprint"] != entry["fingerprint"]:
                raise PackageError("Same name/version has different content; use a new version")
            return f"Already exists: {entry['name']} {entry['version']}"
        if target.exists():
            raise PackageError(f"Unindexed object exists; inspect/reindex before publishing: {target}")
        content = package.read_bytes()
        # Pin the bytes validated above even if the source changes during publication.
        if digest(content) != entry["sha256"]:
            raise PackageError("Package changed during publication")
        atomic_bytes(target, content)
        try:
            atomic_bytes(safe_path(root, "index.json"), encode_index([*entries, entry]))
        except Exception:
            committed = any(e == entry for e in local_entries(root))
            if not committed:
                target.unlink()
            raise
    return f"Published: {entry['name']} {entry['version']}"


def local_unpublish(root, name, version):
    with repository_lock(root):
        entries = local_entries(root)
        entry = select(entries, name, version)
        atomic_bytes(safe_path(root, "index.json"), encode_index([e for e in entries if e != entry]))
        try:
            safe_path(root, entry["path"]).unlink(missing_ok=True)
        except OSError as error:
            raise PackageError(f"Unpublished, but object cleanup failed at {entry['path']}: {error}") from error
    return f"Unpublished: {name} {version}"


def index_repository(root):
    root = root.expanduser().resolve()
    with repository_lock(root):
        entries = []
        for package in sorted(root.glob("*/*/*.zip")):
            entry = entry_for(package)
            if package.relative_to(root).as_posix() != entry["path"]:
                raise PackageError(f"Noncanonical repository object: {package}")
            entries.append(entry)
        atomic_bytes(safe_path(root, "index.json"), encode_index(entries))
    return f"Indexed: {len(entries)} packages"


def s3_key(repo, path):
    return "/".join(p for p in (repo.url.path.strip("/"), path) if p)


def s3_exists(repo, path):
    key = s3_key(repo, path)
    content = command(["aws", "s3api", "list-objects-v2", "--bucket", repo.url.netloc,
                       "--prefix", key, "--max-keys", "2", "--output", "json"])
    return any(item["Key"] == key for item in json.loads(content).get("Contents", []))


def s3_write(repo, *, package=None, name=None, version=None):
    with tempfile.TemporaryDirectory(prefix="aiw-s3-write-") as temporary:
        root = Path(temporary)
        owner = root / "lock-owner"
        owner.write_text(uuid.uuid4().hex, encoding="ascii")
        lock = s3_key(repo, ".aiw-repository-lock")
        command(["aws", "s3api", "put-object", "--bucket", repo.url.netloc, "--key", lock,
                 "--body", str(owner), "--if-none-match", "*"])
        try:
            entries = repo.index() if s3_exists(repo, "index.json") else []
            atomic_bytes(root / "index.json", encode_index(entries))
            if package is not None:
                proposed = entry_for(package)
                path = proposed["path"]
                existing = [e for e in entries if e["name"] == proposed["name"] and e["version"] == proposed["version"]]
                if existing:
                    repo.fetch(existing[0], root)
                elif s3_exists(repo, path):
                    raise PackageError(f"Unindexed S3 object exists; inspect {path}")
                result = local_publish(root, package)
                if result.startswith("Already exists:"):
                    return result
                command(["aws", "s3", "cp", str(safe_path(root, path)), repo.address + "/" + path, "--only-show-errors"])
                try:
                    command(["aws", "s3", "cp", str(root / "index.json"), repo.address + "/index.json", "--only-show-errors"])
                except PackageError as error:
                    # A failed response may follow a successful remote commit. Re-read before deleting.
                    try:
                        committed = any(e == proposed for e in repo.index())
                        if not committed:
                            command(["aws", "s3", "rm", repo.address + "/" + path, "--only-show-errors"])
                    except Exception as cleanup:
                        raise PackageError(f"S3 index outcome/cleanup uncertain for {path}: {cleanup}; original: {error}") from error
                    raise PackageError(f"S3 index upload reported failure; committed={committed}; {error}") from error
                return result
            entry = select(entries, name, version)
            atomic_bytes(root / "index.json", encode_index([e for e in entries if e != entry]))
            command(["aws", "s3", "cp", str(root / "index.json"), repo.address + "/index.json", "--only-show-errors"])
            try:
                command(["aws", "s3", "rm", repo.address + "/" + entry["path"], "--only-show-errors"])
            except PackageError as error:
                raise PackageError(f"Unpublished; S3 object cleanup incomplete: {entry['path']}: {error}") from error
            return f"Unpublished: {name} {version}"
        finally:
            command(["aws", "s3api", "delete-object", "--bucket", repo.url.netloc, "--key", lock])


def ssh_write(repo, *, package=None, name=None, version=None):
    root = repo.url.path.rstrip("/")
    # Remote operators install archive.py and repository.py here explicitly.
    runner = root + "/.aiw-repo/repository.py"
    if package is None:
        return repo.ssh(["python3", runner, "--root", root, "unpublish", identifier(name), identifier(version)]).decode("utf-8")
    entry = entry_for(package)
    staging = root + "/.aiw-upload-" + uuid.uuid4().hex
    remote = staging + "/" + package.name
    repo.ssh(["python3", "-c", "import os,sys; os.mkdir(sys.argv[1])", staging])
    try:
        options = ["-P", str(repo.url.port)] if repo.url.port else []
        command(["scp", *options, "--", str(package.resolve()), repo.host + ":" + shlex.quote(remote)])
        return repo.ssh(["python3", runner, "--root", root, "publish", remote]).decode("utf-8")
    finally:
        repo.ssh(["python3", "-c", "import os,sys; p=sys.argv[1]; os.path.exists(p) and os.unlink(p); os.rmdir(sys.argv[2])", remote, staging])


def configured_repos(explicit=None):
    config, base = {}, Path.cwd()
    candidates = []
    for directory in (Path.cwd(), *Path.cwd().parents):
        matches = [directory / name for name in (".aiw.toml", "aiw.toml") if (directory / name).is_file()]
        if matches:
            candidates.append(matches[0])
            break
    installed = Path(os.environ.get("AIW_ROOT", Path(__file__).resolve().parents[2])) / "aiw.toml"
    candidates.append(installed)
    for candidate in candidates:
        if candidate.is_file():
            if candidate.stat().st_size > INDEX_LIMIT:
                raise PackageError(f"Config too large: {candidate}")
            data = tomllib.loads(candidate.read_text(encoding="utf-8"))
            if "repo" in data:
                config, base = data["repo"], candidate.parent
                break

    def address(value, origin):
        if not isinstance(value, str):
            raise PackageError("Repository address must be a string")
        if not urlsplit(value).scheme or re.match(r"^[A-Za-z]:[\\/]", value):
            return str((origin / Path(value).expanduser()).resolve())
        return value

    read = address(explicit, Path.cwd()) if explicit else address(config.get("url", ""), base)
    if not explicit and not config.get("url"):
        raise PackageError("Configure [repo].url or pass --repo")
    write = address(config["publish_url"], base) if config.get("publish_url") and not explicit else read
    return Repo(read), Repo(write)


if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser(description="Explicit remote filesystem repository writer")
    parser.add_argument("--root", required=True, type=Path)
    sub = parser.add_subparsers(dest="action", required=True)
    sub.add_parser("publish").add_argument("package", type=Path)
    remove = sub.add_parser("unpublish")
    remove.add_argument("name")
    remove.add_argument("version")
    args = parser.parse_args()
    try:
        if args.action == "publish":
            print(local_publish(args.root.resolve(), args.package))
        else:
            print(local_unpublish(args.root.resolve(), args.name, args.version))
    except (PackageError, OSError, ValueError) as error:
        parser.exit(1, f"repository: {error}\n")
