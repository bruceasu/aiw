package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	args, lastMessagePath, cleanup, err := p.argsForGenerate(request)
	if err != nil {
		return Response{}, err
	}
	defer cleanup()
	cmd := exec.CommandContext(ctx, p.command, args...)
	cmd.Dir = request.Workspace
	if p.name == "codex" {
		cmd.Stdin = strings.NewReader(cliPrompt(request))
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return Response{}, fmt.Errorf("%s provider stdout: %w", p.name, err)
	}
	var stderrBuffer bytes.Buffer
	cmd.Stderr = &stderrBuffer
	var events bytes.Buffer
	writer := io.Writer(&events)
	var liveOutput *os.File
	if request.OutputDir != "" {
		if err := os.MkdirAll(request.OutputDir, 0o755); err != nil {
			return Response{}, fmt.Errorf("%s provider output directory: %w", p.name, err)
		}
		livePath := filepath.Join(request.OutputDir, fmt.Sprintf("%04d-live.jsonl", request.TurnNumber))
		liveOutput, err = os.Create(livePath)
		if err != nil {
			return Response{}, fmt.Errorf("%s provider live output: %w", p.name, err)
		}
		defer liveOutput.Close()
		writer = io.MultiWriter(&events, liveOutput)
	}
	if err := cmd.Start(); err != nil {
		return Response{}, fmt.Errorf("%s provider start: %w", p.name, err)
	}
	_, copyErr := io.Copy(writer, stdoutPipe)
	err = cmd.Wait()
	if copyErr != nil && err == nil {
		err = copyErr
	}
	stdout := events.Bytes()
	stderr := stderrBuffer.Bytes()
	exitCode := 1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	threadID := request.ThreadID
	if p.name == "copilot" && threadID == "" {
		threadID = "aiw-" + request.SessionID
	}
	finalOutput := string(stdout)
	if p.name == "codex" {
		threadID = DecodeThreadID(stdout)
		lastMessage, readErr := os.ReadFile(lastMessagePath)
		if readErr == nil && strings.TrimSpace(string(lastMessage)) != "" {
			finalOutput = string(lastMessage)
		} else {
			finalOutput = DecodeCodexFinalOutput(stdout)
		}
		if strings.TrimSpace(finalOutput) == "" {
			finalOutput = diagnosticOutput(stdout)
		}
	}
	result := Response{ThreadID: threadID, FinalOutput: finalOutput, Events: stdout, Stderr: stderr,
		ExitCode: exitCode, Metadata: map[string]string{"provider": p.name, "command": p.command},
		StartedAt: started, CompletedAt: time.Now().UTC()}
	if err != nil {
		if message := strings.TrimSpace(string(stderr)); message != "" {
			return result, fmt.Errorf("%s provider: %w: %s", p.name, err, message)
		}
		if p.name == "codex" {
			if message := DecodeCodexError(stdout); message != "" {
				return result, fmt.Errorf("%s provider: %w: %s", p.name, err, message)
			}
		}
		if message := diagnosticOutput(stdout); message != "" {
			return result, fmt.Errorf("%s provider: %w: %s", p.name, err, message)
		}
		return result, fmt.Errorf("%s provider: %w", p.name, err)
	}
	if p.name == "codex" && strings.TrimSpace(finalOutput) == "" {
		return result, fmt.Errorf("%s provider final output is empty", p.name)
	}
	return result, nil
}

func (p cliProvider) argsForGenerate(request Request) ([]string, string, func(), error) {
	args := p.args(request)
	if p.name != "codex" {
		return args, "", func() {}, nil
	}

	lastMessageFile, err := os.CreateTemp("", "aiw-last-message-*.txt")
	if err != nil {
		return nil, "", func() {}, fmt.Errorf("create %s final output file: %w", p.name, err)
	}
	lastMessagePath := lastMessageFile.Name()
	if err := lastMessageFile.Close(); err != nil {
		_ = os.Remove(lastMessagePath)
		return nil, "", func() {}, fmt.Errorf("close %s final output file: %w", p.name, err)
	}
	cleanupPaths := []string{lastMessagePath}
	cleanup := func() {
		for _, path := range cleanupPaths {
			_ = os.Remove(path)
		}
	}
	flags := []string{"--output-last-message", lastMessagePath}
	if request.OutputSchema == nil {
		withOutput, err := insertCodexFlags(args, flags)
		if err != nil {
			cleanup()
			return nil, "", func() {}, err
		}
		return withOutput, lastMessagePath, cleanup, nil
	}

	schema, err := json.Marshal(request.OutputSchema)
	if err != nil {
		cleanup()
		return nil, "", func() {}, fmt.Errorf("encode %s output schema: %w", p.name, err)
	}
	file, err := os.CreateTemp("", "aiw-output-schema-*.json")
	if err != nil {
		cleanup()
		return nil, "", func() {}, fmt.Errorf("create %s output schema: %w", p.name, err)
	}
	path := file.Name()
	cleanupPaths = append(cleanupPaths, path)
	if _, err := file.Write(append(schema, '\n')); err != nil {
		_ = file.Close()
		cleanup()
		return nil, "", func() {}, fmt.Errorf("write %s output schema: %w", p.name, err)
	}
	if err := file.Close(); err != nil {
		cleanup()
		return nil, "", func() {}, fmt.Errorf("close %s output schema: %w", p.name, err)
	}

	flags = append(flags, "--output-schema", path)
	withOutput, err := insertCodexFlags(args, flags)
	if err != nil {
		cleanup()
		return nil, "", func() {}, err
	}
	return withOutput, lastMessagePath, cleanup, nil
}

func insertCodexFlags(args, flags []string) ([]string, error) {
	promptIndex := -1
	for index, arg := range args {
		if arg == "-" {
			promptIndex = index
			break
		}
	}
	if promptIndex < 0 {
		return nil, errors.New("Codex command is missing stdin prompt argument")
	}
	withFlags := make([]string, 0, len(args)+len(flags))
	withFlags = append(withFlags, args[:promptIndex]...)
	withFlags = append(withFlags, flags...)
	withFlags = append(withFlags, args[promptIndex:]...)
	return withFlags, nil
}

func cliPrompt(request Request) string {
	system := strings.TrimSpace(request.SystemPrompt)
	prompt := strings.TrimSpace(request.Prompt)
	if system == "" {
		return prompt
	}
	if prompt == "" {
		return system
	}
	return system + "\n\n" + prompt
}

func (p cliProvider) Interactive(ctx context.Context, request Request) (Response, error) {
	started := time.Now().UTC()
	cmd := exec.CommandContext(ctx, p.command, p.interactiveArgs(request)...)
	cmd.Dir = request.Workspace
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	result := Response{ThreadID: request.ThreadID, ExitCode: exitCode,
		Metadata:  map[string]string{"provider": p.name, "command": p.command},
		StartedAt: started, CompletedAt: time.Now().UTC()}
	if err != nil {
		return result, fmt.Errorf("%s interactive provider: %w", p.name, err)
	}
	return result, nil
}

func (p cliProvider) interactiveArgs(request Request) []string {
	args := []string{}
	if p.model != "" {
		args = append(args, "--model", p.model)
	}
	if p.name == "codex" && request.ThreadID != "" {
		return append(args, "resume", request.ThreadID, request.Prompt)
	}
	if p.name == "codex" && strings.TrimSpace(request.Prompt) != "" {
		return append(args, request.Prompt)
	}
	if p.name == "copilot" && request.ThreadID != "" && !request.ForceNewThread {
		// Copilot starts interactively by default; --prompt is deliberately not
		// used because it exits after one response.
		return append(args, "--resume="+request.ThreadID)
	}
	return args
}

func (p cliProvider) args(r Request) []string {
	if p.name == "codex" {
		args := []string{"exec"}
		if r.ReadOnly {
			if r.Workspace != "" {
				args = append(args, "--cd", r.Workspace)
			}
			args = append(args, "--sandbox", "read-only")
			for _, path := range r.AdditionalDirs {
				args = append(args, "--add-dir", path)
			}
		}
		if r.ThreadID != "" && !r.ForceNewThread {
			args = append(args, "resume", r.ThreadID)
		}
		return append(args, "--json", "-")
	}
	id := r.ThreadID
	if id == "" || r.ForceNewThread {
		id = "aiw-" + r.SessionID
	}
	args := []string{"--output-format", "text"}
	if r.ReadOnly {
		for _, path := range r.AdditionalDirs {
			args = append(args, "--add-dir", path)
		}
	} else {
		args = append([]string{"--allow-all-tools"}, args...)
	}
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

// DecodeCodexFinalOutput extracts the final assistant message from the JSONL
// event stream emitted by "codex exec --json". Events remain available in
// Response.Events for diagnostics; callers receive only model output.
func DecodeCodexFinalOutput(events []byte) string {
	var output string
	for _, line := range strings.Split(string(events), "\n") {
		var event struct {
			Type string `json:"type"`
			Item struct {
				Type    string          `json:"type"`
				Text    string          `json:"text"`
				Content json.RawMessage `json:"content"`
			} `json:"item"`
		}
		if json.Unmarshal([]byte(line), &event) != nil || event.Type != "item.completed" || event.Item.Type != "agent_message" {
			continue
		}
		if text := strings.TrimSpace(event.Item.Text); text != "" {
			output = text
			continue
		}
		if text := decodeCodexMessageContent(event.Item.Content); text != "" {
			output = text
		}
	}
	return output
}

// DecodeCodexError extracts an error reported by `codex exec --json`.
func DecodeCodexError(events []byte) string {
	for _, line := range strings.Split(string(events), "\n") {
		var event struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Error   struct {
				Message string `json:"message"`
			} `json:"error"`
			Item struct {
				Type    string `json:"type"`
				Message string `json:"message"`
				Error   struct {
					Message string `json:"message"`
				} `json:"error"`
			} `json:"item"`
		}
		if json.Unmarshal([]byte(line), &event) != nil {
			continue
		}
		if !strings.Contains(event.Type, "error") && !strings.Contains(event.Type, "failed") &&
			!strings.Contains(event.Item.Type, "error") && !strings.Contains(event.Item.Type, "failed") {
			continue
		}
		for _, message := range []string{event.Message, event.Error.Message, event.Item.Message, event.Item.Error.Message} {
			if message = strings.TrimSpace(message); message != "" {
				return message
			}
		}
	}
	return ""
}

func diagnosticOutput(output []byte) string {
	message := strings.TrimSpace(string(output))
	const maxLength = 4096
	if len(message) > maxLength {
		return message[:maxLength] + "..."
	}
	return message
}

func decodeCodexMessageContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return strings.TrimSpace(text)
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) != nil {
		return ""
	}
	var texts []string
	for _, part := range parts {
		if part.Type == "output_text" || part.Type == "text" {
			if text := strings.TrimSpace(part.Text); text != "" {
				texts = append(texts, text)
			}
		}
	}
	return strings.Join(texts, "\n")
}
