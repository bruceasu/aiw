#!/usr/bin/env python3
"""aiw-github: 简单的 GitHub 命令行工具。"""

import argparse
import json
import os
import re
import subprocess
import sys
from typing import Optional
from urllib.parse import urlparse

import requests

try:
    from rich.console import Console
    from rich import box
    from rich.panel import Panel
    from rich.text import Text
    from rich.table import Table
    _RICH_AVAILABLE = True
except Exception:
    Console = None
    box = None
    Panel = None
    Text = None
    Table = None
    _RICH_AVAILABLE = False

DEFAULT_API = "https://api.github.com"
HELP_EPILOG = """Examples:

  # auto-detect repo from current git remote
  aiw-github --json list-issue
  aiw-github repo-info

  # explicit repo
  aiw-github create-issue owner/repo --title "Bug report" --body "..."
  aiw-github create-issue owner/repo --title "Bug report" --body-file issue.md
  aiw-github update-issue owner/repo 123 --body-file issue.md
  aiw-github issue-comment owner/repo 123 --body "Looks good"

Use `--json` before COMMAND for machine-readable output.
Use `--body-file -` to read a Markdown body from stdin.
"""


def load_token() -> Optional[str]:
    return os.environ.get("GITHUB_TOKEN")


console = Console() if _RICH_AVAILABLE else None
USE_JSON = False


def run_git(args):
    return subprocess.run(
        ["git", *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def discover_repo() -> Optional[str]:
    try:
        root = run_git(["rev-parse", "--show-toplevel"]).stdout.strip()
        if not root:
            return None
    except Exception:
        return None

    try:
        remote = run_git(["remote", "get-url", "origin"]).stdout.strip()
    except Exception:
        try:
            remote = run_git(["config", "--get", "remote.origin.url"]).stdout.strip()
        except Exception:
            return None

    if not remote:
        return None

    match = re.search(r"[:/]([^/]+/[^/]+?)(?:\.git)?$", remote)
    if match:
        return match.group(1)

    parsed = urlparse(remote)
    candidate = parsed.path.strip("/")
    if candidate.endswith(".git"):
        candidate = candidate[:-4]
    if candidate.count("/") == 1:
        return candidate
    return None


def resolve_repo(repo_arg: Optional[str]) -> str:
    if repo_arg:
        return repo_arg
    discovered = discover_repo()
    if discovered:
        return discovered
    raise SystemExit("repo 未提供，且无法从当前 git 仓库自动发现（请传入 owner/repo）")


def request(method, path, token, params=None, json_body=None):
    url = DEFAULT_API + path
    headers = {"Authorization": f"token {token}", "Accept": "application/vnd.github+json"}
    resp = requests.request(method, url, headers=headers, params=params, json=json_body)
    try:
        data = resp.json()
    except Exception:
        data = resp.text
    if not resp.ok:
        raise SystemExit(f"GitHub API error: {resp.status_code} {data}")
    return data


def emit_json(data):
    print(json.dumps(data, indent=2, ensure_ascii=False))


def format_state(value):
    if not value:
        return "unknown"
    return str(value)


def state_style(value):
    value = (value or "").lower()
    if value == "open":
        return "green"
    if value == "closed":
        return "red"
    if value == "merged":
        return "magenta"
    return "yellow"


def emit_kv_panel(data, title=None):
    if not _RICH_AVAILABLE or USE_JSON:
        emit_json(data)
        return

    if isinstance(data, dict):
        table = Table(show_header=False, box=box.SIMPLE if _RICH_AVAILABLE else None, padding=(0, 1))
        table.add_column("field", style="bold cyan", no_wrap=True)
        table.add_column("value", overflow="fold")
        for key in sorted(data.keys()):
            value = data[key]
            if isinstance(value, (dict, list)):
                value = json.dumps(value, ensure_ascii=False, indent=2)
            table.add_row(str(key), str(value))
        console.print(Panel(table, title=title or "Object", expand=False))
        return

    emit_json(data)


def emit_list_panel(data, title=None, kind="item"):
    if not _RICH_AVAILABLE or USE_JSON:
        emit_json(data)
        return

    if not isinstance(data, list):
        emit_kv_panel(data, title=title)
        return

    table = Table(show_header=True, header_style="bold cyan", box=box.MINIMAL_HEAVY_HEAD)
    table.add_column("#", style="dim", width=4)
    if kind == "issue":
        table.add_column("Number", style="cyan", width=8)
        table.add_column("State", width=9)
        table.add_column("Title", overflow="fold")
        table.add_column("User", overflow="fold")
    elif kind == "pr":
        table.add_column("Number", style="cyan", width=8)
        table.add_column("State", width=9)
        table.add_column("Title", overflow="fold")
        table.add_column("Head", overflow="fold")
    else:
        table.add_column("Value", overflow="fold")

    for idx, item in enumerate(data, start=1):
        if not isinstance(item, dict):
            table.add_row(str(idx), str(item))
            continue
        if kind == "issue":
            state = format_state(item.get("state"))
            user = item.get("user", {}).get("login", "")
            table.add_row(
                str(idx),
                str(item.get("number", "")),
                Text(state, style=state_style(state)),
                str(item.get("title", "")),
                str(user),
            )
        elif kind == "pr":
            state = format_state(item.get("state"))
            head = item.get("head", {}).get("ref", "")
            table.add_row(
                str(idx),
                str(item.get("number", "")),
                Text(state, style=state_style(state)),
                str(item.get("title", "")),
                str(head),
            )
        else:
            table.add_row(str(idx), str(item))

    console.print(Panel(table, title=title or "List", expand=False))


def emit_repo_panel(data, title=None):
    if not _RICH_AVAILABLE or USE_JSON:
        emit_json(data)
        return

    if not isinstance(data, dict):
        emit_json(data)
        return

    grid = Table.grid(padding=(0, 2))
    grid.add_column(style="bold cyan", no_wrap=True)
    grid.add_column(overflow="fold")
    keys = [
        ("full_name", "Repository"),
        ("description", "Description"),
        ("default_branch", "Default branch"),
        ("language", "Language"),
        ("open_issues_count", "Open issues"),
        ("stargazers_count", "Stars"),
        ("forks_count", "Forks"),
        ("html_url", "URL"),
    ]
    for key, label in keys:
        if key in data and data.get(key) not in (None, ""):
            value = data.get(key)
            grid.add_row(label, str(value))

    console.print(Panel(grid, title=title or "Repository", expand=False))


def emit_issue_panel(data, title=None):
    if not _RICH_AVAILABLE or USE_JSON:
        emit_json(data)
        return

    if not isinstance(data, dict):
        emit_json(data)
        return

    header = Text()
    header.append(f"#{data.get('number', '')} ", style="bold cyan")
    header.append(str(data.get("title", "")), style="bold")
    state = format_state(data.get("state"))
    header.append(f" [{state}]", style=state_style(state))

    body = Table.grid(padding=(0, 2))
    body.add_column(style="bold cyan", no_wrap=True)
    body.add_column(overflow="fold")
    body.add_row("User", str(data.get("user", {}).get("login", "")))
    body.add_row("State", state)
    body.add_row("URL", str(data.get("html_url", "")))
    if data.get("labels"):
        labels = ", ".join(label.get("name", "") for label in data.get("labels", []) if isinstance(label, dict))
        body.add_row("Labels", labels)
    if data.get("body"):
        body.add_row("Body", str(data.get("body")))

    console.print(Panel(body, title=header, expand=False))


def emit_pr_panel(data, title=None):
    if not _RICH_AVAILABLE or USE_JSON:
        emit_json(data)
        return

    if not isinstance(data, dict):
        emit_json(data)
        return

    header = Text()
    header.append(f"#{data.get('number', '')} ", style="bold cyan")
    header.append(str(data.get("title", "")), style="bold")
    state = format_state(data.get("state"))
    header.append(f" [{state}]", style=state_style(state))

    body = Table.grid(padding=(0, 2))
    body.add_column(style="bold cyan", no_wrap=True)
    body.add_column(overflow="fold")
    body.add_row("User", str(data.get("user", {}).get("login", "")))
    body.add_row("State", state)
    body.add_row("Head", str(data.get("head", {}).get("ref", "")))
    body.add_row("Base", str(data.get("base", {}).get("ref", "")))
    body.add_row("URL", str(data.get("html_url", "")))
    if data.get("body"):
        body.add_row("Body", str(data.get("body")))

    console.print(Panel(body, title=header, expand=False))


def emit_value(data, title=None):
    emit_json(data)


def read_body(args):
    if getattr(args, "body_file", None) is not None:
        source = args.body_file
        if source == "-":
            return sys.stdin.read()
        try:
            with open(source, "r", encoding="utf-8") as handle:
                return handle.read()
        except OSError as exc:
            raise SystemExit(f"无法读取 Issue body 文件: {source}: {exc}")
    return getattr(args, "body", None) or ""


def issue_identity(owner_repo, data):
    result = dict(data)
    result["repository"] = owner_repo
    result.setdefault("url", result.get("html_url", ""))
    return result


def create_issue(args, token):
    owner_repo = resolve_repo(args.repo)
    payload = {"title": args.title, "body": read_body(args)}
    data = request("POST", f"/repos/{owner_repo}/issues", token, json_body=payload)
    emit_issue_panel(issue_identity(owner_repo, data), "Issue created")


def update_issue(args, token):
    owner_repo = resolve_repo(args.repo)
    payload = {}
    if args.title is not None:
        payload["title"] = args.title
    if args.body is not None or args.body_file is not None:
        payload["body"] = read_body(args)
    if not payload:
        raise SystemExit("update-issue 至少需要 --title、--body 或 --body-file")
    data = request("PATCH", f"/repos/{owner_repo}/issues/{args.number}", token, json_body=payload)
    emit_issue_panel(issue_identity(owner_repo, data), f"Issue #{args.number} updated")


def create_pr(args, token):
    owner_repo = resolve_repo(args.repo)
    payload = {"title": args.title, "head": args.head, "base": args.base, "body": args.body or ""}
    data = request("POST", f"/repos/{owner_repo}/pulls", token, json_body=payload)
    emit_pr_panel(data, "PR created")


def list_issues(args, token):
    owner_repo = resolve_repo(args.repo)
    params = {"state": args.state or "open", "per_page": args.per_page}
    data = request("GET", f"/repos/{owner_repo}/issues", token, params=params)
    emit_list_panel(data, f"Issues: {owner_repo}", kind="issue")


def list_prs(args, token):
    owner_repo = resolve_repo(args.repo)
    params = {"state": args.state or "open", "per_page": args.per_page}
    data = request("GET", f"/repos/{owner_repo}/pulls", token, params=params)
    emit_list_panel(data, f"Pull requests: {owner_repo}", kind="pr")


def merge_pr(args, token):
    owner_repo = resolve_repo(args.repo)
    payload = {"commit_title": args.message or f"Merge PR #{args.number}"}
    data = request("PUT", f"/repos/{owner_repo}/pulls/{args.number}/merge", token, json_body=payload)
    emit_kv_panel(data, f"PR #{args.number} merged")


def get_issue(args, token):
    owner_repo = resolve_repo(args.repo)
    data = request("GET", f"/repos/{owner_repo}/issues/{args.number}", token)
    emit_issue_panel(issue_identity(owner_repo, data), f"Issue #{args.number}")


def get_pr(args, token):
    owner_repo = resolve_repo(args.repo)
    data = request("GET", f"/repos/{owner_repo}/pulls/{args.number}", token)
    emit_pr_panel(data, f"PR #{args.number}")


def comment_issue(args, token):
    owner_repo = resolve_repo(args.repo)
    payload = {"body": args.body}
    data = request("POST", f"/repos/{owner_repo}/issues/{args.number}/comments", token, json_body=payload)
    emit_kv_panel(data, f"Comment added to issue #{args.number}")


def add_labels(args, token):
    owner_repo = resolve_repo(args.repo)
    payload = {"labels": args.labels}
    data = request("POST", f"/repos/{owner_repo}/issues/{args.number}/labels", token, json_body=payload)
    emit_kv_panel(data, f"Labels added to issue #{args.number}")


def close_issue(args, token):
    owner_repo = resolve_repo(args.repo)
    payload = {"state": "closed"}
    data = request("PATCH", f"/repos/{owner_repo}/issues/{args.number}", token, json_body=payload)
    emit_issue_panel(issue_identity(owner_repo, data), f"Issue #{args.number} closed")


def repo_info(args, token):
    owner_repo = resolve_repo(args.repo)
    data = request("GET", f"/repos/{owner_repo}", token)
    emit_repo_panel(data, f"Repository {owner_repo}")


def add_repo_arg(parser):
    parser.add_argument("repo", nargs="?", help="owner/repo; omit to auto-detect from git")


def add_cmd_parser(subparsers, name, help_text):
    return subparsers.add_parser(
        name,
        help=help_text,
        description=help_text,
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )


def build_parser():
    parser = argparse.ArgumentParser(
        prog="aiw-github",
        description="GitHub CLI helpers for issues and pull requests.",
        epilog=HELP_EPILOG,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--json", action="store_true", help="force JSON output")
    sub = parser.add_subparsers(dest="cmd", metavar="COMMAND")

    p_issue = add_cmd_parser(sub, "create-issue", "Create a GitHub issue.")
    add_repo_arg(p_issue)
    p_issue.add_argument("--title", required=True, help="issue title")
    issue_body = p_issue.add_mutually_exclusive_group()
    issue_body.add_argument("--body", help="issue body")
    issue_body.add_argument("--body-file", help="read issue body from a UTF-8 file, or - for stdin")

    p_update = add_cmd_parser(sub, "update-issue", "Update an existing GitHub issue.")
    add_repo_arg(p_update)
    p_update.add_argument("number", type=int, help="issue number")
    p_update.add_argument("--title", help="replacement issue title")
    update_body = p_update.add_mutually_exclusive_group()
    update_body.add_argument("--body", help="replacement issue body")
    update_body.add_argument("--body-file", help="read replacement body from a UTF-8 file, or - for stdin")

    p_pr = add_cmd_parser(sub, "create-pr", "Create a pull request.")
    add_repo_arg(p_pr)
    p_pr.add_argument("--title", required=True, help="PR title")
    p_pr.add_argument("--head", required=True, help="source branch")
    p_pr.add_argument("--base", required=True, help="target branch")
    p_pr.add_argument("--body", help="PR body")

    p_merge = add_cmd_parser(sub, "merge-pr", "Merge a pull request.")
    add_repo_arg(p_merge)
    p_merge.add_argument("number", type=int, help="PR number")
    p_merge.add_argument("--message", help="merge commit title")

    p_list_issues = add_cmd_parser(sub, "list-issue", "List issues.")
    add_repo_arg(p_list_issues)
    p_list_issues.add_argument("--state", choices=["open", "closed", "all"], help="issue state")
    p_list_issues.add_argument("--per-page", type=int, default=30, dest="per_page", help="items per page")

    p_list_pr = add_cmd_parser(sub, "list-pr", "List pull requests.")
    add_repo_arg(p_list_pr)
    p_list_pr.add_argument("--state", choices=["open", "closed", "all"], help="PR state")
    p_list_pr.add_argument("--per-page", type=int, default=30, dest="per_page", help="items per page")

    p_issue_get = add_cmd_parser(sub, "get-issue", "Get a single issue.")
    add_repo_arg(p_issue_get)
    p_issue_get.add_argument("number", type=int, help="issue number")

    p_pr_get = add_cmd_parser(sub, "get-pr", "Get a single pull request.")
    add_repo_arg(p_pr_get)
    p_pr_get.add_argument("number", type=int, help="PR number")

    p_comment = add_cmd_parser(sub, "issue-comment", "Comment on an issue.")
    add_repo_arg(p_comment)
    p_comment.add_argument("number", type=int, help="issue number")
    p_comment.add_argument("--body", required=True, help="comment body")

    p_label = add_cmd_parser(sub, "issue-label-add", "Add labels to an issue.")
    add_repo_arg(p_label)
    p_label.add_argument("number", type=int, help="issue number")
    p_label.add_argument("labels", nargs="+", help="labels to add")

    p_close = add_cmd_parser(sub, "issue-close", "Close an issue.")
    add_repo_arg(p_close)
    p_close.add_argument("number", type=int, help="issue number")

    p_repo = add_cmd_parser(sub, "repo-info", "Show repository metadata.")
    add_repo_arg(p_repo)

    return parser


def main():
    parser = build_parser()

    args = parser.parse_args()
    if not args.cmd:
        parser.print_help()
        sys.exit(1)
    global USE_JSON
    USE_JSON = args.json

    token = load_token()
    if not token:
        print("GITHUB_TOKEN not set in environment")
        sys.exit(2)

    if args.cmd == "create-issue":
        create_issue(args, token)
    elif args.cmd == "update-issue":
        update_issue(args, token)
    elif args.cmd == "create-pr":
        create_pr(args, token)
    elif args.cmd == "merge-pr":
        merge_pr(args, token)
    elif args.cmd == "list-issue":
        list_issues(args, token)
    elif args.cmd == "list-pr":
        list_prs(args, token)
    elif args.cmd == "get-issue":
        get_issue(args, token)
    elif args.cmd == "get-pr":
        get_pr(args, token)
    elif args.cmd == "issue-comment":
        comment_issue(args, token)
    elif args.cmd == "issue-label-add":
        add_labels(args, token)
    elif args.cmd == "issue-close":
        close_issue(args, token)
    elif args.cmd == "repo-info":
        repo_info(args, token)
    else:
        print("unknown command")


if __name__ == "__main__":
    main()
