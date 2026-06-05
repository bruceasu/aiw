#!/usr/bin/env python3
"""aiw-git-un-add wrapper

Unstage files (git reset HEAD -- <path>).
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
	'name': 'aiw git un-add',
	'short': 'Unstage files from the index.',
	'long': 'Removes files from the index (unstage) while keeping them in the working tree.',
	'usage': 'aiw git un-add <path>...',
	'args': [],
	'examples': ['aiw git un-add path/to/file']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if not argv:
		print('usage: aiw git un-add <path>...', file=sys.stderr)
		return 2
	return core.run_cmd(['git', 'reset', 'HEAD', '--'] + argv)


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
