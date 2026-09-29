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
	if !fsx.Exists(task.RuntimeTaskDir(id)) && !fsx.Exists(changeDir) {
		return fmt.Errorf("task not found: %s", id)
	}
	meta, metaErr := task.ReadTaskMeta(task.ResolveTaskMetaPath(id))
	fmt.Print("Read these files first:\n\n")
	planProblem := ""
	files := []string{
		task.ResolveTaskMetaPath(id),
		filepath.Join(task.RuntimeTaskDir(id), "artifacts", "requirement-handoff.md"),
	}
	if metaErr == nil {
		planPath, err := taskworkflow.WorkflowChecklistPath(id)
		if err != nil {
			planProblem = err.Error()
		} else {
			files = append(files, planPath)
		}
	} else {
		files = append(files, task.FeatureDesignPath(id), filepath.Join(changeDir, "tasks.md"))
	}
	files = append(files,
		filepath.Join(changeDir, "proposal.md"),
		filepath.Join(changeDir, "design.md"),
		filepath.Join(changeDir, "notes.md"),
	)
	for _, f := range files {
		if fsx.Exists(f) {
			fmt.Println("-", filepath.ToSlash(f))
		}
	}
	if planProblem != "" {
		fmt.Printf("- Task plan unavailable: %s\n", planProblem)
	}
	if metaErr == nil {
		for _, spec := range meta.Specs {
			fmt.Println("-", filepath.ToSlash(filepath.Join(task.SpecsDir, spec, "spec.md")))
		}
		state, local, summaryErr := workflowStateForMeta(meta)
		if summaryErr != nil {
			fmt.Printf("\nWorkflow runtime: unavailable (%v)\n", summaryErr)
		} else {
			summary := workflow.DeriveSummary(state)
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
			printContextWorkItem(state, summary)
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

func workflowStateForMeta(meta task.TaskMeta) (workflow.RuntimeState, bool, error) {
	store := workflow.NewStore("")
	state, err := store.Load(workflow.TaskID(meta.ID))
	if err == nil {
		return state, true, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return workflow.RuntimeState{}, false, err
	}
	return taskworkflow.WorkflowRuntimeFromMeta(meta), false, nil
}

func printContextWorkItem(state workflow.RuntimeState, summary workflow.TaskSummary) {
	selected := workflow.WorkItemID("")
	label := "Current Work Item"
	if request := state.Automation.PreparedRequest; request != nil {
		selected = request.WorkItemID
	} else if summary.Active != "" {
		for _, attempt := range state.Attempts {
			if attempt.ID == summary.Active {
				selected = attempt.WorkItemID
				break
			}
		}
	}
	if selected == "" {
		label = "Next ready Work Item"
		if item, err := workflow.SelectReadyMappedWorkItem(state); err == nil {
			selected = item.ID
		}
	}
	for _, item := range state.WorkItems {
		if item.ID == selected {
			fmt.Printf("- %s: %s (checklist item %s, %s) %s\n", label, item.ID, item.Checklist.Item, item.State, item.Title)
			return
		}
	}
	fmt.Printf("- %s: none\n", label)
}

func gateIDs(ids []workflow.GateID) []string {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = string(id)
	}
	return result
}
