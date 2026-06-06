# aiw git-track

Short: Track or set upstream for a branch.

Description:
Fetches all remotes and checks out a remote branch as a local tracking branch if --track is used, otherwise sets upstream for an existing local branch or pushes with -u.

Usage:
aiw git-track <branch> [remote] [--push] [--track]

Arguments:
- <branch> — The branch to track or set upstream for.
- [remote] — The remote to track the branch from (default: origin).
- --push — Push the branch to the remote and set upstream.
- --track — Create a new local tracking branch from the remote branch.

Examples:
- aiw git-track feature/login
- aiw git-track main origin --push
- aiw git-track develop origin --track

For full help run: generate_new_plugin_docs.py -h