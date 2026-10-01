package completion

import (
	"fmt"
	"strings"
)

var rootCommands = []string{
	"init", "ask", "completion", "help", "version", "issue", "fd", "wt",
}

var fdCommands = []string{
	"new", "list", "show", "emit", "claim", "resume", "request-review", "refresh-worker", "reopen", "close", "worktree",
}

var fdWorktreeCommands = []string{"add", "status"}
var fdOutcomes = []string{"Complete", "Deferred", "Closed"}

var fdIDCommands = []string{
	"show", "emit", "claim", "resume", "request-review", "refresh-worker", "reopen", "close", "worktree",
}

var worktreeCommands = []string{
	"add", "rm", "commit", "pull", "status", "discard", "list", "prune",
	"lock", "unlock", "repair", "ignore",
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

func render(template string, values ...string) string {
	replacements := make([]string, 0, len(values))
	for i := 0; i < len(values); i += 2 {
		replacements = append(replacements, values[i], values[i+1])
	}
	return strings.NewReplacer(replacements...).Replace(template)
}

func printPowerShell() {
	const script = `Register-ArgumentCompleter -CommandName aiw, aiw.exe -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $aiwArgs = @($commandAst.CommandElements | Select-Object -Skip 1 | ForEach-Object { $_.Value })
    if ($wordToComplete -and $aiwArgs.Count -gt 0) {
        if ($aiwArgs.Count -eq 1) { $aiwArgs = @() }
        else { $aiwArgs = @($aiwArgs[0..($aiwArgs.Count - 2)]) }
    }
    $rootCommands = @(__ROOT_COMMANDS__)
    $fdCommands = @(__FD_COMMANDS__)
    $fdWorktreeCommands = @(__FD_WORKTREE_COMMANDS__)
    $fdOutcomes = @(__FD_OUTCOMES__)
    $fdIdCommands = @(__FD_ID_COMMANDS__)
    $worktreeCommands = @(__WORKTREE_COMMANDS__)
    $fdIds = @()
    foreach ($folder in @("docs/features", "docs/features/archive")) {
        if (Test-Path -LiteralPath $folder) {
            $fdIds += Get-ChildItem -LiteralPath $folder -Filter "FD-*.md" -File |
                ForEach-Object { if ($_.BaseName -match "^(FD-\d+)") { $Matches[1] } }
        }
    }
    $fdIds = @($fdIds | Sort-Object -Unique)
    $candidates = @()

    if ($aiwArgs.Count -eq 0) { $candidates = $rootCommands }
    elseif ($aiwArgs[0] -eq "fd") {
        if ($aiwArgs.Count -eq 1) { $candidates = $fdCommands }
        elseif ($aiwArgs[1] -eq "worktree") {
            if ($aiwArgs.Count -eq 2) { $candidates = $fdWorktreeCommands }
            elseif ($aiwArgs.Count -eq 3) { $candidates = $fdIds }
        }
        elseif ($aiwArgs[1] -in $fdIdCommands) {
            if ($aiwArgs.Count -eq 2) { $candidates = $fdIds }
            elseif ($aiwArgs[1] -eq "close" -and $aiwArgs.Count -eq 3) { $candidates = $fdOutcomes }
        }
    }
    elseif ($aiwArgs[0] -eq "wt") {
        if ($aiwArgs.Count -eq 1) { $candidates = $worktreeCommands }
    }

    $candidates | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
    }
}
`
	fmt.Print(render(script,
		"__ROOT_COMMANDS__", quoted(rootCommands),
		"__FD_COMMANDS__", quoted(fdCommands),
		"__FD_WORKTREE_COMMANDS__", quoted(fdWorktreeCommands),
		"__FD_OUTCOMES__", quoted(fdOutcomes),
		"__FD_ID_COMMANDS__", quoted(fdIDCommands),
		"__WORKTREE_COMMANDS__", quoted(worktreeCommands),
	))
}

func printBash() {
	const script = `_aiw_fd_ids() {
    local path base id
    for path in docs/features/FD-*.md docs/features/archive/FD-*.md; do
        [[ -f "$path" ]] || continue
        base=${path##*/}
        base=${base%.md}
        id=${base%%_*}
        [[ "$id" =~ ^FD-[0-9]+$ ]] && printf '%s\n' "$id"
    done
}

_aiw_complete() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local id
    local -a args candidates fd_ids
    local -a fd_id_commands=(__FD_ID_COMMANDS__)
    args=("${COMP_WORDS[@]:1:COMP_CWORD-1}")
    while IFS= read -r id; do fd_ids+=("$id"); done < <(_aiw_fd_ids)

    if (( ${#args[@]} == 0 )); then candidates=(__ROOT_COMMANDS__)
    elif [[ "${args[0]}" == fd ]]; then
        if (( ${#args[@]} == 1 )); then candidates=(__FD_COMMANDS__)
        elif [[ "${args[1]}" == worktree ]]; then
            if (( ${#args[@]} == 2 )); then candidates=(__FD_WORKTREE_COMMANDS__)
            elif (( ${#args[@]} == 3 )); then candidates=("${fd_ids[@]}")
            fi
        elif [[ " ${fd_id_commands[*]} " == *" ${args[1]} "* ]]; then
            if (( ${#args[@]} == 2 )); then candidates=("${fd_ids[@]}")
            elif [[ "${args[1]}" == close ]] && (( ${#args[@]} == 3 )); then candidates=(__FD_OUTCOMES__)
            fi
        fi
    elif [[ "${args[0]}" == wt ]]; then
        if (( ${#args[@]} == 1 )); then candidates=(__WORKTREE_COMMANDS__)
        fi
    fi

    COMPREPLY=($(compgen -W "${candidates[*]}" -- "$cur"))
}
complete -F _aiw_complete aiw
`
	fmt.Print(render(script,
		"__ROOT_COMMANDS__", strings.Join(rootCommands, " "),
		"__FD_COMMANDS__", strings.Join(fdCommands, " "),
		"__FD_WORKTREE_COMMANDS__", strings.Join(fdWorktreeCommands, " "),
		"__FD_OUTCOMES__", strings.Join(fdOutcomes, " "),
		"__FD_ID_COMMANDS__", strings.Join(fdIDCommands, " "),
		"__WORKTREE_COMMANDS__", strings.Join(worktreeCommands, " "),
	))
}

func printZsh() {
	const script = `#compdef aiw
_aiw_fd_ids() {
    local path base
    for path in docs/features/FD-*.md(N) docs/features/archive/FD-*.md(N); do
        base=${path:t:r}
        print -r -- ${base%%_*}
    done
}

_aiw_complete() {
    local -a fd_ids
    fd_ids=(${(f)"$(_aiw_fd_ids)"})
    if (( CURRENT == 2 )); then
        compadd -- __ROOT_COMMANDS__
    elif [[ $words[2] == fd ]] && (( CURRENT == 3 )); then
        compadd -- __FD_COMMANDS__
    elif [[ $words[2] == fd && $words[3] == worktree ]] && (( CURRENT == 4 )); then
        compadd -- __FD_WORKTREE_COMMANDS__
    elif [[ $words[2] == fd && $words[3] == worktree ]] && (( CURRENT == 5 )); then
        compadd -- "${fd_ids[@]}"
    elif [[ $words[2] == fd && " __FD_ID_COMMANDS__ " == *" $words[3] "* ]] && (( CURRENT == 4 )); then
        compadd -- "${fd_ids[@]}"
    elif [[ $words[2] == fd && $words[3] == close ]] && (( CURRENT == 5 )); then
        compadd -- __FD_OUTCOMES__
    elif [[ $words[2] == wt ]] && (( CURRENT == 3 )); then
        compadd -- __WORKTREE_COMMANDS__
    fi
}
compdef _aiw_complete aiw
`
	fmt.Print(render(script,
		"__ROOT_COMMANDS__", strings.Join(rootCommands, " "),
		"__FD_COMMANDS__", strings.Join(fdCommands, " "),
		"__FD_WORKTREE_COMMANDS__", strings.Join(fdWorktreeCommands, " "),
		"__FD_OUTCOMES__", strings.Join(fdOutcomes, " "),
		"__FD_ID_COMMANDS__", strings.Join(fdIDCommands, " "),
		"__WORKTREE_COMMANDS__", strings.Join(worktreeCommands, " "),
	))
}

func printFish() {
	const script = `function __aiw_fd_ids
    for file in docs/features/FD-*.md docs/features/archive/FD-*.md
        if test -f "$file"
            set base (basename "$file" .md)
            string match -r -g '^FD-[0-9]+' "$base"
        end
    end
end

function __aiw_fd_command_position
    set tokens (commandline -opc)
    test (count $tokens) -eq 2; and test "$tokens[2]" = fd
end

function __aiw_fd_worktree_position
    set tokens (commandline -opc)
    test (count $tokens) -eq 3; and test "$tokens[2]" = fd; and test "$tokens[3]" = worktree
end

function __aiw_fd_id_position
    set tokens (commandline -opc)
    if test (count $tokens) -eq 3; and test "$tokens[2]" = fd
        contains -- "$tokens[3]" __FD_ID_COMMANDS__
    else if test (count $tokens) -eq 4; and test "$tokens[2]" = fd; and test "$tokens[3]" = worktree
        contains -- "$tokens[4]" __FD_WORKTREE_COMMANDS__
    end
end

complete -c aiw -f -n '__fish_use_subcommand' -a '__ROOT_COMMANDS__'
complete -c aiw -f -n '__aiw_fd_command_position' -a '__FD_COMMANDS__'
complete -c aiw -f -n '__aiw_fd_worktree_position' -a '__FD_WORKTREE_COMMANDS__'
complete -c aiw -f -n '__aiw_fd_id_position' -a '(__aiw_fd_ids)'
complete -c aiw -f -n '__fish_seen_subcommand_from wt; and test (count (commandline -opc)) -eq 2' -a '__WORKTREE_COMMANDS__'
`
	fmt.Print(render(script,
		"__ROOT_COMMANDS__", strings.Join(rootCommands, " "),
		"__FD_COMMANDS__", strings.Join(fdCommands, " "),
		"__FD_WORKTREE_COMMANDS__", strings.Join(fdWorktreeCommands, " "),
		"__FD_ID_COMMANDS__", strings.Join(fdIDCommands, " "),
		"__WORKTREE_COMMANDS__", strings.Join(worktreeCommands, " "),
	))
}
