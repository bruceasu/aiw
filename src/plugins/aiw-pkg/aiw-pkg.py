#!/usr/bin/env python3
"""Unified AIW packages, repositories and managed Skill imports (Python 3.11+)."""
import argparse
import json
from pathlib import Path
import sys
import zipfile

from archive import PackageError, identifier, pack_workspace
from manager import Manager, default_destination
from operations import Operation
from repository import configured_repos, index_repository, select
import entries
import skill_management


def format_package_list(records, *, installed=False):
    if not records:
        return "No installed packages." if installed else "No packages found."
    columns = [("NAME", "name"), ("VERSION", "version"), ("KIND", "kind")]
    if installed:
        columns.append(("REASON", "reason"))
    rows = [[str(record.get(key, "manual" if key == "reason" else "-"))
             for _, key in columns]
            for record in sorted(records, key=lambda record: (record["name"], record["version"]))]
    headers = [label for label, _ in columns]
    widths = [max(len(headers[index]), *(len(row[index]) for row in rows))
              for index in range(len(columns))]
    lines = [headers, ["-" * width for width in widths], *rows]
    return "\n".join("  ".join(value.ljust(width) for value, width in zip(row, widths)).rstrip()
                     for row in lines)


def main(arguments=None):
    skill_management.configure_utf8_stdio()
    arguments = list(sys.argv[1:] if arguments is None else arguments)
    if arguments and arguments[0] == "skill":
        return skill_management.main(arguments[1:])
    if arguments and arguments[0].startswith("skill-"):
        return skill_management.main([arguments[0][6:], *arguments[1:]])
    parser = argparse.ArgumentParser(prog="aiw pkg", description=__doc__)
    parser.add_argument("--repo", help="Explicit repository URL or local directory")
    commands = parser.add_subparsers(dest="action", required=True)
    for name in ("install", "remove", "purge", "show"):
        command = commands.add_parser(name)
        command.add_argument("--dest", type=Path, default=default_destination())
        command.add_argument("package", help="Local ZIP path or package name" if name == "install" else "Package name")
        command.add_argument("--repo", default=argparse.SUPPRESS)
        if name == "install":
            command.add_argument("--version")
            command.add_argument("--dry-run", action="store_true")
    update = commands.add_parser("update")
    update.add_argument("--dest", type=Path, default=default_destination())
    update.add_argument("--repo", default=argparse.SUPPRESS)
    update.add_argument("--dry-run", action="store_true")
    scopes = update.add_mutually_exclusive_group()
    scopes.add_argument("--core", action="store_true")
    scopes.add_argument("--all", action="store_true")
    listing = commands.add_parser("list")
    listing.add_argument("--all", action="store_true")
    listing.add_argument("--installed", action="store_true")
    listing.add_argument("--json", action="store_true", help="Print complete JSON records instead of a table")
    listing.add_argument("--dest", type=Path, default=default_destination())
    listing.add_argument("--kind", choices=("program", "plugin", "skill", "resource", "collection"))
    listing.add_argument("--repo", default=argparse.SUPPRESS)
    for name in ("search", "info"):
        query = commands.add_parser(name)
        query.add_argument("query")
        query.add_argument("--repo", default=argparse.SUPPRESS)
        if name == "info":
            query.add_argument("--version")
    entry = commands.add_parser("entries", help="List installed plugin entry points and metadata")
    entry.add_argument("--json", action="store_true")
    commands.add_parser("skill", help="Managed project/user Skill operations; use skill --help")
    for name in ("publish", "unpublish"):
        publish = commands.add_parser(name)
        publish.add_argument("package")
        publish.add_argument("version")
        publish.add_argument("--repo", default=argparse.SUPPRESS)
    commands.add_parser("index").add_argument("repository", type=Path)
    mark = commands.add_parser("mark")
    mark.add_argument("reason", choices=("manual",))
    mark.add_argument("package")
    mark.add_argument("--dest", type=Path, default=default_destination())
    auto = commands.add_parser("autoremove")
    auto.add_argument("--dest", type=Path, default=default_destination())
    auto.add_argument("--dry-run", action="store_true")
    pack = commands.add_parser("pack", help="Package workspace containing info and data/")
    pack.add_argument("workspace", type=Path)
    pack.add_argument("--output", required=True, type=Path)
    args = parser.parse_args(arguments)
    try:
        if args.action == "pack":
            pack_workspace(args.workspace, args.output)
            print(f"Packaged: {args.output}")
            return 0
        if args.action == "index":
            print(index_repository(args.repository))
            return 0
        if args.action == "entries":
            records = entries.list_plugins()
            print(json.dumps({"plugins": records}, ensure_ascii=False, indent=2) if args.json else
                  "\n".join(f"{r['name']}: {r.get('description', '')} ({r['path']})" for r in records))
            return 0
        manager = Manager(args.dest) if hasattr(args, "dest") else None
        if args.action == "show":
            record = manager.record(args.package)
            if record is None:
                raise PackageError(f"Package not recorded: {args.package}")
            print(json.dumps(record, ensure_ascii=False, indent=2))
            return 0
        if args.action == "mark":
            print(manager.mark_manual(args.package))
            return 0
        if args.action == "autoremove":
            print(Operation(manager).autoremove(args.dry_run))
            return 0
        if args.action in {"remove", "purge"}:
            print(manager.remove(args.package, purge=args.action == "purge"))
            print("Orphan packages: " + (", ".join(manager.orphans()) or "none"))
            print("Use aiw pkg autoremove, or aiw pkg mark manual <package> to retain a member.")
            return 0
        if args.action == "list" and args.installed:
            records = [r for r in manager.records().values() if not args.kind or r.get("kind") == args.kind]
            print(json.dumps(records, ensure_ascii=False, indent=2) if args.json else
                  format_package_list(records, installed=True))
            return 0
        if args.action == "install":
            path = Path(args.package).expanduser()
            if path.is_file():
                from archive import read_package
                prepared = read_package(path)
                info = prepared[0]
                if args.version and args.version != info["version"]:
                    raise PackageError("Explicit version does not match local package")
                repo = configured_repos(args.repo)[0] if args.repo or info.get("dependencies") or info.get("members") else None
                print(Operation(manager, repo).install(path, prepared_package=prepared, dry_run=args.dry_run))
            elif path.exists() or "/" in args.package or "\\" in args.package or path.suffix.lower() == ".zip" or ":" in args.package:
                raise PackageError(f"Local package file not found or not regular: {path}")
            else:
                identifier(args.package)
                repo = configured_repos(args.repo)[0]
                print(Operation(manager, repo).install(name=args.package, version=args.version, dry_run=args.dry_run))
            return 0
        repo, writer = configured_repos(args.repo)
        if args.action == "update":
            print(Operation(manager, repo).install(update=True, scope="core" if args.core else "all" if args.all else None, dry_run=args.dry_run))
        elif args.action == "publish":
            from archive import read_package
            package = Path(args.package).expanduser()
            if read_package(package)[0]["version"] != args.version:
                raise PackageError("Publish version must match package metadata")
            print(writer.publish(package))
        elif args.action == "unpublish":
            print(writer.unpublish(args.package, args.version))
        else:
            catalog = repo.index()
            if args.action == "info":
                result = select(catalog, args.query, args.version)
            elif args.action == "search":
                result = [r for r in catalog if args.query.casefold() in r["name"].casefold()]
            else:
                result = catalog if args.all else [select(catalog, n) for n in sorted({r["name"] for r in catalog})]
                result = [r for r in result if not args.kind or r["kind"] == args.kind]
            if args.action == "list" and not args.json:
                print(format_package_list(result))
            else:
                print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0
    except (PackageError, OSError, ValueError, KeyError, TypeError, zipfile.BadZipFile) as error:
        print(f"pkg: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
