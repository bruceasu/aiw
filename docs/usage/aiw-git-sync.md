# aiw git sync

Short: Fetch, rebase the current branch onto the remote, then push.

Description:
Fetches remote, rebases the current branch onto the remote branch, and pushes. Defaults: remote=origin, branch=current.

Usage:
aiw git sync [branch] [remote]

Examples:
- aiw git sync
- aiw git sync main origin

For full help run: generate_git_docs.py -h