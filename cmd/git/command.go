package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

var dryRun bool

// Dispatch routes `aiw git <subcommand> [args]`.
func Dispatch(args []string) error {
	// Strip --dry-run before any dispatch.
	filtered := args[:0]
	for _, a := range args {
		if a == "--dry-run" {
			dryRun = true
		} else {
			filtered = append(filtered, a)
		}
	}
	args = filtered

	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		helpArgs := []string{}
		if len(args) > 1 {
			helpArgs = args[1:]
		}
		gitUsage(helpArgs)
		return nil
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "save":
		return gitSave(rest)
	case "undo":
		return gitUndo(rest)
	case "sync":
		return gitSync(rest)
	case "st", "status":
		return gitStatus()
	case "log":
		return gitLog(rest)
	case "update":
		return gitPullBranch(rest)
	case "change-author":
		return gitChangeAuthor(rest)
	case "delete-branch":
		return gitDeleteBranch(rest)
	case "delete-tag":
		return gitDeleteTag(rest)
	case "get":
		return gitClone(rest)
	case "unpushed":
		return run("git", "log", "@{u}..HEAD")
	case "unpulled":
		return run("git", "log", "HEAD..@{u}")
	case "outstanding":
		return run("git", "rebase", "-i", "@{u}")
	case "whatchanged":
		return gitWhatChanged(rest)
	case "gc":
		if !gitConfirm("gc --aggressive rewrites history objects and cannot be undone. Ensure no reflog recovery is needed first.", rest) {
			fmt.Fprintln(os.Stderr, "aborted")
			return nil
		}
		return run("git", "gc", "--prune=now", "--aggressive")
	case "un-add":
		return gitUnAdd(rest)
	case "ca":
		return run("git", "commit", "-a", "--amend")
	case "caf":
		return run("git", "commit", "-a", "--amend", "--no-edit")
	case "rm-keep":
		return gitRmKeep(rest)
	case "rm-from-commit":
		return gitRmFromCommit(rest)
	case "track":
		return gitTrack(rest)
	case "set-remote-branch":
		return gitSetRemoteBranch(rest)
	case "find-commit-back":
		return gitFindCommitBack(rest)
	case "rename":
		return gitRename(rest)
	case "conflicts":
		return gitConflicts(rest)
	case "mv-to-branch":
		return gitMvToBranch(rest)
	case "get-file-from":
		return gitGetFileFrom(rest)
	case "how-to-split":
		gitHowToSplit()
		return nil
	case "rename-branch":
		return gitRenameBranch(rest)
	case "change-branch-base":
		return gitChangeBranchBase(rest)
	case "restore-file":
		return gitRestoreFile(rest)
	case "set-remote":
		return gitSetRemote(rest)
	case "add-remote":
		return gitAddRemote(rest)
	case "add-mirror":
		return gitAddMirror(rest)
	case "export":
		return gitExport(rest)
	case "revert":
		return gitRevert(rest)
	case "rollback":
		return gitRollback(rest)
	case "clean-all-histories":
		return gitCleanAllHistories(rest)
	case "detach":
		return gitDetach(rest)
	case "subdir-to-root":
		return gitSubdirToRoot(rest)
	default:
		return fmt.Errorf("unknown git subcommand: %s  (run: aiw git help)", sub)
	}
}

// gitConfirm prints a warning and asks the user to confirm.
// Returns true immediately when --force is present in args.
func gitConfirm(prompt string, args []string) bool {
	if hasFlag(args, "--force") {
		return true
	}
	fmt.Fprintf(os.Stderr, "warning: %s\nProceed? [y/N] ", prompt)
	var resp string
	fmt.Scanln(&resp)
	return strings.EqualFold(strings.TrimSpace(resp), "y")
}

// gitRollback shows reflog and prints a step-by-step guide for recovering
// the repository state after any destructive operation.
//
//	aiw git rollback
func gitRollback(_ []string) error {
	if err := run("git", "reflog", "--date=relative", "-20"); err != nil {
		return err
	}
	fmt.Print(`
How to roll back / undo a recent git operation
----------------------------------------------

Step 1 — Find the SHA you want to return to in the reflog above.

Step 2 — Options:

  a) Undo last commit, keep changes staged:
       git reset --soft HEAD~1

  b) Undo last commit, keep changes in working tree:
       git reset HEAD~1           (or: aiw git undo)

  c) Undo last commit AND discard all changes:
       git reset --hard <SHA>     (or: aiw git undo --hard)

  d) Restore a branch pointer to a specific SHA:
       git branch -f <branch> <SHA>

  e) Create a safe recovery branch at any reflog SHA:
       git checkout -b recover-<SHA> <SHA>

  f) Undo a pushed commit safely (creates a new commit):
       aiw git revert <SHA>

Note: reflog entries expire after ~90 days.
      Run "aiw git gc" ONLY after you have recovered what you need.
`)
	return nil
}

// gitSave stages everything and commits.
//
//	aiw git save [message]
//
// If no message is given, the commit message defaults to "wip".
func gitSave(args []string) error {
	msg := "wip"
	if len(args) > 0 {
		msg = strings.Join(args, " ")
	}
	if err := run("git", "add", "-A"); err != nil {
		return err
	}
	return run("git", "commit", "-m", msg)
}

// gitUndo undoes the last commit while keeping the changes in the working tree.
//
//	aiw git undo [--hard] [--force]
//
// --hard   also discards working-tree changes (dangerous; requires confirmation).
// --force  skip confirmation prompt.
func gitUndo(args []string) error {
	if hasFlag(args, "--hard") {
		if !gitConfirm("--hard will permanently discard all working-tree changes.", args) {
			fmt.Fprintln(os.Stderr, "aborted")
			return nil
		}
		return run("git", "reset", "--hard", "HEAD~1")
	}
	return run("git", "reset", "HEAD~1")
}

// gitSync fetches the remote, rebases the current branch, then pushes.
//
//	aiw git sync [branch] [remote]
//
// Defaults: branch=current HEAD branch, remote=origin.
func gitSync(args []string) error {
	remote := "origin"
	if len(args) >= 2 {
		remote = args[1]
	}
	if !hasRemote(remote) {
		return fmt.Errorf("remote %q not found", remote)
	}
	if err := run("git", "fetch", "-p", remote); err != nil {
		return err
	}
	branch, err := currentBranch()
	if err != nil {
		return err
	}
	if len(args) >= 1 {
		branch = args[0]
	}
	if err := run("git", "rebase", remote+"/"+branch); err != nil {
		return err
	}
	return run("git", "push", remote, branch)
}

// gitStatus shows a concise working-tree status.
//
//	aiw git st
func gitStatus() error {
	return run("git", "status", "-sb")
}

// gitLog shows a formatted commit log.
//
//	aiw git log [style] [-n <count>]
//
// Styles (default: lg):
//
//	lg    Colored graph with relative date and author.
//	l     One line per commit.
//	hist  Colored graph with absolute date.
func gitLog(args []string) error {
	style := "lg"
	n := "20"
	for i, a := range args {
		switch a {
		case "lg", "l", "hist":
			style = a
		case "-n":
			if i+1 < len(args) {
				n = args[i+1]
			}
		}
	}
	switch style {
	case "l":
		return run("git", "log", "--pretty=oneline", "-n", n)
	case "hist":
		return run("git", "log",
			"--color",
			"--graph",
			"--pretty=format:%Cred%h%Creset %Cgreen%ad%Creset | %s %C(yellow)%d%Creset %C(bold blue)<%an>%Creset",
			"--date=short",
			"-n", n,
		)
	default: // lg
		return run("git", "log",
			"--all",
			"--color",
			"--graph",
			"--pretty=format:%Cred%h%Creset - %C(yellow)%d%Creset %s %Cgreen[%cr] %C(bold blue)<%an>%Creset",
			"--abbrev-commit",
			"--date=relative",
			"-n", n,
		)
	}
}

// gitClone does a shallow single-branch clone.
//
//	aiw git get <url> [-b <branch>] [-d <dir>] [--depth <n>]
//
// Flags:
//
//	-b, --branch  Branch to clone (default: repo default branch).
//	-d, --dir     Target directory (default: git decides).
//	--depth       History depth (default: 1).
//	--full        Disable shallow clone (omits --depth).
func gitClone(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git get <url> [-b <branch>] [-d <dir>] [--depth <n>] [--full]")
	}
	url := args[0]
	rest := args[1:]

	branch := ""
	dir := ""
	depth := "1"
	full := hasFlag(rest, "--full")

	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "-b", "--branch":
			if i+1 < len(rest) {
				branch = rest[i+1]
				i++
			}
		case "-d", "--dir":
			if i+1 < len(rest) {
				dir = rest[i+1]
				i++
			}
		case "--depth":
			if i+1 < len(rest) {
				depth = rest[i+1]
				i++
			}
		}
	}

	cmdArgs := []string{"clone"}
	if branch != "" {
		cmdArgs = append(cmdArgs, "--branch", branch, "--single-branch")
	}
	if !full {
		cmdArgs = append(cmdArgs, "--depth", depth)
	}
	cmdArgs = append(cmdArgs, url)
	if dir != "" {
		cmdArgs = append(cmdArgs, dir)
	}
	return run("git", cmdArgs...)
}

// gitDeleteTag deletes a local tag and optionally its remote counterpart.
//
//	aiw git delete-tag <tag> [--remote] [--remote-name <name>]
//
// --remote       Also delete the tag from the remote (default: origin).
// --remote-name  Remote name to use (default: origin).
//
// If the standard remote delete fails (e.g. name conflicts with a branch),
// the command automatically retries using the explicit refspec
// refs/tags/<tag>.
func gitDeleteTag(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git delete-tag <tag> [--remote] [--remote-name <name>] [--force]")
	}
	tag := args[0]
	rest := args[1:]

	if !gitConfirm(fmt.Sprintf("This will delete tag %q. Add --force to skip this prompt.", tag), rest) {
		fmt.Fprintln(os.Stderr, "aborted")
		return nil
	}

	// Delete local tag.
	if err := run("git", "tag", "-d", tag); err != nil {
		return err
	}

	if !hasFlag(rest, "--remote") {
		return nil
	}

	remoteName := "origin"
	for i, a := range rest {
		if a == "--remote-name" && i+1 < len(rest) {
			remoteName = rest[i+1]
		}
	}

	// Try standard delete first.
	if err := run("git", "push", "--delete", remoteName, tag); err != nil {
		// Fallback: explicit tag refspec avoids branch-name conflicts.
		fmt.Fprintf(os.Stderr, "standard remote delete failed; retrying with explicit refspec refs/tags/%s\n", tag)
		return run("git", "push", remoteName, ":refs/tags/"+tag)
	}
	return nil
}

// gitDeleteBranch deletes a local and/or remote branch.
//
//	aiw git delete-branch <branch> [--force] [--remote] [--remote-only] [--remote-name <name>]
//
// Flags:
//
//	--force        Force-delete local branch even if not merged (-D).
//	--remote       Delete local branch AND its remote counterpart.
//	--remote-only  Delete only the remote branch (skip local).
//	--remote-name  Remote name to use (default: origin).
func gitDeleteBranch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git delete-branch <branch> [--force] [--remote] [--remote-only] [--remote-name <name>]")
	}
	branch := args[0]
	rest := args[1:]

	force := hasFlag(rest, "--force")
	delRemote := hasFlag(rest, "--remote") || hasFlag(rest, "--remote-only")
	remoteOnly := hasFlag(rest, "--remote-only")
	remoteName := "origin"
	for i, a := range rest {
		if a == "--remote-name" && i+1 < len(rest) {
			remoteName = rest[i+1]
		}
	}

	prompt := fmt.Sprintf("This will delete branch %q", branch)
	if delRemote {
		prompt += " locally AND on remote"
	}
	prompt += ". Add --force to skip this prompt."
	if !gitConfirm(prompt, rest) {
		fmt.Fprintln(os.Stderr, "aborted")
		return nil
	}

	if !remoteOnly {
		localFlag := "-d"
		if force {
			localFlag = "-D"
		}
		if err := run("git", "branch", localFlag, branch); err != nil {
			return err
		}
	}
	if delRemote {
		if err := run("git", "push", remoteName, "--delete", branch); err != nil {
			return err
		}
	}
	return nil
}

// gitChangeAuthor amends the last commit's author name and email,
// or rewrites all history when --all is given.
//
//	aiw git change-author <Name> <email> [--all --filter-name <old> ...] [--force]
//
// Without --all: amend only the last commit (existing behaviour).
// With --all:    rewrite every commit whose author or committer name matches
//               one of the --filter-name values using git filter-branch.
//               At least one --filter-name is required.
//
// ⚠  --all rewrites history; collaborators must re-clone or reset.
func gitChangeAuthor(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: aiw git change-author <Name> <email> [--all --filter-name <old> ...] [--force]")
	}
	name := args[0]
	email := args[1]
	rest := args[2:]

	if !hasFlag(rest, "--all") {
		// Original behaviour: amend the last commit only.
		if !gitConfirm(fmt.Sprintf("This will amend the last commit's author to %q <%s>. Add --force to skip this prompt.", name, email), rest) {
			fmt.Fprintln(os.Stderr, "aborted")
			return nil
		}
		author := fmt.Sprintf("%s <%s>", name, email)
		return run("git", "commit", "--amend", "--author", author, "--no-edit")
	}

	// Collect all --filter-name values (may appear multiple times).
	var filterNames []string
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--filter-name" && i+1 < len(rest) {
			filterNames = append(filterNames, rest[i+1])
			i++
		}
	}
	if len(filterNames) == 0 {
		return fmt.Errorf("--all requires at least one --filter-name <old-name>")
	}

	warning := fmt.Sprintf(
		"change-author --all will rewrite ALL commits whose author/committer matches %v\n"+
			"and replace them with %q <%s>.\n"+
			"This permanently rewrites history. Collaborators must re-clone or reset.",
		filterNames, name, email)
	if !gitConfirm(warning, rest) {
		fmt.Fprintln(os.Stderr, "aborted")
		return nil
	}

	// Build the env-filter shell script.
	// Each filter-name is checked against both author and committer name.
	var conds []string
	for _, fn := range filterNames {
		conds = append(conds,
			fmt.Sprintf(`[ "$GIT_AUTHOR_NAME" = %q ]`, fn),
			fmt.Sprintf(`[ "$GIT_COMMITTER_NAME" = %q ]`, fn),
		)
	}
	ifExpr := strings.Join(conds, " || ")

	script := fmt.Sprintf(
		`an="$GIT_AUTHOR_NAME"
am="$GIT_AUTHOR_EMAIL"
cn="$GIT_COMMITTER_NAME"
cm="$GIT_COMMITTER_EMAIL"
if %s
then
    an=%q
    am=%q
    cn=%q
    cm=%q
fi
export GIT_AUTHOR_NAME="$an"
export GIT_AUTHOR_EMAIL="$am"
export GIT_COMMITTER_NAME="$cn"
export GIT_COMMITTER_EMAIL="$cm"`,
		ifExpr, name, email, name, email)

	return run("git", "filter-branch", "-f", "--env-filter", script, "HEAD")
}

// gitWhatChanged shows files changed per commit (history), or files changed
// in a specific commit when a SHA is provided.
//
//	aiw git whatchanged [sha] [--names]
//
// No args:   full history with file-change details.
// <sha>:     show the commit message + stats for that commit.
// --names:   show only the file names (no stats or diff context).
func gitWhatChanged(args []string) error {
	sha := ""
	namesOnly := hasFlag(args, "--names")
	for _, a := range args {
		if a != "--names" {
			sha = a
			break
		}
	}
	if sha == "" {
		return run("git", "whatchanged")
	}
	if namesOnly {
		return run("git", "diff-tree", "--no-commit-id", "--name-only", "-r", sha)
	}
	return run("git", "show", "--stat", sha)
}

// gitPullBranch fetches and updates a local branch from its remote counterpart.
//
//	aiw git update [branch] [remote]
//
// branch defaults to the remote's default branch (main/master, auto-detected).
// remote defaults to "origin".
//
// If the target branch is the currently checked-out branch, uses
// `git pull --ff-only` (safer and avoids the refspec two-ref trick).
// Otherwise updates the local branch without a checkout.
func gitPullBranch(args []string) error {
	remote := "origin"
	if len(args) >= 2 {
		remote = args[1]
	}

	branch := ""
	if len(args) >= 1 {
		branch = args[0]
	}
	if branch == "" {
		detected, err := detectDefaultBranch(remote)
		if err != nil {
			return err
		}
		branch = detected
		fmt.Fprintf(os.Stderr, "default branch: %s\n", branch)
	}

	cur, err := currentBranch()
	if err != nil {
		return err
	}
	if branch == cur {
		// On the target branch: --ff-only is correct and safe.
		return run("git", "pull", "--ff-only", remote, branch)
	}
	// Off the target branch: update via fetch refspec (no checkout needed).
	refspec := "refs/heads/" + branch + ":refs/heads/" + branch
	return run("git", "fetch", remote, refspec,
		"--recurse-submodules=no", "--progress", "--prune")
}

// detectDefaultBranch returns the default branch name for a remote (e.g. "main" or "master").
// It first queries the remote's HEAD symref, then falls back to checking local branch names.
func detectDefaultBranch(remote string) (string, error) {
	out, err := gitOutput("git", "symbolic-ref", "refs/remotes/"+remote+"/HEAD", "--short")
	if err == nil {
		ref := strings.TrimSpace(out)
		if prefix := remote + "/"; strings.HasPrefix(ref, prefix) {
			return strings.TrimPrefix(ref, prefix), nil
		}
	}
	// Fallback: check local branch names.
	for _, candidate := range []string{"main", "master"} {
		if refExists("refs/heads/" + candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("cannot detect default branch; pass one explicitly, e.g.: aiw git update main")
}

// gitExport archives a commit/branch/tag to a zip file without VCS metadata.
//
//	aiw git export <ref> [output.zip]
//
// ref        Any commit, branch, or tag (e.g. main, v1.2.3, abc1234).
// output     Output file path (default: <ref>.zip).
func gitExport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git export <ref> [output.zip]")
	}
	ref := args[0]
	output := ref + ".zip"
	if len(args) >= 2 {
		output = args[1]
	}
	return run("git", "archive", "--format", "zip", "--output", output, ref)
}

// gitRevert creates a new commit that undoes the changes from a specific commit.
//
//	aiw git revert <commit> [--no-commit]
//
// --no-commit  Stage the revert without committing (useful when reverting multiple commits).
func gitRevert(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git revert <commit> [--no-commit] [--force]")
	}
	sha := ""
	for _, a := range args {
		if a != "--no-commit" && a != "--force" {
			sha = a
			break
		}
	}
	if !gitConfirm(fmt.Sprintf("This will create a new commit that undoes %q. Add --force to skip this prompt.", sha), args) {
		fmt.Fprintln(os.Stderr, "aborted")
		return nil
	}
	cmdArgs := []string{"revert"}
	if hasFlag(args, "--no-commit") {
		cmdArgs = append(cmdArgs, "--no-commit")
	}
	for _, a := range args {
		if a != "--no-commit" && a != "--force" {
			cmdArgs = append(cmdArgs, a)
		}
	}
	return run("git", cmdArgs...)
}

// gitSetRemote changes the URL of an existing remote.
//
//	aiw git set-remote <url> [remote]
//
// remote defaults to "origin".
func gitSetRemote(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git set-remote <url> [remote]")
	}
	url := args[0]
	remote := "origin"
	if len(args) >= 2 {
		remote = args[1]
	}
	return run("git", "remote", "set-url", remote, url)
}

// gitAddRemote adds a new remote to the repository.
//
//	aiw git add-remote <url> [remote]
//
// remote defaults to "origin".
func gitAddRemote(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git add-remote <url> [remote]")
	}
	url := args[0]
	remote := "origin"
	if len(args) >= 2 {
		remote = args[1]
	}
	return run("git", "remote", "add", remote, url)
}

// gitAddMirror adds a mirror remote and optionally pushes a branch to it.
//
//	aiw git add-mirror <url> [remote] [--push [branch]]
//
// remote defaults to "mirror", branch defaults to current branch.
// --push  Push to the mirror remote after adding it.
func gitAddMirror(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git add-mirror <url> [remote] [--push [branch]]")
	}
	push := hasFlag(args, "--push")
	positional := []string{}
	for _, a := range args {
		if a != "--push" {
			positional = append(positional, a)
		}
	}
	url := positional[0]
	remote := "mirror"
	if len(positional) >= 2 {
		remote = positional[1]
	}
	if err := run("git", "remote", "add", remote, url); err != nil {
		return err
	}
	if !push {
		return nil
	}
	branch := ""
	if len(positional) >= 3 {
		branch = positional[2]
	}
	if branch == "" {
		var err error
		branch, err = currentBranch()
		if err != nil {
			return err
		}
	}
	return run("git", "push", remote, branch)
}

// gitRestoreFile restores one or more files to a specific version.
//
//	aiw git restore-file <file...>
//	aiw git restore-file <file...> --from <commit>
//
// No --from:          discard working-tree changes (git restore <file...>).
// --from <commit>:    restore file content from that commit.
//                     Use ^ suffix for the parent: abc1234^
func gitRestoreFile(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git restore-file <file...> [--from <commit>]")
	}
	source := ""
	files := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--from" {
			if i+1 < len(args) {
				source = args[i+1]
				i++
			}
		} else {
			files = append(files, args[i])
		}
	}
	if len(files) == 0 {
		return fmt.Errorf("usage: aiw git restore-file <file...> [--from <commit>]")
	}
	if source == "" {
		cmdArgs := append([]string{"restore"}, files...)
		return run("git", cmdArgs...)
	}
	cmdArgs := append([]string{"restore", "--source=" + source}, files...)
	return run("git", cmdArgs...)
}

// gitChangeBranchBase moves a branch onto a new base using rebase --onto.
//
//	aiw git change-branch-base <new-base> <old-base> [branch]
//
// new-base  The commit/branch to rebase onto.
// old-base  Commits AFTER this point (exclusive) are moved.
// branch    Branch to rebase (default: current branch).
//
// Example — move feature off master onto develop:
//
//	aiw git change-branch-base develop master feature
//
// Equivalent to: git rebase --onto develop master feature
func gitChangeBranchBase(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: aiw git change-branch-base <new-base> <old-base> [branch] [--force]")
	}
	newBase := args[0]
	oldBase := args[1]
	if !gitConfirm(fmt.Sprintf("This will rebase --onto %q from %q. Add --force to skip this prompt.", newBase, oldBase), args) {
		fmt.Fprintln(os.Stderr, "aborted")
		return nil
	}
	cmdArgs := []string{"rebase", "--onto", newBase, oldBase}
	if len(args) >= 3 && args[2] != "--force" {
		cmdArgs = append(cmdArgs, args[2])
	}
	return run("git", cmdArgs...)
}

// gitRenameBranch renames a local branch and optionally updates the remote.
//
//	aiw git rename-branch <new-name> [--remote] [--remote-name <name>]
//	aiw git rename-branch <old-name> <new-name> [--remote] [--remote-name <name>]
//
// One positional arg:   rename the current branch to <new-name>.
// Two positional args:  rename <old-name> to <new-name> (branch need not be checked out).
// --remote              Also update remote: delete old branch, push new branch.
// --remote-name         Remote name (default: origin).
func gitRenameBranch(args []string) error {
	delRemote := hasFlag(args, "--remote")
	remoteName := "origin"
	positional := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--remote":
		case "--remote-name":
			if i+1 < len(args) {
				remoteName = args[i+1]
				i++
			}
		default:
			positional = append(positional, args[i])
		}
	}

	var oldName, newName string
	switch len(positional) {
	case 1:
		newName = positional[0]
		// Rename the current branch.
		var err error
		oldName, err = currentBranch()
		if err != nil {
			return err
		}
		if err := run("git", "branch", "-m", newName); err != nil {
			return err
		}
	case 2:
		oldName, newName = positional[0], positional[1]
		if err := run("git", "branch", "-m", oldName, newName); err != nil {
			return err
		}
	default:
		return fmt.Errorf("usage: aiw git rename-branch <new-name> [--remote] [--remote-name <name>]\n" +
			"       aiw git rename-branch <old-name> <new-name> [--remote] [--remote-name <name>]")
	}

	if !delRemote {
		return nil
	}
	// Delete old remote branch, then push the new name.
	if err := run("git", "push", remoteName, "--delete", oldName); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not delete remote branch %q: %v\n", oldName, err)
	}
	return run("git", "push", remoteName, newName)
}

// gitHowToSplit prints a step-by-step guide for splitting commits across branches.
func gitHowToSplit() {
	fmt.Print(`How to split commits that landed on the wrong branch
=====================================================

Situation: multiple commits are on the current branch but belong to different
feature branches. Use reset + cherry-pick to redistribute them.

Step 1 — See what you have

  aiw git log           # identify each commit hash
  git log --oneline     # compact view

Step 2 — Note the last "good" commit on this branch

  This is the commit BEFORE the ones that need to move.
  Example: abc0000

Step 3 — Create feature branches at the current HEAD
  (they will point to all the commits for now)

  git checkout -b feature-A
  git checkout -b feature-B
  git checkout <original-branch>   # go back

Step 4 — Reset the original branch to the last good commit

  git reset --hard abc0000

Step 5 — Cherry-pick the right commits onto each branch

  git checkout feature-A
  git cherry-pick <sha-for-A>       # copies commit, new hash generated

  git checkout feature-B
  git cherry-pick <sha-for-B>

Key facts about cherry-pick
  • Copies only the named commit, not the whole branch.
  • Generates a new commit hash on the target branch.
  • Original commits remain where they are until pruned.

Useful variants
  git cherry-pick A..B              # pick a range (exclusive A, inclusive B)
  git cherry-pick A^..B             # pick a range (inclusive A and B)
  git cherry-pick --no-commit <sha> # apply changes without committing

Common commands during this workflow
  aiw git log                       # pretty graph log
  aiw git mv-to-branch <branch>     # shortcut when only last commit needs moving
  aiw git get-file-from <branch> <file>  # grab a single file from another branch
`)
}

// gitGetFileFrom copies one or more files from another branch into the current branch.
// The files are staged automatically after the copy.
//
//	aiw git get-file-from <branch> <file...>
func gitGetFileFrom(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: aiw git get-file-from <branch> <file...>")
	}
	branch := args[0]
	files := args[1:]
	cmdArgs := append([]string{"checkout", branch, "--"}, files...)
	return run("git", cmdArgs...)
}

// gitMvToBranch moves accidental commits from the current branch to a new branch.
//
//	aiw git mv-to-branch <new-branch> [reset-to]
//
// new-branch  Name of the branch to create with the current commits.
// reset-to    Commit to reset the current branch back to (default: HEAD^).
//
// Workflow:
//  1. Create new-branch at current HEAD (preserves commits).
//  2. Reset the current branch back to reset-to.
//  3. Switch to new-branch.
func gitMvToBranch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git mv-to-branch <new-branch> [reset-to] [--force]")
	}
	newBranch := args[0]
	resetTo := "HEAD^"
	if len(args) >= 2 && args[1] != "--force" {
		resetTo = args[1]
	}
	if !gitConfirm(fmt.Sprintf("This will reset the current branch to %q. Uncommitted changes will be lost. Add --force to skip.", resetTo), args) {
		fmt.Fprintln(os.Stderr, "aborted")
		return nil
	}
	// Step 1: create branch at current HEAD.
	if err := run("git", "branch", newBranch); err != nil {
		return err
	}
	// Step 2: reset current branch back.
	if err := run("git", "reset", "--hard", resetTo); err != nil {
		return err
	}
	// Step 3: switch to the new branch.
	return run("git", "checkout", newBranch)
}

// gitConflicts helps resolve conflicts after a merge/rebase/pull.
//
//	aiw git conflicts [--diff] [--check] [--staged]
//
// Default: list conflicted file names with a count and a next-step hint.
// --diff    Full diff of all unmerged hunks (verbose).
// --check   Scan for remaining conflict markers (<<<<<<< / =======); use after
//           editing to confirm everything is resolved.
// --staged  Show what is already staged (ready to commit).
func gitConflicts(args []string) error {
	switch {
	case hasFlag(args, "--diff"):
		return run("git", "diff", "--diff-filter=U")
	case hasFlag(args, "--check"):
		return run("git", "diff", "--check")
	case hasFlag(args, "--staged"):
		return run("git", "diff", "--staged")
	default:
		// Show file list with count; a non-zero exit from the diff command
		// is normal when there are conflicts, so we report it as a hint.
		out, err := gitOutput("git", "diff", "--name-only", "--diff-filter=U")
		if err != nil {
			return err
		}
		if out == "" {
			fmt.Println("No unresolved conflicts found.")
			return nil
		}
		files := strings.Split(strings.TrimSpace(out), "\n")
		fmt.Printf("%d conflicted file(s):\n\n", len(files))
		for _, f := range files {
			fmt.Println(" ", f)
		}
		fmt.Println(`
Next steps:
  1. Edit each file above and resolve the conflict markers.
  2. aiw git conflicts --check    # confirm no markers remain
  3. git add <file>               # stage each resolved file
  4. aiw git conflicts --staged   # review what will be committed
  5. git commit (or git rebase --continue)`)
		return nil
	}
}

// gitFindCommitBack shows the reflog and prints a step-by-step recovery guide.
//
//	aiw git find-commit-back
func gitFindCommitBack(_ []string) error {
	if err := run("git", "reflog"); err != nil {
		return err
	}
	fmt.Print(`
How to recover after an accidental hard reset
---------------------------------------------

1. Find the SHA of the commit you want to recover (see reflog above).

2. Create a recovery branch at that SHA and switch to it:

     git checkout -b recover-branch <SHA>

   Or, if you just want to reset the current branch back:

     git reset --hard <SHA>

Note: Git keeps reflog entries for ~90 days by default.
      Run "aiw git gc" only AFTER you have recovered what you need.
`)
	return nil
}

// gitRename renames a file changing only its case without modifying content.
//
//	aiw git rename <old> <new>
func gitRename(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: aiw git rename <old-name> <new-name>")
	}
	return run("git", "mv", "--force", args[0], args[1])
}

// gitRmKeep removes a file from git tracking without deleting it from disk.
//
//	aiw git rm-keep <file...>
func gitRmKeep(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git rm-keep <file...>")
	}
	cmdArgs := append([]string{"rm", "--cached"}, args...)
	return run("git", cmdArgs...)
}

// gitRmFromCommit removes a file from the last commit without deleting it from disk,
// or scrubs it from all (or partial) history when --history is given.
//
//	aiw git rm-from-commit <file> [--history] [--from <commit>] [--force]
//
// Without --history: remove the file from the last commit only (existing behaviour).
// With --history:    use git filter-branch --tree-filter to delete the file from
//                   every commit reachable from HEAD (or from --from..HEAD).
//
// --from <commit>  Only rewrite commits after <commit> (exclusive).
//                  Commits at or before <commit> are NOT touched.
//                  Omitting --from rewrites the entire history.
//
// ⚠  --history rewrites history; collaborators must re-clone or reset.
func gitRmFromCommit(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git rm-from-commit <file> [--history] [--from <commit>] [--force]")
	}

	file := args[0]
	rest := args[1:]

	if !hasFlag(rest, "--history") {
		// Original behaviour: amend the last commit only.
		if err := run("git", "checkout", "HEAD^", "--", file); err != nil {
			return err
		}
		if err := run("git", "add", "-A"); err != nil {
			return err
		}
		return run("git", "commit", "--amend", "--no-edit")
	}

	// --history path: parse --from.
	fromCommit := ""
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--from" && i+1 < len(rest) {
			fromCommit = rest[i+1]
			i++
		}
	}

	var rangeDesc string
	if fromCommit != "" {
		rangeDesc = fmt.Sprintf("commits after %q up to HEAD", fromCommit)
	} else {
		rangeDesc = "ALL commits in history"
	}
	warning := fmt.Sprintf(
		"rm-from-commit --history will permanently delete %q from %s\n"+
			"using git filter-branch. This rewrites history.\n"+
			"Collaborators must re-clone or reset after a force-push.",
		file, rangeDesc)
	if !gitConfirm(warning, rest) {
		fmt.Fprintln(os.Stderr, "aborted")
		return nil
	}

	// Build the tree-filter command: 'git rm -f --ignore-unmatch <file>'
	// --ignore-unmatch prevents failure on commits where the file doesn't exist.
	treeFilter := fmt.Sprintf("git rm -f --ignore-unmatch %s", file)

	cmdArgs := []string{"filter-branch", "-f", "--tree-filter", treeFilter}
	if fromCommit != "" {
		// Rewrite only fromCommit..HEAD; commits before fromCommit are untouched.
		cmdArgs = append(cmdArgs, "--", fromCommit+"..HEAD")
	} else {
		cmdArgs = append(cmdArgs, "HEAD")
	}
	return run("git", cmdArgs...)
}

// gitTrack fetches all remotes and checks out a remote branch as a local tracking branch.
//
//	aiw git track <branch> [remote]
//
// Defaults: remote=origin.
func gitTrack(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git track <branch> [remote]")
	}
	branch := args[0]
	remote := "origin"
	if len(args) >= 2 {
		remote = args[1]
	}
	if err := run("git", "fetch", "--all"); err != nil {
		return err
	}
	return run("git", "checkout", "--track", remote+"/"+branch)
}

// gitSetRemoteBranch associates the current local branch with a remote branch,
// or pushes and sets the upstream in one step.
//
//	aiw git set-remote-branch [remote] [branch] [--push]
//
// --push  Push the current branch and set upstream at the same time.
// Defaults: remote=origin, branch=current branch name.
func gitSetRemoteBranch(args []string) error {
	remote := "origin"
	branch := ""
	push := hasFlag(args, "--push")

	positional := []string{}
	for _, a := range args {
		if a != "--push" {
			positional = append(positional, a)
		}
	}
	if len(positional) >= 1 {
		remote = positional[0]
	}
	if len(positional) >= 2 {
		branch = positional[1]
	}

	if branch == "" {
		var err error
		branch, err = currentBranch()
		if err != nil {
			return err
		}
	}

	if push {
		return run("git", "push", "-u", remote, branch)
	}
	return run("git", "branch", "--set-upstream-to="+remote+"/"+branch)
}

// gitUnAdd unstages files (reverses git add).
//
//	aiw git un-add [file...]
//
// With no arguments, unstages everything.
func gitUnAdd(args []string) error {
	if len(args) == 0 {
		return run("git", "restore", "--staged", ".")
	}
	cmdArgs := append([]string{"restore", "--staged"}, args...)
	return run("git", cmdArgs...)
}

// gitSubdirToRoot makes a subdirectory the new root of the repository across
// the entire commit history using git filter-branch --subdirectory-filter.
//
//	aiw git subdir-to-root <subdir> [--force]
//
// Every commit is rewritten so that <subdir> becomes the project root.
// Commits that did not touch <subdir> are dropped from history.
// This is useful after importing from SVN/CVS where trunk/tags/branches
// appear as top-level directories.
//
// ⚠  Rewrites all history; collaborators must re-clone or reset after force-push.
func gitSubdirToRoot(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git subdir-to-root <subdir> [--force]")
	}
	subdir := args[0]
	warning := fmt.Sprintf(
		"subdir-to-root will rewrite ALL history so that %q becomes the repo root.\n"+
			"Commits that did not touch this directory will be dropped.\n"+
			"This cannot be undone after a force-push.",
		subdir)
	if !gitConfirm(warning, args) {
		fmt.Fprintln(os.Stderr, "aborted")
		return nil
	}
	return run("git", "filter-branch", "-f", "--subdirectory-filter", subdir, "HEAD")
}

// gitDetach removes all history before a given ref, keeping only commits
// from <ref>..HEAD on the target branch.
//
//	aiw git detach <ref> [--branch <name>] [--message <msg>] [--force]
//
// How it works (equivalent shell):
//
//	git checkout --orphan _tmp_ <ref>   # orphan snapshot at <ref>
//	git commit -m <msg>                 # single root commit
//	git rebase --onto _tmp_ <ref> <branch>
//	git branch -D _tmp_
//
// Flags:
//
//	--branch   Branch to detach (default: current branch).
//	--message  Root-commit message (default: "new begin").
//	--force    Skip confirmation.
//
// ⚠  Rewrites history; collaborators must re-clone or reset.
func gitDetach(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aiw git detach <ref> [--branch <name>] [--message <msg>] [--force]")
	}
	ref := args[0]
	rest := args[1:]

	branch := ""
	msg := "new begin"
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--branch":
			if i+1 < len(rest) {
				branch = rest[i+1]
				i++
			}
		case "--message":
			if i+1 < len(rest) {
				msg = rest[i+1]
				i++
			}
		}
	}

	if branch == "" {
		var err error
		branch, err = currentBranch()
		if err != nil {
			return err
		}
	}

	warning := fmt.Sprintf(
		"detach will permanently erase all history before %q on branch %q.\n"+
			"The rewritten branch will be left locally (use 'aiw git push -f' to publish).\n"+
			"This cannot be undone after a force-push.",
		ref, branch)
	if !gitConfirm(warning, args) {
		fmt.Fprintln(os.Stderr, "aborted")
		return nil
	}

	tmpBranch := "_aiw_detach_tmp_"

	// 1. Create orphan branch at <ref> — captures the tree state without history.
	if err := run("git", "checkout", "--orphan", tmpBranch, ref); err != nil {
		return fmt.Errorf("checkout --orphan: %w", err)
	}

	// 2. Commit the snapshot as the new root.
	if err := run("git", "commit", "-m", msg); err != nil {
		_ = run("git", "checkout", branch) // best-effort restore
		return fmt.Errorf("root commit: %w", err)
	}

	// 3. Rebase <ref>..<branch> onto the new root.
	//    git rebase --onto <newbase> <upstream> <branch>
	if err := run("git", "rebase", "--onto", tmpBranch, ref, branch); err != nil {
		fmt.Fprintln(os.Stderr,
			"rebase paused (conflicts?). Resolve, then:\n"+
				"  git rebase --continue   or\n"+
				"  git rebase --abort")
		return err
	}

	// 4. Clean up the temporary orphan branch.
	if err := run("git", "branch", "-D", tmpBranch); err != nil {
		return fmt.Errorf("cleanup temp branch: %w", err)
	}

	fmt.Fprintf(os.Stderr, "done: history before %q removed from branch %q.\n", ref, branch)
	return nil
}

// gitCleanAllHistories squashes the entire repository history into a single
// root commit, force-pushes it to the remote, and re-links the local branch.
//
//	aiw git clean-all-histories [--branch <name>] [--remote <name>] [--message <msg>] [--force]
//
// Flags:
//
//	--branch   Branch to reset (default: master or main, auto-detected).
//	--remote   Remote name (default: origin).
//	--message  Commit message (default: "initial commit").
//	--force    Skip confirmation prompt.
//
// ⚠  This rewrites history and cannot be undone after force-push.
// All collaborators will need to re-clone or reset their local copies.
func gitCleanAllHistories(args []string) error {
	// Parse flags.
	branch := ""
	remote := "origin"
	msg := "initial commit"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--branch":
			if i+1 < len(args) {
				branch = args[i+1]
				i++
			}
		case "--remote":
			if i+1 < len(args) {
				remote = args[i+1]
				i++
			}
		case "--message":
			if i+1 < len(args) {
				msg = args[i+1]
				i++
			}
		}
	}

	// Auto-detect default branch when not supplied.
	if branch == "" {
		out, err := gitOutput("git", "rev-parse", "--abbrev-ref", "HEAD")
		if err == nil {
			branch = strings.TrimSpace(out)
		}
		if branch == "" || branch == "HEAD" {
			branch = "master"
		}
	}

	warning := fmt.Sprintf(
		"clean-all-histories will PERMANENTLY erase all commit history on branch %q\n"+
			"and force-push to remote %q.\n"+
			"All collaborators must re-clone or reset their local copies.\n"+
			"This cannot be undone once pushed.",
		branch, remote)
	if !gitConfirm(warning, args) {
		fmt.Fprintln(os.Stderr, "aborted")
		return nil
	}

	orphanBranch := "_aiw_clean_histories_tmp_"

	// 1. Create orphan branch.
	if err := run("git", "checkout", "--orphan", orphanBranch); err != nil {
		return fmt.Errorf("checkout --orphan: %w", err)
	}

	// 2. Stage all tracked files.
	if err := run("git", "add", "-A"); err != nil {
		_ = run("git", "checkout", branch) // best-effort restore
		return fmt.Errorf("git add -A: %w", err)
	}

	// 3. Initial commit.
	if err := run("git", "commit", "-m", msg); err != nil {
		_ = run("git", "checkout", branch)
		return fmt.Errorf("git commit: %w", err)
	}

	// 4. Delete the original branch.
	if err := run("git", "branch", "-D", branch); err != nil {
		return fmt.Errorf("delete branch %q: %w", branch, err)
	}

	// 5. Rename orphan branch to the original name.
	if err := run("git", "branch", "-m", branch); err != nil {
		return fmt.Errorf("rename branch to %q: %w", branch, err)
	}

	// 6. Force-push to remote.
	if !hasRemote(remote) {
		fmt.Fprintf(os.Stderr, "note: remote %q not found; skipping push.\n", remote)
		return nil
	}
	if err := run("git", "push", "-f", remote, branch); err != nil {
		return fmt.Errorf("force push: %w", err)
	}

	// 7. Re-link local branch to remote tracking.
	return run("git", "branch", "--set-upstream-to="+remote+"/"+branch)
}

// currentBranch returns the name of the currently checked-out branch.
func currentBranch() (string, error) {
	out, err := gitOutput("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", errors.New("cannot determine current branch")
	}
	return strings.TrimSpace(out), nil
}

func gitUsage(args []string) {
	type cmd struct {
		sig  string
		desc string
	}
	type grp struct {
		title string
		cmds  []cmd
	}

	groups := []grp{
		{"Snapshot & Commit", []cmd{
			{"save [message]", `Stage all and commit (default: "wip").`},
			{"undo [--hard]", "Undo last commit; keeps changes unless --hard. \u26a0"},
			{"ca", "Amend last commit interactively (all changes)."},
			{"caf", "Amend last commit silently (all changes)."},
			{"change-author <Name> <email> [--all --filter-name N ...]", "Amend last commit; or rewrite all history matching names. \u26a0"},
			{"rm-from-commit <file> [--history] [--from <commit>]", "Remove file from last commit or full history. \u26a0"},
			{"revert <commit> [--no-commit]", "New commit that undoes a specific commit. \u26a0"},
		}},
		{"History & Status", []cmd{
			{"st", "Short status (git status -sb)."},
			{"log [lg|l|hist] [-n N]", "Formatted log (default: lg style, -n 20)."},
			{"whatchanged [sha] [--names]", "File-change history; or files changed in a specific commit."},
			{"unpushed", "Commits not yet pushed (log @{u}..HEAD)."},
			{"unpulled", "Commits not yet pulled (log HEAD..@{u})."},
		}},
		{"Sync & Remote", []cmd{
			{"sync [branch] [remote]", "Fetch + rebase + push (default remote: origin)."},
			{"update [branch] [remote]", "Update branch from remote; auto-detects main/master; ff-only if current."},
			{"outstanding", "Interactive rebase against upstream (rebase -i @{u})."},
			{"get <url> [-b branch] [-d dir] [--depth N] [--full]", "Shallow single-branch clone (default depth: 1)."},
			{"set-remote-branch [remote] [branch] [--push]", "Link current branch to remote tracking; --push also pushes."},
			{"set-remote <url> [remote]", "Change the URL of a remote (default: origin)."},
			{"add-remote <url> [remote]", "Add a new remote (default name: origin)."},
			{"add-mirror <url> [remote] [--push [branch]]", "Add mirror remote (default: mirror); --push also pushes."},
		}},
		{"Branch", []cmd{
			{"delete-branch <branch> [--force] [--remote] ...", "Delete local and/or remote branch. \u26a0"},
			{"rename-branch <new> / <old> <new> [--remote]", "Rename current or any branch; --remote updates remote."},
			{"track <branch> [remote]", "Checkout a remote branch as local tracking branch."},
			{"mv-to-branch <new-branch> [reset-to]", "Move commits to a new branch, reset current. \u26a0"},
			{"change-branch-base <new-base> <old-base> [branch]", "Rebase --onto: move commits onto a new base. \u26a0"},
		}},
		{"File", []cmd{
			{"un-add [file...]", "Unstage files; all files if no args."},
			{"rm-keep <file...>", "Remove from git tracking but keep on disk."},
			{"restore-file <file...> [--from <commit>]", "Restore file(s) to HEAD or a specific commit."},
			{"get-file-from <branch> <file...>", "Copy files from another branch (auto-staged)."},
			{"rename <old> <new>", "Rename a file (case-only safe, git mv --force)."},
		}},
		{"Conflicts", []cmd{
			{"conflicts [--diff|--check|--staged]", "List conflict files + guide; --diff/--check/--staged for more."},
		}},
		{"Tags & Export", []cmd{
			{"delete-tag <tag> [--remote] [--remote-name <n>]", "Delete local tag; --remote also removes from remote. \u26a0"},
			{"export <ref> [output.zip]", "Archive a commit/branch/tag to zip (no VCS metadata)."},
		}},
		{"Recovery", []cmd{
			{"rollback", "Show reflog + step-by-step guide for undoing any operation."},
			{"find-commit-back", "Show reflog + hard-reset recovery guide."},
			{"gc", "Aggressive garbage collection. Run AFTER recovery. \u26a0"},
			{"detach <ref> [--branch B] [--message M]", "Remove history before <ref>, keep <ref>..HEAD. \u26a0\u26a0"},
			{"subdir-to-root <subdir>", "Make a subdirectory the new repo root across all history. \u26a0\u26a0"},
			{"clean-all-histories [--branch B] [--remote R] [--message M]", "Squash all history into one commit + force-push. \u26a0\u26a0"},
		}},
		{"Guides", []cmd{
			{"how-to-split", "Guide: split commits across branches via cherry-pick."},
			{"help [--alphabet]", "Show this help; --alphabet sorts commands alphabetically."},
		}},
	}

	printLine := func(sig, desc string) {
		if len(sig) <= 46 {
			fmt.Printf("  %-46s %s\n", sig, desc)
		} else {
			fmt.Printf("  %s\n  %48s%s\n", sig, "", desc)
		}
	}

	footer := "\n\u26a0  requires confirmation; use --force to skip.\n" +
		"Global: --dry-run (preview without executing)   --force (skip confirmations)\n\n"

	if hasFlag(args, "--alphabet") {
		type flat struct{ sig, desc string }
		var all []flat
		for _, g := range groups {
			for _, c := range g.cmds {
				all = append(all, flat{c.sig, c.desc})
			}
		}
		sort.Slice(all, func(i, j int) bool { return all[i].sig < all[j].sig })
		fmt.Print("aiw git — simple git shortcuts  (alphabetical)\n\n")
		for _, e := range all {
			printLine(e.sig, e.desc)
		}
		fmt.Print(footer)
		return
	}

	sep := strings.Repeat("\u2500", 54)
	fmt.Print("aiw git — simple git shortcuts\n\n")
	for _, g := range groups {
		fmt.Printf("\u2500\u2500 %s %s\n", g.title, sep)
		for _, c := range g.cmds {
			printLine(c.sig, c.desc)
		}
		fmt.Println()
	}

	fmt.Printf("\u2500\u2500 Examples %s\n", sep)
	examples := [][2]string{
		{"aiw git save", `commit everything with message "wip"`},
		{"aiw git save \"fix typo\"", "commit with custom message"},
		{"aiw git undo", "soft-reset last commit (keeps edits)"},
		{"aiw git undo --hard", "discard last commit AND edits"},
		{"aiw git sync", "fetch origin, rebase, push"},
		{"aiw git sync main upstream", "sync branch main against upstream remote"},
		{"aiw git st", "short status"},
		{"aiw git log", "graph log, last 20 commits"},
		{"aiw git log l -n 50", "one-line log, last 50"},
		{"aiw git whatchanged abc1234", "files changed in a commit"},
		{"aiw git whatchanged abc1234 --names", "file names only"},
		{"aiw git get https://github.com/u/r.git", "shallow clone"},
		{"aiw git get https://github.com/u/r.git -b dev --full", "full clone of a branch"},
		{"aiw git update main", "update local main without checkout"},
		{"aiw git delete-branch old-feat --remote", "delete local + remote branch"},
		{"aiw git rename-branch new-name", "rename current branch"},
		{"aiw git rename-branch old new --remote", "rename + update origin"},
		{"aiw git change-branch-base develop master feature", "rebase feature off master onto develop"},
		{"aiw git restore-file src/main.go", "discard working-tree changes"},
		{"aiw git restore-file src/main.go --from HEAD~3", "restore file from 3 commits back"},
		{"aiw git get-file-from hotfix auth.go", "copy file from another branch"},
		{"aiw git mv-to-branch feature/my-work", "move last commit to a new branch"},
		{"aiw git revert abc1234", "new commit that undoes abc1234"},
		{"aiw git export v2.0.0", "archive tag to v2.0.0.zip"},
		{"aiw git conflicts --check", "scan for remaining conflict markers"},
		{"aiw git rollback", "show reflog + recovery guide"},
		{"aiw git change-author \"Bob\" bob@localhost.localhost", "amend last commit's author ⚠"},
		{"aiw git change-author \"Bob\" bob@localhost.localhost --all --filter-name \"old\"", "rewrite all history ⚠⚠"},
		{"aiw git rm-from-commit secret.txt", "remove file from last commit ⚠"},
		{"aiw git rm-from-commit secret.txt --history", "scrub file from all history ⚠⚠"},
		{"aiw git rm-from-commit secret.txt --history --from abc123", "scrub from abc123..HEAD only ⚠⚠"},
		{"aiw git detach abc123", "drop history before abc123 ⚠⚠"},
		{"aiw git clean-all-histories", "squash all history + force-push ⚠⚠"},
		{"aiw git subdir-to-root trunk", "make trunk/ the new repo root ⚠⚠"},
		{"aiw git save --dry-run", "preview without executing"},
	}
	for _, e := range examples {
		fmt.Printf("  %-44s # %s\n", e[0], e[1])
	}
	fmt.Println()

	fmt.Print(footer)
}

func run(name string, args ...string) error {
	fmt.Fprintf(os.Stderr, "+ %s %s\n", name, strings.Join(args, " "))
	if dryRun {
		fmt.Fprintln(os.Stderr, "  (dry-run: skipped)")
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func gitOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func hasRemote(name string) bool {
	cmd := exec.Command("git", "remote", "get-url", name)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func refExists(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}
