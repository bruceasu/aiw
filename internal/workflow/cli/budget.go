package cli

import (
	"fmt"
	"os/user"
	"strconv"
	"strings"

	"aiw/internal/ai"
	"aiw/internal/workflow"
)

// runWorkflowBudgetCommand is the operator seam for the Task-scoped Schema 10
// budget. The durable Store remains the authority; this adapter only parses
// explicit operator input and projects the committed result.
func runWorkflowBudgetCommand(id string, args []string) error {
	if len(args) == 0 {
		return budgetUsage("")
	}
	store := workflow.NewStore("")
	switch args[0] {
	case "configure":
		var budget workflow.TaskUsageBudget
		var err error
		if len(args) == 1 {
			defaults, loadErr := ai.LoadUsageBudgetDefaults()
			if loadErr != nil { return fmt.Errorf("load Task usage budget defaults: %w", loadErr) }
			budget = workflow.TaskUsageBudget{InputTokenLimit: defaults.InputTokens, OutputTokenLimit: defaults.OutputTokens}
		} else {
			budget, err = parseBudgetLimits(args[1:], false)
		}
		if err != nil {
			return err
		}
		state, err := store.ConfigureTaskUsageBudget(workflow.TaskID(id), budget)
		if err != nil {
			return fmt.Errorf("configure Task usage budget: %w", err)
		}
		fmt.Printf("Task %s usage budget configured\n", id)
		return projectWorkflowState(id, state)
	case "approve":
		actor, reason, replacement, err := parseBudgetApproval(args[1:])
		if err != nil {
			return err
		}
		state, err := store.ApproveTaskUsageBudget(workflow.TaskID(id), actor, reason, replacement)
		if err != nil {
			return fmt.Errorf("approve Task usage budget: %w", err)
		}
		fmt.Printf("Task %s usage budget approved by %s\n", id, actor)
		return projectWorkflowState(id, state)
	case "terminate":
		actor, reason, err := parseBudgetDecision(args[1:], "terminate")
		if err != nil {
			return err
		}
		state, err := store.TerminateTaskUsageBudget(workflow.TaskID(id), actor, reason)
		if err != nil {
			return fmt.Errorf("terminate Task usage budget: %w", err)
		}
		fmt.Printf("Task %s terminated by %s because the usage budget was not increased\n", id, actor)
		return projectWorkflowState(id, state)
	default:
		return fmt.Errorf("unknown budget operation: %s", args[0])
	}
}

func parseBudgetLimits(args []string, allowEmpty bool) (workflow.TaskUsageBudget, error) {
	budget := workflow.TaskUsageBudget{MonetaryLimits: make(map[string]string)}
	hasTokens, hasInput, hasOutput := false, false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--tokens", "--input-tokens", "--output-tokens":
			flag := args[i]
			if i+1 >= len(args) || (flag == "--tokens" && hasTokens) ||
				(flag == "--input-tokens" && hasInput) || (flag == "--output-tokens" && hasOutput) {
				return workflow.TaskUsageBudget{}, budgetUsage("configure")
			}
			value, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || value <= 0 {
				return workflow.TaskUsageBudget{}, fmt.Errorf("%s must be a positive whole number", flag)
			}
			switch flag {
			case "--tokens": budget.TokenLimit, hasTokens = value, true
			case "--input-tokens": budget.InputTokenLimit, hasInput = value, true
			case "--output-tokens": budget.OutputTokenLimit, hasOutput = value, true
			}
			i++
		case "--cost":
			if i+1 >= len(args) {
				return workflow.TaskUsageBudget{}, budgetUsage("configure")
			}
			currency, amount, ok := strings.Cut(args[i+1], "=")
			currency, amount = strings.TrimSpace(currency), strings.TrimSpace(amount)
			if !ok || currency == "" || amount == "" || budget.MonetaryLimits[currency] != "" {
				return workflow.TaskUsageBudget{}, fmt.Errorf("--cost must be CURRENCY=AMOUNT and each currency may appear once")
			}
			budget.MonetaryLimits[currency] = amount
			i++
		default:
			return workflow.TaskUsageBudget{}, budgetUsage("configure")
		}
	}
	if allowEmpty && !hasTokens && !hasInput && !hasOutput && len(budget.MonetaryLimits) == 0 {
		return workflow.TaskUsageBudget{}, nil
	}
	if (hasTokens && (hasInput || hasOutput)) || (hasInput != hasOutput) || (!hasTokens && !hasInput) {
		return workflow.TaskUsageBudget{}, budgetUsage("configure")
	}
	return budget, nil
}

func parseBudgetApproval(args []string) (string, string, *workflow.TaskUsageBudget, error) {
	actor, reason := "", ""
	limitArgs := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--by", "--reason":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" || strings.HasPrefix(args[i+1], "--") {
				return "", "", nil, budgetUsage("approve")
			}
			if args[i] == "--by" {
				if actor != "" { return "", "", nil, fmt.Errorf("--by may be specified only once") }
				actor = args[i+1]
			} else {
				if reason != "" { return "", "", nil, fmt.Errorf("--reason may be specified only once") }
				reason = args[i+1]
			}
			i++
		default:
			limitArgs = append(limitArgs, args[i])
		}
	}
	if reason == "" {
		return "", "", nil, budgetUsage("approve")
	}
	if actor == "" {
		var err error
		actor, err = currentBudgetActor()
		if err != nil { return "", "", nil, err }
	}
	budget, err := parseBudgetLimits(limitArgs, true)
	if err != nil { return "", "", nil, err }
	if budget.TokenLimit == 0 && budget.InputTokenLimit == 0 {
		return actor, reason, nil, nil
	}
	return actor, reason, &budget, nil
}

func parseBudgetDecision(args []string, operation string) (string, string, error) {
	actor, reason := "", ""
	for i := 0; i < len(args); i++ {
		if (args[i] != "--by" && args[i] != "--reason") || i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" || strings.HasPrefix(args[i+1], "--") {
			return "", "", budgetUsage(operation)
		}
		if args[i] == "--by" {
			if actor != "" { return "", "", fmt.Errorf("--by may be specified only once") }
			actor = args[i+1]
		} else {
			if reason != "" { return "", "", fmt.Errorf("--reason may be specified only once") }
			reason = args[i+1]
		}
		i++
	}
	if reason == "" { return "", "", budgetUsage(operation) }
	if actor == "" {
		var err error
		actor, err = currentBudgetActor()
		if err != nil { return "", "", err }
	}
	return actor, reason, nil
}

func currentBudgetActor() (string, error) {
	current, err := user.Current()
	if err != nil || strings.TrimSpace(current.Username) == "" {
		return "", fmt.Errorf("cannot determine current user; provide --by <actor>")
	}
	return current.Username, nil
}

func budgetUsage(operation string) error {
	switch operation {
	case "configure":
		return fmt.Errorf("usage: wf budget <task-id> configure (--input-tokens <positive> --output-tokens <positive> | --tokens <positive>) [--cost <CURRENCY=AMOUNT> ...]")
	case "approve":
		return fmt.Errorf("usage: wf budget <task-id> approve [--by <actor>] --reason <reason> [--input-tokens <positive> --output-tokens <positive> | --tokens <positive>] [--cost <CURRENCY=AMOUNT> ...]")
	case "terminate":
		return fmt.Errorf("usage: wf budget <task-id> terminate [--by <actor>] --reason <reason>")
	default:
		return fmt.Errorf("usage: wf budget <task-id> <configure|approve|terminate> [options]")
	}
}
