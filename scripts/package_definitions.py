"""Shared definitions invoked by each component's explicit packaging script."""
from __future__ import annotations

import argparse
from pathlib import Path
import sys
import tomllib

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))
sys.path.insert(0, str(ROOT / "src/plugins/aiw-pkg"))
from archive import PackageError, write_package
import build as context


def ref(name, version):
    return dict(name=name, version=version)


def metadata(kind):
    path = ROOT / "src" / kind / ("plugin-meta.toml" if kind == "plugins" else "skills-meta.toml")
    data = tomllib.loads(path.read_text(encoding="utf-8"))["packages"]
    core, extension = data["core"], data["extension"]
    if not isinstance(core, list) or not isinstance(extension, list) or len(set(core + extension)) != len(core + extension):
        raise PackageError(f"Invalid release metadata: {path}")
    return core, extension


def collection_members(name):
    plugins, plugin_extensions = metadata("plugins")
    skills, skill_extensions = metadata("skills")
    resources = ["aiw-resource-" + p.stem for p in sorted((ROOT / "src/skills").glob("*.md"))]
    if name.startswith("plugins-"):
        return plugins + (plugin_extensions if name.endswith("all") else [])
    if name.startswith("skills-"):
        return ["aiw-skill-" + s for s in skills + (skill_extensions if name.endswith("all") else [])] + resources
    if name == "aiw-core":
        return ["aiw-main", "plugins-core", "skills-core"]
    if name == "aiw-all":
        return ["aiw-main", "plugins-all", "skills-all", "aiw-gateway", "aiw-http-openai-proxy"]
    raise PackageError(f"Unknown release collection: {name}")


COLLECTIONS = ("plugins-core", "plugins-all", "skills-core", "skills-all", "aiw-core", "aiw-all")

# Source shims keep one implementation; release packages carry private real files.
PLUGIN_PRIVATE_FILES = {
    "file": ("aiw_codec.py",),
    "patch": ("aiw_codec.py",),
    "notify": ("send_teams_msg.py",),
}


def component_dependencies(component):
    if component == "aiw":
        return ["resource-" + p.stem for p in sorted((ROOT / "src/skills").glob("*.md"))]
    if component.startswith("skill-"):
        text = (ROOT / "src/skills" / component[6:] / "SKILL.md").read_text(encoding="utf-8")
        result = ["resource-" + p.stem for p in sorted((ROOT / "src/skills").glob("*.md")) if "skills/" + p.name in text]
        result += [package for command, package in (("aiw git", "git"), ("aiw fd", "fd"), ("aiw issue", "req")) if command in text]
        return result
    return []


def usage_document_owner(stem):
    """Give nested package names priority over their shorter document prefixes."""
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


def build_component(component, version):
    c = context
    name, kind, category, entries, dependencies, members = "aiw-" + component, "plugin", "plugins", [], [], []
    if component in COLLECTIONS:
        name, kind, category = component, "collection", "collections"
        members = [ref(n, version) for n in collection_members(component)]
    elif component.startswith("resource-"):
        stem = component[len("resource-"):]
        name, kind, category = "aiw-resource-" + stem, "resource", "resources"
        entries = [c.one(c.SRC / "skills" / (stem + ".md"), "skills/" + stem + ".md")]
    elif component == "plugin-runtime":
        name, kind, category = "aiw-plugin-runtime", "resource", "resources"
        entries = [c.one(p, "plugins/" + p.name) for p in sorted((c.SRC / "plugins").glob("*.py")) if not p.name.startswith("aiw-")]
    elif component == "docs":
        name, kind, category = "aiw-docs", "resource", "resources"
        entries = c.tree_entries(c.ROOT / "docs/usage", "docs/reference/usage")
        readme = c.ROOT / "README.md"
        if readme.is_file():
            entries.append(c.one(readme, "docs/reference/README.md"))
    elif component.startswith("skill-"):
        skill = component[len("skill-"):]
        name, kind, category = "aiw-skill-" + skill, "skill", "skills"
        directory = c.SRC / "skills" / skill
        entries = c.tree_entries(directory, "skills/" + skill)
        text = (directory / "SKILL.md").read_text(encoding="utf-8")
        dependencies = [ref("aiw-resource-" + p.stem, version) for p in sorted((c.SRC / "skills").glob("*.md"))
                        if "skills/" + p.name in text]
        for command, package in (("aiw git", "aiw-git"), ("aiw fd", "aiw-fd"), ("aiw issue", "aiw-req")):
            if command in text:
                dependencies.append(ref(package, version))
    elif component == "aiw":
        name, kind, category = "aiw-main", "program", "mist"
        entries = [c.one(c.binary("aiw", system), "aiw.exe" if system == "windows" else "aiw", executable=True, system=system)
                   for system in c.SYSTEMS]
        entries += c.tree_entries(c.SRC / "agent-templates", "agent-templates")
        entries.append(c.one(c.ROOT / "aiw.toml.example", "aiw.toml.example"))
        entries.append(c.one(c.ROOT / "aiw.toml.example", "aiw.toml", config=True))
        dependencies = [ref("aiw-resource-" + p.stem, version) for p in sorted((c.SRC / "skills").glob("*.md"))]
    elif component in {"gateway", "http-openai-proxy"}:
        return c.legacy_package_component(component, version)
    else:
        directory = c.SRC / "plugins" / name
        if directory.is_dir():
            private_files = PLUGIN_PRIVATE_FILES.get(component, ())
            excluded = set(private_files) | (set(c.SAY_PROGRAM_ASSETS) if component == "say" else set())
            entries = c.tree_entries(directory, "plugins/" + name,
                                     exclude=excluded)
            entries += [c.one(c.SRC / "plugins" / filename, f"plugins/{name}/{filename}")
                        for filename in private_files]
            if component in c.GO:
                for system in c.SYSTEMS:
                    binary = c.binary(component, system)
                    entries.append(c.one(binary, "plugins/" + name + "/" + binary.name, executable=True, system=system))
            if component == "say":
                source = c.SRC / "programs/aiw-say"
                for filename in ("aiw.toml.example", "aiw-say-hotkeys.ahk", "aiw-say-hotkeys.ps1"):
                    entries.append(c.one(source / filename, "plugins/aiw-say/" + filename))
                entries += c.tree_entries(source / "profiles", "plugins/aiw-say/profiles")
                entries.append(c.one(source / "aiw.toml.example", "plugins/aiw-say/aiw.toml", config=True))
        else:
            source = c.SRC / "plugins" / (name + ".py")
            if component in {"ai-code-index", "ai-gen-index"}:
                source = c.SRC / "programs/ai-code-tools" / ("ai-code-index" if component == "ai-code-index" else "generate-ai-index")
            entries = [c.one(source, "plugins/" + name + ".py")]
    usage = c.ROOT / "docs/usage"
    if kind not in {"collection", "skill", "resource"}:
        for doc in sorted(usage.glob("*.md")):
            if usage_document_owner(doc.stem) == component:
                entries.append(c.one(doc, "docs/usage/" + doc.name))
        if component == "aiw":
            for filename in ("cli-reference.md", "aiw-ask.md", "aiw-auxiliary.md", "requirement-discovery-review.md"):
                entries.append(c.one(usage / filename, "docs/usage/" + filename))
    output = c.DIST / category / version / f"{name}-{version}.zip"
    write_package(output, name, version, kind, entries, dependencies=dependencies, members=members)
    if component == "aiw":
        for system in c.SYSTEMS:
            source = c.binary(component, system)
            destination = c.DIST / "bin" / (system + "-amd64") / source.name
            destination.parent.mkdir(parents=True, exist_ok=True)
            c.shutil.copy2(source, destination)
    print(f"Built: {output}")
    return output


def run(component):
    parser = argparse.ArgumentParser(description="Build exactly one package")
    try:
        context.load_environment()
        default_version = context.os.environ.get("AIW_VERSION")
        parser.add_argument("--version", default=default_version, required=default_version is None)
        args = parser.parse_args()
        context.identifier(args.version)
        build_component(component, args.version)
    except (PackageError, OSError, ValueError) as error:
        parser.exit(1, f"package: {error}\n")
