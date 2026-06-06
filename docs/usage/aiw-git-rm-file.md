# aiw-git-rm-file

Short: Remove a file from commit or full history.

Description:
Removes a file from the last commit, or rewrites history to remove it from all commits using filter-branch.

Usage:
aiw-git-rm-file <file> [--history] [--from <commit>] [--force]

Arguments:
- <file> — The file to remove.
- --history — Remove the file from all commits in history.
- --from <commit> — When used with --history, only remove the file from commits after the specified commit.
- --force — Skip confirmation prompt when using --history.

Examples:
- aiw-git-rm-file secrets.txt
- aiw-git-rm-file secrets.txt --history
- aiw-git-rm-file secrets.txt --history --from abc123

For full help run: generate_new_plugin_docs.py -h