package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/fsx"
	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
	"aiw/internal/workflow"
)

func printContext(id string) error {
	changeDir := task.TaskDir(id)
	if !fsx.Exists(changeDir) {
		return fmt.Errorf("task not found: %s", id)
	}
	fmt.Print("Read these files first:\n\n")
	files := []string{
		task.TaskMetaPath(id),
		filepath.Join(changeDir, "proposal.md"),
		filepath.Join(changeDir, "tasks.md"),
		filepath.Join(changeDir, "design.md"),
		filepath.Join(changeDir, "notes.md"),
	}
	for _, f := range files {
		if fsx.Exists(f) {
			fmt.Println("-", filepath.ToSlash(f))
		}
	}
	meta, err := task.ReadTaskMeta(task.ResolveTaskMetaPath(id))
	if err == nil {
		for _, spec := range meta.Specs {
			fmt.Println("-", filepath.ToSlash(filepath.Join(task.SpecsDir, spec, "spec.md")))
		}
		summary, local, summaryErr := workflowSummaryForMeta(meta)
		if summaryErr != nil {
			fmt.Printf("\nWorkflow runtime: unavailable (%v)\n", summaryErr)
		} else {
			source := "durable metadata compatibility projection"
			if local {
				source = "local runtime projection"
			}
			fmt.Printf("\nWorkflow (%s):\n", source)
			fmt.Printf("- Status: %s\n", summary.Status)
			fmt.Printf("- Planning: %s; execution: %s; validation: %s; delivery: %s\n", summary.Planning, summary.Execution, summary.Validation, summary.Delivery)
			if len(summary.BlockedBy) > 0 {
				fmt.Printf("- Blocking gates: %s\n", strings.Join(gateIDs(summary.BlockedBy), ", "))
			}
		}
	}
	fmt.Print(`
Instruction:
- implement only the scoped task
- avoid unrelated refactors
- preserve backward compatibility
- update TODO and Verification before finishing
- use %% notes instead of guessing
`)
	return nil
}

func workflowSummaryForMeta(meta task.TaskMeta) (workflow.TaskSummary, bool, error) {
	store := workflow.NewStore("")
	state, err := store.Load(workflow.TaskID(meta.ID))
	if err == nil {
		return workflow.DeriveSummary(state), true, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return workflow.TaskSummary{}, false, err
	}
	compatible := taskworkflow.WorkflowRuntimeFromMeta(meta)
	return workflow.DeriveSummary(compatible), false, nil
}

func gateIDs(ids []workflow.GateID) []string {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = string(id)
	}
	return result
}
