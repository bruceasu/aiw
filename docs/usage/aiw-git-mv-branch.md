# aiw-git-mv-branch

Short: Rename a git branch.

Description:
Renames a local branch and optionally updates the remote branch name.

Usage:
aiw-git-mv-branch <new-name> [--remote] [--remote-name <name>]
aiw-git-mv-branch <old-name> <new-name> [--remote] [--remote-name <name>]

Examples:
- aiw-git-mv-branch feature-v2
- aiw-git-mv-branch old-name new-name
- aiw-git-mv-branch main stable --remote
- aiw-git-mv-branch dev release --remote --remote-name upstream

For full help run: generate_new_plugin_docs.py -h