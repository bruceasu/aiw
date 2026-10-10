#!/usr/bin/env python3
"""Restore one or more paths from another branch or commit."""
import sys
import os
import importlib.util

HERE = os.path.dirname(__file__)
CORE_PATH = os.path.join(HERE, 'aiw-git-core.py')
spec = importlib.util.spec_from_file_location('aiw_git_core', CORE_PATH)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

META = {
	'name': 'aiw git restore-path',
	'short': 'Restore one or more paths from another branch or commit.',
	'long': (
		'Writes the selected path versions from a branch or commit into the working '
		'tree without changing the current branch or index. Existing working-tree '
		'changes to those paths may be replaced.'
	),
	'usage': 'aiw git restore-path <commit|branch> <path...>',
	'args': [
		{'flag': '<commit|branch>', 'description': 'The commit or branch from which to restore the paths.'},
		{'flag': '<path...>', 'description': 'One or more file or directory paths to restore.'}
	],
	'examples': [
		'aiw git restore-path origin/main path/to/file',
		'aiw git restore-path HEAD~1 src/main.py README.md',
	]
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if len(argv) < 2:
		print('usage: aiw git restore-path <commit|branch> <path...>', file=sys.stderr)
		return 2
	source, *paths = argv
	return core.run_cmd(['git', 'restore', f'--source={source}', '--worktree', '--'] + paths)


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)


