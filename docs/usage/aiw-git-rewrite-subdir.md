# aiw git rewrite-subdir

Short: Promote a subdirectory to repo root.

Description:
Rewrites full git history so that a subdirectory becomes the repository root using filter-branch. Commits outside the subdir are dropped.

Usage:
aiw git rewrite-subdir <subdir> [--force]

Arguments:
- subdir 鈥?Path to the subdirectory to promote to root. This directory must exist in the current HEAD.
- --force 鈥?By default, the command will ask for confirmation before proceeding. Use --force to skip confirmation.

Examples:
- aiw git rewrite-subdir src
- aiw git rewrite-subdir project --force

For full help run: generate_new_plugin_docs.py -h
