"""Offline compile, release staging and installation with explicit stage boundaries."""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parent
SRC, BIN, DIST = ROOT / "src", ROOT / "bin", ROOT / "dist"
sys.path.insert(0, str(SRC / "plugins" / "aiw-pkg"))
from archive import PackageError, atomic_bytes, identifier, reject_link, safe_path, write_package
from manager import Manager, default_destination

SYSTEMS = ("windows", "linux")
GO = {
    "aiw": (SRC, "./cmd/aiw", "aiw", True),
    "req": (SRC, "./cmd/aiw-req", "aiw-req", True),
    "say": (SRC, "./cmd/aiw-say", "aiw-say", True),
    "gateway": (SRC / "programs/agent-gateway", ".", "agent-gateway", False),
    "packer": (SRC / "programs/aiw-packer", ".", "aiw-packer", False),
    "file-convert": (SRC / "programs/md-converter", ".", "md-converter", False),
}
BIN_TARGETS = ("aiw", "gateway", "http-openai-proxy")
TARGETS = (*GO, "http-openai-proxy", "cz", "plugins", "docs", "skills", "bin", "all")
IGNORE_DIRS = {".git", "__pycache__", "node_modules", ".bin", "tests", ".venv", "dist", "release"}
IGNORE_FILES = {"aiw.toml", "cz.toml", ".cz.toml", "agent-gateway", "aiw-req", "aiw-say", "aiw-packer", "md-converter"}
SAY_PROGRAM_ASSETS = {"aiw.toml.example", "profiles", "aiw-say-hotkeys.ahk", "aiw-say-hotkeys.ps1"}


def run(command: list[str], cwd: Path, environment: dict[str, str]) -> None:
    result = subprocess.run(command, cwd=cwd, env=environment, check=False)
    if result.returncode:
        raise PackageError(f"Command failed ({result.returncode}): {command[0]} in {cwd}")


def tool(name: str) -> str:
    path = shutil.which(name)
    if not path:
        raise PackageError(f"{name} is not available on PATH; dependencies are never downloaded")
    return path


def binary(component: str, system: str) -> Path:
    return BIN / f"{system}-amd64" / (GO[component][2] + (".exe" if system == "windows" else ""))


def plugin_targets() -> list[str]:
    return [directory.name.removeprefix("aiw-") for directory in sorted((SRC / "plugins").iterdir())
            if directory.is_dir() and (directory / "plugin.toml").is_file()] + ["plugin-tools"]


def skill_targets() -> list[str]:
    return ["skills-support"] + ["skill-" + directory.name for directory in sorted((SRC / "skills").iterdir())
                                if directory.is_dir() and (directory / "SKILL.md").is_file()]


def expand(target: str) -> list[str]:
    if target == "bin":
        return list(BIN_TARGETS)
    if target == "plugins":
        return plugin_targets()
    if target == "skills":
        return skill_targets()
    if target == "all":
        return list(dict.fromkeys([*BIN_TARGETS, *plugin_targets(), "docs", *skill_targets()]))
    return [target]


def compile_components(components: list[str], version: str) -> None:
    for component in components:
        if component in GO:
            cwd, target, _name, versioned = GO[component]
            go = tool("go")
            for system in SYSTEMS:
                output = binary(component, system)
                safe_path(ROOT, output.relative_to(ROOT).as_posix())
                output.parent.mkdir(parents=True, exist_ok=True)
                environment = os.environ.copy()
                environment.update(GOPROXY="off", GOSUMDB="off", GOTOOLCHAIN="local", GOFLAGS="",
                                   GOWORK="off", CGO_ENABLED="0", GOOS=system, GOARCH="amd64")
                linker = "-s -w" + (f" -X=aiw/internal/version.Version={version}" if versioned else "")
                run([go, "build", "-trimpath", "-ldflags=" + linker, "-o", str(output), target], cwd, environment)
        elif component == "http-openai-proxy":
            source = SRC / "programs/http-openai-proxy"
            compiler = source / "node_modules/typescript/bin/tsc"
            if not compiler.is_file():
                raise PackageError(f"Local TypeScript compiler missing: {compiler}; install dependencies explicitly")
            BIN.mkdir(parents=True, exist_ok=True)
            with tempfile.TemporaryDirectory(prefix=".proxy-", dir=BIN) as temporary:
                output = Path(temporary)
                run([tool("node"), str(compiler), "--project", str(source / "tsconfig.json"),
                     "--outDir", str(output)], source, os.environ.copy())
                destination = safe_path(ROOT, "bin/http-openai-proxy")
                if destination.exists():
                    shutil.rmtree(destination)
                shutil.copytree(output, destination)
        print(f"Compiled: {component}" if component in GO or component == "http-openai-proxy" else f"No compile needed: {component}")


def tree_entries(source: Path, destination: str, *, include_dependencies=False, exclude=None):
    if not source.is_dir():
        raise PackageError(f"Resource directory missing: {source}")
    reject_link(source)
    excluded = set(exclude or ())
    entries = []
    for directory, directories, names in os.walk(source, followlinks=False):
        directories[:] = sorted(name for name in directories
                                if name not in (IGNORE_DIRS - {"node_modules"} if include_dependencies else IGNORE_DIRS)
                                and (Path(directory) / name).relative_to(source).as_posix() not in excluded)
        for name in directories:
            reject_link(Path(directory) / name)
        for name in sorted(names):
            if name in IGNORE_FILES or name.endswith((".pyc", ".exe")):
                continue
            path = Path(directory) / name
            relative = path.relative_to(source).as_posix()
            if relative in excluded:
                continue
            target = destination + "/" + relative if destination else relative
            entries.append((path, "common/" + target, target, "all", False, os.access(path, os.X_OK)))
    return entries


def one(source: Path, target: str, *, config=False, executable=False, system="all"):
    prefix = "common" if system == "all" else system + "-amd64"
    return (source, prefix + "/" + target, target, system, config, executable)


def package_component(component: str, version: str) -> Path:
    entries = []
    kind, category = "resource", "mist"
    name = "aiw-" + component
    if component == "aiw":
        name, kind = "aiw-main", "program"
        for system in SYSTEMS:
            source = binary(component, system)
            target = "aiw.exe" if system == "windows" else "aiw"
            entries.append(one(source, target, executable=True, system=system))
    elif component == "gateway":
        kind = "program"
        for system in SYSTEMS:
            source = binary(component, system)
            entries.append(one(source, source.name, executable=True, system=system))
        source = SRC / "programs/agent-gateway"
        entries.append(one(source / "gateway-example.json", "gateway-new.json"))
        entries.append(one(source / "gateway-example.json", "gateway.json", config=True))
    elif component == "http-openai-proxy":
        kind = "program"
        source = SRC / "programs/http-openai-proxy"
        entries += tree_entries(BIN / component, component + "/dist")
        for filename in ("http-openai-proxy.js", "package.json", "package-lock.json", "README.md", "tsconfig.json"):
            entries.append(one(source / filename, component + "/" + filename))
        entries += tree_entries(source / "src", component + "/src")
        entries += tree_entries(source / "node_modules", component + "/node_modules", include_dependencies=True)
    elif component == "docs":
        entries += tree_entries(ROOT / "docs/usage", "docs/usage")
        entries += tree_entries(SRC / "agent-templates", "agent-templates")
    elif component == "skills-support":
        category, kind, name = "skills", "skill", "aiw-skills-support"
        for source in sorted((SRC / "skills").glob("*.md")):
            entries.append(one(source, "skills/" + source.name))
    elif component.startswith("skill-"):
        category, kind, name = "skills", "skill", "aiw-" + component
        skill = component.removeprefix("skill-")
        entries += tree_entries(SRC / "skills" / skill, "skills/" + skill)
    elif component == "plugin-tools":
        category, kind, name = "plugins", "plugin", "aiw-plugin-tools"
        for source in sorted((SRC / "plugins").glob("*")):
            if source.is_file() and source.suffix in {".py", ".md"}:
                if source.name in {"aiw-ai-code-index.py", "aiw-ai-gen-index.py"}:
                    continue
                entries.append(one(source, "plugins/" + source.name))
        for source, target in (("ai-code-index", "aiw-ai-code-index.py"), ("generate-ai-index", "aiw-ai-gen-index.py")):
            entries.append(one(SRC / "programs/ai-code-tools" / source, "plugins/" + target))
    else:
        category, kind = "plugins", "plugin"
        source = SRC / "plugins" / name
        entries += tree_entries(source, "plugins/" + name,
                                exclude=SAY_PROGRAM_ASSETS if component == "say" else None)
        if component in GO:
            for system in SYSTEMS:
                compiled = binary(component, system)
                entries.append(one(compiled, "plugins/" + name + "/" + compiled.name,
                                   executable=True, system=system))
        if component == "say":
            source = SRC / "programs/aiw-say"
            for filename in ("aiw.toml.example", "aiw-say-hotkeys.ahk", "aiw-say-hotkeys.ps1"):
                entries.append(one(source / filename, "plugins/aiw-say/" + filename))
            entries += tree_entries(source / "profiles", "plugins/aiw-say/profiles")
            entries.append(one(source / "aiw.toml.example", "plugins/aiw-say/aiw.toml", config=True))
    directory = DIST / category / version if category in {"plugins", "skills"} else DIST / category
    output = directory / f"{name}-{version}.zip"
    write_package(output, name, version, kind, entries)
    if component in {"aiw", "gateway"}:
        for system in SYSTEMS:
            source = binary(component, system)
            destination = DIST / "bin" / (system + "-amd64") / source.name
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)
    print(f"Built: {output}")
    return output


def main(arguments: list[str]) -> int:
    parser = argparse.ArgumentParser(description=__doc__, allow_abbrev=False)
    actions = parser.add_subparsers(dest="action")
    for action in ("compile", "build", "install"):
        command = actions.add_parser(action, allow_abbrev=False)
        command.add_argument("target", choices=TARGETS)
        if action == "install":
            command.add_argument("--dest", type=Path, default=default_destination())
        else:
            command.add_argument("--version", default=os.environ.get("AIW_VERSION", "dev"))
    args = parser.parse_args(arguments)
    if not args.action:
        parser.print_help()
        return 0
    try:
        components = expand(args.target)
        if args.action in {"compile", "build"}:
            identifier(args.version)
            compile_components(components, args.version)
        if args.action == "build":
            packages = [package_component(component, args.version) for component in components]
            by_component = dict(zip(components, packages))
            for component, package in zip(components, packages):
                write_manifest(component, args.version, [package])
            write_manifest(args.target, args.version, packages)
            if args.target == "all":
                for group in ("bin", "plugins", "skills"):
                    write_manifest(group, args.version, [by_component[item] for item in expand(group)])
        elif args.action == "install":
            manifest = DIST / ".manifests" / (args.target + ".json")
            if not manifest.is_file():
                raise PackageError(f"No release manifest for {args.target}; run build explicitly first")
            content = json.loads(manifest.read_text(encoding="utf-8"))
            manager = Manager(args.dest)
            packages = []
            for value in content["packages"]:
                package = safe_path(DIST, value)
                if not package.is_file():
                    raise PackageError(f"Release package missing: {package}")
                packages.append(package)
            for package in packages:
                print(manager.install(package))
        return 0
    except (PackageError, OSError, ValueError, KeyError, TypeError, zipfile.BadZipFile) as error:
        print(f"build.py: {error}", file=sys.stderr)
        return 1


def write_manifest(target: str, version: str, packages: list[Path]) -> None:
    content = dict(version=version, packages=[path.relative_to(DIST).as_posix() for path in packages])
    atomic_bytes(DIST / ".manifests" / (target + ".json"), json.dumps(content, indent=2).encode())


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
