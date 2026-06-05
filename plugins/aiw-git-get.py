#!/usr/bin/env python3
"""aiw-git-get wrapper

Shallow single-branch clone helper.
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
	'name': 'aiw git get',
	'short': 'Shallow single-branch clone helper.',
	'long': 'Performs a shallow single-branch clone with optional branch, directory, and depth flags.',
	'usage': 'aiw git get <url> [-b <branch>] [-d <dir>] [--depth <n>] [--full]',
	'args': [],
	'examples': ['aiw git get https://example.com/repo.git -b main -d repo']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if not argv:
		print('usage: aiw git get <url> [-b <branch>] [-d <dir>] [--depth <n>] [--full]', file=sys.stderr)
		return 2
	url = argv[0]
	rest = argv[1:]
	# simple clone: forward args
	return core.run_cmd(['git', 'clone'] + rest + [url])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
