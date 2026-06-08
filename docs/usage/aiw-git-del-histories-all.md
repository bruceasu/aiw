# aiw git del-histories-all

Short: Delete all git history from the repository.

Description:
Permanently erases all commit history from the repository, force-pushes to remote, and re-links the local branch.

Usage:
aiw git del-histories-all [--branch <name>] [--remote <name>] [--message <msg>] [--force]

Arguments:
- [--branch <name>] 鈥?The branch to delete history from (default: current branch).
- [--remote <name>] 鈥?The remote to force-push to (default: origin).
- [--message <msg>] 鈥?The commit message for the new initial commit.
- [--force] 鈥?Force the operation without confirmation.

Examples:
- aiw git del-histories-all
- aiw git del-histories-all --branch main
- aiw git del-histories-all --remote origin --message "init"

For full help run: generate_new_plugin_docs.py -h
