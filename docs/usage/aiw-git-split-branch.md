# aiw git-mv-current-commit-to-new-branch

Short: Move the current commit to a new branch.

Description:
Creates a new branch at the current HEAD, resets the current branch backward, then switches to the new branch.

Usage:
aiw git-split-branch<new-branch> [reset-to] [--force]

Arguments:
- <new-branch> — The name of the new branch to create.
- [reset-to] — The reference to reset the current branch to (default: HEAD^).
- --force — Skip confirmation prompt.

Examples:
- aiw git-split-branchfeature/login
- aiw git-split-branchhotfix HEAD~3

For full help run: generate_new_plugin_docs.py -h