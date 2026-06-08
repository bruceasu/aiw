# aiw git add-mirror

Short: Add a new mirror for the current repository.

Description:
Adds a new mirror to the current repository.

Usage:
aiw git add-mirror <url> [mirror] [--push [branch]]

Arguments:
- <url> 鈥?The URL of the repository to add as a mirror.
- [mirror] 鈥?The name of the mirror (default: mirror).
- --push [branch] 鈥?Push to the mirror after adding it.

Examples:
- aiw git add-mirror https://github.com/user/repo.git origin

For full help run: generate_new_plugin_docs.py -h
