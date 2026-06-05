#!/usr/bin/env python3
"""aiw-git-whatchanged wrapper

Show what changed between commits (alias for git whatchanged forwarding).
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
	'name': 'aiw git whatchanged',
	'short': 'Show changes between commits (git whatchanged).',
	'long': 'Forwards arguments to `git whatchanged` to show commit-level diffs.',
	'usage': 'aiw git whatchanged [<rev-range>] [-p]',
	'args': [],
	'examples': ['aiw git whatchanged HEAD~5..HEAD -p']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	return core.run_cmd(['git', 'whatchanged'] + argv)


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
