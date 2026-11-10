package aiprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"aiw/internal/ai"
)

const (
	maxRequestBytes      = 1 << 20
	maxTextRunes         = 64 * 1024
	maxOutputBytes       = 4 << 20
	// Keep synchronized with MAX_BRIDGE_OUTPUT_BYTES in the aiw-ai plugin.
	maxBridgeOutputBytes = 25 << 20
)

// request is the private, one-shot protocol used by the Node ai plugin.
type request struct {
	Prompt         string `json:"prompt"`
	SystemPrompt   string `json:"system_prompt"`
	JSON           bool   `json:"json"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type response struct {
	Text     string `json:"text"`
	Provider string `json:"provider"`
	Model    string `json:"model,omitempty"`
}

// Dispatch handles one bounded provider request from the aiw-ai plugin.
func Dispatch() error {
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, maxRequestBytes+1))
	if err != nil {
		return fmt.Errorf("read AIW Provider request: %w", err)
	}
	if len(raw) > maxRequestBytes {
		return errors.New("AIW Provider request exceeds 1 MiB")
	}
	if !utf8.Valid(raw) {
		return errors.New("AIW Provider request is not valid UTF-8")
	}

	var input request
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return fmt.Errorf("invalid AIW Provider request: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("AIW Provider request must contain one JSON object")
		}
		return fmt.Errorf("invalid trailing AIW Provider request data: %w", err)
	}
	if strings.TrimSpace(input.Prompt) == "" || utf8.RuneCountInString(input.Prompt) > maxTextRunes {
		return errors.New("AIW Provider prompt must contain 1-65536 Unicode code points")
	}
	if utf8.RuneCountInString(input.SystemPrompt) > maxTextRunes {
		return errors.New("AIW Provider system prompt exceeds 65536 Unicode code points")
	}
	if input.TimeoutSeconds < 1 || input.TimeoutSeconds > 3660 {
		return errors.New("AIW Provider timeout must be 1-3660 seconds")
	}

	cfg, err := ai.LoadConfig()
	if err != nil {
		return fmt.Errorf("load AIW Provider configuration: %w", err)
	}
	provider := strings.TrimSpace(cfg.Name)
	if provider == "" {
		provider = "auto"
	}
	system := input.SystemPrompt
	if input.JSON {
		const jsonInstruction = "Return exactly one valid JSON object with no Markdown fences or surrounding text."
		if system != "" {
			system += "\n\n"
		}
		system += jsonInstruction
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(input.TimeoutSeconds)*time.Second)
	defer cancel()
	text, err := ai.RunLLMWithContext(ctx, input.Prompt, ai.LLMConfig{
		Provider:       cfg.Name,
		Model:          cfg.Model,
		APIBaseURL:     cfg.BaseURL,
		APIKey:         cfg.APIKey,
		CodexCommand:   cfg.CodexCommand,
		CopilotCommand: cfg.CopilotCommand,
		Workspace:      os.Getenv("AIW_WORKSPACE"),
		ReadOnly:       true,
	}, nil, system)
	if err != nil {
		return fmt.Errorf("AIW Provider request failed: %w", err)
	}
	if len(text) > maxOutputBytes {
		return errors.New("AIW Provider response exceeds 4 MiB")
	}

	encoded, err := json.Marshal(response{Text: text, Provider: provider, Model: cfg.Model})
	if err != nil {
		return fmt.Errorf("encode AIW Provider response: %w", err)
	}
	if len(encoded)+1 > maxBridgeOutputBytes {
		return errors.New("AIW Provider response envelope exceeds 25 MiB")
	}
	encoded = append(encoded, '\n')
	_, err = os.Stdout.Write(encoded)
	return err
}
