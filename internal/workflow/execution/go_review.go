package execution

import (
	"fmt"
	"strings"

	"aiw/internal/workflow"
)

func goReviewRequired(store *workflow.Store, request *workflow.PreparedAgentRequest) (bool, error) {
	if request == nil { return false, fmt.Errorf("Go review requires a prepared request") }
	state, err := store.Load(request.TaskID)
	if err != nil { return false, err }
	for _, item := range state.WorkItems {
		if item.ID != request.WorkItemID { continue }
		if item.Verification == "go" || item.Verification == "mixed" { return true, nil }
		if request.Compile != nil && request.Compile.Request != nil {
			for _, changed := range request.Compile.Request.ChangedPaths {
				lower := strings.ToLower(changed)
				if strings.HasSuffix(lower, ".go") && !strings.HasSuffix(lower, "_test.go") { return true, nil }
			}
		}
		return false, nil
	}
	return false, fmt.Errorf("Go review work item %s is missing", request.WorkItemID)
}
