#!/usr/bin/env python3
"""Show commits available from the current branch's upstream."""

import importlib.util
import os
import subprocess
import sys

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')
spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
    'name': 'aiw git in',
    'short': 'Show commits on the upstream that are not local yet.',
    'long': 'Lists commits in the current branch upstream that are not in HEAD.',
    'usage': 'aiw git in',
    'args': [],
    'examples': ['aiw git in'],
}


def main(argv):
    if any(flag in argv for flag in {'-h', '--help', '-help', '-?'}):
        core.print_help_meta(META)
        return 0
    if argv:
        print('error: aiw git in does not accept arguments', file=sys.stderr)
        core.print_help_meta(META)
        return 2
    try:
        upstream = core.git_output(['git', 'rev-parse', '--abbrev-ref', '@{u}']).strip()
    except subprocess.CalledProcessError:
        upstream = ''
    if not upstream:
        print('no upstream configured', file=sys.stderr)
        return 2
    return core.run_cmd(['git', 'log', '--oneline', f'HEAD..{upstream}'])


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
