#!/usr/bin/env python3
"""
aiw-wt plugin: Python implementation of worktree commands mirroring Go `wt`.
Supports: add, rm, list, prune, lock, unlock, repair, ignore

This plugin uses the same conventions as the Go code: task metadata under
openspec/changes/<id>/task.toml and registry at openspec/registry.json.
"""
import os
import sys
import subprocess
from pathlib import Path
import json
from datetime import datetime

ROOT = Path.cwd()
CHANGES_DIR = ROOT / "openspec" / "changes"
WORKTREE_DIR = Path(".wt")
REGISTRY_FILE = ROOT / "openspec" / "registry.json"


def run_cmd(cmd):
    print(f"> {' '.join(cmd)}", file=sys.stderr)
    p = subprocess.Popen(cmd)
    p.communicate()
    return p.returncode


def task_dir(task_id):
    return CHANGES_DIR / task_id


def task_meta_path(task_id):
    return task_dir(task_id) / "task.toml"


def read_task_meta(path):
    meta = {}
    try:
        with open(path, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#"):
                    continue
                if "=" in line:
                    k, v = line.split("=", 1)
                    meta[k.strip()] = v.strip().strip('"')
    except FileNotFoundError:
        raise
    return meta


def write_task_meta(path, meta):
    content = (
        f'id = "{meta.get("id", "")}"\n'
        f'type = "{meta.get("type", "")}"\n'
        f'status = "{meta.get("status", "")}"\n'
        f'created = "{meta.get("created", "")}"\n'
        f'updated = "{meta.get("updated", "")}"\n'
        f'branch = "{meta.get("branch", "")}"\n'
        f'worktree = "{meta.get("worktree", "")}"\n'
    )
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)


def write_registry():
    entries = []
    if not CHANGES_DIR.exists():
        return
    for d in sorted(CHANGES_DIR.iterdir()):
        if not d.is_dir():
            continue
        try:
            meta = read_task_meta(d / "task.toml")
        except Exception:
            continue
        entries.append({
            "id": meta.get("id", ""),
            "status": meta.get("status", ""),
            "branch": meta.get("branch", ""),
            "worktree": meta.get("worktree", ""),
            "path": str(d).replace('\\', '/'),
            "updated_at": meta.get("updated", ""),
        })
    payload = {"version": "1", "updated": datetime.now().astimezone().isoformat(), "changes": entries}
    with open(REGISTRY_FILE, "w", encoding="utf-8") as f:
        json.dump(payload, f, indent=2)


def ensure_worktree_ignored():
    gitignore = ROOT / ".gitignore"
    entry = str(WORKTREE_DIR) + "/\n"
    if not gitignore.exists():
        gitignore.write_text(entry)
        print("created: .gitignore")
        return 0
    content = gitignore.read_text()
    if entry.strip() in content or str(WORKTREE_DIR) in content:
        print("exists: .gitignore", entry.strip())
        return 0
    if not content.endswith("\n"):
        content += "\n"
    content += entry
    gitignore.write_text(content)
    print("updated: .gitignore", entry.strip())
    return 0


def add(task_id, base):
    td = task_dir(task_id)
    if not td.exists():
        print(f"task not found: {task_id}", file=sys.stderr)
        return 2
    branch = f"feature/{task_id}"
    wt = str((WORKTREE_DIR / task_id).as_posix())
    if run_cmd(["git", "remote", "get-url", "origin"]) == 0:
        run_cmd(["git", "fetch", "origin"])
    if not base:
        # try to detect main/master
        for candidate in ("origin/main", "origin/master", "main", "master"):
            if run_cmd(["git", "rev-parse", "--verify", candidate]) == 0:
                base = candidate
                print("base branch:", base)
                break
        if not base:
            print("cannot detect base branch; pass one explicitly", file=sys.stderr)
            return 2
    if run_cmd(["git", "worktree", "add", wt, "-b", branch, base]) != 0:
        return 2
    meta_path = task_meta_path(task_id)
    meta = read_task_meta(meta_path)
    meta["branch"] = branch
    meta["worktree"] = wt
    # updated field
    from datetime import datetime
    meta["updated"] = datetime.now().strftime("%Y-%m-%d")
    write_task_meta(meta_path, meta)
    write_registry()
    return 0


def rm(task_id, delete_branch=False, force=False):
    td = task_dir(task_id)
    if not td.exists():
        print(f"task not found: {task_id}", file=sys.stderr)
        return 2
    meta_path = task_meta_path(task_id)
    meta = read_task_meta(meta_path)
    branch = meta.get("branch", "").strip() or f"feature/{task_id}"
    wt = meta.get("worktree", "").strip() or str((WORKTREE_DIR / task_id).as_posix())
    cmd = ["git", "worktree", "remove", wt]
    if force:
        cmd.append("--force")
    if run_cmd(cmd) != 0:
        return 2
    meta["worktree"] = ""
    if delete_branch:
        if run_cmd(["git", "branch", "-d", branch]) != 0:
            return 2
        meta["branch"] = ""
    from datetime import datetime
    meta["updated"] = datetime.now().strftime("%Y-%m-%d")
    write_task_meta(meta_path, meta)
    write_registry()
    return 0


def list_cmd(porcelain=False):
    cmd = ["git", "worktree", "list"]
    if porcelain:
        cmd.append("--porcelain")
    return run_cmd(cmd)


def prune(dry_run=False):
    cmd = ["git", "worktree", "prune"]
    if dry_run:
        cmd.extend(["-n", "-v"])
    return run_cmd(cmd)


def lock(task_id, reason):
    wt = (WORKTREE_DIR / task_id).as_posix()
    cmd = ["git", "worktree", "lock", wt]
    if reason:
        cmd.extend(["--reason", reason])
    return run_cmd(cmd)


def unlock(task_id):
    wt = (WORKTREE_DIR / task_id).as_posix()
    return run_cmd(["git", "worktree", "unlock", wt])


def repair():
    return run_cmd(["git", "worktree", "repair"])


def usage():
    print("aiw wt - worktree management")
    print()
    print("  add <task-id> [base]")
    print("  rm  <task-id> [--delete-branch] [--force]")
    print("  list [--porcelain]")
    print("  prune [--dry-run]")
    print("  lock <task-id> [reason]")
    print("  unlock <task-id]")
    print("  repair")
    print("  ignore")


def main():
    args = sys.argv[1:]
    if not args or args[0] in ("help", "-h", "--help"):
        usage()
        return 0
    sub, rest = args[0], args[1:]
    if sub == "add":
        if not rest:
            print("usage: aiw wt add <task-id> [base]", file=sys.stderr)
            return 2
        base = rest[1] if len(rest) >= 2 else ""
        return add(rest[0], base)
    if sub == "rm":
        if not rest:
            print("usage: aiw wt rm <task-id> [--delete-branch] [--force]", file=sys.stderr)
            return 2
        delete_branch = "--delete-branch" in rest[1:]
        force = "--force" in rest[1:]
        return rm(rest[0], delete_branch, force)
    if sub in ("list", "ls"):
        porcelain = "--porcelain" in rest
        return list_cmd(porcelain)
    if sub == "prune":
        dry = "--dry-run" in rest
        return prune(dry)
    if sub == "lock":
        if not rest:
            print("usage: aiw wt lock <task-id> [reason]", file=sys.stderr)
            return 2
        reason = " ".join(rest[1:]).strip()
        return lock(rest[0], reason)
    if sub == "unlock":
        if not rest:
            print("usage: aiw wt unlock <task-id>", file=sys.stderr)
            return 2
        return unlock(rest[0])
    if sub == "repair":
        return repair()
    if sub == "ignore":
        return ensure_worktree_ignored()
    print(f"unknown wt subcommand: {sub}  (run: aiw wt help)", file=sys.stderr)
    return 2


if __name__ == "__main__":
    sys.exit(main())
