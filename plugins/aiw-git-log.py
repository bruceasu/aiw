#!/usr/bin/env python3
"""aiw-git-log wrapper

Formatted commit log with styles.
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
	'name': 'aiw git log',
	'short': 'Show a formatted commit log with several styles.',
	'long': 'Displays the commit history. Supports styles: lg (default), l (one-line), hist (absolute dates).',
	'usage': 'aiw git log [lg|l|hist] [-n <count>]',
	'args': [],
	'examples': ['aiw git log lg', 'aiw git log -n 50']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	# forward all args to git log
	return core.run_cmd(['git', 'log'] + argv)


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
