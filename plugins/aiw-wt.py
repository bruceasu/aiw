#!/usr/bin/env python3
"""
aiw-wt plugin: Python implementation of worktree commands mirroring Go `wt`.
Supports: add, rm, list, prune, lock, unlock, repair, ignore

This plugin uses the same conventions as the Go code: task metadata under
.ai/tasks/<id>/task.toml with legacy tasks.toml fallback.
"""
import os
import shutil
import sys
import subprocess
from fnmatch import fnmatchcase
from pathlib import Path
from datetime import datetime


MERGE_RESOLUTION_MAX_FILES = 8
MERGE_RESOLUTION_MAX_CHARS_PER_VERSION = 3000
MERGE_RESOLUTION_MAX_CHARS = 24000
MERGE_RESOLUTION_PROTECTED_PREFIXES = (".ai/", ".wt/")
MERGE_RESOLUTION_PROTECTED_BASENAMES = {"task.toml", "tasks.toml"}
MERGE_RESOLUTION_LOCK_BASENAMES = {
    "cargo.lock", "composer.lock", "gemfile.lock", "go.sum", "package-lock.json",
    "pipfile.lock", "poetry.lock", "pnpm-lock.yaml", "yarn.lock",
}


def configured_sensitive_path_patterns():
    """Return repo-relative glob patterns excluded from resolution handoffs.

    AIW_MERGE_RESOLUTION_SENSITIVE_PATHS accepts a comma or platform-path-
    separator delimited list. Patterns are matched against slash-normalized,
    repository-relative conflict paths.
    """
    raw = os.environ.get("AIW_MERGE_RESOLUTION_SENSITIVE_PATHS", "")
    return [normalize_repository_path(pattern.strip())
            for pattern in raw.replace(",", os.pathsep).split(os.pathsep)
            if pattern.strip()]


def normalize_repository_path(path):
    normalized = path.replace("\\", "/")
    while normalized.startswith("./"):
        normalized = normalized[2:]
    return normalized


def conflict_exclusion_reason(path):
    """Classify paths that must remain under manual conflict resolution."""
    normalized = normalize_repository_path(path)
    name = Path(normalized).name.lower()
    if not normalized or normalized.startswith("/") or ".." in Path(normalized).parts:
        return "path is not a safe repository-relative path"
    if normalized.startswith(MERGE_RESOLUTION_PROTECTED_PREFIXES):
        return "Workflow Core state or worktree metadata is protected"
    if name in MERGE_RESOLUTION_PROTECTED_BASENAMES:
        return "Task metadata is protected"
    if name in MERGE_RESOLUTION_LOCK_BASENAMES or name.endswith(".lock") or name.endswith(".lock.json"):
        return "lockfile or dependency lockfile is protected"
    for pattern in configured_sensitive_path_patterns():
        if fnmatchcase(normalized, pattern) or normalized.startswith(pattern.rstrip("/") + "/"):
            return "matches configured sensitive-path policy"
    return ""

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
RUNTIME_TASKS_DIR = ROOT / ".ai" / "tasks"
WORKTREE_DIR = Path(".wt")


def terminal_style(text, code, stream=sys.stderr):
    """Apply ANSI styling only for an interactive terminal that permits it."""
    if not stream.isatty() or "NO_COLOR" in os.environ:
        return text
    return f"\033[{code}m{text}\033[0m"


def print_merge_conflict_guidance(task_id):
    stream = sys.stderr
    print(file=stream)
    print(terminal_style("× Merge conflict", "1;31", stream), file=stream)
    print(f"  Task: {terminal_style(task_id, '1', stream)}", file=stream)
    print("  Git listed the conflicted files above.", file=stream)
    print(file=stream)
    print(terminal_style("What to do", "1;33", stream), file=stream)
    print("  The Task branch and worktree were preserved; the parent remains in Git's conflict state.", file=stream)
    print("  1. Resolve conflicts on the parent branch, or run `git merge --abort` to recover it.", file=stream)
    print("  2. Commit the merge result only after review.", file=stream)
    print("  3. Do not rerun `aiw wt pull` after that commit.", file=stream)
    print(file=stream)
    print(terminal_style("Next commands", "1;36", stream), file=stream)
    print("  " + terminal_style(f"aiw task workflow delivery {task_id} merged", "36", stream), file=stream)
    print("  " + terminal_style(f"aiw archive {task_id} --cleanup-wt --delete-branch", "36", stream), file=stream)


def run_cmd(cmd):
    print(f"> {' '.join(cmd)}", file=sys.stderr)
    p = subprocess.Popen(cmd, cwd=ROOT)
    p.communicate()
    return p.returncode


def run_cmd_at(cwd, cmd):
    print(f"> (in {cwd}) {' '.join(cmd)}", file=sys.stderr)
    p = subprocess.Popen(cmd, cwd=cwd)
    p.communicate()
    return p.returncode


def task_dir(task_id):
    return CHANGES_DIR / task_id


def task_meta_path(task_id):
    primary = RUNTIME_TASKS_DIR / task_id / "task.toml"
    legacy = RUNTIME_TASKS_DIR / task_id / "tasks.toml"
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
    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)


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
    task_path = f"openspec/changes/{task_id}/tasks.md"
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
    if delete_branch:
        if run_cmd(["git", "branch", "-d", branch]) != 0:
            return 2
        meta["branch"] = ""
        write_task_meta(meta_path, meta)
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
    binary = aiw_binary()
    if not binary:
        print("discard completed, but the aiw executable was not found to record delivery", file=sys.stderr)
        return 2
    if run_cmd([binary, "task", "workflow", "delivery", task_id, "discarded"]) != 0:
        print("discard completed, but delivery recording failed; run `aiw task workflow delivery " + task_id + " discarded`", file=sys.stderr)
        return 2
    print(f"delivery: discarded {branch}")
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
    print(f"You can run `aiw wt rm $task_id --delete-branch`, and then `aiw wt prune` to clean up.")
    return 0


def worktree_context(task_id):
    td = task_dir(task_id)
    if not td.exists():
        print(f"task not found: {task_id}", file=sys.stderr)
        return None
    meta_path = task_meta_path(task_id)
    meta = read_task_meta(meta_path)
    if meta.get("workspace_kind", "").strip() != "isolated":
        print("operation requires a verified isolated workspace", file=sys.stderr)
        return None
    branch = meta.get("branch", "").strip()
    parent = meta.get("parent_branch", "").strip()
    wt = meta.get("worktree", "").strip()
    if not branch or not parent or not wt:
        print("task metadata must include branch, parent_branch, and worktree", file=sys.stderr)
        return None
    if not Path(wt).is_absolute():
        wt = str((ROOT / wt).resolve())
    if not Path(wt).is_dir():
        print(f"worktree does not exist: {wt}", file=sys.stderr)
        return None
    return meta_path, meta, Path(wt), branch, parent


def print_git_status(label, cwd):
    print(f"{label} git status:")
    status = subprocess.run(
        ["git", "status", "--short", "--branch"], cwd=cwd,
        text=True, capture_output=True, check=False,
    )
    if status.returncode != 0:
        print("  unavailable")
        return False
    lines = status.stdout.splitlines()
    if not lines:
        print("  clean (no status output)")
    else:
        for line in lines:
            print(f"  {line}")
    return not any(line and not line.startswith("##") for line in lines)


def print_last_commit(label, cwd):
    commit = subprocess.run(
        ["git", "log", "-1", "--oneline"], cwd=cwd,
        text=True, capture_output=True, check=False,
    )
    print(f"{label} last commit: {commit.stdout.strip() or 'unavailable'}")


def status_cmd(task_id):
    td = task_dir(task_id)
    if not td.exists():
        print(f"task not found: {task_id}", file=sys.stderr)
        return 2
    meta = read_task_meta(task_meta_path(task_id))
    branch = meta.get("branch", "").strip()
    parent = meta.get("parent_branch", "").strip()
    wt_value = meta.get("worktree", "").strip()
    if not branch or not parent or not wt_value:
        print("status: invalid task metadata", file=sys.stderr)
        return 2

    wt = Path(wt_value)
    if not wt.is_absolute():
        wt = (ROOT / wt).resolve()
    registry = subprocess.run(
        ["git", "worktree", "list", "--porcelain"], cwd=ROOT,
        text=True, capture_output=True, check=False,
    )
    registered_paths = [
        Path(line[9:].strip()).resolve()
        for line in registry.stdout.splitlines()
        if line.startswith("worktree ")
    ]
    registered = any(os.path.normcase(path) == os.path.normcase(wt) for path in registered_paths)
    exists = wt.is_dir()

    print(f"task: {task_id}")
    print(f"workspace: {wt_value}")
    print(f"workspace exists: {'yes' if exists else 'no'}")
    print(f"worktree registered: {'yes' if registered else 'no'}")
    print(f"branch: {branch}")
    print(f"parent: {parent}")
    print(f"delivery: {meta.get('delivery', 'unknown')}")

    reasons = []

    def add_reason(reason, remedy):
        reasons.append((reason, remedy))

    if not exists:
        add_reason(
            "worktree path does not exist",
            f"run `aiw wt add {task_id} {parent}` or repair the task worktree metadata",
        )
    if not registered:
        add_reason(
            "worktree is not registered by Git",
            f"run `git worktree repair` and then `aiw wt status {task_id}`",
        )

    worktree_branch = ""
    worktree_clean = False
    if exists:
        branch_result = subprocess.run(
            ["git", "branch", "--show-current"], cwd=wt,
            text=True, capture_output=True, check=False,
        )
        worktree_branch = branch_result.stdout.strip()
        clean_result = subprocess.run(
            ["git", "status", "--porcelain"], cwd=wt,
            text=True, capture_output=True, check=False,
        )
        worktree_clean = clean_result.returncode == 0 and not clean_result.stdout.strip()
    print(f"current worktree branch: {worktree_branch or 'unavailable'}")
    print(f"worktree clean: {'yes' if worktree_clean else 'no'}")
    if exists:
        print_git_status("worktree", wt)
        print_last_commit("worktree", wt)
    if worktree_branch != branch:
        add_reason(
            "worktree branch does not match task metadata",
            f"switch the worktree to `{branch}` or update task metadata after verifying it",
        )
    if not worktree_clean:
        add_reason(
            "worktree has uncommitted changes or status is unavailable",
            f"run `aiw wt commit {task_id} \"message\"`, or inspect it with `git -C {wt_value} status`",
        )

    root_branch_result = subprocess.run(
        ["git", "branch", "--show-current"], cwd=ROOT,
        text=True, capture_output=True, check=False,
    )
    root_branch = root_branch_result.stdout.strip()
    root_clean_result = subprocess.run(
        ["git", "status", "--porcelain"], cwd=ROOT,
        text=True, capture_output=True, check=False,
    )
    root_clean = root_clean_result.returncode == 0 and not root_clean_result.stdout.strip()
    print(f"current parent workspace branch: {root_branch or 'unavailable'}")
    print(f"parent workspace clean: {'yes' if root_clean else 'no'}")
    print_git_status("parent workspace", ROOT)
    print_last_commit("parent workspace", ROOT)
    if root_branch != parent:
        add_reason(
            "current workspace is not on parent branch",
            f"run `git switch {parent}` in the parent workspace",
        )
    if not root_clean:
        add_reason(
            "parent workspace has uncommitted changes or status is unavailable",
            "commit the parent changes, or run `git stash push -u -m \"before task merge\"`",
        )

    merge_preview_ok = False
    merge_preview_detail = ""
    if not reasons:
        preview = subprocess.run(
            ["git", "merge-tree", "--write-tree", parent, branch], cwd=ROOT,
            text=True, capture_output=True, check=False,
        )
        merge_preview_ok = preview.returncode == 0
        if not merge_preview_ok:
            detail = (preview.stderr or preview.stdout).strip().splitlines()
            merge_preview_detail = detail[0] if detail else "no Git diagnostic was returned"
            add_reason(
                "merge preview reports conflicts or the branches are unavailable",
                f"preview with `git merge-tree {parent} {branch}`; if not merged yet, run `aiw wt pull {task_id}`; if you already merged and committed manually, record delivery instead",
            )
    print(f"merge preview: {'clean' if merge_preview_ok else 'not checked or conflicts'}")
    comparison = subprocess.run(
        ["git", "rev-list", "--left-right", "--count", f"{parent}...{branch}"],
        cwd=ROOT, text=True, capture_output=True, check=False,
    )
    if comparison.returncode == 0 and comparison.stdout.strip():
        behind, ahead = comparison.stdout.split()
        print(f"branch commits: ahead {ahead}, behind {behind} relative to {parent}")
    else:
        print("branch commits: unavailable")
    if reasons:
        print("merge: not-ready")
        print("reasons:")
        for reason, remedy in reasons:
            print(f"  - {reason}")
            print(f"    remedy: {remedy}")
        if merge_preview_detail:
            print(f"    git detail: {merge_preview_detail}")
        return 0
    print("merge: ready")
    print(f"next: aiw wt pull {task_id}")
    return 0


def commit(task_id, message):
    context = worktree_context(task_id)
    if context is None:
        return 2
    _, _, wt, _, _ = context
    if not message.strip():
        print('usage: aiw wt commit <task-id> "message"', file=sys.stderr)
        return 2
    if run_cmd_at(wt, ["git", "add", "-A"]) != 0:
        return 2
    status = subprocess.run(
        ["git", "status", "--porcelain"], cwd=wt, text=True,
        capture_output=True, check=False,
    )
    if status.returncode != 0:
        print("cannot inspect worktree status", file=sys.stderr)
        return 2
    if not status.stdout.strip():
        print("worktree is clean; nothing to commit")
        return 0
    return run_cmd_at(wt, ["git", "commit", "-m", message])


def aiw_binary():
    configured = os.environ.get("AIW_BIN", "").strip()
    if configured:
        return configured
    root = os.environ.get("AIW_ROOT", "").strip()
    if root:
        for name in ("aiw.exe", "aiw"):
            candidate = Path(root) / name
            if candidate.exists():
                return str(candidate)
    return shutil.which("aiw") or shutil.which("aiw.exe")


def conflicted_paths():
    result = subprocess.run(
        ["git", "diff", "--name-only", "--diff-filter=U", "-z"], cwd=ROOT,
        capture_output=True, check=False,
    )
    if result.returncode != 0:
        return None
    return [path.decode("utf-8", "surrogateescape") for path in result.stdout.split(b"\0") if path]


def conflict_stage_text(path, stage):
    result = subprocess.run(
        ["git", "show", f":{stage}:{path}"], cwd=ROOT,
        capture_output=True, check=False,
    )
    if result.returncode != 0:
        return None
    if b"\0" in result.stdout:
        return None
    try:
        text = result.stdout.decode("utf-8")
    except UnicodeDecodeError:
        return None
    if len(text) > MERGE_RESOLUTION_MAX_CHARS_PER_VERSION:
        text = text[:MERGE_RESOLUTION_MAX_CHARS_PER_VERSION] + "\n[truncated]\n"
    return text


def write_merge_resolution_handoff(task_id, parent, branch):
    paths = conflicted_paths()
    if paths is None:
        print("cannot inspect conflicted files for resolution handoff", file=sys.stderr)
        return None
    if not paths:
        return None

    base = subprocess.run(
        ["git", "merge-base", parent, branch], cwd=ROOT, text=True,
        capture_output=True, check=False,
    )
    if base.returncode != 0 or not base.stdout.strip():
        print("cannot determine merge base for resolution handoff", file=sys.stderr)
        return None

    entries = []
    exclusions = []
    used_chars = 0
    for path in paths:
        if len(entries) >= MERGE_RESOLUTION_MAX_FILES:
            exclusions.append((path, "proposal file limit reached"))
            break
        reason = conflict_exclusion_reason(path)
        if reason:
            exclusions.append((path, reason))
            continue
        versions = []
        for label, stage in (("base", 1), ("parent", 2), ("task", 3)):
            text = conflict_stage_text(path, stage)
            if text is None:
                versions = []
                break
            versions.append((label, text))
        entry_chars = sum(len(text) for _, text in versions)
        if not versions:
            exclusions.append((path, "not a UTF-8 text conflict in every merge stage"))
            continue
        if used_chars + entry_chars > MERGE_RESOLUTION_MAX_CHARS:
            exclusions.append((path, "proposal character limit reached"))
            continue
        entries.append((path, versions))
        used_chars += entry_chars

    if not entries:
        print("opt-in resolution found no eligible text conflicts; resolve manually", file=sys.stderr)
        for path, reason in exclusions:
            print(f"excluded from agent resolution: {path} ({reason})", file=sys.stderr)
        return None

    lines = [
        "# Merge resolution proposal request",
        "",
        f"Task: {task_id}",
        f"Parent branch: {parent}",
        f"Task branch: {branch}",
        f"Merge base: {base.stdout.strip()}",
        "",
        "Git is intentionally still in its original conflict state.",
        "This is a proposal request only: do not edit files, stage changes, commit, or complete the merge.",
        "",
        "## Requested response",
        "",
        "Save a unified diff that applies to the current conflicted worktree in `proposal.patch` beside this request.",
        "Save a short explanation of each choice in `proposal.md` beside this request.",
        "The explicit `aiw wt resolve apply <task-id> --confirm` command will accept only a patch for the eligible conflicted files.",
        "If a choice cannot be made safely, say so and leave that file for manual resolution.",
        "",
        "## Eligible conflict context",
    ]
    for path, versions in entries:
        lines.extend(["", f"### {path}"])
        for label, text in versions:
            lines.extend(["", f"#### {label}", "```text", text.rstrip("\n"), "```"])
    if exclusions:
        lines.extend(["", "## Manual-resolution exclusions"])
        for path, reason in exclusions:
            lines.append(f"- `{path}`: {reason}")

    handoff_dir = ROOT / ".ai" / "tasks" / task_id / "merge-resolution"
    handoff_dir.mkdir(parents=True, exist_ok=True)
    handoff = handoff_dir / "proposal-request.md"
    temporary = handoff.with_suffix(".tmp")
    temporary.write_text("\n".join(lines) + "\n", encoding="utf-8")
    temporary.replace(handoff)
    return handoff


def merge_resolution_dir(task_id):
    return ROOT / ".ai" / "tasks" / task_id / "merge-resolution"


def proposal_patch_paths(patch):
    """Return patch targets, rejecting patch forms that can escape a handoff."""
    paths = []
    current_path = None
    expects_new_header = False
    for line in patch.splitlines():
        if line.startswith("diff --git "):
            if current_path is not None and expects_new_header:
                return None
            parts = line.split(" ")
            if len(parts) != 4 or not parts[2].startswith("a/") or not parts[3].startswith("b/"):
                return None
            old_path = normalize_repository_path(parts[2][2:])
            new_path = normalize_repository_path(parts[3][2:])
            if old_path != new_path or not old_path:
                return None
            paths.append(old_path)
            current_path = old_path
            expects_new_header = False
        elif line.startswith("--- "):
            if current_path is None or line[4:].split("\t", 1)[0] != "a/" + current_path:
                return None
            expects_new_header = True
        elif line.startswith("+++ "):
            if not expects_new_header or line[4:].split("\t", 1)[0] != "b/" + current_path:
                return None
            expects_new_header = False
    if not paths or expects_new_header or len(paths) != len(set(paths)):
        return None
    return paths


def review_merge_resolution(task_id):
    directory = merge_resolution_dir(task_id)
    request = directory / "proposal-request.md"
    patch = directory / "proposal.patch"
    explanation = directory / "proposal.md"
    if not request.is_file():
        print(f"no merge-resolution handoff exists for {task_id}", file=sys.stderr)
        return 2
    print(request.read_text(encoding="utf-8"))
    if explanation.is_file():
        print("## Proposed explanation\n")
        print(explanation.read_text(encoding="utf-8"))
    else:
        print("No proposed explanation has been saved yet.")
    if patch.is_file():
        print("## Proposed patch\n")
        print(patch.read_text(encoding="utf-8"))
    else:
        print("No proposed patch has been saved yet.")
    print("Review only: no files were edited, staged, committed, or merged.")
    return 0


def apply_merge_resolution(task_id, confirmed):
    if not confirmed:
        print("refusing to apply a proposal without --confirm", file=sys.stderr)
        return 2
    patch_path = merge_resolution_dir(task_id) / "proposal.patch"
    if not patch_path.is_file():
        print(f"no proposed patch exists for {task_id}", file=sys.stderr)
        return 2
    try:
        targets = proposal_patch_paths(patch_path.read_text(encoding="utf-8"))
    except UnicodeDecodeError:
        targets = None
    if targets is None:
        print("proposal patch must modify only existing repository files with standard diff headers", file=sys.stderr)
        return 2
    conflicts = conflicted_paths()
    if not conflicts:
        print("no unresolved merge conflicts exist; refusing to apply a stale proposal", file=sys.stderr)
        return 2
    conflict_set = {normalize_repository_path(path) for path in conflicts}
    for path in targets:
        if path not in conflict_set:
            print(f"proposal target is not an unresolved conflict: {path}", file=sys.stderr)
            return 2
        reason = conflict_exclusion_reason(path)
        if reason:
            print(f"proposal target remains manual-only: {path} ({reason})", file=sys.stderr)
            return 2
    check = subprocess.run(
        ["git", "apply", "--check", "--recount", str(patch_path)], cwd=ROOT,
        text=True, capture_output=True, check=False,
    )
    if check.returncode != 0:
        print("proposal patch does not apply; conflict state was preserved", file=sys.stderr)
        if check.stderr:
            print(check.stderr, file=sys.stderr, end="")
        return 2
    applied = subprocess.run(["git", "apply", "--recount", str(patch_path)], cwd=ROOT, check=False)
    if applied.returncode != 0:
        print("proposal patch could not be applied; conflict state was preserved", file=sys.stderr)
        return 2
    staged = subprocess.run(["git", "add", "--", *targets], cwd=ROOT, check=False)
    if staged.returncode != 0:
        print("proposal was applied but could not be staged; inspect the working tree before continuing", file=sys.stderr)
        return 2
    print("proposal applied and staged for review; no commit was created and the merge remains incomplete.")
    return 0


def pull(task_id, resolve_agent=False):
    context = worktree_context(task_id)
    if context is None:
        return 2
    meta_path, meta, wt, branch, parent = context
    status = subprocess.run(
        ["git", "status", "--porcelain"], cwd=wt, text=True,
        capture_output=True, check=False,
    )
    if status.returncode != 0:
        print("cannot inspect worktree status", file=sys.stderr)
        return 2
    if status.stdout.strip():
        print(f"worktree has uncommitted changes; run `aiw wt commit {task_id} \"message\"` first", file=sys.stderr)
        return 2
    current = subprocess.run(
        ["git", "branch", "--show-current"], cwd=ROOT, text=True,
        capture_output=True, check=False,
    )
    if current.returncode != 0 or current.stdout.strip() != parent:
        print(f"run pull from parent branch {parent!r}; current branch is {current.stdout.strip()!r}", file=sys.stderr)
        return 2
    if run_cmd(["git", "merge", branch]) != 0:
        binary = aiw_binary()
        if binary:
            run_cmd([binary, "task", "workflow", "delivery-failed", task_id, "merge", "Git merge failed; parent worktree conflict state preserved"])
        if resolve_agent:
            handoff = write_merge_resolution_handoff(task_id, parent, branch)
            if handoff is not None:
                print(f"proposal handoff: {handoff}", file=sys.stderr)
                print("Review it with an agent; it has not edited, staged, committed, or completed the merge.", file=sys.stderr)
        print_merge_conflict_guidance(task_id)
        return 2
    binary = aiw_binary()
    if not binary:
        print("merge completed, but the aiw executable was not found to record delivery", file=sys.stderr)
        return 2
    if run_cmd([binary, "task", "workflow", "delivery", task_id, "merged"]) != 0:
        print("merge completed, but delivery recording failed; run `aiw task workflow delivery " + task_id + " merged`", file=sys.stderr)
        return 2
    print(f"delivery: merged {branch} into {parent}")
    print(f"next: aiw archive {task_id} --cleanup-wt --delete-branch")
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
    print("  commit <task-id> \"message\"             Commit all worktree changes.")
    print("  pull <task-id> [--resolve=agent]         Merge task branch; optionally create a conflict proposal handoff.")
    print("  resolve review <task-id>                 Display an agent resolution proposal without changing Git state.")
    print("  resolve apply <task-id> --confirm        Apply and stage an accepted proposal; never commits or completes a merge.")
    print("  status <task-id>                         Show worktree and merge readiness.")
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
    if sub == "commit":
        if len(rest) != 2:
            print('usage: aiw wt commit <task-id> "message"', file=sys.stderr)
            return 2
        return commit(rest[0], rest[1])
    if sub == "pull":
        if not rest or len(rest) > 2 or (len(rest) == 2 and rest[1] != "--resolve=agent"):
            print("usage: aiw wt pull <task-id> [--resolve=agent]", file=sys.stderr)
            return 2
        return pull(rest[0], "--resolve=agent" in rest[1:])
    if sub == "resolve":
        if len(rest) == 2 and rest[0] == "review":
            return review_merge_resolution(rest[1])
        if len(rest) == 3 and rest[0] == "apply" and rest[2] == "--confirm":
            return apply_merge_resolution(rest[1], True)
        if len(rest) >= 1 and rest[0] == "apply":
            task_id = rest[1] if len(rest) >= 2 else "<task-id>"
            return apply_merge_resolution(task_id, False)
        print("usage: aiw wt resolve review <task-id> | aiw wt resolve apply <task-id> --confirm", file=sys.stderr)
        return 2
    if sub == "status":
        if len(rest) != 1:
            print("usage: aiw wt status <task-id>", file=sys.stderr)
            return 2
        return status_cmd(rest[0])
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
