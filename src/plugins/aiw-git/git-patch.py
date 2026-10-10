#!/usr/bin/env python3
"""Create and apply working-tree patches with recovery guidance."""

import argparse
import subprocess
import sys
from pathlib import Path


META = {
    "name": "patch",
    "short": "Create or apply a patch with practical guidance.",
    "long": "Create a binary-safe diff and companion Markdown guide, or check and apply a patch with failure advice.",
    "usage": "aiw git patch create FILE.patch [--staged | --worktree | --from A --to B]\n  aiw git patch apply FILE.patch",
    "args": [
        {"flag": "create FILE.patch", "description": "Export tracked changes; default includes staged and unstaged changes."},
        {"flag": "--staged", "description": "Export only staged changes."},
        {"flag": "--worktree", "description": "Export only unstaged changes."},
        {"flag": "--from A --to B", "description": "Export the diff from commit or branch A to B."},
        {"flag": "apply FILE.patch", "description": "Check and apply to the working tree, without staging."},
    ],
    "examples": [
        "aiw git patch create changes.patch",
        "aiw git patch create staged.patch --staged",
        "aiw git patch create release.patch --from main --to feature/topic",
        "aiw git patch apply changes.patch",
    ],
}


def git(*args, input_data=None):
    return subprocess.run(["git", *args], input=input_data, capture_output=True)


def error(message):
    print(f"patch: {message}", file=sys.stderr)
    return 1


def git_error(result):
    return result.stderr.decode("utf-8", errors="replace").strip() or f"Git exited with {result.returncode}"


def guide(filename, source, summary, refs=None):
    scope = {
        "all": "HEAD 到当前工作区的已跟踪文件改动（含暂存和未暂存）",
        "staged": "仅暂存区的已跟踪文件改动",
        "worktree": "仅未暂存的已跟踪文件改动",
        "range": "两个提交或分支之间的文件改动",
    }[source]
    provenance = ""
    if refs is not None:
        from_ref, from_id, to_ref, to_id = refs
        provenance = (
            f"\n\n起点：`{from_ref}` → `{from_id}`；终点：`{to_ref}` → `{to_id}`。"
            "比较的是两个端点的文件树，不隐式使用 merge-base。目标设备最好具有起点提交对应的文件版本；"
            "不要求保留相同的分支名称。"
        )
    return (
        f"# 补丁说明：{filename}\n\n"
        f"来源：{scope}。这是 `git diff --binary` 补丁，不包含未跟踪文件，也不包含提交元数据。{provenance}\n\n"
        f"## 改动概览\n\n```text\n{summary.strip()}\n```\n\n"
        "## 应用\n\n"
        f"在目标仓库执行 `aiw git patch apply {filename}`。命令先检查，再应用到工作区；不会自动暂存或提交。"
        "先用 `git status --short` 查看目标仓库状态，应用后用 `git diff` 检查并按需 `git add`。\n\n"
        "## 应用失败时\n\n"
        "读取命令输出的 Git 错误和针对性建议。先确认补丁是否已应用、目标文件是否已有本地改动，"
        "以及目标仓库是否具有相应的文件版本。可用 `git apply --stat` 查看补丁涉及的文件。"
        "保留现有改动，必要时在临时干净分支中评估 `git apply --3way` 并人工解决冲突；"
        "不要盲目执行 `git reset --hard`。命令不会自动尝试 `--3way` 或 `--reject`。\n"
    )


def resolve_commit(ref):
    result = git("rev-parse", "--verify", "--quiet", "--end-of-options", f"{ref}^{{commit}}")
    if result.returncode or not result.stdout.strip():
        raise ValueError(f"cannot resolve commit or branch: {ref}")
    return result.stdout.decode("ascii").strip()


def create(filename, source, from_ref=None, to_ref=None):
    patch = Path(filename)
    if patch.suffix.lower() != ".patch":
        return error("output filename must end in .patch")
    note = Path(str(patch) + ".md")
    if patch.exists() or note.exists():
        return error("patch or companion guide already exists; choose another filename")
    args = ["diff", "--binary", "--full-index", "--no-ext-diff", "--no-textconv"]
    refs = None
    try:
        if source == "range":
            from_id = resolve_commit(from_ref)
            to_id = resolve_commit(to_ref)
            refs = (from_ref, from_id, to_ref, to_id)
            args.extend((from_id, to_id))
        elif source == "all":
            args.append("HEAD")
        elif source == "staged":
            args.append("--cached")
        args.append("--")
        result = git(*args)
    except (OSError, ValueError) as exc:
        return error(f"cannot create diff: {exc}")
    if result.returncode:
        return error(f"cannot create diff: {git_error(result)}")
    if not result.stdout:
        return error(f"no {source} tracked changes to export")
    try:
        stat = git("apply", "--stat", input_data=result.stdout)
    except OSError as exc:
        return error(f"cannot summarize patch: {exc}")
    if stat.returncode:
        return error(f"cannot summarize patch: {git_error(stat)}")
    created_patch = False
    created_note = False
    try:
        with patch.open("xb") as output:
            created_patch = True
            output.write(result.stdout)
        with note.open("x", encoding="utf-8", newline="\n") as output:
            created_note = True
            output.write(guide(patch.name, source, stat.stdout.decode("utf-8", errors="replace"), refs))
    except OSError as exc:
        if created_note:
            note.unlink(missing_ok=True)
        if created_patch:
            patch.unlink(missing_ok=True)
        return error(f"cannot write patch and guide: {exc}")
    print(f"Created {patch} and {note}")
    print("The patch includes tracked changes only; untracked files need separate handling.")
    return 0


def apply(filename):
    patch = Path(filename)
    if not patch.is_file():
        return error(f"patch file does not exist: {patch}")
    try:
        if patch.stat().st_size == 0:
            return error("patch file is empty")
        path = str(patch.resolve())
        checked = git("apply", "--check", path)
        if checked.returncode:
            print(f"patch: apply check failed: {git_error(checked)}", file=sys.stderr)
            reversed_check = git("apply", "--reverse", "--check", path)
            if reversed_check.returncode == 0:
                print("Suggestion: the patch may already be applied. Inspect `git diff` and `git status --short` before retrying.", file=sys.stderr)
            else:
                print("Suggestion: run `git apply --stat <patch>` to list affected files, then inspect `git status --short` and `git diff`. Check the base version; consider a temporary clean branch for manual conflict resolution. No patch was applied.", file=sys.stderr)
            return checked.returncode
        applied = git("apply", path)
    except OSError as exc:
        return error(f"cannot check or apply patch: {exc}")
    if applied.returncode:
        print(f"patch: apply failed: {git_error(applied)}", file=sys.stderr)
        print("Suggestion: inspect `git status --short` and `git diff` for possible partial changes before retrying.", file=sys.stderr)
        return applied.returncode
    print(f"Applied {patch} to the working tree. Inspect `git diff` and stage changes when ready.")
    return 0


def main(argv):
    parser = argparse.ArgumentParser(prog="aiw git patch", description=META["short"])
    actions = parser.add_subparsers(dest="action", required=True)
    create_parser = actions.add_parser("create", help="Create a patch and companion guide")
    create_parser.add_argument("file", metavar="FILE.patch")
    scope = create_parser.add_mutually_exclusive_group()
    scope.add_argument("--staged", action="store_true")
    scope.add_argument("--worktree", action="store_true")
    create_parser.add_argument("--from", dest="from_ref", metavar="REF")
    create_parser.add_argument("--to", dest="to_ref", metavar="REF")
    apply_parser = actions.add_parser("apply", help="Check and apply a patch")
    apply_parser.add_argument("file", metavar="FILE.patch")
    args = parser.parse_args(argv)
    if args.action == "create":
        if bool(args.from_ref) != bool(args.to_ref):
            return error("--from and --to must be provided together")
        if args.from_ref and (args.staged or args.worktree):
            return error("--from/--to cannot be combined with --staged or --worktree")
        source = "range" if args.from_ref else "staged" if args.staged else "worktree" if args.worktree else "all"
        return create(args.file, source, args.from_ref, args.to_ref)
    return apply(args.file)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
