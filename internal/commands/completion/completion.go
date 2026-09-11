package completion

import (
	"fmt"
	"strings"
)

var rootCommands = []string{
	"init", "new", "list", "show", "status", "done", "archive",
	"context", "decision", "spec", "requirement", "prompts", "workflow",
	"workspace", "turn", "chat", "wt", "session", "cxs", "cz", "help",
}

var taskCommands = []string{
	"new", "list", "show", "status", "done", "archive", "context",
	"decision", "spec", "workflow", "requirement", "prompts", "workspace",
	"turn", "chat", "help",
}

var workflowCommands = []string{
	"plan", "sync", "advance", "run", "supervise", "repair", "attempt",
	"evidence", "gate", "delivery", "complete", "retry-policy", "reopen",
	"force-close", "diagnose", "recover", "help",
}

var worktreeCommands = []string{
	"add", "rm", "commit", "pull", "status", "discard", "list", "prune",
	"lock", "unlock", "repair", "ignore", "help",
}

func Dispatch(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: aiw completion <powershell|bash|zsh|fish>")
	}
	switch strings.ToLower(args[0]) {
	case "powershell", "pwsh":
		printPowerShell()
	case "bash":
		printBash()
	case "zsh":
		printZsh()
	case "fish":
		printFish()
	default:
		return fmt.Errorf("unsupported completion shell: %s", args[0])
	}
	return nil
}

func quoted(values []string) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprintf("'%s'", value)
	}
	return strings.Join(parts, ", ")
}

func printPowerShell() {
	fmt.Printf(`Register-ArgumentCompleter -CommandName aiw, aiw.exe -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $tokens = @($commandAst.CommandElements | ForEach-Object { $_.Value })
    $commands = @(%s)
    $taskCommands = @(%s)
    $workflowCommands = @(%s)
    $worktreeCommands = @(%s)
    $taskIds = @()
    if (Test-Path "openspec/changes") {
        $taskIds = @(Get-ChildItem "openspec/changes" -Directory | Select-Object -ExpandProperty Name)
    }
    $candidates = @()
    if ($tokens.Count -le 1) { $candidates = $commands }
    elseif ($tokens[1] -eq "task" -and $tokens.Count -le 2) { $candidates = $taskCommands }
    elseif (($tokens[1] -eq "workflow" -or ($tokens[1] -eq "task" -and $tokens[2] -eq "workflow")) -and $tokens.Count -le 3) { $candidates = $workflowCommands }
    elseif (($tokens[1] -eq "wt") -and $tokens.Count -le 2) { $candidates = $worktreeCommands }
    elseif ($tokens[1] -in @("show", "status", "done", "archive", "context", "workflow", "turn", "chat", "wt") -or $tokens[2] -in @("show", "status", "done", "archive", "context", "run", "supervise", "delivery", "complete")) { $candidates = $taskIds }
    $candidates | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object { [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_) }
}
`, quoted(rootCommands), quoted(taskCommands), quoted(workflowCommands), quoted(worktreeCommands))
}

func printBash() {
	fmt.Printf(`_aiw_complete() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local commands=(%s)
    local task_commands=(%s)
    local workflow_commands=(%s)
    local worktree_commands=(%s)
    local task_ids=()
    if [[ -d openspec/changes ]]; then
        task_ids=($(find openspec/changes -mindepth 1 -maxdepth 1 -type d -printf '%%f\n' 2>/dev/null))
    fi
    local candidates=()
    if (( COMP_CWORD == 1 )); then candidates=("${commands[@]}")
    elif [[ "${COMP_WORDS[1]}" == task && COMP_CWORD == 2 ]]; then candidates=("${task_commands[@]}")
    elif [[ "${COMP_WORDS[1]}" == workflow && COMP_CWORD == 2 ]] || [[ "${COMP_WORDS[1]}" == task && "${COMP_WORDS[2]}" == workflow && COMP_CWORD == 3 ]]; then candidates=("${workflow_commands[@]}")
    elif [[ "${COMP_WORDS[1]}" == wt && COMP_CWORD == 2 ]]; then candidates=("${worktree_commands[@]}")
    else candidates=("${task_ids[@]}")
    fi
    COMPREPLY=($(compgen -W "${candidates[*]}" -- "$cur"))
}
complete -F _aiw_complete aiw
`, strings.Join(rootCommands, " "), strings.Join(taskCommands, " "), strings.Join(workflowCommands, " "), strings.Join(worktreeCommands, " "))
}

func printZsh() {
	fmt.Printf(`#compdef aiw
_arguments '1:command:(%s)' '*:task-id:_files -W openspec/changes'
`, strings.Join(rootCommands, " "))
}

func printFish() {
	for _, command := range rootCommands {
		fmt.Printf("complete -c aiw -f -n '__fish_use_subcommand' -a '%s'\n", command)
	}
	fmt.Println("complete -c aiw -f -n '__fish_seen_subcommand_from show status done archive context workflow turn chat' -a '(string replace -r '^openspec/changes/' '' -- openspec/changes/*)'")
}
