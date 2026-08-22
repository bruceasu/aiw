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
if spec is None or spec.loader is None:
    raise RuntimeError(f'Cannot load aiw git core module from {CORE_PATH}')
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'show',
    'short': 'Inspect repository state, history, files, and conflicts.',
    'long': 'Collection of read-only git inspection helpers including repository status, commit logs, file evolution, blame, historical file content, upstream differences, and conflicts.',
    'usage': 'aiw git show <view> [args]',
    'args': [
        {'flag': 'st', 'description': 'Short subcommand for status.'},
        {'flag': 'lg', 'description': 'Short subcommand for graph log (built-in last 20).'},
        {'flag': 'l1', 'description': 'Short subcommand for one-line log (built-in last 20).'},
        {'flag': 'fh', 'description': 'Short subcommand for full file history (built-in options).'},
        {'flag': 'in|out', 'description': 'Short subcommands for unpulled / unpushed.'},
        {'flag': 'cf', 'description': 'Short subcommand for commit-files status mode.'},
        {'flag': 'c|ck', 'description': 'Short subcommands for conflict list / conflict check.'},
        {'flag': 'file-oneline|file-patch|file-stat|file-graph', 'description': 'Clear file history mode commands.'},
        {'flag': 'log-date', 'description': 'Clear date log command (built-in last 20).'},
        {'flag': 'commit-files-names|commit-files-root', 'description': 'Clear commit-files mode commands.'},
        {'flag': 'conflicts-diff|conflicts-staged', 'description': 'Clear conflict detail commands.'},
        {'flag': 'whatchanged-names', 'description': 'Clear whatchanged names-only command.'},
        {'flag': 'file', 'description': 'Show one file history, following renames by default.'},
        {'flag': 'file-hist', 'description': 'Show full file history (patch + stat), following renames.'},
        {'flag': 'blame', 'description': 'Show line-by-line attribution for one file.'},
        {'flag': 'file-at', 'description': 'Show one file as it existed at a revision.'},
        {'flag': 'lines', 'description': 'Track a line range or function with git log -L.'},
        {'flag': '<view>', 'description': 'Other views: commit-files, conflicts, log, status, unpulled, unpushed, whatchanged.'},
    ],
    'examples': [
        'aiw git show st',
        'aiw git show lg',
        'aiw git show l1',
        'aiw git show file-oneline src/App.java',
        'aiw git show file-patch src/App.java',
        'aiw git show file-stat src/App.java',
        'aiw git show file-graph src/App.java',
        'aiw git show fh src/App.java',
        'aiw git show in',
        'aiw git show out',
        'aiw git show commit-files-names HEAD~1',
        'aiw git show commit-files-root HEAD',
        'aiw git show conflicts-diff',
        'aiw git show conflicts-staged',
        'aiw git show ck',
        'aiw git show whatchanged-names HEAD~1',
        'aiw git show file-hist src/App.java',
        'aiw git show status',
        'aiw git show log -n 20',
        'aiw git show file --full src/App.java',
        'aiw git show blame src/App.java',
        'aiw git show file-at HEAD~3 README.md',
        'aiw git show lines 10,30 src/App.java',
        'aiw git show lines :main src/main.py',
        'aiw git show commit-files HEAD~1',
        'aiw git show conflicts --check',
    ],
}

QUICK_COMMANDS = [
    ('st', 'status'),
    ('lg [-n N]', 'graph log (user-controlled count)'),
    ('l1 [-n N]', 'one-line log (user-controlled count)'),
    ('fh <path>', 'full file history'),
    ('in / out', 'upstream delta'),
    ('cf [rev]', 'commit files status'),
    ('c / ck', 'conflicts list / check'),
    ('file-oneline|file-patch|file-stat|file-graph <path>', 'clear file modes'),
    ('log-date [-n N]', 'clear date log (user-controlled count)'),
    ('commit-files-names|commit-files-root [rev]', 'clear commit-files modes'),
    ('conflicts-diff|conflicts-staged', 'clear conflict detail modes'),
    ('whatchanged-names [rev]', 'clear whatchanged names-only'),
]

VIEW_GROUPS = [
    ('High-frequency short commands', [
        'st', 'lg', 'l1', 'fh', 'in', 'out', 'cf', 'c', 'ck',
    ]),
    ('Clear word commands', [
        'log-date', 'file-oneline', 'file-patch', 'file-stat',
        'file-graph', 'commit-files-names', 'commit-files-root',
        'conflicts-diff', 'conflicts-staged', 'whatchanged-names',
    ]),
    ('Compatible long commands', [
        'status', 'log', 'file', 'file-hist', 'blame', 'file-at',
        'lines', 'commit-files', 'conflicts', 'unpulled', 'unpushed',
        'whatchanged',
    ]),
]


def print_available_views():
    print('\nAvailable views:')
    for title, commands in VIEW_GROUPS:
        print(f'\n{title}:')
        for command in commands:
            print(f'  {command}')


def print_quick_start():
    print('Daily Quick Start:')
    for command, description in QUICK_COMMANDS:
        print(f'  {command:12} {description}')

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
    'long': 'Displays the commit history. Styles: lg/g (default), l/1 (one-line), hist/d (absolute dates). No default commit count limit unless -n is provided.',
    'usage': 'log [lg|g|l|1|hist|d] [-n <count>]',
    'args': [
        {'flag': 'lg|g', 'description': 'Style with graph, relative dates, and decorations.'},
        {'flag': 'l|1', 'description': 'One-line format for compact view.'},
        {'flag': 'hist|d', 'description': 'Graph with absolute dates for detailed history.'},
        {'flag': '-n <count>', 'description': 'Limit the number of commits shown.'}
    ],
    'examples': ['log lg', 'log 1 -n 50', 'log d -n 30']
}

def cmd_log(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_LOG)
        return 0
    style, n, i, remain = "lg", None, 0, []
    style_aliases = {
        'g': 'lg',
        '1': 'l',
        'd': 'hist',
    }
    while i < len(argv):
        if argv[i] in ("lg", "l", "hist", "g", "1", "d"):
            style = style_aliases.get(argv[i], argv[i])
        elif argv[i] == "-n" and i + 1 < len(argv):
            n = argv[i + 1]
            i += 1
        else: remain.append(argv[i])
        i += 1
    if style == "l":
        cmd = ["git", "log", "--pretty=oneline"]
        if n is not None:
            cmd.extend(["-n", n])
        return core.run_cmd(cmd)
    if style == "hist":
        cmd = ["git", "log", "--color", "--graph", "--pretty=format:%Cred%h%Creset %Cgreen%ad%Creset | %s %C(yellow)%d%Creset %C(bold blue)<%an>%Creset", "--date=short"]
        if n is not None:
            cmd.extend(["-n", n])
        return core.run_cmd(cmd)
    if style == "lg":
        cmd = ["git", "log", "--all", "--color", "--graph", "--pretty=format:%Cred%h%Creset - %C(yellow)%d%Creset %s %Cgreen[%cr] %C(bold blue)<%an>%Creset", "--abbrev-commit", "--date=relative"]
        if n is not None:
            cmd.extend(["-n", n])
        return core.run_cmd(cmd)
    cmd = ['git', 'log']
    if n is not None:
        cmd.extend(['-n', n])
    return core.run_cmd(cmd + remain)

# --- Subcommand: file ---
META_FILE = {
    'name': 'file',
    'short': 'Show the commit history for one file.',
    'long': 'Follows renames by default and supports concise, patch, statistics, graph, and full-evolution output.',
    'usage': 'file [--oneline|-1|--patch|-p|--stat|-s|--graph|-g|--full|-f] [--no-follow] <path>',
    'args': [
        {'flag': '--oneline|-1', 'description': 'Show compact one-line commits.'},
        {'flag': '--patch|-p', 'description': 'Show the diff from every commit.'},
        {'flag': '--stat|-s', 'description': 'Show file change statistics.'},
        {'flag': '--graph|-g', 'description': 'Show a decorated one-line commit graph.'},
        {'flag': '--full|-f', 'description': 'Show the recommended patch and statistics view.'},
        {'flag': '--no-follow', 'description': 'Do not follow history across renames.'},
        {'flag': '<path>', 'description': 'Repository-relative file path.'},
    ],
    'examples': [
        'file README.md',
        'file -1 src/App.java',
        'file -f src/App.java',
    ],
}

META_FILE_HIST = {
    'name': 'file-hist',
    'short': 'Show full commit history for one file (patch + stat).',
    'long': 'High-frequency shortcut for `file --full`. Follows renames by default.',
    'usage': 'file-hist [--no-follow] <path>',
    'args': [
        {'flag': '--no-follow', 'description': 'Do not follow history across renames.'},
        {'flag': '<path>', 'description': 'Repository-relative file path.'},
    ],
    'examples': [
        'file-hist README.md',
        'file-hist --no-follow src/App.java',
    ],
}

FILE_MODES = {
    '--oneline': ['--oneline'],
    '-1': ['--oneline'],
    '--patch': ['-p'],
    '-p': ['-p'],
    '--stat': ['--stat'],
    '-s': ['--stat'],
    '--graph': ['--graph', '--decorate', '--oneline'],
    '-g': ['--graph', '--decorate', '--oneline'],
    '--full': ['-p', '--stat'],
    '-f': ['-p', '--stat'],
}


def usage_error(meta, message):
    print(f"error: {message}", file=sys.stderr)
    core.print_help_meta(meta)
    return 2


def ensure_no_args(meta, argv, command_name):
    if argv:
        return usage_error(meta, f'{command_name} does not accept extra arguments')
    return None


def parse_single_path(meta, argv, command_name):
    paths = argv[1:] if argv[:1] == ['--'] else argv
    if len(paths) != 1:
        usage_error(meta, f'{command_name} requires exactly one path')
        return None
    return paths[0]


def parse_optional_revision(meta, argv, command_name):
    if any(arg.startswith('-') for arg in argv):
        usage_error(meta, f'{command_name} only accepts an optional revision')
        return None
    if len(argv) > 1:
        usage_error(meta, f'{command_name} accepts at most one revision')
        return None
    return argv[0] if argv else 'HEAD'


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


def cmd_file_hist(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_FILE_HIST)
        return 0

    follow, paths_only = True, False
    paths = []
    for arg in argv:
        if paths_only:
            paths.append(arg)
        elif arg == '--':
            paths_only = True
        elif arg == '--no-follow':
            follow = False
        elif arg.startswith('-'):
            return usage_error(META_FILE_HIST, f"unknown option: {arg}")
        else:
            paths.append(arg)

    if len(paths) != 1:
        return usage_error(META_FILE_HIST, 'file-hist requires exactly one path')

    cmd = ['git', 'log']
    if follow:
        cmd.append('--follow')
    cmd.extend(['-p', '--stat', '--', paths[0]])
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


# --- Subcommand: commit-files ---
META_COMMIT_FILES = {
    'name': 'commit-files',
    'short': 'List files changed by one commit.',
    'usage': 'commit-files [--names] [--root] [<revision>]',
    'args': [
        {'flag': '--names', 'description': 'Show paths only, without change status.'},
        {'flag': '--root', 'description': 'Include files introduced by an initial commit.'},
        {'flag': '<revision>', 'description': 'Commit to inspect; defaults to HEAD.'},
    ],
    'examples': ['commit-files', 'commit-files HEAD~1', 'commit-files --names HEAD'],
}

def cmd_commit_files(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_COMMIT_FILES)
        return 0
    names_only = '--names' in argv
    include_root = '--root' in argv
    positional = [arg for arg in argv if not arg.startswith('-')]
    if len(positional) > 1 or any(arg.startswith('-') and arg not in {'--names', '--root'} for arg in argv):
        return usage_error(META_COMMIT_FILES, 'invalid commit-files arguments')
    command = ['git', 'diff-tree', '--no-commit-id', '--name-only' if names_only else '--name-status', '-r']
    if include_root:
        command.append('--root')
    command.append(positional[0] if positional else 'HEAD')
    return core.run_cmd(command)
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


META_ST = {
    'name': 'st',
    'short': 'Short subcommand for concise status.',
    'usage': 'st',
    'args': [],
    'examples': ['st'],
}


def cmd_st(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_ST)
        return 0
    err = ensure_no_args(META_ST, argv, 'st')
    if err is not None:
        return err
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


META_IN = {
    'name': 'in',
    'short': 'Short subcommand for unpulled commits.',
    'usage': 'in',
    'args': [],
    'examples': ['in'],
}


def cmd_in(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_IN)
        return 0
    err = ensure_no_args(META_IN, argv, 'in')
    if err is not None:
        return err
    return cmd_unpulled([])

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


META_OUT = {
    'name': 'out',
    'short': 'Short subcommand for unpushed commits.',
    'usage': 'out',
    'args': [],
    'examples': ['out'],
}


def cmd_out(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_OUT)
        return 0
    err = ensure_no_args(META_OUT, argv, 'out')
    if err is not None:
        return err
    return cmd_unpushed([])

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


META_LG = {
    'name': 'lg',
    'short': 'Short subcommand for graph log style (user-controlled count).',
    'usage': 'lg [-n <count>]',
    'args': [
        {'flag': '-n <count>', 'description': 'Optional commit count limit. Not limited by default.'},
    ],
    'examples': ['lg', 'lg -n 30'],
}


def cmd_lg(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_LG)
        return 0
    return cmd_log(['lg', *argv])


META_L1 = {
    'name': 'l1',
    'short': 'Short subcommand for one-line recent commits (user-controlled count).',
    'usage': 'l1 [-n <count>]',
    'args': [
        {'flag': '-n <count>', 'description': 'Optional commit count limit. Not limited by default.'},
    ],
    'examples': ['l1', 'l1 -n 50'],
}


def cmd_l1(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_L1)
        return 0
    return cmd_log(['1', *argv])


META_LOG_DATE = {
    'name': 'log-date',
    'short': 'Date log style (user-controlled count).',
    'usage': 'log-date [-n <count>]',
    'args': [
        {'flag': '-n <count>', 'description': 'Optional commit count limit. Not limited by default.'},
    ],
    'examples': ['log-date', 'log-date -n 40'],
}


def cmd_log_date(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_LOG_DATE)
        return 0
    return cmd_log(['hist', *argv])


META_FILE_ONELINE = {
    'name': 'file-oneline',
    'short': 'File one-line history (follow rename).',
    'usage': 'file-oneline <path>',
    'args': [{'flag': '<path>', 'description': 'Repository-relative file path.'}],
    'examples': ['file-oneline src/App.java'],
}


def cmd_file_oneline(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_FILE_ONELINE)
        return 0
    path = parse_single_path(META_FILE_ONELINE, argv, 'file-oneline')
    if path is None:
        return 2
    return core.run_cmd(['git', 'log', '--follow', '--oneline', '--', path])


META_FILE_PATCH = {
    'name': 'file-patch',
    'short': 'File patch history (follow rename).',
    'usage': 'file-patch <path>',
    'args': [{'flag': '<path>', 'description': 'Repository-relative file path.'}],
    'examples': ['file-patch src/App.java'],
}


def cmd_file_patch(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_FILE_PATCH)
        return 0
    path = parse_single_path(META_FILE_PATCH, argv, 'file-patch')
    if path is None:
        return 2
    return core.run_cmd(['git', 'log', '--follow', '-p', '--', path])


META_FILE_STAT = {
    'name': 'file-stat',
    'short': 'File stat history (follow rename).',
    'usage': 'file-stat <path>',
    'args': [{'flag': '<path>', 'description': 'Repository-relative file path.'}],
    'examples': ['file-stat src/App.java'],
}


def cmd_file_stat(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_FILE_STAT)
        return 0
    path = parse_single_path(META_FILE_STAT, argv, 'file-stat')
    if path is None:
        return 2
    return core.run_cmd(['git', 'log', '--follow', '--stat', '--', path])


META_FILE_GRAPH = {
    'name': 'file-graph',
    'short': 'File graph history (follow rename).',
    'usage': 'file-graph <path>',
    'args': [{'flag': '<path>', 'description': 'Repository-relative file path.'}],
    'examples': ['file-graph src/App.java'],
}


def cmd_file_graph(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_FILE_GRAPH)
        return 0
    path = parse_single_path(META_FILE_GRAPH, argv, 'file-graph')
    if path is None:
        return 2
    return core.run_cmd(['git', 'log', '--follow', '--graph', '--decorate', '--oneline', '--', path])


META_FH = {
    'name': 'fh',
    'short': 'Short subcommand for full file history (patch + stat).',
    'usage': 'fh <path>',
    'args': [
        {'flag': '<path>', 'description': 'Repository-relative file path.'},
    ],
    'examples': ['fh src/App.java'],
}


def cmd_fh(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_FH)
        return 0
    path = parse_single_path(META_FH, argv, 'fh')
    if path is None:
        return 2
    return core.run_cmd(['git', 'log', '--follow', '-p', '--stat', '--', path])


META_CF = {
    'name': 'cf',
    'short': 'Short subcommand for commit-files with built-in defaults.',
    'usage': 'cf [<revision>]',
    'args': [
        {'flag': '<revision>', 'description': 'Commit to inspect; defaults to HEAD.'},
    ],
    'examples': ['cf', 'cf HEAD~1'],
}


def cmd_cf(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_CF)
        return 0
    revision = parse_optional_revision(META_CF, argv, 'cf')
    if revision is None:
        return 2
    return core.run_cmd(['git', 'diff-tree', '--no-commit-id', '--name-status', '-r', revision])


META_COMMIT_FILES_NAMES = {
    'name': 'commit-files-names',
    'short': 'Commit files names-only (built-in mode).',
    'usage': 'commit-files-names [<revision>]',
    'args': [{'flag': '<revision>', 'description': 'Commit to inspect; defaults to HEAD.'}],
    'examples': ['commit-files-names', 'commit-files-names HEAD~1'],
}


def cmd_commit_files_names(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_COMMIT_FILES_NAMES)
        return 0
    revision = parse_optional_revision(META_COMMIT_FILES_NAMES, argv, 'commit-files-names')
    if revision is None:
        return 2
    return core.run_cmd(['git', 'diff-tree', '--no-commit-id', '--name-only', '-r', revision])


META_COMMIT_FILES_ROOT = {
    'name': 'commit-files-root',
    'short': 'Commit files with root files included (built-in mode).',
    'usage': 'commit-files-root [<revision>]',
    'args': [{'flag': '<revision>', 'description': 'Commit to inspect; defaults to HEAD.'}],
    'examples': ['commit-files-root', 'commit-files-root HEAD'],
}


def cmd_commit_files_root(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_COMMIT_FILES_ROOT)
        return 0
    revision = parse_optional_revision(META_COMMIT_FILES_ROOT, argv, 'commit-files-root')
    if revision is None:
        return 2
    return core.run_cmd(['git', 'diff-tree', '--no-commit-id', '--name-status', '-r', '--root', revision])


META_C = {
    'name': 'c',
    'short': 'Short subcommand for unresolved conflict files list.',
    'usage': 'c',
    'args': [],
    'examples': ['c'],
}


def cmd_c(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_C)
        return 0
    err = ensure_no_args(META_C, argv, 'c')
    if err is not None:
        return err
    return cmd_conflicts([])


META_CK = {
    'name': 'ck',
    'short': 'Short subcommand for conflict marker check.',
    'usage': 'ck',
    'args': [],
    'examples': ['ck'],
}


def cmd_ck(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_CK)
        return 0
    err = ensure_no_args(META_CK, argv, 'ck')
    if err is not None:
        return err
    return core.run_cmd(['git', 'diff', '--check'])


META_CONFLICTS_DIFF = {
    'name': 'conflicts-diff',
    'short': 'Short subcommand for unmerged conflict diff.',
    'usage': 'conflicts-diff',
    'args': [],
    'examples': ['conflicts-diff'],
}


def cmd_conflicts_diff(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_CONFLICTS_DIFF)
        return 0
    err = ensure_no_args(META_CONFLICTS_DIFF, argv, 'conflicts-diff')
    if err is not None:
        return err
    return core.run_cmd(['git', 'diff', '--diff-filter=U'])


META_CONFLICTS_STAGED = {
    'name': 'conflicts-staged',
    'short': 'Short subcommand for staged conflict diff.',
    'usage': 'conflicts-staged',
    'args': [],
    'examples': ['conflicts-staged'],
}


def cmd_conflicts_staged(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_CONFLICTS_STAGED)
        return 0
    err = ensure_no_args(META_CONFLICTS_STAGED, argv, 'conflicts-staged')
    if err is not None:
        return err
    return core.run_cmd(['git', 'diff', '--staged'])


META_WHATCHANGED_NAMES = {
    'name': 'whatchanged-names',
    'short': 'Whatchanged names-only shortcut.',
    'usage': 'whatchanged-names [<revision>]',
    'args': [{'flag': '<revision>', 'description': 'Commit to inspect; defaults to HEAD.'}],
    'examples': ['whatchanged-names', 'whatchanged-names HEAD~1'],
}


def cmd_whatchanged_names(argv):
    if any(f in argv for f in {'-h', '--help'}):
        core.print_help_meta(META_WHATCHANGED_NAMES)
        return 0
    revision = parse_optional_revision(META_WHATCHANGED_NAMES, argv, 'whatchanged-names')
    if revision is None:
        return 2
    return core.run_cmd(['git', 'diff-tree', '--no-commit-id', '--name-only', '-r', revision])

SUBCOMMANDS = {
    'blame': cmd_blame,
    'c': cmd_c,
    'ck': cmd_ck,
    'cf': cmd_cf,
    'commit-files-names': cmd_commit_files_names,
    'commit-files-root': cmd_commit_files_root,
    'commit-files': cmd_commit_files,
    'conflicts': cmd_conflicts,
    'conflicts-diff': cmd_conflicts_diff,
    'conflicts-staged': cmd_conflicts_staged,
    'file': cmd_file,
    'file-oneline': cmd_file_oneline,
    'file-patch': cmd_file_patch,
    'file-stat': cmd_file_stat,
    'file-graph': cmd_file_graph,
    'fh': cmd_fh,
    'file-hist': cmd_file_hist,
    'file-at': cmd_file_at,
    'in': cmd_in,
    'l1': cmd_l1,
    'lg': cmd_lg,
    'log-date': cmd_log_date,
    'lines': cmd_lines,
    'log': cmd_log,
    'out': cmd_out,
    'st': cmd_st,
    'status': cmd_status,
    'unpulled': cmd_unpulled,
    'unpushed': cmd_unpushed,
    'whatchanged-names': cmd_whatchanged_names,
    'whatchanged': cmd_whatchanged,
}

SUBCOMMAND_ALIASES = {
    's': 'st',
    'l': 'lg',
    'f': 'fh',
}

def main(argv):
    if not argv or argv[0] in {'-h', '--help'}:
        core.print_help_meta(META)
        print('')
        print_quick_start()
        print_available_views()
        return 0
    sub = SUBCOMMAND_ALIASES.get(argv[0], argv[0])
    if sub in SUBCOMMANDS:
        return SUBCOMMANDS[sub](argv[1:])
    print(f"Unknown subcommand: {sub}", file=sys.stderr)
    return 1

if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))


