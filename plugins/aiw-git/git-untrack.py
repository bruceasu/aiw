#!/usr/bin/env python3
"""aiw git untrack wrapper

Remove file from HEAD while keeping working tree copy.
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
	'name': 'aiw git untrack',
	'short': 'Remove file from HEAD while keeping working tree copy.',
	'long': 'Removes a path from the index but leaves the file in the working tree (safe remove from history/commit).',
	'usage': 'aiw git untrack <path>',
	'args': [
		{'flag': '<path>', 'description': 'The path to un-track.'}
	],
	'examples': ['aiw git untrack path/to/file']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if not argv:
		print('usage: aiw git untrack <path>', file=sys.stderr)
		return 2
	return core.run_cmd(['git', 'rm', '--cached'] + argv)


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)


