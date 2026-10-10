from __future__ import annotations

import os
import shutil
import subprocess
import sys
from pathlib import Path
from typing import Dict, List, Optional, Set

ROOT = Path(__file__).resolve().parent
SRC = ROOT / "src"
BIN = ROOT / "bin"
DIST = ROOT / "dist"
INSTALL_DIR = Path(os.environ.get("AIW_INSTALL_DIR", r"C:\green\aiw"))
ACTIONS = ("windows", "linux", "bin", "req", "say", "gateway", "cz", "plugins", "docs", "skills", "all")


def go_environment(**overrides: str) -> Dict[str, str]:
    environment = os.environ.copy()
    environment.update({"GOPROXY": "off", "GOSUMDB": "off", "GOTOOLCHAIN": "local", "GOFLAGS": ""})
    environment.update(overrides)
    return environment


def run(command: List[str], *, cwd: Path, environment: Dict[str, str]) -> bool:
    try:
        result = subprocess.run(command, cwd=cwd, env=environment, check=False)
    except OSError as error:
        print(f"Error: could not run {command[0]}: {error}", file=sys.stderr)
        return False
    return result.returncode == 0


def go_command() -> Optional[str]:
    command = shutil.which("go")
    if command is None:
        print("Error: Go is not available on PATH.", file=sys.stderr)
    return command


def build_go(target: str, output: Path, *, cwd: Path, goos: str, versioned: bool) -> bool:
    go = go_command()
    if go is None:
        return False
    output.parent.mkdir(parents=True, exist_ok=True)
    linker = "-s -w"
    if versioned:
        linker += f" -X=aiw/internal/version.Version={os.environ.get('AIW_VERSION', 'dev')}"
    environment = go_environment(GOOS=goos, GOARCH="amd64")
    command = [go, "build", "-trimpath", "-ldflags=" + linker, "-o", str(output), target]
    return run(command, cwd=cwd, environment=environment)


def build_main(goos: str) -> bool:
    suffix = ".exe" if goos == "windows" else ""
    output = BIN / f"aiw-{goos}-amd64{suffix}"
    if not build_go("./cmd/aiw", output, cwd=SRC, goos=goos, versioned=True):
        print(f"Error: {goos.capitalize()} build failed.", file=sys.stderr)
        return False
    destination = INSTALL_DIR / ("aiw.exe" if goos == "windows" else "aiw")
    shutil.copy2(output, destination)
    print(f"Installation complete. aiw is now available in {INSTALL_DIR}.")
    return True


def build_plugin_binary(name: str, target: str, goos: str) -> bool:
    suffix = ".exe" if goos == "windows" else ""
    output = DIST / "plugins" / f"aiw-{name}" / f"aiw-{name}{suffix}"
    if not build_go(target, output, cwd=SRC, goos=goos, versioned=True):
        print(f"Error: {goos.capitalize()} {name} build failed.", file=sys.stderr)
        return False
    return True


def build_req() -> bool:
    if not build_plugin_binary("req", "./cmd/aiw-req", "windows"):
        return False
    if not build_plugin_binary("req", "./cmd/aiw-req", "linux"):
        return False
    req_stage = DIST / "plugins" / "aiw-req"
    print(f"Req plugin binaries staged in {req_stage}; use 'python build.py plugins' or 'all' to install them.")
    return True


def build_say() -> bool:
    if not build_plugin_binary("say", "./cmd/aiw-say", "windows"):
        return False
    if not build_plugin_binary("say", "./cmd/aiw-say", "linux"):
        return False
    say_source = SRC / "programs" / "aiw-say"
    say_stage = DIST / "plugins" / "aiw-say"
    shutil.copy2(say_source / "aiw.toml.example", say_stage / "aiw.toml.example")
    shutil.copytree(say_source / "profiles", say_stage / "profiles", dirs_exist_ok=True)
    print(f"Say plugin binaries and samples staged in {say_stage}; use 'python build.py plugins' or 'all' to install them.")
    return True


def build_gateway() -> bool:
    gateway = SRC / "programs" / "agent-gateway"
    for goos, suffix in (("windows", ".exe"), ("linux", "")):
        output = BIN / f"agent-gateway{suffix}"
        go = go_command()
        if go is None or not run(
            [go, "build", "-trimpath", "-ldflags=-s -w", "-o", str(output), "."],
            cwd=gateway,
            environment=go_environment(GOOS=goos, GOARCH="amd64"),
        ):
            print(f"Error: {goos.capitalize()} agent-gateway build failed.", file=sys.stderr)
            return False
    print("Agent Gateway standalone binaries built in bin.")
    shutil.copy2(gateway / "gateway-example.json", INSTALL_DIR / "gateway-new.json")
    for name in ("agent-gateway.exe", "agent-gateway"):
        shutil.copy2(BIN / name, INSTALL_DIR / name)
    print(f"Gateway installed in {INSTALL_DIR}; existing gateway.json preserved.")
    return True


def build_cz() -> bool:
    source = SRC / "plugins" / "aiw-cz"
    release = DIST / "plugins" / "aiw-cz" / "release"
    if release.exists():
        shutil.rmtree(release)
    release.mkdir(parents=True)
    for pattern in ("aiw-cz.py", "cz_*.py", "requirements.txt"):
        for path in source.glob(pattern):
            shutil.copy2(path, release / path.name)
    shutil.copytree(source / "locales", release / "locales", dirs_exist_ok=True)
    print(f"Python cz plugin release staged in {release}; use 'python build.py plugins' or 'all' to install it.")
    return True


def copy_tree(source: Path, destination: Path, *, ignore: Optional[Set[str]] = None) -> None:
    ignored = ignore or set()
    shutil.copytree(
        source,
        destination,
        dirs_exist_ok=True,
        ignore=lambda _directory, names: [name for name in names if name in ignored],
    )


def install_plugins() -> bool:
    if not build_req() or not build_say() or not build_cz():
        return False
    install_plugins_dir = INSTALL_DIR / "plugins"
    install_plugins_dir.mkdir(parents=True, exist_ok=True)
    copy_tree(
        SRC / "plugins",
        install_plugins_dir,
        ignore={"aiw-cz", "aiw-say", "aiw-gw"},
    )
    copy_tree(DIST / "plugins" / "aiw-req", install_plugins_dir / "aiw-req")
    copy_tree(DIST / "plugins" / "aiw-say", install_plugins_dir / "aiw-say", ignore={"aiw.toml"})
    copy_tree(
        DIST / "plugins" / "aiw-cz" / "release",
        install_plugins_dir / "aiw-cz",
        ignore={"cz.toml", ".cz.toml"},
    )
    print(f"Plugins installed in {install_plugins_dir}: repository plugins, req, say, and cz.")
    return True


def install_docs() -> bool:
    usage_destination = INSTALL_DIR / "docs" / "usage"
    templates_destination = INSTALL_DIR / "agent-templates"
    copy_tree(ROOT / "docs" / "usage", usage_destination)
    copy_tree(SRC / "agent-templates", templates_destination)
    print(f"Usage documentation installed in {usage_destination}.")
    print(f"Agent templates installed in {templates_destination}.")
    return True


def install_skills() -> bool:
    destination = INSTALL_DIR / "skills"
    copy_tree(SRC / "skills", destination)
    print(f"Skills installed in {destination}.")
    return True


def run_action(action: str) -> bool:
    actions = {
        "windows": lambda: build_main("windows"),
        "linux": lambda: build_main("linux"),
        "req": build_req,
        "say": build_say,
        "gateway": build_gateway,
        "cz": build_cz,
        "plugins": install_plugins,
        "docs": install_docs,
        "skills": install_skills,
        "bin": lambda: build_main("windows") and build_main("linux") and build_gateway(),
        "all": lambda: build_main("windows") and build_main("linux") and build_gateway() and install_plugins() and install_docs() and install_skills(),
    }
    operation = actions.get(action.lower())
    if operation is None:
        print(f"Error: Unknown build action: {action}", file=sys.stderr)
        print("Run build.py --help for available actions.", file=sys.stderr)
        return False
    try:
        return operation()
    except OSError as error:
        print(f"Error: action {action} failed: {error}", file=sys.stderr)
        return False


def show_help() -> None:
    print(f"Usage: build.py [{' '.join(ACTIONS)}]")
    print("\nActions can be combined and run in the specified order.")
    print("With no arguments, only the Windows build is performed.\n")
    print("  windows  Build and install the Windows executable.")
    print("  linux    Build and install the Linux executable.")
    print("  bin      Build and install AIW and Gateway binaries; preserve Gateway config.")
    print("  req      Build Windows and Linux req plugin binaries.")
    print("  say      Build Windows and Linux say binaries and configuration samples.")
    print("  gateway  Build and install standalone Windows and Linux Gateway binaries beside AIW.")
    print("  cz       Prepare the Python cz release.")
    print("  plugins  Build and install plugins, including Say; preserve Say aiw.toml.")
    print("  docs     Copy usage documentation to the install directory.")
    print("  skills   Install skills individually.")
    print("  all      Build and install AIW, standalone Gateway, plugins, docs, and skills.")
    print("\nHelp: -h, --help, help, /h, /?")


def main(arguments: List[str]) -> int:
    if arguments and arguments[0].lower() in {"-h", "--help", "help", "/h", "/?"}:
        show_help()
        return 0
    BIN.mkdir(exist_ok=True)
    INSTALL_DIR.mkdir(parents=True, exist_ok=True)
    actions = arguments or ["windows"]
    for action in actions:
        if not run_action(action):
            return 1
        print(f"Action completed: {action}.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
