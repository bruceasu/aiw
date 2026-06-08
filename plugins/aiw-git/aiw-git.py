#!/usr/bin/env python3
import importlib.util
import os
import shutil
import subprocess
import sys

META = {
    "name": "git",
    "short": "Git command bundle with discoverable subcommands.",
    "long": "Single entrypoint for aiw git subcommands with concise overview help and detailed per-command help.",
    "usage": "aiw git <subcommand> [args...]",
    "examples": [
        "aiw git show status",
        "aiw git add-remote origin https://example/repo.git",
    ],
}

HERE = os.path.dirname(__file__)
HELP_FLAGS = {"-h", "--help", "-help", "-?"}
SUPPORTED_EXTENSIONS = {"", ".py", ".bat", ".cmd", ".sh", ".ps1"}
EXTENSION_PRIORITY = {
    ".py": 0,
    ".bat": 1,
    ".cmd": 1,
    ".sh": 2,
    ".ps1": 3,
    "": 4,
}


def module_name_for_path(path):
    stem = os.path.splitext(os.path.basename(path))[0]
    return stem.replace("-", "_")


def load_python_module(path):
    spec = importlib.util.spec_from_file_location(module_name_for_path(path), path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def parse_subcommand_name(filename):
    stem, ext = os.path.splitext(filename)
    if ext.lower() not in SUPPORTED_EXTENSIONS:
        return ""
    if not stem.startswith("git-"):
        return ""
    return stem[len("git-") :]


def discover_subcommands(base_dir):
    commands = {}
    for entry in sorted(os.listdir(base_dir)):
        full_path = os.path.join(base_dir, entry)
        if not os.path.isfile(full_path):
            continue
        subcommand = parse_subcommand_name(entry)
        if not subcommand:
            continue
        ext = os.path.splitext(entry)[1].lower()
        command = {
            "name": subcommand,
            "path": full_path,
            "ext": ext,
            "meta": default_meta(subcommand),
        }
        if ext == ".py":
            module = load_python_module(full_path)
            command["module"] = module
            meta = getattr(module, "META", None)
            if isinstance(meta, dict):
                command["meta"] = normalized_meta(subcommand, meta)
            else:
                command["meta"] = meta_from_module(subcommand, module)
        old = commands.get(subcommand)
        if old is None or EXTENSION_PRIORITY[ext] < EXTENSION_PRIORITY[old["ext"]]:
            commands[subcommand] = command
    return commands


def default_meta(subcommand):
    return {
        "name": subcommand,
        "short": "",
        "long": "",
        "usage": f"aiw git {subcommand}",
        "args": [],
        "examples": [f"aiw git {subcommand}"],
    }


def normalized_meta(subcommand, meta):
    normalized = default_meta(subcommand)
    normalized.update(meta)
    normalized["name"] = subcommand
    normalized["args"] = list(normalized.get("args") or [])
    normalized["examples"] = list(normalized.get("examples") or [])
    return normalized


def meta_from_module(subcommand, module):
    meta = default_meta(subcommand)
    summary = summary_from_docstring(getattr(module, "__doc__", "") or "")
    if summary:
        meta["short"] = summary
    return meta


def summary_from_docstring(docstring):
    for raw_line in docstring.splitlines():
        line = raw_line.strip()
        if not line:
            continue
        if line.startswith("aiw git ") or line.startswith("aiw-git") or line.startswith("git-"):
            continue
        return line
    return ""


def render_overview_help(commands):
    lines = [
        "Usage:",
        "  aiw git <subcommand> [args...]",
        "",
        "Git helper commands:",
        "",
        "Commands:",
    ]
    width = max([len(name) for name in commands] + [4])
    for name in sorted(commands):
        meta = commands[name]["meta"]
        short = meta.get("short", "").strip() or "No summary available."
        lines.append(f"  {name:<{width}}  {short}")
        example = first_example(name, meta)
        if example:
            lines.append(f"  {'':<{width}}  e.g. {example}")
    lines.extend(
        [
            "",
            "Run `aiw git help <subcommand>` for detailed help and more examples.",
        ]
    )
    return "\n".join(lines)


def render_detailed_help(command):
    meta = command["meta"]
    lines = [f"aiw git {command['name']}"]
    if meta.get("short"):
        lines.extend(["", meta["short"]])
    if meta.get("long"):
        lines.extend(["", meta["long"]])
    if meta.get("usage"):
        lines.extend(["", "Usage:", f"  {normalize_usage(meta['usage'], command['name'])}"])
    if meta.get("args"):
        lines.extend(["", "Arguments:"])
        for arg in meta["args"]:
            flag = arg.get("flag", "")
            desc = arg.get("description", "")
            lines.append(f"  {flag:16} {desc}".rstrip())
    if meta.get("examples"):
        lines.extend(["", "Examples:"])
        for example in meta["examples"]:
            lines.append(f"  {normalize_example(command['name'], example)}")
    return "\n".join(lines)


def normalize_usage(usage, subcommand):
    stripped = usage.strip()
    if stripped.startswith("aiw git "):
        return stripped
    if stripped.startswith("aiw "):
        return stripped.replace("aiw ", "aiw git ", 1)
    if stripped.startswith(subcommand):
        return f"aiw git {stripped}"
    return f"aiw git {subcommand} {stripped}".strip()


def normalize_example(subcommand, example):
    stripped = example.strip()
    if not stripped:
        return ""
    if stripped.startswith("aiw git "):
        return stripped
    if stripped.startswith("aiw "):
        return stripped.replace("aiw ", "aiw git ", 1)
    return f"aiw git {stripped}"


def first_example(subcommand, meta):
    examples = meta.get("examples") or []
    if not examples:
        return ""
    return normalize_example(subcommand, examples[0])


def print_overview(commands):
    print(render_overview_help(commands))


def print_detailed(commands, subcommand):
    command = commands.get(subcommand)
    if command is None:
        print(f"Unknown git subcommand: {subcommand}", file=sys.stderr)
        print("Run `aiw git` to see available commands.", file=sys.stderr)
        return 1
    print(render_detailed_help(command))
    return 0


def build_external_command(path, ext, args):
    if ext in {".bat", ".cmd"}:
        if os.name == "nt":
            return ["cmd", "/C", path, *args]
        return [path, *args]
    if ext == ".ps1":
        shell = "powershell" if os.name == "nt" else "pwsh"
        return [shell, "-File", path, *args]
    if ext == ".sh":
        shell = shutil.which("bash") or shutil.which("sh") or "sh"
        return [shell, path, *args]
    return [path, *args]


def dispatch_command(command, argv):
    if command["ext"] == ".py":
        module = command.get("module") or load_python_module(command["path"])
        if not hasattr(module, "main"):
            print(f"Python subcommand has no main(argv): {command['path']}", file=sys.stderr)
            return 1
        return module.main(argv)
    proc = subprocess.run(build_external_command(command["path"], command["ext"], argv))
    return proc.returncode


def main(argv):
    commands = discover_subcommands(HERE)
    if not argv or argv[0] in HELP_FLAGS:
        print_overview(commands)
        return 0
    if argv[0] == "help":
        if len(argv) == 1 or argv[1] in HELP_FLAGS:
            print_overview(commands)
            return 0
        return print_detailed(commands, argv[1])

    subcommand = argv[0]
    command = commands.get(subcommand)
    if command is None:
        print(f"Unknown git subcommand: {subcommand}", file=sys.stderr)
        print("Run `aiw git` to see available commands.", file=sys.stderr)
        return 1
    if any(arg in HELP_FLAGS for arg in argv[1:]):
        return print_detailed(commands, subcommand)
    return dispatch_command(command, argv[1:])


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
