"""Resolve build-local package graphs from original ZIPs, without a repository copy."""
from pathlib import Path

from archive import PackageError, identifier, read_package, safe_path
from manager import host_platform
from repository import entry_for


class LocalPackages:
    CATEGORIES = ("mist", "plugins", "skills", "resources", "collections")

    def __init__(self, root: Path, packages: list[Path]):
        self.root = root
        self._entries, self._paths, self._files = {}, {}, {}
        self._sources = set()
        self.roots = []
        for path in packages:
            entry = self._add(path)
            if entry not in self.roots:
                self.roots.append(entry)
        pending = list(self._entries.values())
        while pending:
            entry = pending.pop()
            for ref in entry["dependencies"] + entry["members"]:
                key = (ref["name"], ref["version"])
                if key not in self._entries:
                    pending.append(self._add(self._find(*key)))

    def _source(self, path: Path) -> Path:
        try:
            relative = path.relative_to(self.root).as_posix()
        except ValueError as error:
            raise PackageError(f"Package source is outside dist: {path}") from error
        return safe_path(self.root, relative)

    def _add(self, path: Path) -> dict:
        path = self._source(path)
        entry = entry_for(path)
        self._sources.add(path)
        key = (entry["name"], entry["version"])
        if key in self._entries:
            if self._entries[key] != entry:
                raise PackageError(f"Conflicting local packages: {key[0]} {key[1]}")
            return self._entries[key]
        self._entries[key], self._paths[key] = entry, path
        self._files[key] = read_package(path)[0]["files"]
        return entry

    def _find(self, name: str, version: str) -> Path:
        identifier(name)
        identifier(version)
        candidates = [safe_path(self.root, f"{category}/{version}/{name}-{version}.zip")
                      for category in self.CATEGORIES]
        candidates = [path for path in candidates if path.is_file()]
        if len(candidates) != 1:
            raise PackageError(f"Expected one local ZIP for {name} {version}, found {len(candidates)}; rebuild the target")
        path = candidates[0]
        entry = entry_for(path)
        if (entry["name"], entry["version"]) != (name, version):
            raise PackageError(f"Local dependency identity mismatch: {path}")
        return path

    def index(self) -> list[dict]:
        return list(self._entries.values())

    def fetch(self, entry: dict, folder) -> Path:
        # Operation owns extraction/transaction staging; the original input ZIP stays in dist.
        key = (entry["name"], entry["version"])
        if key not in self._entries or entry != self._entries[key]:
            raise PackageError("Unknown local package entry")
        path = self._source(self._paths[key])
        if entry_for(path) != entry:
            raise PackageError(f"Local package changed after selection: {path}")
        return path

    def preflight_destination(self, destination: Path) -> None:
        root = destination.expanduser().resolve(strict=False)
        sources = {self._source(path).resolve(strict=True) for path in self._sources}
        system = host_platform()
        for files in self._files.values():
            for item in files:
                if item["platform"] not in {"all", system}:
                    continue
                target = safe_path(root, item["target"]).resolve(strict=False)
                if any(target == source or target in source.parents for source in sources):
                    raise PackageError(f"Installation would overwrite an input ZIP or its parent: {target}")
