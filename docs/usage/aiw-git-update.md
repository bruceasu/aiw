# aiw git update

Short: Fetch updates, optionally rebase and push.

Description:
Fetches remote updates and either pulls or rebases the current branch. Optionally pushes after rebase or pull. Defaults: remote=origin, branch=current.

Usage:
aiw git update [branch] [remote] [--rebase] [--push] [--merge]

Examples:
- aiw git update
- aiw git update main origin --rebase
- aiw git update develop origin --push

For full help run: generate_new_plugin_docs.py -h
