#!/usr/bin/env python3
"""aiw-git-ca wrapper

Amend last commit including all changes.
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
    'name': 'aiw git-export',
    'short': 'Export the current working directory to a tar archive.',
    'long': 'Creates a tar archive of the current working directory, including all tracked files.',
    'usage': 'aiw git-export <ref> [file.zip]',
    'args': [
        {'flag': '<ref>', 'description': 'The reference to export.'},
        {'flag': '[file.zip]', 'description': 'The output zip file (default: <ref>.zip).'}
    ],
    'examples': ['aiw git-export myproject.zip']
}


def main(argv):
    help_flags = {'-h', '--help', '-help', '-?'}
    if any(f in argv for f in help_flags):
        core.print_help_meta(META)
        return 0
    if len(argv) == 0:
        ref = "HEAD"
    else:
        ref = argv[0]
    output = ref + ".zip"
    if len(argv) >= 2:
        output = argv[1]
    
    return core.run_cmd(['git', 'archive', '--format=zip', '-o', output, ref])

if __name__ == '__main__':
    rc = main(sys.argv[1:])
    sys.exit(rc)
