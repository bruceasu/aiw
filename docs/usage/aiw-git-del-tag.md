# aiw git delete-tag

Short: Delete a local tag and optionally remove it from a remote.

Description:
Deletes local tag and can push deletion to remote. Falls back to explicit refspec if remote delete fails.

Usage:
aiw git delete-tag <tag> [--remote] [--remote-name <name>] [--force]

Arguments:
- <tag> 鈥?The name of the tag to delete.
- --remote 鈥?Also delete the tag from the remote (default: origin).
- --remote-name <name> 鈥?Remote name to use (default: origin).
- --force 鈥?Skip confirmation prompt.

Examples:
- aiw git delete-tag v1.2.3 --remote

For full help run: generate_new_plugin_docs.py -h
