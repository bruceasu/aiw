#!/usr/bin/env python3
"""Offline package lifecycle. Repository lookup is reserved for a later FD."""
import argparse
from pathlib import Path
import sys
import zipfile

from archive import PackageError, identifier, pack_workspace
from manager import Manager, default_destination


def main(arguments=None):
    parser = argparse.ArgumentParser(prog="aiw pkg", description=__doc__)
    commands = parser.add_subparsers(dest="action", required=True)
    for name in ("install", "remove", "purge"):
        command = commands.add_parser(name)
        command.add_argument("--dest", type=Path, default=default_destination())
        command.add_argument("package", help="Local ZIP path or package name" if name == "install" else "Package name")
    pack = commands.add_parser("pack", help="Package workspace containing info and data/")
    pack.add_argument("workspace", type=Path)
    pack.add_argument("--output", required=True, type=Path)
    args = parser.parse_args(arguments)
    try:
        if args.action == "pack":
            pack_workspace(args.workspace, args.output)
            print(f"Packaged: {args.output}")
            return 0
        manager = Manager(args.dest)
        if args.action == "install":
            path = Path(args.package).expanduser()
            if path.is_file():
                print(manager.install(path))
            elif path.exists() or "/" in args.package or "\\" in args.package or path.suffix.lower() == ".zip" or ":" in args.package:
                raise PackageError(f"Local package file not found or not regular: {path}")
            else:
                identifier(args.package)
                raise PackageError("Package-name lookup requires repository support, not implemented yet; provide a local ZIP file")
        else:
            print(manager.remove(args.package, purge=args.action == "purge"))
        return 0
    except (PackageError, OSError, ValueError, KeyError, TypeError, zipfile.BadZipFile) as error:
        print(f"pkg: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
