#!/usr/bin/env python3
import argparse
import ast
import json
import tomllib
from pathlib import Path

META = {
    "name": "aiw-plugin",
    "short": "list AIW plugins and their metadata",
    "description": "List AIW plugin entry points and their basic metadata.",
    "commands": ["list"],
    "readOnly": True,
    "mutatesFiles": False,
    "requiresConfirmation": False,
    "outputFormat": "json",
}


def _plugin_dir() -> Path:
    return Path(__file__).resolve().parent.parent


def _load_meta(path: Path) -> dict:
    try:
        source = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        source = path.read_text(encoding="utf-8-sig")
    try:
        tree = ast.parse(source, filename=str(path))
    except SyntaxError:
        return {}
    for node in tree.body:
        if isinstance(node, ast.Assign):
            for target in node.targets:
                if isinstance(target, ast.Name) and target.id == "META":
                    try:
                        meta = ast.literal_eval(node.value)
                    except Exception:
                        return {}
                    if isinstance(meta, dict):
                        return meta
    return {}


def list_plugins() -> list[dict]:
    records = []
    for path in sorted(_plugin_dir().glob("aiw-*.py")):
        if path.name == "aiw-plugin.py":
            continue
        meta = _load_meta(path)
        records.append(
            {
                "name": meta.get("name", path.stem),
                "short": meta.get("short", ""),
                "description": meta.get("description", ""),
                "path": str(path),
                "commands": meta.get("commands", []),
                "readOnly": bool(meta.get("readOnly", False)),
                "mutatesFiles": bool(meta.get("mutatesFiles", False)),
                "requiresConfirmation": bool(meta.get("requiresConfirmation", False)),
                "outputFormat": meta.get("outputFormat", "text"),
            }
        )
    for directory in sorted(_plugin_dir().iterdir()):
        manifest = directory / "plugin.toml"
        if not manifest.is_file():
            continue
        for meta in tomllib.loads(manifest.read_text(encoding="utf-8")).get("plugins", []):
            entry = meta.get("entrypoint")
            paths = [directory / entry] if entry else sorted(directory.glob("aiw-*"))
            path = paths[0] if paths else directory
            source_meta = {}
            if (path.suffix == ".py" and path.is_file()
                    and path.resolve().is_relative_to(directory.resolve())):
                source_meta = _load_meta(path)
            records.append(dict(source_meta, **meta, path=str(path)))
    return records


def main() -> int:
    parser = argparse.ArgumentParser(prog="aiw pkg entries")
    sub = parser.add_subparsers(dest="command")
    list_parser = sub.add_parser("list")
    list_parser.add_argument("--json", action="store_true")
    args = parser.parse_args()
    if args.command == "list" and args.json:
        print(json.dumps({"plugins": list_plugins()}, ensure_ascii=False, indent=2))
        return 0
    parser.error("use: aiw pkg entries --json")
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
