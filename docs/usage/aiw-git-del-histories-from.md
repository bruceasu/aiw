# aiw git del-histories-from

Short: Delete history before a ref.

Description:
Rewrites history so that only commits after a given ref remain, creating a new root commit and rebasing the branch onto it.

Usage:
aiw git del-histories-from <ref> [--branch <name>] [--message <msg>] [--force]

Arguments:
- <ref> 鈥?The reference before which to delete history.
- --branch <name> 鈥?The branch to modify (default: current branch).
- --message <msg> 鈥?The commit message for the new root commit.
- --force 鈥?Skip confirmation prompt.

Examples:
- aiw git del-histories-from HEAD~10
- aiw git del-histories-from abc123 --branch main
- aiw git del-histories-from v1.0 --message "new begin"

For full help run: generate_new_plugin_docs.py -h
