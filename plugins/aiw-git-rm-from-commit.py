#!/usr/bin/env python3
"""aiw-git-rm-from-commit wrapper

Remove a file from a historical commit (rewrite history assistance).
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
	'name': 'aiw git rm-from-commit',
	'short': 'Remove a file from a historical commit (rewrite history assistance).',
	'long': 'Helper to remove a file introduced in a particular commit. Typically used to strip accidental files from history.',
	'usage': 'aiw git rm-from-commit <commit> <path>',
	'args': [],
	'examples': ['aiw git rm-from-commit abc123 path/to/secret']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	# Complex history rewrite is out of scope; show guidance
	print('Use git filter-branch or git filter-repo to remove files from history.', file=sys.stderr)
	return 0


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
