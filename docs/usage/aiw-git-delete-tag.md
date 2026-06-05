# aiw git delete-tag

Short: Delete a local tag and optionally remove it from a remote.

Description:
Deletes local tag and can push deletion to remote. Falls back to explicit refspec if remote delete fails.

Usage:
aiw git delete-tag <tag> [--remote] [--remote-name <name>] [--force]

Examples:
- aiw git delete-tag v1.2.3 --remote

For full help run: generate_git_docs.py -h