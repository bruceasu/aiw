#!/usr/bin/env python3
"""aiw-git-track wrapper

Show or set upstream tracking for the current branch.
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
	'name': 'aiw git track',
	'short': 'Show or set upstream tracking for the current branch.',
	'long': 'With no args, shows the current branch upstream. With one arg, sets the upstream branch (git branch --set-upstream-to).',
	'usage': 'aiw git track [remote/branch]',
	'args': [],
	'examples': ['aiw git track', 'aiw git track origin/main']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if not argv:
		up = core.git_output(['git', 'rev-parse', '--abbrev-ref', '@{u}'])
		if not up:
			print('no upstream configured for current branch', file=sys.stderr)
			return 2
		print(up)
		return 0
	# set upstream
	return core.run_cmd(['git', 'branch', '--set-upstream-to', argv[0]])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
