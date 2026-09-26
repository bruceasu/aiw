package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"aiw/internal/ai"
	"aiw/internal/workflow"
)

// RemediationAnalyzer is a read-only seam. Implementations may inspect the
// frozen problem, but they cannot receive a Store or mutate Workflow state.
type RemediationAnalyzer interface {
	Analyze(context.Context, workflow.RemediationProblem, []workflow.RemediationOption) (workflow.RemediationDiagnosis, error)
}

// LLMRemediationAnalyzer asks one configured model for a structured diagnosis.
// It is advisory: Core validates the selected action before anything runs.
type LLMRemediationAnalyzer struct {
	Workspace string
	Provider  string
	Model     string
}

func (a LLMRemediationAnalyzer) Analyze(_ context.Context, problem workflow.RemediationProblem, options []workflow.RemediationOption) (workflow.RemediationDiagnosis, error) {
	config, err := ai.LoadConfig()
	if err != nil { return workflow.RemediationDiagnosis{}, err }
	if strings.TrimSpace(a.Provider) != "" { config.Name = a.Provider }
	if strings.TrimSpace(a.Model) != "" { config.Model = a.Model }
	if strings.TrimSpace(a.Provider) != "" || strings.TrimSpace(a.Model) != "" {
		config, err = ai.ConfigFor(a.Provider, a.Model)
		if err != nil { return workflow.RemediationDiagnosis{}, err }
	}
	input := struct {
		Problem workflow.RemediationProblem `json:"problem"`
		Options []workflow.RemediationOption `json:"options"`
	}{problem, options}
	body, err := json.Marshal(input)
	if err != nil { return workflow.RemediationDiagnosis{}, err }
	prompt := "Analyze this Workflow failure as read-only data. Choose only one action from the supplied options. Do not propose commands, file edits, authorization changes, budget changes, or arbitrary retries. Return JSON only.\n" + string(body)
	output, err := ai.RunLLMWithSystemPrompt(prompt, ai.LLMConfig{Provider: config.Name, Model: config.Model, APIBaseURL: config.BaseURL, APIKey: config.APIKey, CodexCommand: config.CodexCommand, CopilotCommand: config.CopilotCommand, Workspace: a.Workspace, ReadOnly: true}, map[string]any{"type": "object"}, "Return only JSON with category, summary, confidence, evidence, recommended_action, and reason.")
	if err != nil { return workflow.RemediationDiagnosis{}, fmt.Errorf("remediation analysis failed: %w", err) }
	var diagnosis workflow.RemediationDiagnosis
	if err := json.Unmarshal([]byte(output), &diagnosis); err != nil { return diagnosis, fmt.Errorf("decode remediation analysis: %w", err) }
	if err := diagnosis.Validate(problem, options); err != nil { return diagnosis, err }
	if diagnosis.Confidence < 0.70 { return diagnosis, errors.New("remediation analysis confidence is below the automatic threshold") }
	return diagnosis, nil
}
