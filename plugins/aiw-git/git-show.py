#!/usr/bin/env python3
"""aiw git show

Unified wrapper for git inspection commands.
Consolidates repository and file history inspection commands.
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
    'short': 'Inspect repository state, history, files, and conflicts.',
    'long': 'Collection of read-only git inspection helpers including repository status, commit logs, file evolution, blame, historical file content, upstream differences, and conflicts.',
    'usage': 'aiw git show <view> [args]',
    'args': [
        {'flag': 'file', 'description': 'Show one file history, following renames by default.'},
        {'flag': 'blame', 'description': 'Show line-by-line attribution for one file.'},
        {'flag': 'file-at', 'description': 'Show one file as it existed at a revision.'},
        {'flag': 'lines', 'description': 'Track a line range or function with git log -L.'},
        {'flag': '<view>', 'description': 'Other views: conflicts, log, status, unpulled, unpushed, whatchanged.'},
    ],
    'examples': [
        'aiw git show status',
        'aiw git show log -n 20',
        'aiw git show file --full src/App.java',
        'aiw git show blame src/App.java',
        'aiw git show file-at HEAD~3 README.md',
        'aiw git show lines 10,30 src/App.java',
        'aiw git show lines :main src/main.py',
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

# --- Subcommand: file ---
META_FILE = {
    'name': 'file',
    'short': 'Show the commit history for one file.',
    'long': 'Follows renames by default and supports concise, patch, statistics, graph, and full-evolution output.',
    'usage': 'file [--oneline|--patch|--stat|--graph|--full] [--no-follow] <path>',
    'args': [
        {'flag': '--oneline', 'description': 'Show compact one-line commits.'},
        {'flag': '--patch', 'description': 'Show the diff from every commit.'},
        {'flag': '--stat', 'description': 'Show file change statistics.'},
        {'flag': '--graph', 'description': 'Show a decorated one-line commit graph.'},
        {'flag': '--full', 'description': 'Show the recommended patch and statistics view.'},
        {'flag': '--no-follow', 'description': 'Do not follow history across renames.'},
        {'flag': '<path>', 'description': 'Repository-relative file path.'},
    ],
    'examples': [
        'file README.md',
        'file --oneline src/App.java',
        'file --full src/App.java',
    ],
}

FILE_MODES = {
    '--oneline': ['--oneline'],
    '--patch': ['-p'],
    '--stat': ['--stat'],
    '--graph': ['--graph', '--decorate', '--oneline'],
    '--full': ['-p', '--stat'],
}


def usage_error(meta, message):
    print(f"error: {message}", file=sys.stderr)
    core.print_help_meta(meta)
    return 2


def cmd_file(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_FILE)
        return 0

    modes, paths = [], []
    follow, paths_only = True, False
    for arg in argv:
        if paths_only:
            paths.append(arg)
        elif arg == '--':
            paths_only = True
        elif arg == '--no-follow':
            follow = False
        elif arg in FILE_MODES:
            modes.append(arg)
        elif arg.startswith('-'):
            return usage_error(META_FILE, f"unknown option: {arg}")
        else:
            paths.append(arg)

    if len(modes) > 1:
        return usage_error(META_FILE, 'choose only one output mode')
    if len(paths) != 1:
        return usage_error(META_FILE, 'file requires exactly one path')

    cmd = ['git', 'log']
    if follow:
        cmd.append('--follow')
    if modes:
        cmd.extend(FILE_MODES[modes[0]])
    cmd.extend(['--', paths[0]])
    return core.run_cmd(cmd)


# --- Subcommand: blame ---
META_BLAME = {
    'name': 'blame',
    'short': 'Show who last changed each line of a file.',
    'usage': 'blame <path>',
    'args': [
        {'flag': '<path>', 'description': 'Repository-relative file path.'},
    ],
    'examples': ['blame src/App.java'],
}


def cmd_blame(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_BLAME)
        return 0
    paths = argv[1:] if argv[:1] == ['--'] else argv
    if len(paths) != 1:
        return usage_error(META_BLAME, 'blame requires exactly one path')
    return core.run_cmd(['git', 'blame', '--', paths[0]])


# --- Subcommand: file-at ---
META_FILE_AT = {
    'name': 'file-at',
    'short': 'Show a file as it existed at a revision.',
    'usage': 'file-at <revision> <path>',
    'args': [
        {'flag': '<revision>', 'description': 'Commit, tag, branch, or other Git revision.'},
        {'flag': '<path>', 'description': 'Repository-relative file path at that revision.'},
    ],
    'examples': [
        'file-at HEAD~3 README.md',
        'file-at a17b2f3 src/App.java',
    ],
}


def cmd_file_at(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_FILE_AT)
        return 0
    if len(argv) != 2:
        return usage_error(META_FILE_AT, 'file-at requires a revision and path')
    revision, path = argv
    return core.run_cmd(['git', 'show', f'{revision}:{path}'])


# --- Subcommand: lines ---
META_LINES = {
    'name': 'lines',
    'short': 'Track the history of a line range or function.',
    'long': 'Passes a selector and file path to git log -L. Use start,end for lines or :name for a function.',
    'usage': 'lines <selector> <path>',
    'args': [
        {'flag': '<selector>', 'description': 'Line range such as 10,30 or function such as :main.'},
        {'flag': '<path>', 'description': 'Repository-relative file path.'},
    ],
    'examples': [
        'lines 10,30 src/App.java',
        'lines :main src/main.py',
    ],
}


def cmd_lines(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_LINES)
        return 0
    if len(argv) != 2:
        return usage_error(META_LINES, 'lines requires a selector and path')
    selector, path = argv
    return core.run_cmd(['git', 'log', '-L', f'{selector}:{path}'])

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
    'blame': cmd_blame,
    'conflicts': cmd_conflicts,
    'file': cmd_file,
    'file-at': cmd_file_at,
    'lines': cmd_lines,
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


