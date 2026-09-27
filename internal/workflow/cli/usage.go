package cli

import (
	"fmt"
	"time"

	"aiw/internal/workflow"
)

func parseUsageReportArgs(args []string) (workflow.UsageReportQuery, string, error) {
	query := workflow.UsageReportQuery{}
	format := "table"
	seen := make(map[string]bool)
	for i := 0; i < len(args); i++ {
		flag := args[i]
		if seen[flag] {
			return query, "", fmt.Errorf("usage option %s may be specified only once", flag)
		}
		seen[flag] = true
		if i+1 >= len(args) || args[i+1] == "" || args[i+1][0] == '-' {
			return query, "", fmt.Errorf("usage option %s requires a value", flag)
		}
		value := args[i+1]
		i++
		switch flag {
		case "--work-item":
			query.WorkItemID = workflow.WorkItemID(value)
		case "--attempt":
			query.AttemptID = workflow.AttemptID(value)
		case "--provider":
			query.Provider = value
		case "--profile":
			query.Profile = value
		case "--from", "--to":
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return query, "", fmt.Errorf("%s must be an RFC3339 timestamp: %w", flag, err)
			}
			if flag == "--from" {
				query.From = &parsed
			} else {
				query.To = &parsed
			}
		case "--format":
			if value != "table" && value != "json" {
				return query, "", fmt.Errorf("usage format must be table or json")
			}
			format = value
		default:
			return query, "", fmt.Errorf("unknown usage option: %s", flag)
		}
	}
	if query.From != nil && query.To != nil && query.From.After(*query.To) {
		return query, "", fmt.Errorf("--from must not be after --to")
	}
	return query, format, nil
}
