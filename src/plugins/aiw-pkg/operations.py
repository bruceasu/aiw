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

    def install(self, package=None, *, packages=None, root_requests=None, prepared_package=None,
                name=None, version=None, update=False, scope=None, dry_run=False):
        if packages is not None and (package is not None or name is not None or update):
            raise PackageError("Multiple package roots cannot be combined with another selection")
        if root_requests is not None and (package is not None or packages is not None or name is not None or update):
            raise PackageError("Index roots cannot be combined with another selection")
        if prepared_package is not None and package is None:
            raise PackageError("Prepared input requires a single local package")
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
            old = self.manager.records(inspect_targets=False)

            def from_entry(entry):
                current = old.get(entry["name"])
                if (current and current["version"] == entry["version"]
                        and current.get("fingerprint") == entry["fingerprint"]):
                    # The index supplies the complete graph even if a previous
                    # collection update retained only some installed members.
                    info = dict(entry, files=current["files"])
                    return None, (info, None, None, entry["fingerprint"])
                prepare = getattr(self.repo, "prepare", None)
                if prepare is not None:
                    return prepare(entry, folder)
                path = self.repo.fetch(entry, folder)
                return path, read_package(path)

            # Root selection is complete before dependency expansion: conflicts never pick an arbitrary winner.
            requests = []
            if package is not None or packages is not None:
                root_packages = [package] if package is not None else packages
                if not root_packages:
                    raise PackageError("No installation roots selected")
                for root_package in root_packages:
                    prepared = prepared_package if prepared_package is not None else read_package(root_package)
                    info = prepared[0]
                    previous = selected.get(info["name"])
                    if previous and previous[1][3] != prepared[3]:
                        raise PackageError(f"Conflicting installation roots: {info['name']}")
                    selected[info["name"]] = (root_package, prepared)
                    roots.add(info["name"])
            elif root_requests is not None:
                requests = list(root_requests)
                if not requests:
                    raise PackageError("No installation roots selected")
            elif update:
                requests = [("aiw-" + scope, None)] if scope else [(n, None) for n in sorted(old)]
            else:
                requests = [(name, version)]
            for requested, wanted in requests:
                entry = select(catalog(), requested, wanted)
                previous = selected.get(requested)
                if previous and (previous[1][0]["version"] != entry["version"]
                                 or previous[1][3] != entry["fingerprint"]):
                    raise PackageError(f"Conflicting installation roots: {requested}")
                if previous is None:
                    selected[requested] = from_entry(entry)
                roots.add(requested)

            actual_members = {}

            def visit(package_name, wanted=None):
                if package_name not in selected:
                    entry = select(catalog(), package_name, wanted)
                    selected[package_name] = from_entry(entry)
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
            if sum(path.stat().st_size for path, _ in selected.values() if path is not None) > MAX_BYTES:
                raise PackageError("Combined operation exceeds package staging limit")

            def preflight():
                current = self.manager.records(inspect_targets=False)
                if current != old:
                    raise PackageError("Installation state changed during planning; retry")
                final = dict(current)
                system = host_platform()
                targets = {}
                for package_name, (path, prepared) in selected.items():
                    info, _, _, fingerprint = prepared
                    if path is None:
                        restricted = set(info["platforms"]) - {"all"}
                        if restricted and system not in restricted:
                            raise PackageError(f"No {system} payload: {package_name}")
                        final[package_name] = dict(current[package_name],
                                                   dependencies=info["dependencies"],
                                                   members=actual_members[package_name])
                        continue
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
                    self.manager.preflight(package_name, files, current.get(package_name),
                                           adopt_existing=True, check_modified=False)
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
                        if path is None:
                            record = self.manager.record(package_name, inspect_targets=False)
                            if record != old[package_name]:
                                raise PackageError(f"Installation record changed during operation: {package_name}")
                            retained = dict(record,
                                            reason="manual" if reason == "manual" or record.get("reason", "manual") == "manual" else "auto",
                                            dependencies=info["dependencies"],
                                            members=actual_members[package_name])
                            if retained != record:
                                self.manager.save_record(self.manager.package_dir(package_name), retained)
                            results.append(f"Already installed: {package_name} {info['version']}")
                        else:
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
