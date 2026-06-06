# aiw git-show

Short: Consolidates conflicts, log, status, unpulled, unpushed, and whatchanged.

Description:
Consolidates conflicts, log, status, unpulled, unpushed, and whatchanged, etc.



Usage:
aiw git-track <sub-cmd> [options]


Sub Commands:
- conflicts': cmd_conflicts,
- log': cmd_log,
- status': cmd_status,
- unpulled': cmd_unpulled,
- unpushed': cmd_unpushed,
- whatchanged': cmd_whatchanged,


Arguments:
- <conflicts> Inspect and help resolve git conflicts. Shows conflicted files, diffs, staging status, and provides guidance for resolving merge/rebase conflicts.
   - usage: conflicts [--diff] [--check] [--staged]
   - args:
     - --diff  Show all unmerged hunks in diff format.
     - --check Detect remaining conflict markers in files.
     - --stage Show staged resolved files.
   - examples:
     - conflicts
     - confiicts --diff
- <log> — Show a formatted commit log with several styles. Displays the commit history. Styles: lg (default), l (one-line), hist (absolute dates)
   - usage: log [lg|l|hists] [-n <count>]
   - args:
     - [lg]   Style with graph, relative dates, and decorations.
     - [l]    One-line format for compact view.
     - [hist] Graph with absolute dates for detailed history.
     - [-n <count>] Limit the number of commits shown.
   - examples:
     - log lg
     - log hist -n 50
- <status> Show concise working-tree status.
- <unpulled> Show commits present on upstream but not pulled locally.
- <unpushed> Show commits not pushed to upstream.
- <whatchanged> Show changes between commits (git whatchanged).

