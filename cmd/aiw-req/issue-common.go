package main

import (
	"aiw/internal/issue"
	"fmt"
	"os/user"
	"strings"
)

type issueChatPlan struct {
	RequirementID string
	SessionID     string
	Phase         string
	Provider      string
	Model         string
}

type issuePendingAction struct {
	AutoNumber          bool     `json:"auto_number,omitempty"`
	Facts               []string `json:"facts,omitempty"`
	Kind                string   `json:"kind"`
	RequirementID       string   `json:"requirement_id"`
	Title               string   `json:"title,omitempty"`
	Artifact            string   `json:"artifact,omitempty"`
	Source              string   `json:"source,omitempty"`
	Decision            string   `json:"decision,omitempty"`
	By                  string   `json:"by,omitempty"`
	Reason              string   `json:"reason,omitempty"`
}

// approvalCheckpoint binds advice to the exact displayed action and evidence.
// It is checked again before chat approval; direct approve CLI is unchanged.
type approvalCheckpoint struct {
	Action        string
	HistoryDigest string
	FactsDigest   string
}

type captureCheckpoint struct {
	Revision   int
	Action     string
	Digest     string
	References []issue.CoverageReference
}

func parseTerminalArgs(args []string, command string) (string, string, string, error) {
	if len(args) < 3 || !issue.ValidID(args[0]) {
		return "", "", "", fmt.Errorf("usage: aiw req %s <id> [--by <actor>] --reason <reason>", command)
	}
	by, reason := "", ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--by":
			if i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "--") {
				return "", "", "", fmt.Errorf("usage: aiw req %s <id> [--by <actor>] --reason <reason>", command)
			}
			by, i = args[i+1], i+1
		case "--reason":
			if i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "--") {
				return "", "", "", fmt.Errorf("usage: aiw req %s <id> [--by <actor>] --reason <reason>", command)
			}
			reason, i = args[i+1], i+1
		default:
			return "", "", "", fmt.Errorf("usage: aiw req %s <id> [--by <actor>] --reason <reason>", command)
		}
	}
	if reason == "" {
		return "", "", "", fmt.Errorf("usage: aiw req %s <id> [--by <actor>] --reason <reason>", command)
	}
	if by == "" {
		current, err := user.Current()
		if err != nil || current.Username == "" {
			return "", "", "", fmt.Errorf("cannot determine current user; provide --by <actor>")
		}
		by = current.Username
	}
	return args[0], by, reason, nil
}
