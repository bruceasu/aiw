#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
generate-ai-index v1.2.0

Generate AI-friendly repository indexes.

Markdown:
  .ai/PROJECT_MAP.md
  .ai/API_INDEX.md

JSONL:
  .ai/symbols.jsonl
  .ai/apis.jsonl
  .ai/files.jsonl
  .ai/metadata.json

V1.2.0 additions:
  - line_start / line_end for symbols
  - line for API route annotation / route declaration
  - stable-ish symbol id
  - location-aware Markdown output

No third-party dependencies.
"""

from __future__ import annotations

import argparse
import json
import re
from collections import Counter
from dataclasses import dataclass, field, asdict
from datetime import datetime, timezone
from pathlib import Path
from typing import Iterable


VERSION = "1.2.0"

IGNORE_DIRS = {
    ".git", ".idea", ".vscode", ".gradle", "target", "build", "dist", "out",
    "node_modules", ".next", ".nuxt", ".turbo", ".cache", "__pycache__",
    "venv", ".venv", "coverage", ".pytest_cache", ".ai"
}

SOURCE_EXTS = {".java", ".go", ".py", ".js", ".jsx", ".ts", ".tsx"}

EXT_LANGUAGE = {
    ".java": "java",
    ".go": "go",
    ".py": "python",
    ".js": "javascript",
    ".jsx": "javascript",
    ".ts": "typescript",
    ".tsx": "typescript",
}


@dataclass
class Symbol:
    type: str
    name: str
    file: str
    kind: str = ""
    owner: str = ""
    signature: str = ""
    package: str = ""
    language: str = ""
    line_start: int | None = None
    line_end: int | None = None
    id: str = ""
    keywords: list[str] = field(default_factory=list)


@dataclass
class ApiRoute:
    method: str
    path: str
    handler: str
    file: str
    framework: str = ""
    repo: str = ""
    line: int | None = None
    id: str = ""
    keywords: list[str] = field(default_factory=list)


@dataclass
class FileRecord:
    file: str
    kind: str
    language: str
    line_count: int = 0
    keywords: list[str] = field(default_factory=list)


@dataclass
class Index:
    repo: str = ""
    modules: list[str] = field(default_factory=list)
    symbols: list[Symbol] = field(default_factory=list)
    routes: list[ApiRoute] = field(default_factory=list)
    files: list[FileRecord] = field(default_factory=list)


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def read_text(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        return path.read_text(encoding="utf-8", errors="ignore")


def rel(path: Path, root: Path) -> str:
    return path.relative_to(root).as_posix()


def should_skip(path: Path) -> bool:
    return any(part in IGNORE_DIRS for part in path.parts)


def walk_sources(root: Path) -> Iterable[Path]:
    for path in root.rglob("*"):
        if path.is_file() and path.suffix in SOURCE_EXTS and not should_skip(path):
            yield path


def line_no(text: str, pos: int) -> int:
    return text.count("\n", 0, pos) + 1


def find_block_end_line(text: str, open_brace_pos: int) -> int:
    """
    Best-effort brace matching for Java/Go/JS-like syntax.
    Returns line number of the matching closing brace, or open line if not found.
    """
    depth = 0
    in_string = None
    escape = False
    for i in range(open_brace_pos, len(text)):
        ch = text[i]
        if in_string:
            if escape:
                escape = False
            elif ch == "\\":
                escape = True
            elif ch == in_string:
                in_string = None
            continue
        if ch in ("'", '"', "`"):
            in_string = ch
            continue
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                return line_no(text, i)
    return line_no(text, open_brace_pos)


def find_python_block_end_line(text: str, start_pos: int) -> int:
    lines = text.splitlines()
    start_line = line_no(text, start_pos)
    if start_line > len(lines):
        return start_line

    start_text = lines[start_line - 1]
    base_indent = len(start_text) - len(start_text.lstrip(" "))

    end = start_line
    for i in range(start_line, len(lines)):
        raw = lines[i]
        stripped = raw.strip()
        if not stripped or stripped.startswith("#"):
            end = i + 1
            continue
        indent = len(raw) - len(raw.lstrip(" "))
        if indent <= base_indent:
            break
        end = i + 1
    return end


def split_camel(s: str) -> list[str]:
    if not s:
        return []
    s = re.sub(r"([a-z0-9])([A-Z])", r"\1 \2", s)
    s = re.sub(r"[^A-Za-z0-9_/\-{}:.]+", " ", s)
    parts = []
    for token in s.replace("/", " ").replace("-", " ").replace("_", " ").replace(".", " ").split():
        token = token.strip()
        if token:
            parts.append(token)
    return parts


def uniq(items: Iterable[str]) -> list[str]:
    seen = set()
    out = []
    for x in items:
        if x is None:
            continue
        x = str(x).strip()
        if not x:
            continue
        key = x.lower()
        if key not in seen:
            seen.add(key)
            out.append(x)
    return out


def normalize_path(p: str) -> str:
    if not p:
        return "/"
    p = p.strip().strip('"').strip("'").strip("`")
    p = re.sub(r"\s*\+\s*", "", p)
    p = p.replace("//", "/")
    if not p.startswith("/"):
        p = "/" + p
    return p


def join_paths(a: str, b: str) -> str:
    a = normalize_path(a)
    b = normalize_path(b)
    if a == "/":
        return b
    if b == "/":
        return a
    return (a.rstrip("/") + "/" + b.lstrip("/")).replace("//", "/")


def symbol_keywords(*values: str) -> list[str]:
    tokens = []
    for v in values:
        tokens.extend(split_camel(v or ""))
        if v:
            tokens.append(v)
    return uniq(tokens)


def api_keywords(method: str, path: str, handler: str, framework: str = "") -> list[str]:
    tokens = [method, path, handler, framework]
    tokens.extend(split_camel(path))
    tokens.extend(split_camel(handler))
    return uniq(tokens)


def short_signature(signature: str) -> str:
    return re.sub(r"\s+", " ", signature or "").strip()


def make_symbol_id(language: str, owner: str, name: str, signature: str, file: str, line: int | None) -> str:
    loc = str(line or 0)
    core = f"{owner}#{signature}" if owner else (signature or name)
    raw = f"{language}:{core}:{file}:{loc}"
    return re.sub(r"\s+", "", raw)


def make_api_id(method: str, path: str, handler: str, file: str, line: int | None) -> str:
    raw = f"api:{method}:{path}:{handler}:{file}:{line or 0}"
    return re.sub(r"\s+", "", raw)


# -------------------------
# Module detection
# -------------------------

def detect_modules(root: Path) -> list[str]:
    modules = []
    for marker in ["pom.xml", "build.gradle", "build.gradle.kts", "package.json", "pyproject.toml", "go.mod", "bun.lockb", "bun.lock"]:
        for f in root.rglob(marker):
            if should_skip(f):
                continue
            parent = f.parent
            name = "." if parent == root else rel(parent, root)
            modules.append(name)
    return sorted(set(modules))


# -------------------------
# Java / Spring
# -------------------------

JAVA_CLASS_RE = re.compile(
    r"\b(public\s+)?(abstract\s+|final\s+)?(class|interface|enum|record)\s+([A-Za-z_][A-Za-z0-9_]*)"
)

JAVA_PACKAGE_RE = re.compile(r"^\s*package\s+([A-Za-z0-9_.]+)\s*;", re.MULTILINE)

JAVA_METHOD_RE = re.compile(
    r"(?:public|private|protected)?\s*(?:static\s+)?(?:final\s+)?"
    r"[A-Za-z0-9_<>\[\].?,\s]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^;{}]*)\)\s*(?:throws\s+[^{]+)?\{",
    re.MULTILINE
)

SPRING_STEREOTYPES = {
    "RestController": "Controller",
    "Controller": "Controller",
    "Service": "Service",
    "Repository": "Repository",
    "Component": "Component",
    "Configuration": "Configuration",
}

SPRING_MAPPING_METHODS = {
    "GetMapping": "GET",
    "PostMapping": "POST",
    "PutMapping": "PUT",
    "DeleteMapping": "DELETE",
    "PatchMapping": "PATCH",
    "RequestMapping": "ANY",
}


def extract_annotation_value(annotation_text: str) -> str:
    m = re.search(r'\(\s*["`]([^"`]+)["`]\s*\)', annotation_text)
    if m:
        return m.group(1)
    m = re.search(r'\b(?:value|path)\s*=\s*(?:\{\s*)?["`]([^"`]+)["`]', annotation_text)
    if m:
        return m.group(1)
    return "/"


def extract_request_methods(annotation_text: str) -> list[str]:
    methods = re.findall(r"RequestMethod\.([A-Z]+)", annotation_text)
    return methods or ["ANY"]


def parse_java(path: Path, root: Path, index: Index) -> None:
    text = read_text(path)
    rpath = rel(path, root)

    package = ""
    pm = JAVA_PACKAGE_RE.search(text)
    if pm:
        package = pm.group(1)

    class_match = JAVA_CLASS_RE.search(text)
    if not class_match:
        return

    class_name = class_match.group(4)
    class_line = line_no(text, class_match.start())

    class_open = text.find("{", class_match.end())
    class_end = find_block_end_line(text, class_open) if class_open >= 0 else class_line

    before_class = text[:class_match.start()]
    recent_annotations = "\n".join(before_class.splitlines()[-20:])

    kind = "JavaClass"
    for anno, mapped_kind in SPRING_STEREOTYPES.items():
        if re.search(r"@" + anno + r"\b", recent_annotations):
            kind = mapped_kind
            break

    class_sig = class_name
    index.symbols.append(
        Symbol(
            type="class",
            kind=kind,
            name=class_name,
            signature=class_sig,
            file=rpath,
            package=package,
            language="java",
            line_start=class_line,
            line_end=class_end,
            id=make_symbol_id("java", "", class_name, class_sig, rpath, class_line),
            keywords=symbol_keywords(class_name, kind, package, rpath),
        )
    )

    for mm in JAVA_METHOD_RE.finditer(text):
        method_name = mm.group(1)
        if method_name in {"if", "for", "while", "switch", "catch"}:
            continue
        params = re.sub(r"\s+", " ", mm.group(2).strip())
        signature = short_signature(f"{method_name}({params})")
        start = line_no(text, mm.start())
        open_pos = text.find("{", mm.end() - 1)
        end = find_block_end_line(text, open_pos) if open_pos >= 0 else start
        index.symbols.append(
            Symbol(
                type="method",
                kind="JavaMethod",
                name=method_name,
                owner=class_name,
                signature=signature,
                file=rpath,
                package=package,
                language="java",
                line_start=start,
                line_end=end,
                id=make_symbol_id("java", class_name, method_name, signature, rpath, start),
                keywords=symbol_keywords(class_name, method_name, signature, package, rpath),
            )
        )

    base_path = "/"
    class_mapping_match = re.search(
        r"@(RequestMapping|GetMapping|PostMapping|PutMapping|DeleteMapping|PatchMapping)\s*(\([^)]*\))",
        recent_annotations,
        re.DOTALL,
    )
    if class_mapping_match:
        base_path = extract_annotation_value(class_mapping_match.group(0))

    mapping_re = re.compile(
        r"@(?P<anno>GetMapping|PostMapping|PutMapping|DeleteMapping|PatchMapping|RequestMapping)\s*(?P<body>\([^)]*\))?"
        r"(?P<between>(?:\s|@\w+(?:\([^)]*\))?)*)"
        r"(?:public|private|protected)?\s*(?:static\s+)?(?:final\s+)?"
        r"[A-Za-z0-9_<>\[\].?,\s]+\s+"
        r"(?P<method>[A-Za-z_][A-Za-z0-9_]*)\s*\(",
        re.DOTALL,
    )

    for m in mapping_re.finditer(text):
        anno = m.group("anno")
        body = m.group("body") or ""
        annotation_text = f"@{anno}{body}"
        path_value = extract_annotation_value(annotation_text)
        full_path = join_paths(base_path, path_value)
        route_line = line_no(text, m.start())

        methods = extract_request_methods(annotation_text) if anno == "RequestMapping" else [SPRING_MAPPING_METHODS[anno]]
        for method in methods:
            handler = f"{class_name}#{m.group('method')}"
            index.routes.append(
                ApiRoute(
                    repo=index.repo,
                    method=method,
                    path=full_path,
                    handler=handler,
                    file=rpath,
                    framework="Spring",
                    line=route_line,
                    id=make_api_id(method, full_path, handler, rpath, route_line),
                    keywords=api_keywords(method, full_path, handler, "Spring"),
                )
            )


# -------------------------
# Go
# -------------------------

GO_PACKAGE_RE = re.compile(r"^\s*package\s+([A-Za-z_][A-Za-z0-9_]*)", re.MULTILINE)
GO_FUNC_RE = re.compile(r"^\s*func\s+(?:\(([^)]+)\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)", re.MULTILINE)


def parse_go(path: Path, root: Path, index: Index) -> None:
    text = read_text(path)
    rpath = rel(path, root)

    pm = GO_PACKAGE_RE.search(text)
    package = pm.group(1) if pm else ""

    for fm in GO_FUNC_RE.finditer(text):
        receiver = re.sub(r"\s+", " ", (fm.group(1) or "").strip())
        name = fm.group(2)
        params = re.sub(r"\s+", " ", fm.group(3).strip())
        owner = receiver
        signature = short_signature(f"{name}({params})")
        start = line_no(text, fm.start())
        open_pos = text.find("{", fm.end() - 1)
        end = find_block_end_line(text, open_pos) if open_pos >= 0 else start
        index.symbols.append(
            Symbol(
                type="function",
                kind="GoFunc",
                name=name,
                owner=owner,
                signature=signature,
                file=rpath,
                package=package,
                language="go",
                line_start=start,
                line_end=end,
                id=make_symbol_id("go", owner, name, signature, rpath, start),
                keywords=symbol_keywords(name, signature, package, rpath),
            )
        )

    route_re = re.compile(
        r"\b[A-Za-z_][A-Za-z0-9_]*\.(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)\s*"
        r'\(\s*["`]([^"`]+)["`]\s*,\s*([A-Za-z0-9_./]+)',
        re.MULTILINE,
    )
    for m in route_re.finditer(text):
        method = m.group(1)
        p = normalize_path(m.group(2))
        handler = m.group(3)
        ln = line_no(text, m.start())
        index.routes.append(ApiRoute(repo=index.repo, method=method, path=p, handler=handler, file=rpath, framework="Go Router", line=ln, id=make_api_id(method, p, handler, rpath, ln), keywords=api_keywords(method, p, handler, "Go Router")))

    http_re = re.compile(r'\bhttp\.HandleFunc\s*\(\s*["`]([^"`]+)["`]\s*,\s*([A-Za-z0-9_./]+)', re.MULTILINE)
    for m in http_re.finditer(text):
        method = "ANY"
        p = normalize_path(m.group(1))
        handler = m.group(2)
        ln = line_no(text, m.start())
        index.routes.append(ApiRoute(repo=index.repo, method=method, path=p, handler=handler, file=rpath, framework="net/http", line=ln, id=make_api_id(method, p, handler, rpath, ln), keywords=api_keywords(method, p, handler, "net/http")))


# -------------------------
# Python
# -------------------------

PY_CLASS_RE = re.compile(r"^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)", re.MULTILINE)
PY_FUNC_RE = re.compile(r"^\s*(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)", re.MULTILINE)


def parse_python(path: Path, root: Path, index: Index) -> None:
    text = read_text(path)
    rpath = rel(path, root)

    for cm in PY_CLASS_RE.finditer(text):
        name = cm.group(1)
        start = line_no(text, cm.start())
        end = find_python_block_end_line(text, cm.start())
        sig = name
        index.symbols.append(Symbol(type="class", kind="PythonClass", name=name, signature=sig, file=rpath, language="python", line_start=start, line_end=end, id=make_symbol_id("python", "", name, sig, rpath, start), keywords=symbol_keywords(name, rpath)))

    for fm in PY_FUNC_RE.finditer(text):
        name = fm.group(1)
        params = re.sub(r"\s+", " ", fm.group(2).strip())
        signature = short_signature(f"{name}({params})")
        start = line_no(text, fm.start())
        end = find_python_block_end_line(text, fm.start())
        index.symbols.append(Symbol(type="function", kind="PythonFunc", name=name, signature=signature, file=rpath, language="python", line_start=start, line_end=end, id=make_symbol_id("python", "", name, signature, rpath, start), keywords=symbol_keywords(name, signature, rpath)))

    py_route_re = re.compile(
        r"@(?:app|router|blueprint|bp)\.(get|post|put|delete|patch|options|head|route)"
        r'\s*\(\s*["\']([^"\']+)["\'](?:[^)]*)\)\s*'
        r"(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(",
        re.IGNORECASE | re.MULTILINE,
    )
    for m in py_route_re.finditer(text):
        method = m.group(1).upper()
        if method == "ROUTE":
            method = "ANY"
        p = normalize_path(m.group(2))
        handler = m.group(3)
        ln = line_no(text, m.start())
        index.routes.append(ApiRoute(repo=index.repo, method=method, path=p, handler=handler, file=rpath, framework="FastAPI/Flask", line=ln, id=make_api_id(method, p, handler, rpath, ln), keywords=api_keywords(method, p, handler, "FastAPI/Flask")))

    django_re = re.compile(r'\bpath\s*\(\s*["\']([^"\']+)["\']\s*,\s*([A-Za-z0-9_.]+)', re.MULTILINE)
    for m in django_re.finditer(text):
        method = "ANY"
        p = normalize_path(m.group(1))
        handler = m.group(2)
        ln = line_no(text, m.start())
        index.routes.append(ApiRoute(repo=index.repo, method=method, path=p, handler=handler, file=rpath, framework="Django", line=ln, id=make_api_id(method, p, handler, rpath, ln), keywords=api_keywords(method, p, handler, "Django")))


# -------------------------
# Node / Bun / TypeScript
# -------------------------

JS_CLASS_RE = re.compile(r"\bclass\s+([A-Za-z_][A-Za-z0-9_]*)")
JS_FUNC_RE = re.compile(r"\b(?:function\s+([A-Za-z_][A-Za-z0-9_]*)|const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:async\s*)?\([^)]*\)\s*=>)")


def parse_js_ts(path: Path, root: Path, index: Index) -> None:
    text = read_text(path)
    rpath = rel(path, root)
    language = EXT_LANGUAGE.get(path.suffix, "javascript")

    for cm in JS_CLASS_RE.finditer(text):
        name = cm.group(1)
        start = line_no(text, cm.start())
        open_pos = text.find("{", cm.end() - 1)
        end = find_block_end_line(text, open_pos) if open_pos >= 0 else start
        sig = name
        index.symbols.append(Symbol(type="class", kind="JSClass", name=name, signature=sig, file=rpath, language=language, line_start=start, line_end=end, id=make_symbol_id(language, "", name, sig, rpath, start), keywords=symbol_keywords(name, rpath)))

    for fm in JS_FUNC_RE.finditer(text):
        name = fm.group(1) or fm.group(2)
        if not name:
            continue
        start = line_no(text, fm.start())
        open_pos = text.find("{", fm.end() - 1)
        end = find_block_end_line(text, open_pos) if open_pos >= 0 else start
        signature = f"{name}(...)"
        index.symbols.append(Symbol(type="function", kind="JSFunc", name=name, signature=signature, file=rpath, language=language, line_start=start, line_end=end, id=make_symbol_id(language, "", name, signature, rpath, start), keywords=symbol_keywords(name, signature, rpath)))

    js_route_re = re.compile(
        r"\b(?:app|router|server|api)\.(get|post|put|delete|patch|options|head|all)"
        r'\s*\(\s*["`\'"]([^"`\'"]+)["`\'"]\s*,\s*([A-Za-z0-9_.$]+)?',
        re.IGNORECASE | re.MULTILINE,
    )
    for m in js_route_re.finditer(text):
        method = m.group(1).upper()
        if method == "ALL":
            method = "ANY"
        p = normalize_path(m.group(2))
        handler = m.group(3) or "(inline handler)"
        ln = line_no(text, m.start())
        index.routes.append(ApiRoute(repo=index.repo, method=method, path=p, handler=handler, file=rpath, framework="Express/Hono/Elysia", line=ln, id=make_api_id(method, p, handler, rpath, ln), keywords=api_keywords(method, p, handler, "Express/Hono/Elysia")))

    controller_re = re.compile(
        r"@Controller\s*\(\s*['\"`]([^'\"`]*)['\"`]\s*\)\s*"
        r"(?:export\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)[\s\S]*?\{([\s\S]*?)\n\}",
        re.MULTILINE,
    )
    method_decorator_re = re.compile(
        r"@(Get|Post|Put|Delete|Patch|Options|Head|All)\s*(?:\(\s*['\"`]([^'\"`]*)['\"`]\s*\))?"
        r"[\s\S]*?\n\s*(?:async\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\(",
        re.MULTILINE,
    )
    for c in controller_re.finditer(text):
        base = c.group(1)
        cls = c.group(2)
        body = c.group(3)
        body_base_pos = c.start(3)
        for m in method_decorator_re.finditer(body):
            method = m.group(1).upper()
            if method == "ALL":
                method = "ANY"
            sub = m.group(2) or "/"
            p = join_paths(base, sub)
            handler = f"{cls}#{m.group(3)}"
            ln = line_no(text, body_base_pos + m.start())
            index.routes.append(ApiRoute(repo=index.repo, method=method, path=p, handler=handler, file=rpath, framework="NestJS", line=ln, id=make_api_id(method, p, handler, rpath, ln), keywords=api_keywords(method, p, handler, "NestJS")))


# -------------------------
# Files index
# -------------------------

def classify_file(path: Path, root: Path, text: str) -> FileRecord:
    rpath = rel(path, root)
    language = EXT_LANGUAGE.get(path.suffix, path.suffix.lstrip("."))
    kind = "source"
    lower = rpath.lower()

    if "/test/" in lower or lower.endswith("test.java") or lower.endswith("_test.go") or lower.startswith("test/") or lower.startswith("tests/"):
        kind = "test"
    elif "controller" in lower or "handler" in lower or "route" in lower:
        kind = "controller"
    elif "service" in lower:
        kind = "service"
    elif "repository" in lower or "dao" in lower:
        kind = "repository"
    elif "config" in lower or "configuration" in lower:
        kind = "config"

    words = split_camel(rpath)
    identifiers = re.findall(r"\b[A-Za-z_][A-Za-z0-9_]{2,}\b", text[:20000])
    common = [w for w, _ in Counter(identifiers).most_common(30)]
    keywords = uniq(words + common)

    return FileRecord(file=rpath, kind=kind, language=language, line_count=len(text.splitlines()), keywords=keywords[:80])


# -------------------------
# Deduplication and writers
# -------------------------

def dedupe_symbols(symbols: list[Symbol]) -> list[Symbol]:
    seen = set()
    out = []
    for s in symbols:
        key = (s.type, s.kind, s.name, s.owner, s.signature, s.file, s.line_start)
        if key in seen:
            continue
        seen.add(key)
        out.append(s)
    return out


def dedupe_routes(routes: list[ApiRoute]) -> list[ApiRoute]:
    seen = set()
    out = []
    for r in routes:
        key = (r.method, r.path, r.handler, r.file, r.line)
        if key in seen:
            continue
        seen.add(key)
        out.append(r)
    return out


def group_symbols(symbols: list[Symbol]) -> dict[str, list[Symbol]]:
    grouped: dict[str, list[Symbol]] = {}
    for s in sorted(symbols, key=lambda x: (x.kind, x.name, x.file, x.line_start or 0)):
        grouped.setdefault(s.kind, []).append(s)
    return grouped


def loc(file: str, start: int | None, end: int | None = None) -> str:
    if not start:
        return file
    if end and end != start:
        return f"{file}:{start}-{end}"
    return f"{file}:{start}"


def display_symbol(s: Symbol) -> str:
    if s.owner and s.signature:
        return f"{s.owner}#{s.signature}"
    return s.signature or s.name


def write_project_map(index: Index, out: Path) -> None:
    grouped = group_symbols(index.symbols)
    lines = []
    lines.append("# Project Map")
    lines.append("")
    lines.append(f"Generated by `generate-ai-index` v{VERSION}.")
    lines.append("")
    lines.append(f"- Repository: `{index.repo}`")
    lines.append("")
    lines.append("## Modules")
    lines.append("")
    if index.modules:
        for m in index.modules:
            lines.append(f"- `{m}`")
    else:
        lines.append("- No module markers detected.")

    lines.append("")
    lines.append("## Symbols")
    lines.append("")

    preferred_order = [
        "Controller", "Service", "Repository", "Component", "Configuration",
        "JavaClass", "JavaMethod",
        "GoFunc",
        "PythonClass", "PythonFunc",
        "JSClass", "JSFunc"
    ]

    for kind in preferred_order + [k for k in grouped.keys() if k not in preferred_order]:
        items = grouped.get(kind)
        if not items:
            continue
        lines.append(f"### {kind}")
        lines.append("")
        lines.append("| Symbol | Owner | Location | Package |")
        lines.append("|---|---|---|---|")
        for s in items:
            lines.append(f"| `{display_symbol(s)}` | `{s.owner}` | `{loc(s.file, s.line_start, s.line_end)}` | `{s.package}` |")
        lines.append("")

    out.write_text("\n".join(lines) + "\n", encoding="utf-8")


def write_api_index(index: Index, out: Path) -> None:
    routes = sorted(index.routes, key=lambda r: (r.path, r.method, r.handler, r.file, r.line or 0))
    lines = []
    lines.append("# API Index")
    lines.append("")
    lines.append(f"Generated by `generate-ai-index` v{VERSION}.")
    lines.append("")
    lines.append(f"- Repository: `{index.repo}`")
    lines.append("")
    lines.append("| Method | Path | Handler | Framework | Location |")
    lines.append("|---|---|---|---|---|")
    if routes:
        for r in routes:
            lines.append(f"| `{r.method}` | `{r.path}` | `{r.handler}` | `{r.framework}` | `{loc(r.file, r.line)}` |")
    else:
        lines.append("| - | No API routes detected | - | - | - |")
    lines.append("")
    out.write_text("\n".join(lines), encoding="utf-8")


def dataclass_to_clean_dict(obj) -> dict:
    d = asdict(obj)
    return {k: v for k, v in d.items() if v not in ("", None, [], {})}


def write_jsonl(items: list, out: Path) -> None:
    with out.open("w", encoding="utf-8") as f:
        for item in items:
            f.write(json.dumps(dataclass_to_clean_dict(item), ensure_ascii=False, sort_keys=True) + "\n")


def write_jsonl_outputs(index: Index, out_dir: Path) -> None:
    write_jsonl(index.symbols, out_dir / "symbols.jsonl")
    write_jsonl(index.routes, out_dir / "apis.jsonl")
    write_jsonl(index.files, out_dir / "files.jsonl")
    metadata = {
        "repo": index.repo,
        "generator": "generate-ai-index",
        "version": VERSION,
        "generated_at": utc_now(),
        "modules": index.modules,
        "symbols": len(index.symbols),
        "apis": len(index.routes),
        "files": len(index.files),
        "schema": {
            "symbols": "v1.2",
            "apis": "v1.2",
            "files": "v1.2"
        }
    }
    (out_dir / "metadata.json").write_text(json.dumps(metadata, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


# -------------------------
# Main
# -------------------------

def parse_source(path: Path, root: Path, index: Index) -> None:
    suffix = path.suffix
    if suffix == ".java":
        parse_java(path, root, index)
    elif suffix == ".go":
        parse_go(path, root, index)
    elif suffix == ".py":
        parse_python(path, root, index)
    elif suffix in {".js", ".jsx", ".ts", ".tsx"}:
        parse_js_ts(path, root, index)


def build_index(root: Path, repo_name: str | None = None) -> Index:
    index = Index(repo=repo_name or root.name)
    index.modules = detect_modules(root)
    for path in walk_sources(root):
        text = read_text(path)
        index.files.append(classify_file(path, root, text))
        parse_source(path, root, index)
    index.symbols = dedupe_symbols(index.symbols)
    index.routes = dedupe_routes(index.routes)
    return index


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", default=".", help="Project root. Default: current directory.")
    parser.add_argument("--out", default=".ai", help="Output directory relative to root. Default: .ai")
    parser.add_argument("--repo-name", default=None, help="Repository name to write into JSONL records. Default: root directory name.")
    parser.add_argument("--version", action="store_true", help="Print version and exit.")
    args = parser.parse_args()

    if args.version:
        print(VERSION)
        return

    root = Path(args.root).resolve()
    out_dir = root / args.out
    out_dir.mkdir(parents=True, exist_ok=True)

    index = build_index(root, repo_name=args.repo_name)
    write_project_map(index, out_dir / "PROJECT_MAP.md")
    write_api_index(index, out_dir / "API_INDEX.md")
    write_jsonl_outputs(index, out_dir)

    print(f"Generated: {out_dir / 'PROJECT_MAP.md'}")
    print(f"Generated: {out_dir / 'API_INDEX.md'}")
    print(f"Generated: {out_dir / 'symbols.jsonl'}")
    print(f"Generated: {out_dir / 'apis.jsonl'}")
    print(f"Generated: {out_dir / 'files.jsonl'}")
    print(f"Symbols: {len(index.symbols)}")
    print(f"Routes: {len(index.routes)}")
    print(f"Files: {len(index.files)}")


if __name__ == "__main__":
    main()
