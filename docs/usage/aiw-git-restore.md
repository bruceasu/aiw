# aiw git restore

Short: Restore files from the working tree or a commit.

Description:
Restores one or more files either from the current working tree or from a specific commit using git restore.

Usage:
aiw git restore <file...>
aiw git restore <file...> --from <commit>

Arguments:
- <file...> 鈥?The file(s) to restore.
- --from 鈥?The commit from which to restore the file(s).

Examples:
- aiw git restore main.py
- aiw git restore a.txt b.txt
- aiw git restore main.py --from HEAD~1
- aiw git restore app.py --from abc1234^

For full help run: generate_new_plugin_docs.py -h
