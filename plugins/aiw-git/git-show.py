#!/usr/bin/env python3
"""aiw git show

Unified wrapper for git inspection commands.
Consolidates conflicts, log, status, unpulled, unpushed, and whatchanged.
"""
import sys
import os
import importlib.util

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')
spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'show',
    'short': 'Inspect repository state, history, and conflict views.',
    'long': 'Collection of read-only git inspection helpers including status, log, unpulled, unpushed, conflicts, and whatchanged.',
    'usage': 'aiw git show <view> [args]',
    'args': [
        {'flag': '<view>', 'description': 'One of: conflicts, log, status, unpulled, unpushed, whatchanged.'},
    ],
    'examples': [
        'aiw git show status',
        'aiw git show log -n 20',
        'aiw git show conflicts --check',
    ],
}

# --- Subcommand: conflicts ---
META_CONFLICTS = {
    'name': 'conflicts',
    'short': 'Inspect and help resolve git conflicts.',
    'long': 'Shows conflicted files, diffs, staging status, and provides guidance for resolving merge/rebase conflicts.',
    'usage': 'conflicts [--diff] [--check] [--staged]',
    'args': [
        {'flag': '--diff', 'description': 'Show all unmerged hunks in diff format.'},
        {'flag': '--check', 'description': 'Detect remaining conflict markers in files.'},
        {'flag': '--staged', 'description': 'Show staged resolved files.'}
    ],
    'examples': [
        'conflicts',
        'conflicts --diff',
    ],
}

def cmd_conflicts(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_CONFLICTS)
        return 0
    if core.has_flag(argv, '--diff'):
        return core.run_cmd(['git', 'diff', '--diff-filter=U'])
    if core.has_flag(argv, '--check'):
        return core.run_cmd(['git', 'diff', '--check'])
    if core.has_flag(argv, '--staged'):
        return core.run_cmd(['git', 'diff', '--staged'])
    out = core.git_output(['git', 'diff', '--name-only', '--diff-filter=U'])
    if not out:
        print('No unresolved conflicts found.')
        return 0
    files = out.strip().split('\n')
    print(f'{len(files)} conflicted file(s):\n')
    for f in files:
        print(' ', f)
    print("\nNext steps:\n  1. Edit each file above and resolve markers.\n  2. aiw git show conflicts --check\n  3. git add <file>\n  4. git commit")
    return 0

# --- Subcommand: log ---
META_LOG = {
    'name': 'log',
    'short': 'Show a formatted commit log with several styles.',
    'long': 'Displays the commit history. Styles: lg (default), l (one-line), hist (absolute dates).',
    'usage': 'log [lg|l|hist] [-n <count>]',
    'args': [
        {'flag': 'lg', 'description': 'Style with graph, relative dates, and decorations.'},
        {'flag': 'l', 'description': 'One-line format for compact view.'},
        {'flag': 'hist', 'description': 'Graph with absolute dates for detailed history.'},
        {'flag': '-n <count>', 'description': 'Limit the number of commits shown.'}
    ],
    'examples': ['log lg', 'log -n 50']
}

def cmd_log(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_LOG)
        return 0
    style, n, i, remain = "lg", "20", 0, []
    while i < len(argv):
        if argv[i] in ("lg", "l", "hist"): style = argv[i]
        elif argv[i] == "-n" and i + 1 < len(argv):
            n = argv[i + 1]
            i += 1
        else: remain.append(argv[i])
        i += 1
    if style == "l":
        return core.run_cmd(["git", "log", "--pretty=oneline", "-n", n])
    if style == "hist":
        return core.run_cmd(["git", "log", "--color", "--graph", "--pretty=format:%Cred%h%Creset %Cgreen%ad%Creset | %s %C(yellow)%d%Creset %C(bold blue)<%an>%Creset", "--date=short", "-n", n])
    if style == "lg":
        return core.run_cmd(["git", "log", "--all", "--color", "--graph", "--pretty=format:%Cred%h%Creset - %C(yellow)%d%Creset %s %Cgreen[%cr] %C(bold blue)<%an>%Creset", "--abbrev-commit", "--date=relative", "-n", n])
    return core.run_cmd(['git', 'log', '-n', n] + remain)

# --- Subcommand: status ---
META_STATUS = {
    'name': 'status',
    'short': 'Show concise working-tree status.',
    'usage': 'status',
    'args': [],
    'examples': ['status']
}

def cmd_status(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_STATUS)
        return 0
    return core.run_cmd(['git', 'status', '-sb'])

# --- Subcommand: unpulled ---
META_UNPULLED = {
    'name': 'unpulled',
    'short': 'Show commits present on upstream but not pulled locally.',
    'usage': 'unpulled',
    'args': [],
    'examples': ['unpulled']
}

def cmd_unpulled(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_UNPULLED)
        return 0
    up = core.git_output(['git', 'rev-parse', '--abbrev-ref', '@{u}']).strip()
    if not up:
        print('no upstream configured', file=sys.stderr)
        return 2
    return core.run_cmd(['git', 'log', '--oneline', f'HEAD..{up}'])

# --- Subcommand: unpushed ---
META_UNPUSHED = {
    'name': 'unpushed',
    'short': 'Show commits not pushed to upstream.',
    'usage': 'unpushed',
    'args': [],
    'examples': ['unpushed']
}

def cmd_unpushed(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_UNPUSHED)
        return 0
    up = core.git_output(['git', 'rev-parse', '--abbrev-ref', '@{u}']).strip()
    if not up:
        print('no upstream configured', file=sys.stderr)
        return 2
    return core.run_cmd(['git', 'log', '--oneline', f'{up}..HEAD'])

# --- Subcommand: whatchanged ---
META_WHATCHANGED = {
    'name': 'whatchanged',
    'short': 'Show changes between commits (git whatchanged).',
    'usage': 'whatchanged [<rev-range>] [--names]',
    'args': [],
    'examples': ['whatchanged']
}

def cmd_whatchanged(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_WHATCHANGED)
        return 0
    sha, names_only = "", core.has_flag(argv, "--names")
    for arg in argv:
        if arg != "--names":
            sha = arg
            break
    if sha == "":
        return core.run_cmd(["git", "whatchanged"])
    if names_only:
        return core.run_cmd(["git", "diff-tree", "--no-commit-id", "--name-only", "-r", sha])
    return core.run_cmd(["git", "show", "--stat", sha])

SUBCOMMANDS = {
    'conflicts': cmd_conflicts,
    'log': cmd_log,
    'status': cmd_status,
    'unpulled': cmd_unpulled,
    'unpushed': cmd_unpushed,
    'whatchanged': cmd_whatchanged,
}

def main(argv):
    if not argv or argv[0] in {'-h', '--help'}:
        core.print_help_meta(META)
        print("\nAvailable views:")
        for sub in sorted(SUBCOMMANDS.keys()):
            print(f"  {sub}")
        return 0
    sub = argv[0]
    if sub in SUBCOMMANDS:
        return SUBCOMMANDS[sub](argv[1:])
    print(f"Unknown subcommand: {sub}", file=sys.stderr)
    return 1

if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))


