package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type cliProvider struct {
	name    string
	command string
	model   string
}

func NewCLIProvider(name string, cfg Config) Provider {
	command := strings.TrimSpace(cfg.Command)
	if command == "" {
		command = name
	}
	return cliProvider{name: name, command: command, model: cfg.Model}
}

func (p cliProvider) Name() string { return p.name }

func (p cliProvider) Generate(ctx context.Context, request Request) (Response, error) {
	started := time.Now().UTC()
	args := p.args(request)
	cmd := exec.CommandContext(ctx, p.command, args...)
	cmd.Dir = request.Workspace
	if p.name == "codex" {
		cmd.Stdin = strings.NewReader(request.Prompt)
	}
	stdout, err := cmd.Output()
	stderr := []byte{}
	if exit, ok := err.(*exec.ExitError); ok {
		stderr = exit.Stderr
	}
	exitCode := 1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	threadID := request.ThreadID
	if p.name == "copilot" && threadID == "" {
		threadID = "aiw-" + request.SessionID
	}
	if p.name == "codex" {
		threadID = DecodeThreadID(stdout)
	}
	result := Response{ThreadID: threadID, FinalOutput: string(stdout), Events: stdout, Stderr: stderr,
		ExitCode: exitCode, Metadata: map[string]string{"provider": p.name, "command": p.command},
		StartedAt: started, CompletedAt: time.Now().UTC()}
	if err != nil {
		return result, fmt.Errorf("%s provider: %w", p.name, err)
	}
	return result, nil
}

func (p cliProvider) args(r Request) []string {
	if p.name == "codex" {
		args := []string{"exec"}
		if r.ThreadID != "" && !r.ForceNewThread {
			args = append(args, "resume", r.ThreadID)
		}
		return append(args, "-", "--json")
	}
	id := r.ThreadID
	if id == "" || r.ForceNewThread {
		id = "aiw-" + r.SessionID
	}
	args := []string{"--allow-all-tools", "--output-format", "text"}
	if p.model != "" {
		args = append([]string{"--model", p.model}, args...)
	}
	if r.ThreadID != "" && !r.ForceNewThread {
		args = append(args, "--resume="+id)
	} else {
		args = append(args, "--session-id="+id)
	}
	return append(args, "--prompt", r.Prompt)
}

func DecodeThreadID(events []byte) string {
	var id string
	for _, line := range strings.Split(string(events), "\n") {
		var event map[string]any
		if json.Unmarshal([]byte(line), &event) != nil {
			continue
		}
		if value, ok := event["thread_id"].(string); ok && value != "" {
			id = value
		}
		if value, ok := event["session_id"].(string); ok && value != "" {
			id = value
		}
	}
	return id
}
