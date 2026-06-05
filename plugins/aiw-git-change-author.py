#!/usr/bin/env python3
"""aiw-git-change-author wrapper

Change the author on the last commit or advise history rewrite.
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
	'name': 'aiw git change-author',
	'short': 'Change the author on the last commit (or rewrite history).',
	'long': 'Convenience helper to amend the last commit author with --author "Name <email>". For wide rewrites use proper history-rewrite tools.',
	'usage': 'aiw git change-author "Name <email>"',
	'args': [],
	'examples': ['aiw git change-author "Alice <alice@example.com>"']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if not argv:
		print('usage: aiw git change-author "Name <email>"', file=sys.stderr)
		return 2
	author = argv[0]
	return core.run_cmd(['git', 'commit', '--amend', '--author', author, '--no-edit'])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
