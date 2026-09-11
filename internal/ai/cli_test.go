package ai

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDecodeCodexFinalOutputUsesCompletedAgentMessage(t *testing.T) {
	events := []byte("{\"type\":\"thread.started\",\"thread_id\":\"thread-1\"}\n" +
		"{\"type\":\"item.completed\",\"item\":{\"type\":\"reasoning\",\"text\":\"hidden\"}}\n" +
		"{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"{\\\"candidates\\\":[{\\\"type\\\":\\\"fix\\\",\\\"subject\\\":\\\"handle codex output\\\"}]}\"}}\n" +
		"{\"type\":\"turn.completed\"}\n")

	want := `{"candidates":[{"type":"fix","subject":"handle codex output"}]}`
	if got := DecodeCodexFinalOutput(events); got != want {
		t.Fatalf("DecodeCodexFinalOutput() = %q, want %q", got, want)
	}
}

func TestDecodeCodexFinalOutputSupportsContentParts(t *testing.T) {
	events := []byte(`{"type":"item.completed","item":{"type":"agent_message","content":[{"type":"output_text","text":"first"},{"type":"output_text","text":"second"}]}}`)

	if got, want := DecodeCodexFinalOutput(events), "first\nsecond"; got != want {
		t.Fatalf("DecodeCodexFinalOutput() = %q, want %q", got, want)
	}
}

func TestDiagnosticOutputPreservesCodexEventsWhenNoMessageIsDecoded(t *testing.T) {
	events := []byte(`{"type":"thread.started","thread_id":"thread-1"}`)

	if got, want := diagnosticOutput(events), string(events); got != want {
		t.Fatalf("diagnosticOutput() = %q, want %q", got, want)
	}
}

func TestDecodeCodexErrorUsesFailedEventMessage(t *testing.T) {
	events := []byte(`{"type":"turn.failed","error":{"message":"unsupported --output-schema"}}`)

	if got, want := DecodeCodexError(events), "unsupported --output-schema"; got != want {
		t.Fatalf("DecodeCodexError() = %q, want %q", got, want)
	}
}

func TestCodexReadOnlyArgumentsUseSupportedApprovalFlags(t *testing.T) {
	p := NewCLIProvider("codex", Config{Name: "codex", Command: "codex"})
	args := p.(cliProvider).args(Request{
		Workspace:      t.TempDir(),
		AdditionalDirs: []string{filepath.Join(t.TempDir(), "shared")},
		ReadOnly:       true,
	})
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "--sandbox read-only") {
		t.Fatalf("Codex read-only arguments = %q, want read-only sandbox", joined)
	}
	if strings.Contains(joined, "--ask-for-approval") {
		t.Fatalf("Codex read-only arguments = %q, contains unsupported approval flag", joined)
	}
	jsonIndex, stdinIndex := -1, -1
	for index, arg := range args {
		if arg == "--json" {
			jsonIndex = index
		}
		if arg == "-" {
			stdinIndex = index
		}
	}
	if jsonIndex < 0 || stdinIndex < 0 || jsonIndex > stdinIndex {
		t.Fatalf("Codex arguments must place --json before stdin prompt marker: %q", joined)
	}
}

func TestCodexStructuredArgumentsUseOutputSchema(t *testing.T) {
	p := NewCLIProvider("codex", Config{Name: "codex", Command: "codex"})
	args, lastMessagePath, cleanup, err := p.(cliProvider).argsForGenerate(Request{
		OutputSchema: map[string]any{"type": "object"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "--output-schema ") {
		t.Fatalf("Codex structured arguments = %q, want output schema path", joined)
	}
	if lastMessagePath == "" || !strings.Contains(joined, "--output-last-message ") {
		t.Fatalf("Codex structured arguments = %q, want final output file", joined)
	}
}

func TestCLIProviderPromptIncludesSystemPrompt(t *testing.T) {
	prompt := cliPrompt(Request{SystemPrompt: "Return JSON only.", Prompt: "Answer the question."})
	if !strings.Contains(prompt, "Return JSON only.") || !strings.Contains(prompt, "Answer the question.") {
		t.Fatalf("CLI prompt = %q, want system and user prompts", prompt)
	}
}

func TestInteractiveProviderUsesWorkspaceAndProviderSpecificArguments(t *testing.T) {
	for _, provider := range []string{"codex", "copilot"} {
		t.Run(provider, func(t *testing.T) {
			workspace := t.TempDir()
			logPath := filepath.Join(workspace, "invocation.log")
			command := writeRecordingCommand(t, logPath)
			t.Setenv("AIW_TEST_LOG", logPath)

			p := NewCLIProvider(provider, Config{Name: provider, Command: command, Model: "model-test"})
			result, err := p.Interactive(context.Background(), Request{
				Workspace: workspace,
				Prompt:    "handoff prompt",
				ThreadID:  "thread-test",
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.ExitCode != 0 {
				t.Fatalf("interactive exit code = %d", result.ExitCode)
			}
			contents, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSpace(string(contents)), "\n")
			if len(lines) < 2 || strings.TrimSpace(lines[0]) != workspace {
				t.Fatalf("interactive command workspace = %q, want %q; log=%q", lines, workspace, contents)
			}
			args := lines[1]
			if !strings.Contains(args, "model-test") || !strings.Contains(args, "thread-test") {
				t.Fatalf("provider-specific arguments missing from %q", args)
			}
			if provider == "codex" && !strings.Contains(args, "resume") {
				t.Fatalf("Codex interactive command did not resume thread: %q", args)
			}
			if provider == "copilot" && !strings.Contains(args, "--resume=thread-test") {
				t.Fatalf("Copilot interactive command did not resume thread: %q", args)
			}
		})
	}
}

func writeRecordingCommand(t *testing.T, logPath string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(t.TempDir(), "provider.bat")
		contents := "@>\"%AIW_TEST_LOG%\" echo %CD%\r\n@>>\"%AIW_TEST_LOG%\" echo %*\r\n"
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	path := filepath.Join(t.TempDir(), "provider.sh")
	contents := "#!/bin/sh\nprintf '%s\\n' \"$PWD\" > \"$AIW_TEST_LOG\"\nprintf '%s\\n' \"$*\" >> \"$AIW_TEST_LOG\"\n"
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
