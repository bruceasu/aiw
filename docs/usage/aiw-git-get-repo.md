# aiw git get-repo

Short: Shallow or full git clone.

Description:
Clones a repository with optional branch selection, target directory, and shallow depth control.

Usage:
aiw git get-repo <url> [-b <branch>] [-d <dir>] [--depth <n>] [--full]

Arguments:
- <url> 鈥?The URL of the repository to clone.
- -b <branch> 鈥?The branch to clone.
- -d <dir> 鈥?The directory into which to clone the repository.
- --depth <n> 鈥?The depth of the shallow clone.
- --full 鈥?Perform a full clone.

Examples:
- aiw git get-repo https://github.com/user/repo.git
- aiw git get-repo repo.git -b main -d myrepo
- aiw git get-repo repo.git --depth 5
- aiw git get-repo repo.git --full

For full help run: generate_new_plugin_docs.py -h
