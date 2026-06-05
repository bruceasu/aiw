#!/usr/bin/env python3
"""aiw-git-undo wrapper

Implements undo behavior and help metadata.
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
	'name': 'aiw git undo',
	'short': 'Undo last commit while keeping changes (or discard with --hard).',
	'long': 'Resets HEAD to the previous commit. By default changes are kept in the working tree. Use --hard to discard changes (dangerous).',
	'usage': 'aiw git undo [--hard] [--force]',
	'args': [
		{'flag': '--hard', 'desc': 'Also discard working-tree changes.'},
		{'flag': '--force', 'desc': 'Skip confirmation prompts.'},
	],
	'examples': ['aiw git undo', 'aiw git undo --hard --force']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if '--hard' in argv:
		if not core.git_confirm('--hard will permanently discard all working-tree changes.', argv):
			print('aborted', file=sys.stderr)
			return 0
		return core.run_cmd(['git', 'reset', '--hard', 'HEAD~1'])
	return core.run_cmd(['git', 'reset', 'HEAD~1'])


if __name__ == '__main__':
	try:
		rc = main(sys.argv[1:])
		sys.exit(rc)
	except SystemExit:
		raise
