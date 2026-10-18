"""Preflight a complete package graph before applying per-package transactions."""
from __future__ import annotations

from contextlib import nullcontext
from pathlib import Path
import tempfile

from archive import MAX_BYTES, PackageError, read_package
from manager import host_platform
from repository import select


class Operation:
    def __init__(self, manager, repo=None):
        self.manager, self.repo = manager, repo

    def install(self, package=None, *, name=None, version=None, update=False, scope=None, dry_run=False):
        # Pin source files, imports and all archive bytes before self-update can replace the engine.
        with tempfile.TemporaryDirectory(prefix="aiw-plan-") as temporary:
            folder = Path(temporary)
            entries = None

            def catalog():
                nonlocal entries
                if entries is None:
                    if self.repo is None:
                        raise PackageError("Package dependencies require a configured repository")
                    entries = self.repo.index()
                return entries

            selected, roots, order, visiting, done = {}, set(), [], set(), set()
            old = self.manager.records()
            # Root selection is complete before dependency expansion: conflicts never pick an arbitrary winner.
            requests = []
            if package is not None:
                prepared = read_package(package)
                info = prepared[0]
                selected[info["name"]] = (package, prepared)
                roots.add(info["name"])
            elif update:
                requests = [("aiw-" + scope, None)] if scope else [(n, None) for n in sorted(old)]
            else:
                requests = [(name, version)]
            for requested, wanted in requests:
                entry = select(catalog(), requested, wanted)
                path = self.repo.fetch(entry, folder)
                selected[requested] = (path, read_package(path))
                roots.add(requested)

            actual_members = {}

            def visit(package_name, wanted=None):
                if package_name not in selected:
                    entry = select(catalog(), package_name, wanted)
                    path = self.repo.fetch(entry, folder)
                    selected[package_name] = (path, read_package(path))
                info = selected[package_name][1][0]
                if wanted is not None and info["version"] != wanted:
                    raise PackageError(f"Exact version conflict: {package_name} needs {wanted}, selected {info['version']}")
                if package_name in visiting:
                    raise PackageError(f"Dependency/member cycle: {package_name}")
                if package_name in done:
                    return
                visiting.add(package_name)
                members = info.get("members", [])
                if update and scope is None and package_name in old and info["kind"] == "collection":
                    previous = {r["name"] for r in old[package_name].get("members", [])}
                    members = [r for r in members if r["name"] in previous and r["name"] in old]
                actual_members[package_name] = members
                for ref in info.get("dependencies", []) + members:
                    visit(ref["name"], ref["version"])
                visiting.remove(package_name)
                done.add(package_name)
                order.append(package_name)

            for root in sorted(roots):
                visit(root)
            if sum(path.stat().st_size for path, _ in selected.values()) > MAX_BYTES:
                raise PackageError("Combined operation exceeds package staging limit")

            def preflight():
                current = self.manager.records()
                if current != old:
                    raise PackageError("Installation state changed during planning; retry")
                final = dict(current)
                system = host_platform()
                targets = {}
                for package_name, (path, prepared) in selected.items():
                    info, _, _, fingerprint = prepared
                    restricted = {f["platform"] for f in info["files"] if f["platform"] != "all"}
                    if restricted and system not in restricted:
                        raise PackageError(f"No {system} payload: {package_name}")
                    files = [f for f in info["files"] if f["platform"] in {"all", system}]
                    if not files and info["kind"] != "collection":
                        raise PackageError(f"No platform payload: {package_name}")
                    for item in files:
                        if self.manager.target(item["target"]).resolve(strict=False) == path.resolve(strict=True):
                            raise PackageError("Package would overwrite its input ZIP")
                        key = item["target"].casefold()
                        if any(key == p or key.startswith(p + "/") or p.startswith(key + "/") for p in targets):
                            raise PackageError(f"Packages conflict at target: {item['target']}")
                        targets[key] = package_name
                    self.manager.preflight(package_name, files, current.get(package_name), adopt_existing=True)
                    final[package_name] = dict(info, fingerprint=fingerprint, files=files,
                                               members=actual_members[package_name], status="installed")
                visited, active = set(), set()

                def check(package_name):
                    if package_name in active:
                        raise PackageError(f"Final dependency cycle: {package_name}")
                    if package_name in visited:
                        return
                    active.add(package_name)
                    record = final[package_name]
                    for ref in record.get("dependencies", []) + record.get("members", []):
                        dependency = final.get(ref["name"])
                        if not dependency or dependency["version"] != ref["version"]:
                            raise PackageError(f"Final state violates {package_name}: needs {ref['name']} {ref['version']}")
                        check(ref["name"])
                    active.remove(package_name)
                    visited.add(package_name)

                for package_name in final:
                    check(package_name)

            with nullcontext() if dry_run else self.manager.locked():
                preflight()
                results = []
                for package_name in order:
                    path, prepared = selected[package_name]
                    info = prepared[0]
                    reason = old.get(package_name, {}).get("reason", "manual" if package_name in old else "auto")
                    if (not update and package_name in roots) or (scope and package_name in roots):
                        reason = "manual"
                    prefix = "Update" if package_name in old else "Install"
                    print(f"{prefix}: {package_name} {info['version']} ({reason}; {info['kind']})")
                    if dry_run:
                        continue
                    try:
                        results.append(self.manager.install(path, prepared=prepared, reason=reason,
                                                            members=actual_members[package_name]))
                    except Exception as error:
                        raise PackageError(f"Stopped at {package_name}; completed packages: {', '.join(order[:len(results)]) or 'none'}. {error}") from error
                return "Dry run: no installation changes" if dry_run else "\n".join(results) or "No installed packages to update"

    def autoremove(self, dry_run=False):
        with nullcontext() if dry_run else self.manager.locked():
            names = self.manager.orphans()
            if dry_run:
                return "Orphan packages (dry run): " + (", ".join(names) or "none")
            results = [self.manager.remove(name, removing=names) for name in names]
            return "\n".join(results) or "No orphan packages"
