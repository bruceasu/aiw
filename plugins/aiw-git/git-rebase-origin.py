#!/usr/bin/env python3
"""aiw git rebase-origin wrapper

Interactive rebase against upstream.
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
	'name': 'aiw git rebase-origin',
	'short': 'Interactive rebase against upstream.',
	'long': 'Runs an interactive rebase against the upstream branch (@{u}).',
	'usage': 'aiw git rebase-origin',
	'args': [],
	'examples': ['aiw git rebase-origin']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	return core.run_cmd(['git', 'rebase', '-i', '@{u}'])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)


