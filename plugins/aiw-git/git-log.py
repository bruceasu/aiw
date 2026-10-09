#!/usr/bin/env python3
"""Convenience views for commit history and changed files."""

import importlib.util
import os
import sys

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')
spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git log',
    'short': 'View commit history or the files changed by a commit.',
    'long': (
        'The default view is a graph across all refs. Choose --oneline or '
        '--date for another style. Other arguments are passed to git log. '
        'Use --files to list changed paths across history or for one commit.'
    ),
    'usage': (
        'aiw git log [--graph|--oneline|--date] [-n <count>] '
        '[git-log arguments...]\n'
        'aiw git log --files [--names] [--root] [<revision>]'
    ),
    'args': [
        {'flag': '--graph', 'description': 'Graph with decorations and relative dates (default).'},
        {'flag': '--oneline', 'description': 'Show one line per commit.'},
        {'flag': '--date', 'description': 'Graph with absolute dates.'},
        {'flag': '-n <count>', 'description': 'Limit the number of commits.'},
        {'flag': '--files', 'description': 'List changed files across history, or for one commit if a revision is given.'},
        {'flag': '--names', 'description': 'With --files, show paths without change status.'},
        {'flag': '--root', 'description': 'With --files, include files from a root commit.'},
    ],
    'examples': [
        'aiw git log -n 20',
        'aiw git log --oneline -n 50',
        'aiw git log --date -n 30',
        'aiw git log --files HEAD~1',
        'aiw git log --files --names HEAD',
    ],
}

GRAPH_FORMAT = (
    '%Cred%h%Creset - %C(yellow)%d%Creset %s '
    '%Cgreen[%cr] %C(bold blue)<%an>%Creset'
)
DATE_FORMAT = (
    '%Cred%h%Creset %Cgreen%ad%Creset | %s '
    '%C(yellow)%d%Creset %C(bold blue)<%an>%Creset'
)


def usage_error(message):
    print(f'error: {message}', file=sys.stderr)
    core.print_help_meta(META)
    return 2


def show_files(argv):
    names_only = '--names' in argv
    include_root = '--root' in argv
    revisions = [arg for arg in argv if arg not in {'--names', '--root'}]
    if any(arg.startswith('-') for arg in revisions):
        return usage_error('unrecognized option for --files')
    if len(revisions) > 1:
        return usage_error('--files accepts at most one revision')

    if not revisions:
        command = ['git', 'log', '--name-only' if names_only else '--name-status']
        if include_root:
            command.append('--root')
        return core.run_cmd(command)

    command = [
        'git', 'diff-tree', '--no-commit-id',
        '--name-only' if names_only else '--name-status', '-r',
    ]
    if include_root:
        command.append('--root')
    command.append(revisions[0])
    return core.run_cmd(command)


def main(argv):
    if any(flag in argv for flag in {'-h', '--help', '-help', '-?'}):
        core.print_help_meta(META)
        return 0

    if '--files' in argv:
        file_args = [arg for arg in argv if arg != '--files']
        return show_files(file_args)
    if '--names' in argv or '--root' in argv:
        return usage_error('--names and --root require --files')

    styles = [style for style in ('--graph', '--oneline', '--date') if style in argv]
    if len(styles) > 1:
        return usage_error('choose only one log style')
    style = styles[0] if styles else '--graph'
    args = [arg for arg in argv if arg != style] if styles else list(argv)

    if style == '--oneline':
        command = ['git', 'log', '--pretty=oneline', *args]
    elif style == '--date':
        command = [
            'git', 'log', '--color', '--graph',
            f'--pretty=format:{DATE_FORMAT}', '--date=short', *args,
        ]
    else:
        command = [
            'git', 'log', '--all', '--color', '--graph',
            f'--pretty=format:{GRAPH_FORMAT}', '--abbrev-commit',
            '--date=relative', *args,
        ]
    return core.run_cmd(command)


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
