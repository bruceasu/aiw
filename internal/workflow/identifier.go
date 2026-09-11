package workflow

import (
	"fmt"
	"strconv"
	"strings"
)

const workItemPrefix = "wi-"

// NewWorkItemID creates a persistent identifier. Callers assign it once when
// planning a Work Item and keep it unchanged when the Markdown checklist moves.
func NewWorkItemID(sequence uint) WorkItemID {
	if sequence == 0 {
		return ""
	}
	return WorkItemID(fmt.Sprintf("%s%04d", workItemPrefix, sequence))
}

func ParseWorkItemID(value string) (WorkItemID, error) {
	if !strings.HasPrefix(value, workItemPrefix) {
		return "", fmt.Errorf("invalid work item id %q", value)
	}
	sequence, err := strconv.ParseUint(strings.TrimPrefix(value, workItemPrefix), 10, 0)
	if err != nil || sequence == 0 {
		return "", fmt.Errorf("invalid work item id %q", value)
	}
	return WorkItemID(value), nil
}

func workItemSequence(value WorkItemID) (uint, bool) {
	parsed, err := ParseWorkItemID(string(value))
	if err != nil {
		return 0, false
	}
	sequence, err := strconv.ParseUint(strings.TrimPrefix(string(parsed), workItemPrefix), 10, 0)
	if err != nil {
		return 0, false
	}
	return uint(sequence), true
}
