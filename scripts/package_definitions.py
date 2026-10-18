"""Build the program packages and the two complete plugin/skill bundles."""
from __future__ import annotations

import argparse
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))
sys.path.insert(0, str(ROOT / "src/plugins/aiw-pkg"))
from archive import PackageError, write_package
import build as context

COLLECTIONS = ("aiw-core", "aiw-all")
PLUGIN_PRIVATE_FILES = {
    "file": ("aiw_codec.py",),
    "patch": ("aiw_codec.py",),
    "notify": ("send_teams_msg.py",),
}


def ref(name, version):
    return dict(name=name, version=version)


def collection_members(name):
    members = ["aiw-main", "aiw-plugins", "aiw-skills"]
    if name == "aiw-core":
        return members
    if name == "aiw-all":
        return members + ["aiw-gateway", "aiw-http-openai-proxy"]
    raise PackageError(f"Unknown release collection: {name}")


def usage_document_owner(stem):
    """Select a source component's usage documents before aggregating plugins."""
    matches = []
    for component in ["aiw", *context.plugin_targets()]:
        name = "aiw-main" if component == "aiw" else "aiw-" + component
        prefixes = [name]
        if component == "cz":
            prefixes.append("cz")
        elif component == "req":
            prefixes.append("aiw-issue")
        for prefix in prefixes:
            if stem == prefix or stem.startswith(prefix + "-"):
                matches.append((len(prefix), component))
    return max(matches)[1] if matches else None


def plugin_entries(component):
    """Collect one source plugin, preserving native/config/private-helper rules."""
    c = context
    name = "aiw-" + component
    directory = c.SRC / "plugins" / name
    if directory.is_dir():
        private_files = PLUGIN_PRIVATE_FILES.get(component, ())
        excluded = set(private_files) | (set(c.SAY_PROGRAM_ASSETS) if component == "say" else set())
        entries = c.tree_entries(directory, "plugins/" + name, exclude=excluded)
        entries += [c.one(c.SRC / "plugins" / filename, f"plugins/{name}/{filename}")
                    for filename in private_files]
        if component in c.GO:
            for system in c.SYSTEMS:
                native = c.binary(component, system)
                entries.append(c.one(native, "plugins/" + name + "/" + native.name,
                                     executable=True, system=system))
        if component == "say":
            source = c.SRC / "programs/aiw-say"
            for filename in ("aiw.toml.example", "aiw-say-hotkeys.ahk", "aiw-say-hotkeys.ps1"):
                entries.append(c.one(source / filename, "plugins/aiw-say/" + filename))
            entries += c.tree_entries(source / "profiles", "plugins/aiw-say/profiles")
            entries.append(c.one(source / "aiw.toml.example", "plugins/aiw-say/aiw.toml", config=True))
    else:
        source = c.SRC / "plugins" / (name + ".py")
        if component in {"ai-code-index", "ai-gen-index"}:
            source = c.SRC / "programs/ai-code-tools" / (
                "ai-code-index" if component == "ai-code-index" else "generate-ai-index")
        entries = [c.one(source, "plugins/" + name + ".py")]
    for doc in sorted((c.ROOT / "docs/usage").glob("*.md")):
        if usage_document_owner(doc.stem) == component:
            entries.append(c.one(doc, "docs/usage/" + doc.name))
    return entries


def build_component(component, version):
    c = context
    name = "aiw-main" if component == "aiw" else "aiw-" + component
    members, entries = [], []
    if component in COLLECTIONS:
        name, kind, category = component, "collection", "collections"
        members = [ref(member, version) for member in collection_members(component)]
    elif component == "plugins":
        kind, category = "plugin", "plugins"
        entries = [entry for plugin in c.plugin_targets() for entry in plugin_entries(plugin)]
    elif component == "skills":
        kind, category = "skill", "skills"
        entries = c.tree_entries(c.SRC / "skills", "skills")
    elif component == "docs":
        kind, category = "resource", "resources"
        entries = c.tree_entries(c.ROOT / "docs/usage", "docs/reference/usage")
        readme = c.ROOT / "README.md"
        if readme.is_file():
            entries.append(c.one(readme, "docs/reference/README.md"))
    elif component == "aiw":
        kind, category = "program", "mist"
        entries = [c.one(c.binary("aiw", system), "aiw.exe" if system == "windows" else "aiw",
                         executable=True, system=system) for system in c.SYSTEMS]
        entries += c.tree_entries(c.SRC / "agent-templates", "agent-templates")
        entries.append(c.one(c.ROOT / "aiw.toml.example", "aiw.toml.example"))
        entries.append(c.one(c.ROOT / "aiw.toml.example", "aiw.toml", config=True))
        for doc in sorted((c.ROOT / "docs/usage").glob("*.md")):
            if usage_document_owner(doc.stem) == "aiw":
                entries.append(c.one(doc, "docs/usage/" + doc.name))
        for filename in ("cli-reference.md", "aiw-ask.md", "aiw-auxiliary.md", "requirement-discovery-review.md"):
            entries.append(c.one(c.ROOT / "docs/usage" / filename, "docs/usage/" + filename))
    elif component in {"gateway", "http-openai-proxy"}:
        return c.legacy_package_component(component, version)
    else:
        raise PackageError(f"Unknown package target: {component}")
    output = c.DIST / category / version / f"{name}-{version}.zip"
    write_package(output, name, version, kind, entries, dependencies=[], members=members)
    if component == "aiw":
        for system in c.SYSTEMS:
            source = c.binary("aiw", system)
            destination = c.DIST / "bin" / (system + "-amd64") / source.name
            destination.parent.mkdir(parents=True, exist_ok=True)
            c.shutil.copy2(source, destination)
    print(f"Built: {output}")
    return output


def run(component):
    parser = argparse.ArgumentParser(description="Build exactly one release package")
    try:
        context.load_environment()
        default_version = context.os.environ.get("AIW_VERSION")
        parser.add_argument("--version", default=default_version, required=default_version is None)
        args = parser.parse_args()
        context.identifier(args.version)
        build_component(component, args.version)
    except (PackageError, OSError, ValueError) as error:
        parser.exit(1, f"package: {error}\n")
