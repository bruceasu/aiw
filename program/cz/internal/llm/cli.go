package llm

import (
	czdata "aiw-cz/internal/cz"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var lookPathFn = exec.LookPath
var execCommandContextFn = exec.Command

func RunCodexCLI(prompt string, cfg czdata.Config) (string, error) {
	cmdName := strings.TrimSpace(cfg.CodexCommand)
	if cmdName == "" {
		cmdName = "codex"
	}
	if _, err := lookPathFn(cmdName); err != nil {
		return "", fmt.Errorf("codex cli not found: %w", err)
	}
	schemaFile, err := os.CreateTemp("", "aiw-cz-codex-schema-*.json")
	if err != nil {
		return "", fmt.Errorf("create codex output schema: %w", err)
	}
	schemaPath := schemaFile.Name()
	defer os.Remove(schemaPath)
	if _, err := schemaFile.WriteString(czCandidatesJSONSchema); err != nil {
		schemaFile.Close()
		return "", fmt.Errorf("write codex output schema: %w", err)
	}
	if err := schemaFile.Close(); err != nil {
		return "", fmt.Errorf("close codex output schema: %w", err)
	}

	args := []string{"exec", "--output-schema", schemaPath, "-"}
	out, errOut, err := runCLIWithInput(cmdName, args, prompt)
	if err != nil {
		return "", fmt.Errorf("codex cli failed: %w (%s)", err, strings.TrimSpace(errOut))
	}
	content := strings.TrimSpace(out)
	if content == "" {
		return "", fmt.Errorf("codex cli returned empty output")
	}
	return content, nil
}

const czCandidatesJSONSchema = `{
	"type": "object",
	"properties": {
		"candidates": {
			"type": "array",
			"minItems": 1,
			"items": {
				"type": "object",
				"properties": {
					"type": {"type": "string"},
					"scope": {"type": "string"},
					"subject": {"type": "string"},
					"body": {"type": "string"},
					"breaking": {"type": "string"},
					"footer": {"type": "string"}
				},
				"required": ["type", "scope", "subject", "body", "breaking", "footer"],
				"additionalProperties": false
			}
		}
	},
	"required": ["candidates"],
	"additionalProperties": false
}`

func RunCopilotCLI(prompt string, cfg czdata.Config) (string, error) {
	cmdName := strings.TrimSpace(cfg.CopilotCommand)
	if cmdName == "" {
		cmdName = "copilot"
	}
	if _, err := lookPathFn(cmdName); err != nil {
		return "", fmt.Errorf("copilot cli not found: %w", err)
	}
	args := []string{"--model", "gpt-5.4-mini", "--silent", "--output-format", "json", "--prompt", prompt}
	out, errOut, err := runCLI(cmdName, args)
	if err != nil {
		return "", fmt.Errorf("copilot cli failed: %w (%s)", err, strings.TrimSpace(errOut))
	}
	content := strings.TrimSpace(out)
	if content == "" {
		return "", fmt.Errorf("copilot cli returned empty output")
	}
	return content, nil
}

func codexCLIAvailable(cfg czdata.Config) bool {
	name := strings.TrimSpace(cfg.CodexCommand)
	if name == "" {
		name = "codex"
	}
	_, err := lookPathFn(name)
	return err == nil
}

func copilotCLIAvailable(cfg czdata.Config) bool {
	name := strings.TrimSpace(cfg.CopilotCommand)
	if name == "" {
		name = "copilot"
	}
	_, err := lookPathFn(name)
	if err != nil {
		return false
	}
	_, _, probeErr := runCLI(name, []string{"--help"})
	return probeErr == nil
}

func runCLI(name string, args []string) (string, string, error) {
	return runCLIWithInput(name, args, "")
}

func runCLIWithInput(name string, args []string, input string) (string, string, error) {
	cmd := execCommandContextFn(name, args...)
	var outb bytes.Buffer
	var errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	err := cmd.Run()
	return outb.String(), errb.String(), err
}

func cliNullPath() string {
	if runtime.GOOS == "windows" {
		return "NUL"
	}
	return "/dev/null"
}
