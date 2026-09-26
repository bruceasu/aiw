package workflowadapter

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"

	"aiw/internal/session"
	"aiw/internal/task"
	"aiw/internal/workflow"
)

// prepareFrozenAgentContext stages E01 behind the schema-10 migration owned
// by E02. Do not enable half of the new acceptance/budget protocol on schema 9.
func prepareFrozenAgentContext(id, handoff, prompt string, sessions *session.Store, status session.Status) (*session.FrozenTurn, error) {
	store := workflow.NewStore("")
	state, err := store.Load(workflow.TaskID(id))
	if err != nil { return nil, err }
	if state.SchemaVersion < 10 { return nil, nil }
	r := state.Automation.PreparedRequest
	if r == nil || r.SessionID != status.Session.ID || r.ExpectedSessionTurn != status.Session.LastTurn+1 { return nil, fmt.Errorf("fixed context requires the exact prepared Session turn") }
	if r.DispatchedAt != "" { return nil, fmt.Errorf("the fixed request is already dispatched; reconcile its result") }
	var input workflow.ExecutionInput
	if r.InputReference != nil {
		if err := store.ReadExecutionArtifact(r.TaskID, *r.InputReference, &input); err != nil { return nil, err }
		if input.RequestID != workflow.ExecutionRequestID(*r) || input.SchemaVersion != 1 { return nil, fmt.Errorf("frozen input identity is not applicable") }
	} else {
		root, err := task.ExecutionWorkspace(r.Workspace)
		if err != nil { return nil, err }
		var sources []workflow.InputSource
		var instructions, memory string
		if r.ReportOrigin == nil {
			sources, err = task.LoadExecutionSources(id, root, handoff)
			if err != nil { return nil, err }
			dependencies, err := task.ValidateDependencyInputs(state, r.WorkItemID, root)
			if err != nil { return nil, err }
			sources = append(sources, dependencies...)
			instructions, err = sessions.ReadText(r.SessionID, status.Instructions.File)
			if err != nil { return nil, err }
			memory, err = sessions.ReadText(r.SessionID, status.Instructions.MemoryFile)
			if err != nil { return nil, err }
			taskMemory, sourceVersion, err := store.TaskMemoryContext(r.TaskID)
			if err != nil { return nil, err }
			// Keep user-authored Session text intact; only the frozen turn gains
			// a versioned Task projection reconstructed from original facts.
			memory += "\n\n"+taskMemory
			taskSum := sha256.Sum256([]byte(taskMemory))
			sources = append(sources, workflow.InputSource{Kind: "task-memory", Path: "task-source:"+sourceVersion, Required: true, SHA256: hex.EncodeToString(taskSum[:]), Status: "loaded", Content: taskMemory})
			for _, value := range []struct { kind, path, content string }{{"instructions", status.Instructions.File, instructions}, {"memory", status.Instructions.MemoryFile, memory}} {
				sum := sha256.Sum256([]byte(value.content))
				sources = append(sources, workflow.InputSource{Kind: value.kind, Path: value.path, Required: true, SHA256: hex.EncodeToString(sum[:]), Status: "loaded", Content: value.content})
			}
		}
		origin := r
		if r.ReportOrigin != nil {
			origin = r.ReportOrigin
			if origin.InputReference == nil { return nil, fmt.Errorf("original input version is unknown; manual report reconciliation required") }
			var originalInput workflow.ExecutionInput
			if err := store.ReadExecutionArtifact(r.TaskID, *origin.InputReference, &originalInput); err != nil { return nil, err }
			sources = append([]workflow.InputSource(nil), originalInput.Sources...)
			for _, source := range sources {
				if source.Kind == "instructions" { instructions = source.Content }
				if source.Kind == "memory" { memory = source.Content }
			}
			originalReport, err := store.ReadExecutionReport(*origin)
			if err != nil { return nil, err }
			sources = append(sources, workflow.InputSource{Kind: "original-response", Path: "reports/executions/"+workflow.ExecutionRequestID(*origin)+".json", Required: true, Status: "loaded", Content: originalReport.Output, SHA256: originalReport.OutputSHA256})
			prompt = "Write the missing report for the original generation below. This turn is read-only. Do not edit files, run tests or compilers, or repeat implementation work. Use unknown or unverified with a reason when the original evidence is unavailable."
		}
		prompt += task.RenderExecutionSources(sources)
		inputPath := filepath.ToSlash(filepath.Join(task.RuntimeTaskDir(id), "reports", "inputs", workflow.ExecutionRequestID(*origin)+".json"))
		prompt += fmt.Sprintf("\n\n[Implementation report]\nReturn one JSON object with outcome, detail, blocked_category when blocked, and report. The report must contain schema_version=1, request_id=%q, task_id=%q, work_item_id=%q, attempt_id=%q, session_id=%q, turn=%d, actor=coder, and input_sha256 (SHA256 of the saved input file at %s). Include coverage, interfaces, side_effects, test_entrypoints, decisions, limitations, risks, validation, and history. Each section must be an object with state (known, empty, unknown, or unverified), items (an explicit string array), and reason. Known requires facts; empty, unknown, and unverified require a reason. Include changes and references as explicit arrays of {kind,path,sha256}. Changed files use kind=content and their current SHA256. Deleted files use kind=deleted without a digest. References use workspace-relative paths and exact content digests. Do not claim that your own statement is controlled validation evidence. Keep prior generation identities in history. Do not invent interfaces or facts.\n", workflow.ExecutionRequestID(*origin), origin.TaskID, origin.WorkItemID, origin.AttemptID, origin.SessionID, origin.ExpectedSessionTurn, inputPath)
		input = workflow.ExecutionInput{SchemaVersion: 1, RequestID: workflow.ExecutionRequestID(*r), TaskID: r.TaskID, WorkItemID: r.WorkItemID, AttemptID: r.AttemptID, SessionID: r.SessionID, Turn: r.ExpectedSessionTurn, Actor: workflow.ActorCoder, Workspace: r.Workspace, AISelection: r.AISelection, AllowedPaths: []string{"."}, Sources: sources, Prompt: session.ComposePrompt(instructions, memory, "handoff", prompt)}
		if r.ReportOrigin == nil {
			var knowledge []workflow.InputSource
			input.Prompt, knowledge, err = store.InjectKnowledge(r.TaskID, root, input.Prompt, r.AISelection)
			if err != nil { return nil, err }
			input.Sources = append(input.Sources, knowledge...)
		}
		baseline, err := workflow.CaptureValidationInputs(root, []string{"."}, nil, "generation-content")
		if err != nil { return nil, err }
		input.WorkspaceInputs = &baseline
		if r.ReportOrigin != nil { input.AllowedPaths = []string{"read-only"} }
		ref, err := store.PersistExecutionInput(*r, input)
		if err != nil { return nil, err }
		r.InputReference = &ref
		if _, err := store.RecordAutomation(r.TaskID, state.Automation.PlanFingerprint, state.Automation.Cursor, r); err != nil { return nil, err }
	}
	if err := store.ValidateFrozenTaskSources(input); err != nil { return nil, err }
	frozen := &session.FrozenTurn{Prompt: input.Prompt, ExpectedTurn: r.ExpectedSessionTurn, ReadOnly: r.ReportOrigin != nil}
	for _, source := range input.Sources {
		if source.Kind == "instructions" { frozen.Instructions = source.Content }
		if source.Kind == "memory" { frozen.Memory = source.Content }
	}
	return frozen, nil
}
