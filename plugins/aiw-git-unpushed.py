#!/usr/bin/env python3
"""aiw-git-unpushed wrapper

Show commits not pushed to upstream.
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
	'name': 'aiw git unpushed',
	'short': 'Show commits not pushed to upstream.',
	'long': 'Shows the commits present locally but not on the upstream branch.',
	'usage': 'aiw git unpushed',
	'args': [],
	'examples': ['aiw git unpushed']
}


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	# find upstream
	up = core.git_output(['git', 'rev-parse', '--abbrev-ref', '@{u}'])
	if not up:
		print('no upstream configured for current branch', file=sys.stderr)
		return 2
	return core.run_cmd(['git', 'log', '--oneline', f'{up}..HEAD'])


if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)
