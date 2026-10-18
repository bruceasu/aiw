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
from operations import Operation
from repository import Repo, entry_for

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
            if directory.is_dir() and (directory / "plugin.toml").is_file()] + [
                path.stem.removeprefix("aiw-") for path in sorted((SRC / "plugins").glob("aiw-*.py"))]


def skill_targets() -> list[str]:
    return ["resource-" + path.stem for path in sorted((SRC / "skills").glob("*.md"))] + ["skill-" + directory.name for directory in sorted((SRC / "skills").iterdir())
                                if directory.is_dir() and (directory / "SKILL.md").is_file()]


def expand(target: str) -> list[str]:
    from scripts.package_definitions import COLLECTIONS, collection_members, component_dependencies
    if target in COLLECTIONS:
        result = []
        for name in collection_members(target):
            if name in COLLECTIONS:
                result += expand(name)
            elif name == "aiw-main":
                result += expand("aiw")
            else:
                result += expand(name.removeprefix("aiw-"))
        return list(dict.fromkeys([*result, target]))
    if target == "bin":
        return list(dict.fromkeys(item for component in BIN_TARGETS for item in expand(component)))
    if target == "plugins":
        return ["plugin-runtime", *plugin_targets(), "plugins-core", "plugins-all"]
    if target == "skills":
        return list(dict.fromkeys([*(item for component in skill_targets() for item in expand(component)), "skills-core", "skills-all"]))
    if target == "all":
        return list(dict.fromkeys([*BIN_TARGETS, "plugin-runtime", *plugin_targets(), *skill_targets(), *COLLECTIONS]))
    return list(dict.fromkeys([*(item for dependency in component_dependencies(target) for item in expand(dependency)), target]))


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


def legacy_package_component(component: str, version: str) -> Path:
    entries = []
    if component == "gateway":
        for system in SYSTEMS:
            source = binary(component, system)
            entries.append(one(source, source.name, executable=True, system=system))
        source = SRC / "programs/agent-gateway"
        entries.append(one(source / "gateway-example.json", "gateway-new.json"))
        entries.append(one(source / "gateway-example.json", "gateway.json", config=True))
    elif component == "http-openai-proxy":
        source = SRC / "programs/http-openai-proxy"
        entries += tree_entries(BIN / component, component + "/dist")
        for filename in ("http-openai-proxy.js", "package.json", "package-lock.json", "README.md", "tsconfig.json"):
            entries.append(one(source / filename, component + "/" + filename))
        entries += tree_entries(source / "src", component + "/src")
        entries += tree_entries(source / "node_modules", component + "/node_modules", include_dependencies=True)
    else:
        raise PackageError(f"Unknown standalone program: {component}")
    name = "aiw-" + component
    doc = ROOT / "docs/usage" / (name + ".md")
    if doc.is_file():
        entries.append(one(doc, "docs/usage/" + doc.name))
    output = DIST / "mist" / version / f"{name}-{version}.zip"
    write_package(output, name, version, "program", entries)
    if component == "gateway":
        for system in SYSTEMS:
            source = binary(component, system)
            destination = DIST / "bin" / (system + "-amd64") / source.name
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)
    print(f"Built: {output}")
    return output


def package_component(component: str, version: str) -> Path:
    from scripts.package_definitions import build_component
    if component in plugin_targets() or component.startswith("skill-"):
        script = ROOT / "scripts/packages" / (("skill-" + component[6:]) if component.startswith("skill-") else "plugin-aiw-" + component)
        script = script.with_suffix(".py")
        if not script.is_file():
            raise PackageError(f"Independent package script missing: {script}")
        result = subprocess.run([sys.executable, str(script), "--version", version], cwd=ROOT, capture_output=True, text=True, check=False)
        if result.returncode:
            raise PackageError(f"Package script failed: {script}: {result.stderr}")
        print(result.stdout.strip())
        name = "aiw-" + component
        return DIST / ("skills" if component.startswith("skill-") else "plugins") / version / f"{name}-{version}.zip"
    return build_component(component, version)


def export_repository(packages, root):
    repo = Repo(str(root))
    for package in packages:
        print(repo.publish(package))


def main(arguments: list[str]) -> int:
    parser = argparse.ArgumentParser(description=__doc__, allow_abbrev=False)
    actions = parser.add_subparsers(dest="action")
    from scripts.package_definitions import COLLECTIONS
    choices = sorted(set(TARGETS) - {"docs"} | set(plugin_targets()) | set(skill_targets()) | set(COLLECTIONS) | {"plugin-runtime"})
    for action in ("compile", "build", "install", "export"):
        command = actions.add_parser(action, allow_abbrev=False)
        command.add_argument("target", choices=choices)
        if action == "install":
            command.add_argument("--dest", type=Path, default=default_destination())
        elif action == "export":
            command.add_argument("--repository", type=Path, required=True)
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
            export_repository(packages, DIST / "repository")
            if args.target == "all":
                for group in ("bin", "plugins", "skills"):
                    write_manifest(group, args.version, [by_component[item] for item in expand(group)])
        elif args.action in {"install", "export"}:
            manifest = DIST / ".manifests" / (args.target + ".json")
            if not manifest.is_file():
                raise PackageError(f"No release manifest for {args.target}; run build explicitly first")
            content = json.loads(manifest.read_text(encoding="utf-8"))
            packages = []
            for value in content["packages"]:
                package = safe_path(DIST, value)
                if not package.is_file():
                    raise PackageError(f"Release package missing: {package}")
                packages.append(package)
            if args.action == "export":
                export_repository(packages, args.repository)
            else:
                from scripts.package_definitions import COLLECTIONS
                root = DIST / "repository"
                repo = Repo(str(root))
                manager = Manager(args.dest)
                roots = [p for p in packages if entry_for(p)["name"] == args.target] if args.target in COLLECTIONS else packages
                for package in roots:
                    print(Operation(manager, repo).install(package))
        return 0
    except (PackageError, OSError, ValueError, KeyError, TypeError, zipfile.BadZipFile) as error:
        print(f"build.py: {error}", file=sys.stderr)
        return 1


def write_manifest(target: str, version: str, packages: list[Path]) -> None:
    content = dict(version=version, packages=[path.relative_to(DIST).as_posix() for path in packages])
    atomic_bytes(DIST / ".manifests" / (target + ".json"), json.dumps(content, indent=2).encode())


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
