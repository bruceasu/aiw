package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aiw/internal/session"
	"aiw/internal/taskx"
	"aiw/internal/ui"
	"aiw/internal/workflow"
)

func runWorkflowSupervisor(args []string) (runErr error) {
	if len(args) < 3 || !safeID(args[1]) {
		return fmt.Errorf("usage: task workflow supervise <task-id> <start|status|stop> [--provider NAME] [--model MODEL]")
	}
	id, action := args[1], args[2]
	defer func() {
		if runErr != nil {
			printSupervisorFailureGuidance(id, runErr)
		}
	}()
	provider, model, err := parseProviderModelOverrides(args[3:])
	if err != nil {
		return err
	}
	if action != "start" && (provider != "" || model != "") {
		return fmt.Errorf("provider/model overrides are only valid for supervisor start")
	}
	store := workflow.NewStore("")
	state, err := store.Load(workflow.TaskID(id))
	if err != nil {
		return err
	}
	switch action {
	case "status":
		printSupervisorStatus(state)
		printSupervisorRuntimeStatus(state)
		return nil
	case "start":
		leaseID := fmt.Sprintf("supervisor-%d", time.Now().UTC().UnixNano())
		if _, err := store.StartSupervisor(workflow.TaskID(id), leaseID); err != nil {
			return err
		}
		current, err := store.Load(workflow.TaskID(id))
		if err != nil {
			return err
		}
		for _, repair := range current.Automation.ProjectionRepairs {
			if repair.ResolvedAt != "" {
				continue
			}
			if err := repairWorkflowState(id); err != nil {
				_, _ = store.RecordSupervisorObservation(workflow.TaskID(id), leaseID, current.LastEventSequence, "repair-paused", repair.Target)
				return err
			}
			current, err = store.Load(workflow.TaskID(id))
			if err != nil {
				return err
			}
			_, err = store.RecordSupervisorObservation(workflow.TaskID(id), leaseID, current.LastEventSequence, "repair-resolved", repair.Target)
			return err
		}
		for {
			if _, err := syncWorkflowChecklist(id, store); err != nil {
				return err
			}
			updated, err := store.Load(workflow.TaskID(id))
			if err != nil || updated.Automation.Supervisor.LeaseID != leaseID {
				return err
			}
			if workflow.SupervisorDue(updated, time.Now().UTC()) || updated.Automation.Supervisor.ObservedEvent == 0 {
				if _, err := store.RenewSupervisorLease(workflow.TaskID(id), leaseID); err != nil {
					return err
				}
				if err := runWorkflowWithOverrides(id, true, true, false, provider, model); err != nil {
					failed, loadErr := store.Load(workflow.TaskID(id))
					if loadErr == nil && failed.Automation.PreparedRequest != nil {
						if failed, loadErr = store.RecordAttemptOutcome(workflow.TaskID(id), failed.Automation.PreparedRequest.AttemptID, false); loadErr == nil {
							_ = projectWorkflowState(id, failed)
						}
					}
					_, _ = store.PauseSupervisor(workflow.TaskID(id), leaseID, "runner-paused", err.Error(), time.Now().UTC().Add(30*time.Second))
					return err
				}
				updated, err = store.Load(workflow.TaskID(id))
				if err != nil {
					return err
				}
				if request := updated.Automation.PreparedRequest; request != nil {
					// A model turn may outlive the one-minute supervisor lease. Renew
					// before committing its result so the completion and the next
					// outcome remain owned by this supervisor invocation.
					if _, err = store.RenewSupervisorLease(workflow.TaskID(id), leaseID); err != nil {
						return err
					}
					if err = recordSupervisorSessionEvidence(store, request); err != nil {
						_, _ = store.PauseSupervisor(workflow.TaskID(id), leaseID, "session-result-unknown", err.Error(), time.Now().UTC().Add(30*time.Second))
						return err
					}
					updated, err = store.RecordAttemptOutcome(workflow.TaskID(id), request.AttemptID, true)
					if err != nil {
						return err
					}
					if err = projectWorkflowState(id, updated); err != nil {
						return err
					}
				}
				// The execution path leaves the old automation cursor in place.
				// Re-evaluate the durable state after closing the Attempt so the
				// Supervisor records the actual next outcome instead of repeating
				// agent-request-prepared forever.
				if err = runWorkflowWithOptions(id, false, true, false); err != nil {
					return err
				}
				updated, err = store.Load(workflow.TaskID(id))
				if err != nil {
					return err
				}
				// A fresh prepared request is external work for the next
				// supervisor iteration. Do not advance the observation cursor over
				// it, or the request would remain prepared forever.
				if updated.Automation.Cursor.Result == string(workflow.RunnerPrepared) {
					continue
				}
				if _, err = store.RecordSupervisorObservation(workflow.TaskID(id), leaseID, updated.LastEventSequence, updated.Automation.Cursor.Result, updated.Automation.Cursor.Detail); err != nil {
					return err
				}
				switch updated.Automation.Cursor.Result {
				case string(workflow.RunnerGate), string(workflow.RunnerNoWork), "repair-required", "blocked":
					_, err = store.PauseSupervisor(workflow.TaskID(id), leaseID, "supervisor-paused", updated.Automation.Cursor.Result, time.Now().UTC().Add(30*time.Second))
					return err
				}
			}
			time.Sleep(2 * time.Second)
		}
	case "stop":
		if state.Automation.Supervisor.LeaseID == "" {
			return nil
		}
		_, err := store.StopSupervisor(workflow.TaskID(id), state.Automation.Supervisor.LeaseID)
		return err
	default:
		return fmt.Errorf("usage: task workflow supervise <task-id> <start|status|stop>")
	}
}

func parseProviderModelOverrides(args []string) (string, string, error) {
	provider, model := "", ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "--model":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				return "", "", fmt.Errorf("%s requires a value", args[i])
			}
			if args[i] == "--provider" {
				provider = args[i+1]
			} else {
				model = args[i+1]
			}
			i++
		default:
			return "", "", fmt.Errorf("unknown supervisor option: %s", args[i])
		}
	}
	return provider, model, nil
}

func printSupervisorFailureGuidance(id string, cause error) {
	fmt.Fprintf(os.Stderr, "supervisor: stopped with an error for task=%s\n", id)
	fmt.Fprintf(os.Stderr, "next: aiw task workflow diagnose %s\n", id)
	store := workflow.NewStore("")
	diagnostics, err := store.Diagnose(workflow.TaskID(id))
	if err != nil {
		fmt.Fprintf(os.Stderr, "diagnosis unavailable: %v\n", err)
	} else {
		for _, diagnostic := range diagnostics {
			fmt.Fprintf(os.Stderr, "detected %s: %s\nnext: %s\n", diagnostic.Code, diagnostic.Message, diagnostic.Repair)
		}
	}
	if meta, metaErr := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(id)); metaErr == nil && resolvedWorkspaceKind(meta) == "isolated" && !verifiedTaskWorktree(meta) {
		fmt.Fprintf(os.Stderr, "detected workspace-binding-invalid: the isolated worktree is missing or unregistered\nnext: aiw wt repair\n")
	}
	message := strings.ToLower(cause.Error())
	switch {
	case strings.Contains(message, "rename") || strings.Contains(message, "access is denied"):
		fmt.Fprintf(os.Stderr, "file access failure: wait for another process to release the state file; if diagnose reports event-pending, run: aiw task workflow recover %s\n", id)
	case strings.Contains(message, "invalid isolated worktree") || strings.Contains(message, "worktree"):
		fmt.Fprintln(os.Stderr, "workspace failure: inspect the worktree binding; repair it before retrying the supervisor")
	case strings.Contains(message, "session"):
		fmt.Fprintln(os.Stderr, "session failure: inspect the recorded Session output before retrying; preserve the Attempt so its result remains auditable")
	case strings.Contains(message, "gate"):
		fmt.Fprintln(os.Stderr, "gate failure: resolve or waive only the reported Gate after the required human decision")
	}
	fmt.Fprintf(os.Stderr, "then inspect: aiw task workflow supervise %s status\n", id)
	fmt.Fprintln(os.Stderr, "do not start another supervisor until diagnosis or recovery completes")
}

func printSupervisorStatus(state workflow.RuntimeState) {
	terminal := ui.NewTerminal(os.Stdout)
	supervisor := state.Automation.Supervisor
	status := "stopped"
	if supervisor.LeaseID != "" {
		status = "lease-valid"
		if supervisor.LeaseExpiresAt != "" {
			expires, err := time.Parse(time.RFC3339, supervisor.LeaseExpiresAt)
			if err == nil && !time.Now().UTC().Before(expires) {
				status = "lease-expired"
			}
		}
	}
	terminal.Section("Supervisor")
	terminal.State("snapshot", status)
	terminal.Field("lease", supervisor.LeaseID)
	terminal.Field("expires", supervisor.LeaseExpiresAt)
	terminal.Field("observed event", fmt.Sprintf("%d", supervisor.ObservedEvent))
	if supervisor.Result != "" {
		terminal.Section("Last result")
		terminal.Field("result", supervisor.Result)
		terminal.Field("detail", supervisor.Detail)
	}
	if supervisor.RetryAfter != "" {
		retryStatus := "due"
		if retryAt, err := time.Parse(time.RFC3339, supervisor.RetryAfter); err == nil && time.Now().UTC().Before(retryAt) {
			retryStatus = "waiting"
		}
		terminal.Section("Retry")
		terminal.State("status", retryStatus)
		terminal.Field("due at", supervisor.RetryAfter)
	}
}

func printSupervisorRuntimeStatus(state workflow.RuntimeState) {
	terminal := ui.NewTerminal(os.Stdout)
	if request := state.Automation.PreparedRequest; request != nil {
		terminal.Section("Agent request")
		terminal.State("state", "prepared")
		terminal.Field("work item", string(request.WorkItemID))
		terminal.Field("attempt", string(request.AttemptID))
		terminal.Field("session", request.SessionID)
		terminal.Field("workspace", request.Workspace)
		if request.SessionID != "" {
			terminal.Field("session output", fmt.Sprintf(".ai/sessions/%s/outputs", request.SessionID))
		}
	}
	if state.WriteLease != nil {
		terminal.Section("Workspace")
		terminal.State("write lease", "held")
		terminal.Field("attempt", string(state.WriteLease.AttemptID))
		terminal.Field("acquired", state.WriteLease.AcquiredAt)
		terminal.Field("path", state.WriteLease.Workspace)
	}
	for _, gate := range state.Gates {
		if gate.State == workflow.GateOpen {
			terminal.Section("Open gate")
			terminal.Field("id", string(gate.ID))
			terminal.Field("kind", string(gate.Kind))
			terminal.Field("work item", string(gate.WorkItemID))
			terminal.Field("reason", gate.Reason)
		}
	}
	for _, repair := range state.Automation.ProjectionRepairs {
		if repair.ResolvedAt == "" {
			terminal.Section("Projection repair")
			terminal.Field("event", fmt.Sprintf("%d", repair.EventSequence))
			terminal.Field("target", repair.Target)
			terminal.Field("next", repair.Recommended)
		}
	}
}

// runSupervisedWorkflow keeps the foreground CLI informative while the
// external Agent CLI is running. The Agent output remains persisted in the
// Session output files; these messages are only lifecycle heartbeats.
func runSupervisedWorkflow(id, leaseID string) error {
	terminal := ui.NewTerminal(os.Stdout)
	started := time.Now()
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- runWorkflowWithOptions(id, true, true, false)
	}()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	terminal.Line(fmt.Sprintf("supervisor: processing task=%s work_item=agent-turn", id))
	for {
		select {
		case err := <-resultCh:
			if err != nil {
				terminal.Line(fmt.Sprintf("supervisor: processing failed task=%s elapsed=%s", id, time.Since(started).Round(time.Second)))
				return err
			}
			terminal.Line(fmt.Sprintf("supervisor: processing completed task=%s elapsed=%s", id, time.Since(started).Round(time.Second)))
			return nil
		case <-ticker.C:
			state, err := workflow.NewStore("").Load(workflow.TaskID(id))
			if err != nil {
				terminal.Line(fmt.Sprintf("supervisor: processing task=%s elapsed=%s state=unavailable", id, time.Since(started).Round(time.Second)))
				continue
			}
			if _, err := workflow.NewStore("").RenewSupervisorLease(workflow.TaskID(id), leaseID); err != nil {
				terminal.Line(fmt.Sprintf("supervisor: processing task=%s elapsed=%s lease=lost", id, time.Since(started).Round(time.Second)))
				continue
			}
			workItem := state.Automation.Cursor.Detail
			if workItem == "" {
				workItem = "pending"
			}
			terminal.Line(fmt.Sprintf("supervisor: processing task=%s work_item=%s elapsed=%s lease=renewed%s", id, workItem, time.Since(started).Round(time.Second), supervisorLiveProgress(state)))
		}
	}
}

func supervisorLiveProgress(state workflow.RuntimeState) string {
	request := state.Automation.PreparedRequest
	if request == nil || request.SessionID == "" {
		return ""
	}
	status, err := session.NewStore("").Load(request.SessionID)
	if err != nil {
		return " agent=unavailable"
	}
	path := filepath.Join(".ai", "sessions", request.SessionID, "outputs", fmt.Sprintf("%04d-live.jsonl", status.Session.LastTurn+1))
	content, err := os.ReadFile(path)
	if err != nil || len(content) == 0 {
		return " agent=starting"
	}
	for index := len(strings.Split(string(content), "\n")) - 1; index >= 0; index-- {
		line := strings.TrimSpace(strings.Split(string(content), "\n")[index])
		if line == "" {
			continue
		}
		var event struct {
			Type string `json:"type"`
			Item struct {
				Type string `json:"type"`
			} `json:"item"`
		}
		if json.Unmarshal([]byte(line), &event) == nil {
			if event.Item.Type != "" {
				return fmt.Sprintf(" agent_event=%s", event.Item.Type)
			}
			if event.Type != "" {
				return fmt.Sprintf(" agent_event=%s", event.Type)
			}
		}
		return " agent=active"
	}
	return " agent=active"
}

func recordSupervisorSessionEvidence(store *workflow.Store, request *workflow.PreparedAgentRequest) error {
	if request == nil || request.SessionID == "" {
		return fmt.Errorf("managed Session binding is required")
	}
	status, err := session.NewStore("").Load(request.SessionID)
	if err != nil {
		return err
	}
	if status.Result.Status != "completed" || status.Result.FinalOutputFile == "" {
		return fmt.Errorf("managed Session result is incomplete")
	}
	evidenceID := workflow.EvidenceID("agent-output-" + string(request.AttemptID))
	state, err := store.Load(request.TaskID)
	if err != nil {
		return err
	}
	for _, existing := range state.Evidence {
		if existing.ID == evidenceID {
			return nil
		}
	}
	_, err = store.RecordEvidence(request.TaskID, workflow.Evidence{
		ID:         evidenceID,
		WorkItemID: request.WorkItemID,
		Kind:       workflow.EvidenceManual,
		State:      workflow.EvidencePassed,
		Reference:  ".ai/sessions/" + request.SessionID + "/" + status.Result.FinalOutputFile,
	})
	return err
}
