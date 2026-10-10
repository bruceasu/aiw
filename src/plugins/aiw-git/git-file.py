#!/usr/bin/env python3
"""Inspect a file's history, contents, or line attribution."""

import importlib.util
import os
import sys

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')
spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git file',
    'short': 'Inspect one file across Git history.',
    'long': (
        'By default, shows the file commit history and follows renames. '
        'Choose an output mode, or use --blame, --at, or --lines for a '
        'different file view.'
    ),
    'usage': (
        'aiw git file <path> [--oneline|--patch|--stat|--graph|--full] '
        '[--no-follow]\n'
        'aiw git file <path> --blame\n'
        'aiw git file <path> --at <revision>\n'
        'aiw git file <path> --lines <range|:function>'
    ),
    'args': [
        {'flag': '<path>', 'description': 'Repository-relative path to one file.'},
        {'flag': '--oneline', 'description': 'Show compact one-line commits.'},
        {'flag': '--patch', 'description': 'Show the diff from every commit.'},
        {'flag': '--stat', 'description': 'Show file change statistics.'},
        {'flag': '--graph', 'description': 'Show a decorated commit graph.'},
        {'flag': '--full', 'description': 'Show patches and file statistics.'},
        {'flag': '--no-follow', 'description': 'Do not follow history across renames.'},
        {'flag': '--blame', 'description': 'Show line-by-line attribution.'},
        {'flag': '--at <revision>', 'description': 'Show the file as it existed at a revision.'},
        {'flag': '--lines <range|:function>', 'description': 'Trace a line range or function history.'},
    ],
    'examples': [
        'aiw git file README.md',
        'aiw git file src/App.java --full',
        'aiw git file src/App.java --blame',
        'aiw git file README.md --at HEAD~3',
        'aiw git file src/main.py --lines :main',
    ],
}

HISTORY_MODES = {
    '--oneline': ['--oneline'],
    '--patch': ['-p'],
    '--stat': ['--stat'],
    '--graph': ['--graph', '--decorate', '--oneline'],
    '--full': ['-p', '--stat'],
}
SPECIAL_MODES = {'--blame', '--at', '--lines'}


def usage_error(message):
    print(f'error: {message}', file=sys.stderr)
    core.print_help_meta(META)
    return 2


def main(argv):
    if any(flag in argv for flag in {'-h', '--help', '-help', '-?'}):
        core.print_help_meta(META)
        return 0

    paths = []
    options = []
    paths_only = False
    i = 0
    while i < len(argv):
        arg = argv[i]
        if paths_only:
            paths.append(arg)
        elif arg == '--':
            paths_only = True
        elif arg in HISTORY_MODES or arg in SPECIAL_MODES or arg == '--no-follow':
            options.append(arg)
            if arg in {'--at', '--lines'}:
                i += 1
                if i >= len(argv):
                    return usage_error(f'{arg} requires a value')
                options.append(argv[i])
        elif arg.startswith('-'):
            return usage_error(f'unknown option: {arg}')
        else:
            paths.append(arg)
        i += 1

    if len(paths) != 1:
        return usage_error('exactly one file path is required')

    path = paths[0]
    selected_modes = [option for option in options if option in HISTORY_MODES or option in SPECIAL_MODES]
    if len(selected_modes) > 1:
        return usage_error('choose only one file view')
    mode = selected_modes[0] if selected_modes else None
    no_follow = '--no-follow' in options

    if mode == '--blame':
        if no_follow:
            return usage_error('--no-follow only applies to file history')
        return core.run_cmd(['git', 'blame', '--', path])
    if mode == '--at':
        if no_follow:
            return usage_error('--no-follow only applies to file history')
        revision = options[options.index('--at') + 1]
        return core.run_cmd(['git', 'show', f'{revision}:{path}'])
    if mode == '--lines':
        if no_follow:
            return usage_error('--no-follow only applies to file history')
        selector = options[options.index('--lines') + 1]
        return core.run_cmd(['git', 'log', '-L', f'{selector}:{path}'])

    command = ['git', 'log']
    if not no_follow:
        command.append('--follow')
    if mode:
        command.extend(HISTORY_MODES[mode])
    command.extend(['--', path])
    return core.run_cmd(command)


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
