#!/usr/bin/env python3
"""
aiw-wt plugin: Python implementation of worktree commands mirroring Go `wt`.
Supports: add, rm, list, prune, lock, unlock, repair, ignore

This plugin uses the same conventions as the Go code: task metadata under
openspec/changes/<id>/task.toml (with legacy tasks.toml fallback) and registry
at openspec/registry.json.
"""
import os
import sys
import subprocess
from pathlib import Path
import json
from datetime import datetime

def resolve_root():
    result = subprocess.run(
        ["git", "rev-parse", "--show-toplevel"],
        text=True, capture_output=True, check=False,
    )
    if result.returncode != 0:
        raise RuntimeError("not inside a Git worktree")
    listing = subprocess.run(
        ["git", "worktree", "list", "--porcelain"],
        text=True, capture_output=True, check=False,
    )
    for line in listing.stdout.splitlines():
        if line.startswith("worktree "):
            return Path(line[9:].strip()).resolve()
    return Path(result.stdout.strip()).resolve()


ROOT = resolve_root()
CHANGES_DIR = ROOT / "openspec" / "changes"
WORKTREE_DIR = Path(".wt")
REGISTRY_FILE = ROOT / "openspec" / "registry.json"


def run_cmd(cmd):
    print(f"> {' '.join(cmd)}", file=sys.stderr)
    p = subprocess.Popen(cmd, cwd=ROOT)
    p.communicate()
    return p.returncode


def task_dir(task_id):
    return CHANGES_DIR / task_id


def task_meta_path(task_id):
    primary = task_dir(task_id) / "task.toml"
    legacy = task_dir(task_id) / "tasks.toml"
    if primary.exists():
        return primary
    if legacy.exists():
        print(f"warning: using legacy task metadata {legacy}; rename it to task.toml", file=sys.stderr)
        return legacy
    return primary


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
    ordered = (
        "id", "type", "status", "created", "updated", "branch",
        "parent_branch", "worktree", "workspace_kind", "delivery", "session",
    )
    lines = [f'{key} = "{meta.get(key, "")}"' for key in ordered]
    for key in ("specs", "tags"):
        if meta.get(key):
            lines.append(f'{key} = {meta[key]}')
    content = "\n".join(lines) + "\n"
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
            meta = read_task_meta(task_meta_path(d.name))
        except Exception:
            continue
        entries.append({
            "id": meta.get("id", ""),
            "status": meta.get("status", ""),
            "branch": meta.get("branch", ""),
            "worktree": meta.get("worktree", ""),
            "workspace_kind": meta.get("workspace_kind", ""),
            "delivery": meta.get("delivery", ""),
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
    if not base:
        meta = read_task_meta(task_meta_path(task_id))
        base = meta.get("parent_branch", "").strip()
        if not base:
            print("task has no parent_branch; pass one explicitly", file=sys.stderr)
            return 2
    task_path = f"openspec/changes/{task_id}/task.toml"
    if run_cmd(["git", "cat-file", "-e", f"{base}:{task_path}"]) != 0:
        print(f"task artifacts are not committed on {base}", file=sys.stderr)
        return 2
    status = subprocess.run(
        ["git", "status", "--porcelain", "--", f"openspec/changes/{task_id}"],
        cwd=ROOT, text=True, capture_output=True, check=False,
    )
    if status.returncode != 0 or status.stdout.strip():
        print("task artifacts have uncommitted changes", file=sys.stderr)
        return 2
    if run_cmd(["git", "worktree", "add", wt, "-b", branch, base]) != 0:
        return 2
    meta_path = task_meta_path(task_id)
    meta = read_task_meta(meta_path)
    meta["branch"] = branch
    meta["worktree"] = wt
    meta["workspace_kind"] = "isolated"
    meta["delivery"] = "pending"
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
    kind = meta.get("workspace_kind", "").strip()
    if not kind:
        wt_value = meta.get("worktree", "").strip()
        if not wt_value:
            kind = "unassigned"
        elif (ROOT / wt_value).resolve() == ROOT:
            kind = "primary"
        else:
            listing = subprocess.run(["git", "worktree", "list", "--porcelain"], cwd=ROOT, text=True, capture_output=True, check=False)
            registered = [line[9:].strip() for line in listing.stdout.splitlines() if line.startswith("worktree ")]
            target = os.path.normcase(str((ROOT / wt_value).resolve()))
            kind = "isolated" if any(os.path.normcase(os.path.abspath(path)) == target for path in registered) else "unknown"
    if kind != "isolated":
        print("refusing to remove a non-isolated or legacy-unknown workspace", file=sys.stderr)
        return 2
    branch = meta.get("branch", "").strip()
    wt = meta.get("worktree", "").strip()
    registered = subprocess.run(
        ["git", "worktree", "list", "--porcelain"], cwd=ROOT,
        text=True, capture_output=True, check=False,
    )
    target = str((ROOT / wt).resolve())
    registered_paths = [line[9:].strip() for line in registered.stdout.splitlines() if line.startswith("worktree ")]
    if not any(os.path.normcase(os.path.abspath(path)) == os.path.normcase(target) for path in registered_paths):
        print("refusing to remove a workspace not registered by Git", file=sys.stderr)
        return 2
    cmd = ["git", "worktree", "remove", wt]
    if force:
        cmd.append("--force")
    if run_cmd(cmd) != 0:
        return 2
    meta["worktree"] = ""
    meta["workspace_kind"] = "unassigned"
    meta["updated"] = datetime.now().strftime("%Y-%m-%d")
    write_task_meta(meta_path, meta)
    write_registry()
    if delete_branch:
        if run_cmd(["git", "branch", "-d", branch]) != 0:
            return 2
        meta["branch"] = ""
        write_task_meta(meta_path, meta)
        write_registry()
    return 0


def discard(task_id, yes=False):
    if not yes:
        print("discard requires --yes", file=sys.stderr)
        return 2
    meta_path = task_meta_path(task_id)
    meta = read_task_meta(meta_path)
    if meta.get("workspace_kind", "").strip() != "isolated":
        print("discard requires a verified isolated workspace", file=sys.stderr)
        return 2
    branch = meta.get("branch", "").strip()
    if rm(task_id, delete_branch=False, force=True) != 0:
        return 2
    if run_cmd(["git", "branch", "-D", branch]) != 0:
        return 2
    meta = read_task_meta(meta_path)
    meta["branch"] = ""
    meta["status"] = "CANCELLED"
    meta["delivery"] = "discarded"
    meta["updated"] = datetime.now().strftime("%Y-%m-%d")
    write_task_meta(meta_path, meta)
    notes = task_dir(task_id) / "notes.md"
    with open(notes, "a", encoding="utf-8") as f:
        f.write("\n%% Cancelled: isolated experiment explicitly discarded.\n")
    write_registry()
    return 0


def list_cmd(porcelain=False):
    cmd = ["git", "worktree", "list"]
    if porcelain:
        cmd.append("--porcelain")
    return run_cmd(cmd)


def push(task_id):
    td = task_dir(task_id)
    if not td.exists():
        print(f"task not found: {task_id}", file=sys.stderr)
        return 2
    meta_path = task_meta_path(task_id)
    meta = read_task_meta(meta_path)
    branch = meta.get("branch", "").strip() or f"feature/{task_id}"
    wt = meta.get("worktree", "").strip() or str((WORKTREE_DIR / task_id).as_posix())
    cmd = ["git", "worktree", "-C", wt,  "push", "origin", branch]
    if run_cmd(cmd) != 0:
        return 2
    meta["status"] = "PUSHED"
    from datetime import datetime
    meta["updated"] = datetime.now().strftime("%Y-%m-%d")
    write_task_meta(meta_path, meta)
    write_registry()
    print(f"You can run `aiw wt rm $task_id --delete-branch`, and then `aiw wt prune` to clean up.")
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
    print("Usage: aiw wt <command> [args...]")
    print()
    print("Commands:")
    print("  add <task-id> [base]                 Create a task worktree.")
    print("  rm <task-id> [--delete-branch] [--force]  Remove a worktree.")
    print("  discard <task-id> --yes              Discard an isolated experiment.")
    print("  list [--porcelain]                   List worktrees.")
    print("  prune [--dry-run]                    Remove stale metadata.")
    print("  lock <task-id> [reason]              Lock a worktree.")
    print("  unlock <task-id>                     Unlock a worktree.")
    print("  repair                               Repair worktree links.")
    print("  ignore                               Add .wt/ to .gitignore.")
    print()
    print("Examples:")
    print("  aiw wt add payment-retry")
    print("  aiw wt list")
    print("  aiw wt prune --dry-run")


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
    if sub == "discard":
        if not rest:
            print("usage: aiw wt discard <task-id> --yes", file=sys.stderr)
            return 2
        return discard(rest[0], "--yes" in rest[1:])
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
