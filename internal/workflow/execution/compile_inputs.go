package execution

import (
	"fmt"

	"aiw/internal/task"
	"aiw/internal/workflow"
)

func captureCompileInputs(request *workflow.PreparedAgentRequest) (workflow.ValidationInputs, error) {
	root, err := task.ExecutionWorkspace(request.Workspace)
	if err != nil { return workflow.ValidationInputs{}, err }
	if request.Compile == nil || request.Compile.Plan == nil { return workflow.ValidationInputs{}, fmt.Errorf("compile input plan is missing") }
	bindings, err := workflow.CompileInputBindings(request.Compile.Plan)
	if err != nil { return workflow.ValidationInputs{}, err }
	// Read binary identity without executing version probes. Script targets
	// retain the script itself in the content manifest. Environment changes
	// that affect compiler selection invalidate the previous run too.
	toolchain, err := workflow.ValidationToolchainIdentity()
	if err != nil { return workflow.ValidationInputs{}, err }
	for _, target := range request.Compile.Plan.Targets {
		if target.Kind == "script" {
			// A script can select additional tools or external environments.
			// E03 must supply that adapter's complete environment contract;
			// the generic executable fingerprint is not enough to prove reuse.
			toolchain = "incomplete:script-environment\n" + toolchain
			break
		}
	}
	return workflow.CaptureValidationInputs(root, []string{"."}, bindings, toolchain)
}

func requireApplicableCompile(store *workflow.Store, request *workflow.PreparedAgentRequest) error {
	if request.InputReference == nil { return nil }
	if request.Compile == nil || request.Compile.Request == nil || request.Compile.Result == nil || request.Compile.Result.RequestID != request.Compile.Request.ID { return fmt.Errorf("controlled compile result binding is missing") }
	current, err := captureCompileInputs(request)
	if err != nil { return err }
	applicable, reason := workflow.EvidenceApplicability(request.Compile.Inputs, current, request.Compile.Result != nil, request.Compile.Result != nil && request.Compile.Result.Status == workflow.ActorResultAccepted)
	if !applicable { return fmt.Errorf("compile evidence is not applicable: %s", reason) }
	return nil
}
