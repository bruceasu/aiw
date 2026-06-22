#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
ai-code-index v1.2.0

Local cross-repository code index / search / LLM context generator.

Reads:
  repo/.ai/symbols.jsonl
  repo/.ai/apis.jsonl
  repo/.ai/files.jsonl

Stores:
  ~/.ai-code-index/repos.json
  ~/.ai-code-index/db.sqlite

V1.2.0 additions:
  - imports symbol line_start / line_end
  - imports API line
  - imports stable-ish id
  - location-aware search/context output
  - global generate-ai-index command support

No third-party dependencies.
"""

from __future__ import annotations

import argparse
import json
import re
import sqlite3
import subprocess
import sys
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Iterable


VERSION = "1.2.0"

APP_DIR = Path.home() / ".ai-code-index"
REPOS_JSON = APP_DIR / "repos.json"
DB_PATH = APP_DIR / "db.sqlite"

JSONL_FILES = {
    "symbols": ".ai/symbols.jsonl",
    "apis": ".ai/apis.jsonl",
    "files": ".ai/files.jsonl",
}

STOPWORDS = {
    "the", "a", "an", "of", "to", "in", "on", "for", "and", "or",
    "is", "are", "was", "were", "with", "by", "from", "at",
    "请", "帮", "我", "一下", "哪里", "哪个", "如何", "怎么", "为什么",
    "接口", "功能", "代码", "相关", "定位", "分析",
}

SYNONYMS = {
    "接口": ["api", "endpoint", "route", "controller"],
    "路由": ["api", "endpoint", "route"],
    "创建": ["create", "add", "insert", "save", "new"],
    "新增": ["create", "add", "insert", "save", "new"],
    "保存": ["save", "create", "insert"],
    "删除": ["delete", "remove"],
    "更新": ["update", "modify", "edit"],
    "修改": ["update", "modify", "edit"],
    "查询": ["query", "search", "find", "get", "list"],
    "获取": ["get", "find", "query", "list"],
    "用户": ["user", "account", "customer"],
    "订单": ["order", "trade"],
    "支付": ["payment", "pay"],
    "入金": ["deposit"],
    "出金": ["withdrawal", "withdraw"],
}


@dataclass
class SearchHit:
    source: str
    repo: str
    repo_path: str
    title: str
    file: str
    location: str
    body: str
    score: float
    reason: list[str]
    raw: dict[str, Any]


def now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def ensure_app_dir() -> None:
    APP_DIR.mkdir(parents=True, exist_ok=True)


def read_json(path: Path, default: Any) -> Any:
    if not path.exists():
        return default
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return default


def write_json(path: Path, data: Any) -> None:
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def repo_name_from_path(path: Path) -> str:
    return path.resolve().name


def normalize_repo_path(path: str | Path) -> str:
    return str(Path(path).expanduser().resolve())


def load_repos() -> list[dict[str, Any]]:
    ensure_app_dir()
    return read_json(REPOS_JSON, [])


def save_repos(repos: list[dict[str, Any]]) -> None:
    ensure_app_dir()
    write_json(REPOS_JSON, repos)


def open_db() -> sqlite3.Connection:
    ensure_app_dir()
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    init_db(conn)
    return conn


def init_db(conn: sqlite3.Connection) -> None:
    conn.executescript(
        """
        CREATE TABLE IF NOT EXISTS records (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          record_id TEXT,
          source TEXT NOT NULL,
          repo TEXT NOT NULL,
          repo_path TEXT NOT NULL,
          type TEXT,
          kind TEXT,
          name TEXT,
          owner TEXT,
          signature TEXT,
          method TEXT,
          path TEXT,
          handler TEXT,
          file TEXT,
          package TEXT,
          language TEXT,
          framework TEXT,
          line_start INTEGER,
          line_end INTEGER,
          line INTEGER,
          location TEXT,
          keywords TEXT,
          title TEXT NOT NULL,
          body TEXT NOT NULL,
          raw_json TEXT NOT NULL,
          updated_at TEXT NOT NULL
        );

        CREATE INDEX IF NOT EXISTS idx_records_repo ON records(repo);
        CREATE INDEX IF NOT EXISTS idx_records_source ON records(source);
        CREATE INDEX IF NOT EXISTS idx_records_file ON records(file);
        CREATE INDEX IF NOT EXISTS idx_records_name ON records(name);
        CREATE INDEX IF NOT EXISTS idx_records_path ON records(path);
        CREATE INDEX IF NOT EXISTS idx_records_location ON records(location);
        CREATE INDEX IF NOT EXISTS idx_records_record_id ON records(record_id);

        CREATE VIRTUAL TABLE IF NOT EXISTS records_fts USING fts5(
          title,
          body,
          repo,
          file,
          location,
          content='records',
          content_rowid='id',
          tokenize='unicode61'
        );

        CREATE TRIGGER IF NOT EXISTS records_ai AFTER INSERT ON records BEGIN
          INSERT INTO records_fts(rowid, title, body, repo, file, location)
          VALUES (new.id, new.title, new.body, new.repo, new.file, new.location);
        END;

        CREATE TRIGGER IF NOT EXISTS records_ad AFTER DELETE ON records BEGIN
          INSERT INTO records_fts(records_fts, rowid, title, body, repo, file, location)
          VALUES('delete', old.id, old.title, old.body, old.repo, old.file, old.location);
        END;

        CREATE TRIGGER IF NOT EXISTS records_au AFTER UPDATE ON records BEGIN
          INSERT INTO records_fts(records_fts, rowid, title, body, repo, file, location)
          VALUES('delete', old.id, old.title, old.body, old.repo, old.file, old.location);
          INSERT INTO records_fts(rowid, title, body, repo, file, location)
          VALUES (new.id, new.title, new.body, new.repo, new.file, new.location);
        END;
        """
    )
    conn.commit()


def jsonl_rows(path: Path) -> Iterable[dict[str, Any]]:
    if not path.exists():
        return
    with path.open("r", encoding="utf-8", errors="ignore") as f:
        for line_no, line in enumerate(f, 1):
            line = line.strip()
            if not line:
                continue
            try:
                row = json.loads(line)
                if isinstance(row, dict):
                    yield row
            except Exception:
                print(f"warning: invalid jsonl {path}:{line_no}", file=sys.stderr)


def split_camel(s: str) -> list[str]:
    if not s:
        return []
    s = re.sub(r"([a-z0-9])([A-Z])", r"\1 \2", s)
    s = re.sub(r"[^A-Za-z0-9_\-/{}:.]+", " ", s)
    parts = []
    for token in s.replace("-", " ").replace("_", " ").replace("/", " ").replace(".", " ").split():
        token = token.strip().lower()
        if token and token not in STOPWORDS:
            parts.append(token)
    return parts


def normalize_query(query: str) -> list[str]:
    tokens = split_camel(query)
    for k, vals in SYNONYMS.items():
        if k in query:
            tokens.extend(vals)
    for m in re.findall(r"\b(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)\b", query, flags=re.I):
        tokens.append(m.lower())
    for p in re.findall(r"/[A-Za-z0-9_/{}/.-]+", query):
        tokens.extend(split_camel(p))
        tokens.append(p.lower())
    seen = set()
    out = []
    for t in tokens:
        if t and t not in seen and t not in STOPWORDS:
            seen.add(t)
            out.append(t)
    return out


def fts_query(tokens: list[str]) -> str:
    cleaned = []
    for t in tokens:
        t = re.sub(r'["]', "", t)
        if re.match(r"^[A-Za-z0-9_]+$", t):
            cleaned.append(t)
    return " OR ".join(cleaned)


def make_location(file: str, line_start: Any = None, line_end: Any = None, line: Any = None) -> str:
    ln = line if line is not None else line_start
    if not ln:
        return file or ""
    if line_end and line_end != ln:
        return f"{file}:{ln}-{line_end}"
    return f"{file}:{ln}"


def make_record(source: str, repo: dict[str, Any], row: dict[str, Any]) -> dict[str, Any]:
    repo_name = row.get("repo") or repo["name"]
    repo_path = repo["path"]

    line_start = row.get("line_start")
    line_end = row.get("line_end")
    line = row.get("line")
    location = make_location(row.get("file", ""), line_start, line_end, line)

    if source == "apis":
        title = f"{row.get('method', 'ANY')} {row.get('path', '')} {row.get('handler', '')}".strip()
        body_parts = [
            title,
            location,
            row.get("framework", ""),
            row.get("file", ""),
            " ".join(row.get("keywords", []) if isinstance(row.get("keywords"), list) else []),
        ]
    elif source == "symbols":
        owner = row.get("owner", "")
        signature = row.get("signature", "")
        name = row.get("name", "")
        symbol_name = f"{owner}#{signature}" if owner and signature else (signature or name)
        title = symbol_name
        body_parts = [
            title,
            location,
            row.get("type", ""),
            row.get("kind", ""),
            row.get("package", ""),
            row.get("file", ""),
            " ".join(row.get("keywords", []) if isinstance(row.get("keywords"), list) else []),
        ]
    else:
        title = row.get("file", "") or row.get("name", "") or "(file)"
        body_parts = [
            title,
            row.get("language", ""),
            row.get("kind", ""),
            str(row.get("line_count", "")),
            " ".join(row.get("keywords", []) if isinstance(row.get("keywords"), list) else []),
        ]

    body = " ".join(str(x) for x in body_parts if x)

    return {
        "record_id": row.get("id", ""),
        "source": source,
        "repo": repo_name,
        "repo_path": repo_path,
        "type": row.get("type", ""),
        "kind": row.get("kind", ""),
        "name": row.get("name", ""),
        "owner": row.get("owner", ""),
        "signature": row.get("signature", ""),
        "method": row.get("method", ""),
        "path": row.get("path", ""),
        "handler": row.get("handler", ""),
        "file": row.get("file", ""),
        "package": row.get("package", ""),
        "language": row.get("language", ""),
        "framework": row.get("framework", ""),
        "line_start": line_start,
        "line_end": line_end,
        "line": line,
        "location": location,
        "keywords": json.dumps(row.get("keywords", []), ensure_ascii=False),
        "title": title,
        "body": body,
        "raw_json": json.dumps(row, ensure_ascii=False),
        "updated_at": now_iso(),
    }


def insert_records(conn: sqlite3.Connection, records: list[dict[str, Any]], repo_path: str) -> None:
    conn.execute("DELETE FROM records WHERE repo_path = ?", (repo_path,))
    if not records:
        conn.commit()
        return
    keys = list(records[0].keys())
    placeholders = ",".join("?" for _ in keys)
    sql = f"INSERT INTO records ({','.join(keys)}) VALUES ({placeholders})"
    conn.executemany(sql, [[r.get(k, None) for k in keys] for r in records])
    conn.commit()


def find_generator(repo_dir: Path) -> list[str] | None:
    from shutil import which
    global_cmd = which("generate-ai-index")
    if global_cmd:
        return [global_cmd]
    local_py = repo_dir / "scripts" / "generate-ai-index.py"
    if local_py.exists():
        return [sys.executable, str(local_py)]
    return None


def run_generator_for_repo(repo_dir: Path) -> None:
    cmd = find_generator(repo_dir)
    if not cmd:
        print("warning: generate-ai-index not found in PATH and repo-local scripts/generate-ai-index.py not found", file=sys.stderr)
        return
    subprocess.run(cmd + ["--root", str(repo_dir)], check=False)


def scan_repo(repo_path: str, run_generator: bool = False) -> int:
    repo_path = normalize_repo_path(repo_path)
    repo_dir = Path(repo_path)
    if not repo_dir.exists():
        raise SystemExit(f"repo path not found: {repo_path}")

    if run_generator:
        run_generator_for_repo(repo_dir)

    repos = load_repos()
    repo = next((r for r in repos if r["path"] == repo_path), None)
    if repo is None:
        repo = {"name": repo_name_from_path(repo_dir), "path": repo_path, "added_at": now_iso()}
        repos.append(repo)
        save_repos(repos)

    records = []
    for source, rel_file in JSONL_FILES.items():
        f = repo_dir / rel_file
        for row in jsonl_rows(f) or []:
            records.append(make_record(source, repo, row))

    conn = open_db()
    insert_records(conn, records, repo_path)
    conn.close()
    return len(records)


def score_row(row: sqlite3.Row, query: str, tokens: list[str], current_repo_path: str | None = None) -> tuple[float, list[str]]:
    score = 0.0
    reason = []

    title = (row["title"] or "").lower()
    body = (row["body"] or "").lower()
    file = (row["file"] or "").lower()
    name = (row["name"] or "").lower()
    path = (row["path"] or "").lower()
    handler = (row["handler"] or "").lower()
    method = (row["method"] or "").lower()
    repo = (row["repo"] or "").lower()
    location = (row["location"] or "").lower()
    q = query.lower()

    if path and path in q:
        score += 100
        reason.append("exact API path match")
    if name and name in q:
        score += 80
        reason.append("exact symbol match")
    if handler and handler in q:
        score += 70
        reason.append("handler match")
    if file and Path(file).name.lower() in q:
        score += 50
        reason.append("filename match")
    if method and re.search(rf"\b{re.escape(method)}\b", q):
        score += 30
        reason.append("HTTP method match")
    if current_repo_path and row["repo_path"] == current_repo_path:
        score += 10
        reason.append("current repo boost")
    if row["source"] == "apis":
        score += 15
        reason.append("API record")
    if row["kind"] in ("Controller", "Service", "Repository"):
        score += 10
        reason.append(f"{row['kind']} symbol")
    if row["line_start"] or row["line"]:
        score += 3
        reason.append("location available")

    matched = 0
    for t in tokens:
        if not t:
            continue
        if t in title:
            score += 12
            matched += 1
        elif t in body:
            score += 6
            matched += 1
        elif t in file:
            score += 8
            matched += 1
        elif t in repo:
            score += 6
            matched += 1
        elif t in location:
            score += 4
            matched += 1
    if matched:
        reason.append(f"{matched} keyword matches")

    negative_patterns = ["test", "/test/", "deprecated", "archive", "generated", "target/", "build/"]
    if any(p in file for p in negative_patterns):
        score -= 25
        reason.append("penalized test/deprecated/generated file")

    return score, reason


def sql_candidates(conn: sqlite3.Connection, query: str, tokens: list[str], limit: int = 200) -> list[sqlite3.Row]:
    candidates: dict[int, sqlite3.Row] = {}
    fq = fts_query(tokens)
    if fq:
        try:
            for row in conn.execute(
                """
                SELECT r.*
                FROM records_fts f
                JOIN records r ON r.id = f.rowid
                WHERE records_fts MATCH ?
                LIMIT ?
                """,
                (fq, limit),
            ):
                candidates[row["id"]] = row
        except sqlite3.OperationalError:
            pass

    like_terms = tokens[:8] or split_camel(query)[:8]
    for t in like_terms:
        pattern = f"%{t}%"
        for row in conn.execute(
            """
            SELECT * FROM records
            WHERE title LIKE ?
               OR body LIKE ?
               OR file LIKE ?
               OR repo LIKE ?
               OR path LIKE ?
               OR handler LIKE ?
               OR location LIKE ?
            LIMIT ?
            """,
            (pattern, pattern, pattern, pattern, pattern, pattern, pattern, limit),
        ):
            candidates[row["id"]] = row
    return list(candidates.values())


def search(query: str, top: int, current_repo: bool = False) -> list[SearchHit]:
    tokens = normalize_query(query)
    current_repo_path = None
    if current_repo:
        cwd = normalize_repo_path(Path.cwd())
        repos = sorted(load_repos(), key=lambda r: len(r["path"]), reverse=True)
        for r in repos:
            if cwd.startswith(r["path"]):
                current_repo_path = r["path"]
                break

    conn = open_db()
    rows = sql_candidates(conn, query, tokens, limit=max(200, top * 40))
    hits = []
    for row in rows:
        score, reason = score_row(row, query, tokens, current_repo_path)
        if score <= 0:
            continue
        raw = json.loads(row["raw_json"])
        hits.append(SearchHit(source=row["source"], repo=row["repo"], repo_path=row["repo_path"], title=row["title"], file=row["file"], location=row["location"], body=row["body"], score=score, reason=reason, raw=raw))
    conn.close()
    hits.sort(key=lambda h: h.score, reverse=True)
    return hits[:top]


def format_search(hits: list[SearchHit]) -> str:
    if not hits:
        return "No matching records found."
    lines = []
    for i, h in enumerate(hits, 1):
        lines.append(f"{i}. [{h.source}] {h.title}")
        lines.append(f"   repo: {h.repo}")
        lines.append(f"   location: {h.location or h.file}")
        lines.append(f"   score: {h.score:.0f}")
        lines.append(f"   reason: {', '.join(h.reason)}")
    return "\n".join(lines)


def format_context(hits: list[SearchHit], query: str) -> str:
    lines = []
    lines.append("# Candidate Context")
    lines.append("")
    lines.append(f"Query: {query}")
    lines.append("")
    if not hits:
        lines.append("No matching records found.")
        return "\n".join(lines)

    for i, h in enumerate(hits, 1):
        raw = h.raw
        lines.append(f"## {i}. {h.title}")
        lines.append(f"- Repo: `{h.repo}`")
        lines.append(f"- Repo path: `{h.repo_path}`")
        if raw.get("method") or raw.get("path"):
            lines.append(f"- API: `{raw.get('method', 'ANY')} {raw.get('path', '')}`")
        if raw.get("handler"):
            lines.append(f"- Handler: `{raw.get('handler')}`")
        if raw.get("kind") or raw.get("type"):
            lines.append(f"- Symbol: `{raw.get('kind') or raw.get('type')}` `{raw.get('name', '')}`")
        if raw.get("signature"):
            lines.append(f"- Signature: `{raw.get('signature')}`")
        if h.location:
            lines.append(f"- Location: `{h.location}`")
        elif h.file:
            lines.append(f"- File: `{h.file}`")
        if raw.get("framework"):
            lines.append(f"- Framework: `{raw.get('framework')}`")
        if raw.get("id"):
            lines.append(f"- ID: `{raw.get('id')}`")
        lines.append(f"- Score: `{h.score:.0f}`")
        lines.append(f"- Reason: {', '.join(h.reason)}")
        lines.append("")
    return "\n".join(lines)


def cmd_repo_add(args: argparse.Namespace) -> None:
    repos = load_repos()
    path = normalize_repo_path(args.path)
    if any(r["path"] == path for r in repos):
        print(f"already registered: {path}")
        return
    repos.append({"name": args.name or repo_name_from_path(Path(path)), "path": path, "added_at": now_iso()})
    save_repos(repos)
    print(f"registered: {path}")


def cmd_repo_list(args: argparse.Namespace) -> None:
    repos = load_repos()
    if not repos:
        print("No repos registered.")
        return
    for r in repos:
        print(f"{r['name']}\t{r['path']}")


def cmd_repo_remove(args: argparse.Namespace) -> None:
    path = normalize_repo_path(args.path)
    repos = [r for r in load_repos() if r["path"] != path]
    save_repos(repos)
    conn = open_db()
    conn.execute("DELETE FROM records WHERE repo_path = ?", (path,))
    conn.commit()
    conn.close()
    print(f"removed: {path}")


def cmd_scan(args: argparse.Namespace) -> None:
    count = scan_repo(args.path, run_generator=args.run_generator)
    print(f"indexed {count} records from {normalize_repo_path(args.path)}")


def cmd_update(args: argparse.Namespace) -> None:
    total = 0
    for repo in load_repos():
        try:
            count = scan_repo(repo["path"], run_generator=args.run_generator)
            total += count
            print(f"indexed {count} records from {repo['name']}")
        except Exception as e:
            print(f"error indexing {repo.get('path')}: {e}", file=sys.stderr)
    print(f"total indexed records: {total}")


def cmd_search(args: argparse.Namespace) -> None:
    hits = search(args.query, args.top, current_repo=args.current_repo)
    print(format_search(hits))


def cmd_context(args: argparse.Namespace) -> None:
    hits = search(args.query, args.top, current_repo=args.current_repo)
    print(format_context(hits, args.query))


def cmd_stats(args: argparse.Namespace) -> None:
    conn = open_db()
    total = conn.execute("SELECT COUNT(*) AS c FROM records").fetchone()["c"]
    print(f"version: {VERSION}")
    print(f"db: {DB_PATH}")
    print(f"records: {total}")
    for row in conn.execute("SELECT repo, source, COUNT(*) AS c FROM records GROUP BY repo, source ORDER BY repo, source"):
        print(f"{row['repo']}\t{row['source']}\t{row['c']}")
    conn.close()


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="ai-code-index")
    parser.add_argument("--version", action="store_true", help="Print version and exit.")
    sub = parser.add_subparsers(dest="cmd")

    repo = sub.add_parser("repo")
    repo_sub = repo.add_subparsers(dest="repo_cmd", required=True)

    p = repo_sub.add_parser("add")
    p.add_argument("path")
    p.add_argument("--name", default=None)
    p.set_defaults(func=cmd_repo_add)

    p = repo_sub.add_parser("list")
    p.set_defaults(func=cmd_repo_list)

    p = repo_sub.add_parser("remove")
    p.add_argument("path")
    p.set_defaults(func=cmd_repo_remove)

    p = sub.add_parser("scan")
    p.add_argument("path")
    p.add_argument("--run-generator", action="store_true", help="Run generate-ai-index before importing .ai/*.jsonl")
    p.set_defaults(func=cmd_scan)

    p = sub.add_parser("update")
    p.add_argument("--run-generator", action="store_true", help="Run generate-ai-index for each repo before importing .ai/*.jsonl")
    p.set_defaults(func=cmd_update)

    p = sub.add_parser("search")
    p.add_argument("query")
    p.add_argument("--top", type=int, default=10)
    p.add_argument("--current-repo", action="store_true", help="Boost results from current working repo")
    p.set_defaults(func=cmd_search)

    p = sub.add_parser("context")
    p.add_argument("query")
    p.add_argument("--top", type=int, default=8)
    p.add_argument("--current-repo", action="store_true", help="Boost results from current working repo")
    p.set_defaults(func=cmd_context)

    p = sub.add_parser("stats")
    p.set_defaults(func=cmd_stats)

    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()
    if args.version:
        print(VERSION)
        return
    if not hasattr(args, "func"):
        parser.print_help()
        return
    args.func(args)


if __name__ == "__main__":
    main()
