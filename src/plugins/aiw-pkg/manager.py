"""Package transactions shared by build.py and aiw pkg; no historical rollback."""
from __future__ import annotations

from contextlib import contextmanager
import json
import os
from pathlib import Path
import platform
import shutil
import stat
import subprocess
import sys
import tempfile
import uuid

from archive import (PackageError, atomic_bytes, digest, identifier, read_package,
                     references, relative, reject_link, safe_path)


def default_destination() -> Path:
    value = os.environ.get("AIW_INSTALL_DIR")
    return Path(value).expanduser() if value else (
        Path(r"C:\green\aiw") if os.name == "nt" else Path.home() / "apps" / "aiw")


def host_platform() -> str:
    system = platform.system().lower()
    if system not in {"windows", "linux"} or platform.machine().lower() not in {"amd64", "x86_64"}:
        raise PackageError("Installation supports Windows/Linux amd64 only")
    return system


def read_json(path: Path):
    reject_link(path)
    if path.stat().st_size > 16 * 1024 * 1024:
        raise PackageError(f"State file too large: {path}")
    return json.loads(path.read_text(encoding="utf-8"))


def write_json(path: Path, value) -> None:
    atomic_bytes(path, json.dumps(value, ensure_ascii=False, indent=2).encode("utf-8"))


class Manager:
    def __init__(self, destination: Path):
        # The explicit destination may use a user-selected junction/symlink alias.
        # Pin its physical root once; safe_path still rejects links beneath it.
        try:
            self.root = destination.expanduser().resolve(strict=False)
        except RuntimeError as error:
            raise PackageError(f"Cannot resolve installation root: {destination}: {error}") from error
        self.state = safe_path(self.root, ".aiw/packages")
        self._lock_held = False

    def target(self, value: str) -> Path:
        if value.casefold() == ".aiw" or value.casefold().startswith(".aiw/"):
            raise PackageError("Payload target overlaps engine state")
        return safe_path(self.root, value)

    def package_dir(self, name: str) -> Path:
        return safe_path(self.state, identifier(name))

    @contextmanager
    def locked(self):
        if self._lock_held:
            yield
            return
        safe_path(self.root, ".aiw/packages")
        self.state.mkdir(parents=True, exist_ok=True)
        lock = safe_path(self.state, ".lock")
        try:
            handle = os.open(lock, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
        except FileExistsError as error:
            raise PackageError(f"Installation is locked; inspect {lock}. No automatic takeover.") from error
        try:
            with os.fdopen(handle, "w") as stream:
                stream.write(f"pid={os.getpid()}\n")
            # An incomplete transaction in any package blocks this installation root.
            for directory in self.state.iterdir():
                reject_link(directory)
                if directory.is_dir() and (directory / "transaction").exists():
                    raise PackageError(f"Incomplete transaction; preserve and inspect {directory / 'transaction'}")
            self._lock_held = True
            yield
        finally:
            self._lock_held = False
            lock.unlink()

    def record(self, name: str) -> dict | None:
        path = safe_path(self.package_dir(name), "current.json")
        if not path.exists():
            return None
        record = read_json(path)
        if safe_path(self.package_dir(name), "previous").exists():
            raise PackageError(f"Legacy backup state for {name}; reinstall into a clean destination; no automatic migration")
        if (record.get("name") != name or record.get("status") not in {"installed", "removed"}
                or not isinstance(record.get("files"), list)):
            raise PackageError(f"Invalid installation record: {path}")
        for item in record["files"]:
            self.target(item["target"])
            if type(item["config"]) is not bool:
                raise PackageError(f"Invalid configuration record: {path}")
        references(record.get("dependencies", []))
        references(record.get("members", []))
        if record.get("reason", "manual") not in {"manual", "auto"}:
            raise PackageError(f"Invalid installation reason: {path}")
        return record

    def records(self, *, removed=False):
        result = {}
        if self.state.exists():
            for directory in self.state.iterdir():
                reject_link(directory)
                if directory.is_dir():
                    record = self.record(directory.name)
                    if record and (removed or record["status"] == "installed"):
                        result[directory.name] = record
        return result

    def mark_manual(self, name):
        with self.locked():
            record = self.record(name)
            if not record or record["status"] != "installed":
                raise PackageError(f"Package not installed: {name}")
            self.save_record(self.package_dir(name), dict(record, reason="manual"))
        return f"Marked manual: {name}"

    def orphans(self):
        records = self.records()
        needed = set()

        def visit(name):
            if name in needed or name not in records:
                return
            needed.add(name)
            for ref in records[name].get("dependencies", []) + records[name].get("members", []):
                visit(ref["name"])

        for name, record in records.items():
            if record.get("reason", "manual") == "manual":
                visit(name)
        return sorted(set(records) - needed)

    def owners(self, name: str) -> dict[str, str]:
        result = {}
        if not self.state.exists():
            return result
        for directory in self.state.iterdir():
            reject_link(directory)
            if not directory.is_dir() or directory.name == name:
                continue
            other = self.record(directory.name)
            if other:
                for item in other["files"]:
                    result[item["target"].casefold()] = directory.name
        return result

    def preflight(self, name: str, files: list[dict], current: dict | None, *, adopt_existing=False) -> None:
        owners = self.owners(name)
        old = {item["target"].casefold(): item for item in (current or {}).get("files", [])}
        for item in files:
            value = item["target"]
            key = value.casefold()
            path = self.target(value)
            if any(key == p or key.startswith(p + "/") or p.startswith(key + "/") for p in owners):
                raise PackageError(f"Target owned by another package: {value}")
            if key in old and old[key]["config"] != item["config"]:
                raise PackageError(f"Cannot change configuration classification: {value}")
            if path.exists():
                if not path.is_file():
                    raise PackageError(f"Target is not a regular file: {path}")
                if key not in old and not item["config"] and not adopt_existing:
                    raise PackageError(f"Refusing to overwrite unowned file during restoration: {path}")
                # Only installation adopts ordinary files through its transaction snapshot.
            for parent in path.parents:
                if parent == self.root:
                    break
                if parent.exists() and not parent.is_dir():
                    raise PackageError(f"Target parent is a file: {parent}")
        self.unchanged(current)

    def unchanged(self, record: dict | None) -> None:
        for item in (record or {}).get("files", []):
            path = self.target(item["target"])
            if not item["config"] and path.exists():
                if not path.is_file() or digest(path.read_bytes()) != item["sha256"]:
                    raise PackageError(f"Managed file was modified; preserve it before continuing: {path}")

    def snapshot(self, folder: Path, current: dict | None, files: list[dict]) -> None:
        folder.mkdir(parents=True)
        seen, saved = set(), []
        for item in files:
            value = item["target"]
            if item["config"] or value in seen:
                continue
            seen.add(value)
            source = self.target(value)
            entry = {"target": value, "exists": source.exists()}
            if source.exists():
                if not source.is_file():
                    raise PackageError(f"Cannot snapshot non-file: {source}")
                stored = digest(value.encode())
                shutil.copy2(source, folder / stored)
                entry.update(stored=stored, sha256=digest(source.read_bytes()), mode=stat.S_IMODE(source.stat().st_mode))
            saved.append(entry)
        write_json(folder / "snapshot.json", {"record": current, "files": saved})
        if current and current["status"] == "installed":
            package = self.package_dir(current["name"]) / "current.zip"
            reject_link(package)
            if package.is_file():
                shutil.copy2(package, folder / "package.zip")

    def restore(self, folder: Path) -> dict | None:
        snapshot = read_json(folder / "snapshot.json")
        # Verify every snapshot file before changing any live file.
        for item in snapshot["files"]:
            self.target(item["target"])
            if item["exists"]:
                source = safe_path(folder, item["stored"])
                if digest(source.read_bytes()) != item["sha256"]:
                    raise PackageError(f"Damaged recovery snapshot: {source}")
        for item in snapshot["files"]:
            path = self.target(item["target"])
            if item["exists"]:
                atomic_bytes(path, (folder / item["stored"]).read_bytes())
                path.chmod(item["mode"])
            else:
                path.unlink(missing_ok=True)
        directory = folder.parent.parent if folder.name == "before" else folder.parent
        saved_package = folder / "package.zip"
        current_package = safe_path(directory, "current.zip")
        if saved_package.is_file():
            reject_link(saved_package)
            atomic_bytes(current_package, saved_package.read_bytes())
        else:
            current_package.unlink(missing_ok=True)
        return snapshot["record"]

    def save_record(self, directory: Path, record: dict | None) -> None:
        path = safe_path(directory, "current.json")
        if record is None:
            path.unlink(missing_ok=True)
        else:
            write_json(path, record)

    @staticmethod
    def retained_configs(record: dict | None) -> list[dict]:
        return [item for item in (record or {}).get("files", []) if item["config"]]

    def merged_configs(self, record: dict | None, candidates: list[dict], name: str) -> dict | None:
        configs = {item["target"].casefold(): item for item in self.retained_configs(record)}
        configs.update({item["target"].casefold(): item for item in candidates
                        if item["config"] and self.target(item["target"]).is_file()})
        if record is None and configs:
            record = {"name": name, "status": "removed", "files": []}
        if record is not None:
            record = dict(record)
            record["files"] = [item for item in record["files"] if not item["config"]] + list(configs.values())
        return record

    def failed(self, directory: Path, transaction: Path, error: Exception, configs: list[dict]) -> None:
        try:
            restored = self.restore(transaction / "before")
            restored = self.merged_configs(restored, configs, directory.name)
            self.save_record(directory, restored)
            write_json(transaction / "failure.json", {"error": str(error), "recovered": True})
            diagnostic = directory / ("failed-" + uuid.uuid4().hex)
            os.replace(transaction, diagnostic)
        except Exception as recovery:
            raise PackageError(f"Operation failed: {error}; recovery incomplete: {recovery}. Preserve {transaction}. "
                               "Exit related programs before recovery; do not start another installation while recovery is incomplete. "
                               f'Independent main-program entry: python "{self.root / "plugins/aiw-pkg/aiw-pkg.py"}" install <ZIP> --dest "{self.root}"') from error
        raise PackageError(f"Operation failed and managed files restored: {error}. Diagnostics: {diagnostic}. "
                           "If a file is in use, exit related programs. Retry main-program installation with "
                           f'python "{self.root / "plugins/aiw-pkg/aiw-pkg.py"}" install <ZIP> --dest "{self.root}"') from error

    def hook(self, name: str, scripts: dict[str, bytes], stage: Path, info: dict,
             action: str, current: dict | None, transaction: Path) -> None:
        if name not in scripts:
            return
        script = stage / name
        script.write_bytes(scripts[name])
        environment = os.environ.copy()
        environment.update(AIW_PKG_STAGE=str(stage / "data"), AIW_INSTALL_DIR=str(self.root),
                           AIW_PKG_ACTION=action)
        with (transaction / (name + ".log")).open("wb") as log:
            result = subprocess.run([sys.executable, str(script), info["name"], info["version"],
                                     action, (current or {}).get("version", ""), str(self.root)],
                                    cwd=stage, env=environment, stdout=log, stderr=subprocess.STDOUT,
                                    check=False)
        if result.returncode:
            raise PackageError(f"{name} exited {result.returncode}; see {transaction / (name + '.log')}")

    def install(self, package: Path, *, prepared=None, reason="manual", members=None) -> str:
        info, payload, scripts, fingerprint = prepared or read_package(package)
        system = host_platform()
        restricted = {item["platform"] for item in info["files"] if item["platform"] != "all"}
        if restricted and system not in restricted:
            raise PackageError(f"Package has no platform payload for {system}")
        files = [dict(item) for item in info["files"] if item["platform"] in {"all", system}]
        if not files and info["kind"] != "collection":
            raise PackageError(f"No payload for {system}")
        input_package = package.resolve(strict=True)
        for item in files:
            if self.target(item["target"]).resolve(strict=False) == input_package:
                raise PackageError(f"Package target overlaps its local input ZIP: {item['target']}")
        if scripts and (sys.version_info < (3, 11) or not Path(sys.executable).is_file()):
            raise PackageError("Python 3.11+ interpreter required before installation")
        # Temporary materialization is outside the final destination and never owns the input ZIP.
        with tempfile.TemporaryDirectory(prefix="aiw-pkg-") as temporary:
            stage = Path(temporary)
            for source, content in payload.items():
                target = safe_path(stage / "data", source)
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(content)
            with self.locked():
                name = info["name"]
                directory = self.package_dir(name)
                directory.mkdir(exist_ok=True)
                current = self.record(name)
                if current and current["status"] == "installed" and current.get("version") == info["version"] and current.get("fingerprint") == fingerprint:
                    self.save_record(directory, dict(current,
                        reason="manual" if reason == "manual" or current.get("reason", "manual") == "manual" else "auto",
                        dependencies=info.get("dependencies", []),
                        members=info.get("members", []) if members is None else members))
                    return f"Already installed: {name} {info['version']}"
                self.preflight(name, files, current, adopt_existing=True)
                transaction = directory / "transaction"
                before_files = (current or {}).get("files", []) + files
                try:
                    self.snapshot(transaction / "before", current, before_files)
                except Exception:
                    if transaction.exists():
                        shutil.rmtree(transaction)
                    raise
                try:
                    action = "upgrade" if current and current["status"] == "installed" else "install"
                    self.hook("preinst", scripts, stage, info, action, current, transaction)
                    selected = {item["target"].casefold() for item in files}
                    for old in (current or {}).get("files", []):
                        if not old["config"] and old["target"].casefold() not in selected:
                            self.target(old["target"]).unlink(missing_ok=True)
                    for item in files:
                        path = self.target(item["target"])
                        if item["config"] and path.exists():
                            continue
                        atomic_bytes(path, payload[item["source"]])
                        path.chmod(0o755 if item["executable"] else 0o644)
                    self.hook("postinst", scripts, stage, info, action, current, transaction)
                    # Store actual installed bytes, including deliberate hook edits to managed files.
                    for item in files:
                        path = self.target(item["target"])
                        if not path.is_file():
                            raise PackageError(f"Managed file missing after hook: {path}")
                        item["sha256"] = digest(path.read_bytes())
                    record = dict(name=name, version=info["version"], status="installed",
                                  fingerprint=fingerprint, files=files, kind=info["kind"],
                                  dependencies=info.get("dependencies", []),
                                  members=info.get("members", []) if members is None else members,
                                  reason="manual" if reason == "manual" or (current and current.get("reason", "manual") == "manual") else "auto")
                    record = self.merged_configs(record, self.retained_configs(current), name)
                    safe_path(directory, "current.zip").unlink(missing_ok=True)
                    self.save_record(directory, record)
                    write_json(transaction / "complete.json", {"installed": fingerprint})
                except Exception as error:
                    self.failed(directory, transaction, error, files)
                try:
                    shutil.rmtree(transaction)
                except OSError as error:
                    raise PackageError(f"Installed {name}, but transaction cleanup failed: {transaction}: {error}") from error
                return f"Installed: {name} {info['version']} in {self.root}"

    def remove(self, name: str, *, purge: bool = False, removing=()) -> str:
        identifier(name)
        host_platform()
        with self.locked():
            directory = self.package_dir(name)
            current = self.record(name)
            if not current:
                raise PackageError(f"Package is not recorded: {name}")
            users = [other for other, record in self.records().items() if other != name and other not in removing
                     and any(ref["name"] == name for ref in record.get("dependencies", []) + record.get("members", []))]
            if users:
                raise PackageError(f"Package is required by: {', '.join(users)}")
            restored = None
            self.unchanged(current)
            affected = current["files"]
            # Purge includes configurations in rollback snapshots, but no other action overwrites them.
            snapshot_files = [dict(item, config=False) if purge else item for item in affected]
            transaction = directory / "transaction"
            try:
                self.snapshot(transaction / "before", current, snapshot_files)
            except Exception:
                if transaction.exists():
                    shutil.rmtree(transaction)
                raise
            try:
                for item in current["files"]:
                    if purge or not item["config"]:
                        self.target(item["target"]).unlink(missing_ok=True)
                safe_path(directory, "current.zip").unlink(missing_ok=True)
                if not purge:
                    restored = self.merged_configs(restored, self.retained_configs(current), name)
                self.save_record(directory, restored)
                write_json(transaction / "complete.json", {"removed": name, "purge": purge})
            except Exception as error:
                self.failed(directory, transaction, error, [] if purge else self.retained_configs(current))
            try:
                shutil.rmtree(transaction)
                if purge:
                    # Delete only this engine-owned package state, never the installation root.
                    shutil.rmtree(directory)
            except OSError as error:
                raise PackageError(f"Files removed; state cleanup incomplete at {directory}: {error}") from error
            return f"{'Purged' if purge else 'Removed'}: {name}"
