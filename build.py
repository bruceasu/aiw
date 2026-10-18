"""Offline compile, release staging and installation with explicit stage boundaries."""
from __future__ import annotations

import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import time
import zipfile

ROOT = Path(__file__).resolve().parent
SRC, BIN, DIST = ROOT / "src", ROOT / "bin", ROOT / "dist"
sys.path.insert(0, str(SRC / "plugins" / "aiw-pkg"))
from archive import PackageError, atomic_bytes, identifier, reject_link, safe_path, write_package
from manager import Manager, default_destination
from operations import Operation
from repository import Repo

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
TARGETS = {
    "programs": BIN_TARGETS,
    "resources": ("docs", "plugin-runtime"),
    "groups": ("bin", "plugins", "skills", "all"),
}
IGNORE_DIRS = {".git", "__pycache__", "node_modules", ".bin", "tests", ".venv", "dist", "release"}
IGNORE_FILES = {"aiw.toml", "cz.toml", ".cz.toml", "agent-gateway", "aiw-req", "aiw-say", "aiw-packer", "md-converter"}
SAY_PROGRAM_ASSETS = {"aiw.toml.example", "profiles", "aiw-say-hotkeys.ahk", "aiw-say-hotkeys.ps1"}


def load_environment() -> None:
    """Load checkout-local defaults; never execute or expand dotenv values."""
    path = ROOT / ".env"
    if not path.is_file():
        return
    for number, line in enumerate(path.read_text(encoding="utf-8-sig").splitlines(), 1):
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("export "):
            line = line[7:].lstrip()
        key, separator, value = line.partition("=")
        key, value = key.strip(), value.strip()
        if not separator or not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*", key):
            raise PackageError(f"Invalid .env assignment at line {number}")
        if value.startswith(("'", '"')):
            quote = value[0]
            end = value.find(quote, 1)
            if end < 0 or (value[end + 1:].strip() and not value[end + 1:].lstrip().startswith("#")):
                raise PackageError(f"Invalid .env quoted value at line {number}")
            value = value[1:end]
        else:
            value = re.split(r"\s+#", value, maxsplit=1)[0].rstrip()
        os.environ.setdefault(key, value)


def run(command: list[str], cwd: Path, environment: dict[str, str]) -> None:
    result = subprocess.run(command, cwd=cwd, env=environment, check=False)
    if result.returncode:
        raise PackageError(f"Command failed ({result.returncode}): {command[0]} in {cwd}")


def tool(name: str) -> str:
    configured = os.environ.get("AIW_" + name.upper(), name)
    path = shutil.which(configured)
    if not path:
        raise PackageError(f"Tool unavailable: {configured}; configure AIW_{name.upper()} or PATH; dependencies are never downloaded")
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


def target_catalog() -> dict[str, tuple[str, ...]]:
    """Separate package kinds from compatibility groups; discover source owners."""
    from scripts.package_definitions import COLLECTIONS
    skills = skill_targets()
    return {**TARGETS, "plugins": tuple(plugin_targets()),
            "skills": tuple(item for item in skills if item.startswith("skill-")),
            "resources": (*TARGETS["resources"], *(item for item in skills if item.startswith("resource-"))),
            "collections": tuple(COLLECTIONS)}


def package_aliases() -> dict[str, str]:
    from scripts.package_definitions import COLLECTIONS
    return {(name if name in COLLECTIONS else "aiw-main" if name == "aiw" else "aiw-" + name): name
            for kind, names in target_catalog().items() if kind != "groups" for name in names}


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
        return list(dict.fromkeys(item for kind, names in target_catalog().items() if kind != "groups"
                                  for name in names for item in expand(name)))
    return list(dict.fromkeys([*(item for dependency in component_dependencies(target) for item in expand(dependency)), target]))


def compile_components(components: list[str]) -> None:
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
                linker = "-s -w" + (" -X=aiw/internal/version.Version=snapshot" if versioned else "")
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


def output_files(directory: Path):
    """Walk output without following links or silently skipping unreadable paths."""
    reject_link(directory)
    if not directory.is_dir():
        raise PackageError(f"Output directory missing or invalid: {directory}")

    def failed(error):
        raise error

    for current, directories, names in os.walk(directory, followlinks=False, onerror=failed):
        for name in directories:
            reject_link(Path(current) / name)
        for name in sorted(names):
            path = Path(current) / name
            reject_link(path)
            if not path.is_file():
                raise PackageError(f"Output is not a regular file: {path}")
            yield path


def native_inventory(components: list[str], *, missing: list[str] | None = None) -> dict[str, str]:
    files = []
    for component in components:
        if component in GO:
            files += [safe_path(ROOT, binary(component, system).relative_to(ROOT).as_posix()) for system in SYSTEMS]
        elif component == "http-openai-proxy":
            directory = safe_path(ROOT, "bin/http-openai-proxy")
            try:
                directory.stat()
            except FileNotFoundError:
                if missing is None:
                    raise PackageError("Compiled proxy output missing; run compile again")
                missing.append("http-openai-proxy")
                continue
            paths = list(output_files(directory))
            if not paths:
                if missing is None:
                    raise PackageError("Compiled proxy output is empty; run compile again")
                missing.append("http-openai-proxy")
            files += paths
    inventory = {}
    for path in files:
        try:
            info = path.stat()
        except FileNotFoundError:
            if missing is None:
                raise PackageError(f"Compiled output missing: {path}; run compile again")
            missing.append(path.relative_to(BIN).as_posix())
            continue
        if not stat.S_ISREG(info.st_mode):
            raise PackageError(f"Compiled output is not a regular file: {path}")
        with path.open("rb") as source:
            inventory[path.relative_to(BIN).as_posix()] = hashlib.file_digest(source, "sha256").hexdigest()
    return inventory


def compile_records() -> dict:
    path = safe_path(ROOT, "bin/.compile.json")
    if not path.is_file():
        return {}
    content = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(content, dict) or content.get("schema") != 1 or not isinstance(content.get("records"), dict):
        raise PackageError("Invalid compile records; run clean then compile again")
    records = content["records"]
    for target, record in records.items():
        if not isinstance(target, str) or not isinstance(record, dict):
            raise PackageError("Invalid compile record")
        version, components, outputs = record.get("version"), record.get("components"), record.get("outputs")
        completed = record.get("compiled_at_ns")
        if (not isinstance(version, str) or not isinstance(components, list) or not components
                or any(not isinstance(item, str) for item in components) or not isinstance(outputs, dict)
                or any(not isinstance(name, str) or not isinstance(digest, str)
                       or not re.fullmatch(r"[0-9a-f]{64}", digest) for name, digest in outputs.items())
                or not isinstance(completed, int) or isinstance(completed, bool) or completed <= 0):
            raise PackageError("Invalid compile record; run clean then compile again")
        identifier(version)
    return records


def save_compile_records(records: dict) -> None:
    path = safe_path(ROOT, "bin/.compile.json")
    atomic_bytes(path, json.dumps(dict(schema=1, records=records), indent=2).encode())


def compile_target(target: str) -> None:
    components = expand(target)
    native = {item for item in components if item in GO or item == "http-openai-proxy"}
    # Invalidate before the first compiler can overwrite output or fail midway.
    records = {name: record for name, record in compile_records().items()
               if name != target and not native.intersection(record["components"])}
    save_compile_records(records)
    compile_components(components)
    records[target] = dict(version="snapshot", components=components, outputs=native_inventory(components),
                           compiled_at_ns=time.time_ns())
    save_compile_records(records)


def latest_compile(target: str | None, *, allow_missing: bool = False) -> dict | None:
    required = set(expand(target)) if target is not None else set()
    candidates = [record for record in compile_records().values()
                  if required.issubset(record["components"]) and record["version"] == "snapshot"]
    if not candidates:
        if allow_missing:
            # Even without provenance, unsafe existing output must not be hidden by recompilation.
            native_inventory(list(required), missing=[])
        return None
    record = max(candidates, key=lambda item: item["compiled_at_ns"])
    missing = [] if allow_missing else None
    inventory = native_inventory(record["components"], missing=missing)
    expected = record["outputs"]
    if (inventory.keys() - expected.keys()
            or any(inventory[name] != expected[name] for name in inventory.keys() & expected.keys())):
        raise PackageError("Snapshot output was added or changed; run compile again")
    if missing or expected.keys() - inventory.keys():
        if allow_missing:
            return None
        raise PackageError("Snapshot output missing; run compile again")
    return record


def clean_outputs() -> None:
    root = ROOT.resolve()
    outputs = []
    # Complete preflight of both trees before the first recursive deletion.
    for name in ("bin", "dist"):
        path = safe_path(ROOT, name)
        if path.resolve() != root / name or path.parent.resolve() != root:
            raise PackageError(f"Refusing to clean outside checkout outputs: {path}")
        if path.exists():
            for _ in output_files(path):
                pass
            outputs.append(path)
    for path in outputs:
        shutil.rmtree(path)
        print(f"Cleaned: {path}")
    if not outputs:
        print("Nothing to clean")


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
    if component == "docs" or component in plugin_targets() or component.startswith("skill-"):
        script = ROOT / "scripts/packages" / ("docs" if component == "docs" else ("skill-" + component[6:]) if component.startswith("skill-") else "plugin-aiw-" + component)
        script = script.with_suffix(".py")
        if not script.is_file():
            raise PackageError(f"Independent package script missing: {script}")
        result = subprocess.run([sys.executable, str(script), "--version", version], cwd=ROOT, capture_output=True, text=True, check=False)
        if result.returncode:
            raise PackageError(f"Package script failed: {script}: {result.stderr}")
        print(result.stdout.strip())
        name = "aiw-" + component
        category = "resources" if component == "docs" else "skills" if component.startswith("skill-") else "plugins"
        return DIST / category / version / f"{name}-{version}.zip"
    return build_component(component, version)


def export_repository(packages, root):
    repo = Repo(str(root))
    for package in packages:
        print(repo.publish(package))


def build_packages(target: str, version: str) -> list[Path]:
    identifier(version)
    components = expand(target)
    if latest_compile(target, allow_missing=True) is None:
        print("Snapshot output missing; compiling snapshot", flush=True)
        compile_target(target)
    else:
        print("Using snapshot output", flush=True)
    packages = [package_component(component, version) for component in components]
    by_component = dict(zip(components, packages))
    built_at_ns = time.time_ns()
    for component in components:
        write_manifest(component, version, [by_component[item] for item in expand(component)], built_at_ns=built_at_ns)
    write_manifest(target, version, packages, built_at_ns=built_at_ns)
    if target == "all":
        for group in ("bin", "plugins", "skills"):
            write_manifest(group, version, [by_component[item] for item in expand(group)],
                           built_at_ns=built_at_ns)
    return packages


def build_version(explicit: str | None) -> str:
    version = explicit if explicit is not None else os.environ.get("AIW_VERSION", "").strip()
    if explicit is None and not version:
        version = "snapshot-" + datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S%fZ")
    identifier(version)
    return version


def latest_version(target: str | None) -> str:
    directory = DIST / ".manifests"
    paths = [directory / (target + ".json")] if target is not None else sorted(directory.glob("*.json"))
    if not paths or any(not path.is_file() for path in paths):
        raise PackageError(f"No build recorded for {target or 'any target'}; run build first")
    candidates = []
    for path in paths:
        content = json.loads(path.read_text(encoding="utf-8"))
        if not isinstance(content, dict):
            raise PackageError(f"Invalid build manifest: {path}")
        completed = content.get("built_at_ns", path.stat().st_mtime_ns)
        if not isinstance(completed, int) or isinstance(completed, bool) or completed <= 0:
            raise PackageError(f"Invalid build timestamp: {path}")
        candidates.append((completed, path.name, content))
    content = max(candidates, key=lambda candidate: candidate[:2])[2]
    version, packages = content.get("version"), content.get("packages")
    if not isinstance(version, str) or not isinstance(packages, list) or not packages:
        raise PackageError("Invalid latest build manifest: version and packages are required")
    identifier(version)
    for package in packages:
        if not isinstance(package, str) or not safe_path(DIST, package).is_file():
            raise PackageError(f"Latest build package missing or invalid: {package}")
    return version


def publish_destination() -> Path:
    address = os.environ.get("AIW_PACKAGE_REPO", "").strip()
    if not address:
        raise PackageError("Set AIW_PACKAGE_REPO in .env or the process environment before publishing")
    if Repo(address).kind != "local":
        raise PackageError("AIW_PACKAGE_REPO must be a local repository path")
    destination = Path(address).expanduser()
    return (destination if destination.is_absolute() else ROOT / destination).resolve()


def main(arguments: list[str]) -> int:
    try:
        load_environment()
    except (PackageError, OSError, ValueError) as error:
        print(f"build.py: {error}", file=sys.stderr)
        return 1
    parser = argparse.ArgumentParser(description=__doc__, allow_abbrev=False)
    actions = parser.add_subparsers(dest="action")
    aliases = package_aliases()
    packages = set(aliases.values())
    choices = sorted(packages | set(TARGETS["groups"]) | set(aliases))
    for action in ("compile", "build", "install", "export"):
        command = actions.add_parser(action, allow_abbrev=False)
        command.add_argument("target", choices=choices)
        if action == "install":
            command.add_argument("--dest", type=Path, default=default_destination())
        elif action == "export":
            command.add_argument("--repository", type=Path, required=True)
        elif action == "build":
            command.add_argument("--version", help="Package version; otherwise AIW_VERSION or UTC timestamp. Compilation always uses snapshot")
    for action in ("latest", "latest-compile"):
        command = actions.add_parser(action, help="Print latest package version" if action == "latest" else "Validate current compile output and print snapshot", allow_abbrev=False)
        command.add_argument("target", nargs="?", choices=choices)
    command = actions.add_parser("clean", help="Remove checkout bin/ and dist/ after path/link preflight", allow_abbrev=False)
    command.set_defaults(target=None)
    command = actions.add_parser("publish", help="Publish an existing package version and its dependencies locally", allow_abbrev=False)
    command.add_argument("target", metavar="package", choices=sorted(packages | set(aliases)))
    command.add_argument("version")
    command = actions.add_parser("publish-all", help="Publish all existing packages at a version locally", allow_abbrev=False)
    command.add_argument("version")
    args = parser.parse_args(arguments)
    if not args.action:
        parser.print_help()
        return 0
    try:
        target = "all" if args.action == "publish-all" else aliases.get(args.target, args.target)
        if args.action == "latest":
            print(latest_version(target))
        elif args.action == "latest-compile":
            record = latest_compile(target)
            if record is None:
                raise PackageError(f"No reusable compile recorded for {target or 'any target'}; run compile first")
            print(record["version"])
        elif args.action == "clean":
            clean_outputs()
        elif args.action in {"publish", "publish-all"}:
            from scripts.local_packages import LocalPackages
            destination = publish_destination()
            name = None if args.action == "publish-all" else next(name for name, component in aliases.items() if component == target)
            release = LocalPackages.from_release(DIST, args.version, name)
            packages = [release.fetch(entry, None) for entry in release.index()]
            export_repository(packages, destination)
        elif args.action == "compile":
            print("Compiling snapshot", flush=True)
            compile_target(target)
        elif args.action == "build":
            version = build_version(args.version)
            print(f"Version: {version}", flush=True)
            build_packages(target, version)
        elif args.action in {"install", "export"}:
            manifest = DIST / ".manifests" / (target + ".json")
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
                from scripts.local_packages import LocalPackages
                repo = LocalPackages(DIST, packages)
                roots = repo.roots
                if target not in TARGETS["groups"]:
                    names = {name for name, component in aliases.items() if component == target}
                    roots = [entry for entry in roots if entry["name"] in names and entry["version"] == content["version"]]
                    if len(roots) != 1:
                        raise PackageError(f"Expected one manifest root for {target}; rebuild the target")
                repo.preflight_destination(args.dest)
                manager = Manager(args.dest)
                print(Operation(manager, repo).install(packages=[repo.fetch(entry, None) for entry in roots]))
        return 0
    except (PackageError, OSError, ValueError, KeyError, TypeError, zipfile.BadZipFile) as error:
        print(f"build.py: {error}", file=sys.stderr)
        return 1


def write_manifest(target: str, version: str, packages: list[Path], *, built_at_ns: int | None = None) -> None:
    content = dict(version=version, packages=[path.relative_to(DIST).as_posix() for path in packages],
                   built_at_ns=time.time_ns() if built_at_ns is None else built_at_ns,
                   compile_version="snapshot")
    atomic_bytes(DIST / ".manifests" / (target + ".json"), json.dumps(content, indent=2).encode())


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
