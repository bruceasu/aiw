#!/usr/bin/env python3
"""aiw git guide

This is a helper plugin that provides guidance.
Find a commit a number of steps back.
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
	'name': 'aiw git guide',
	'short': 'Find a commit a number of steps back (helper).',
	'long': 'Searches the commit history backward to locate a commit matching criteria or a specific offset.',
	'usage': 'aiw git guide <how-to>',
	'args': [
		{'flag': '<how-to>', 'description': 'The guidance topic to display.'}
	],
	'examples': ['aiw git guide', 'rollback']
}


def commit_recovery():
	core.run_cmd(['git', 'rev-list', '--all'])
	core.run_cmd(['git', 'reflog'])
	print("""
How to recover after an accidental hard reset
---------------------------------------------

1. Find the SHA of the commit you want to recover (see reflog above).

2. Create a recovery branch at that SHA and switch to it:

     git checkout -b recover-branch <SHA>

   Or, if you just want to reset the current branch back:

     git reset --hard <SHA>

Note: Git keeps reflog entries for ~90 days by default.
      Run "aiw git gc" only AFTER you have recovered what you need.
""")
	return 0



def rollback():
	print("""
How to roll back / undo a recent git operation
----------------------------------------------

Step 1 鈥?Find the SHA you want to return to in the reflog above.

Step 2 鈥?Options:

  a) Undo last commit, keep changes staged:
       git reset --soft HEAD~1

  b) Undo last commit, keep changes in working tree:
       git reset HEAD~1           (or: aiw git undo)

  c) Undo last commit AND discard all changes:
       git reset --hard <SHA>     (or: aiw git undo --hard)

  d) Restore a branch pointer to a specific SHA:
       git branch -f <branch> <SHA>

  e) Create a safe recovery branch at any reflog SHA:
       git checkout -b recover-<SHA> <SHA>

  f) Undo a pushed commit safely (creates a new commit):
       aiw git revert <SHA>

Note: reflog entries expire after ~90 days.
      Run "aiw git gc" ONLY after you have recovered what you need.
""")

	return 0


def split():
    print("""How to split commits that landed on the wrong branch
=====================================================

Situation: multiple commits are on the current branch but belong to different
feature branches. Use reset + cherry-pick to redistribute them.

Step 1 鈥?See what you have

  aiw git log           # identify each commit hash
  git log --oneline     # compact view

Step 2 鈥?Note the last "good" commit on this branch

  This is the commit BEFORE the ones that need to move.
  Example: abc0000

Step 3 鈥?Create feature branches at the current HEAD
  (they will point to all the commits for now)

  git checkout -b feature-A
  git checkout -b feature-B
  git checkout <original-branch>   # go back

Step 4 鈥?Reset the original branch to the last good commit

  git reset --hard abc0000

Step 5 鈥?Cherry-pick the right commits onto each branch

  git checkout feature-A
  git cherry-pick <sha-for-A>       # copies commit, new hash generated

  git checkout feature-B
  git cherry-pick <sha-for-B>

Key facts about cherry-pick
  鈥?Copies only the named commit, not the whole branch.
  鈥?Generates a new commit hash on the target branch.
  鈥?Original commits remain where they are until pruned.

Useful variants
  git cherry-pick A..B              # pick a range (exclusive A, inclusive B)
  git cherry-pick A^..B             # pick a range (inclusive A and B)
  git cherry-pick --no-commit <sha> # apply changes without committing

Common commands during this workflow
  aiw git log                       # pretty graph log
  aiw git mv-to-branch <branch>     # shortcut when only last commit needs moving
  aiw git get-file-from <branch> <file>  # grab a single file from another branch
""")


def main(argv):
	help_flags = {'-h', '--help', '-help', '-?'}
	if any(f in argv for f in help_flags):
		core.print_help_meta(META)
		return 0
	if not argv:
		print('usage: aiw git guide <how-to>', file=sys.stderr)
		return 2
	topic = argv[0]
	if topic == 'list':
		print('Available topics: commit-recovery, rollback, split')
		return 0
	if topic == 'commit-recovery':
		return commit_recovery()
	elif topic == 'rollback':
		return rollback()
	elif topic == 'split':
		return split()
	else:
		print(f'Unknown topic: {topic}', file=sys.stderr)
		return 2
	
if __name__ == '__main__':
	rc = main(sys.argv[1:])
	sys.exit(rc)


