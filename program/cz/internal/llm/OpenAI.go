package llm

import (
	czdata "aiw-cz/internal/cz"
	"bytes"
	"encoding/json"
	"io"
	"time"

	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type openAIChatRequest struct {
	Model          string               `json:"model"`
	Messages       []openAIChatMessage  `json:"messages"`
	ResponseFormat *openAIResponseShape `json:"response_format,omitempty"`
	Temperature    float64              `json:"temperature,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseShape struct {
	Type       string            `json:"type"`
	JSONSchema *openAIJSONSchema `json:"json_schema,omitempty"`
}

type openAIJSONSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func RunOpenAI(prompt string, cfg czdata.Config) (string, error) {
	cwdEnv, aiwEnv, exeEnv, err := loadLLMEnvFromDotEnv()
	if err != nil {
		return "", err
	}

	model, modelSource := resolveLLMValue(cfg.LLMModel, "OPENAI_MODEL", cwdEnv, aiwEnv, exeEnv, "gpt-4o-mini")
	baseURL, baseURLSource := resolveLLMValue(cfg.APIBaseURL, "OPENAI_BASE_URL", cwdEnv, aiwEnv, exeEnv, "https://api.openai.com/v1")
	baseURL = strings.TrimRight(baseURL, "/")

	fmt.Fprintf(os.Stderr, "+ openai chat.completions model=%s endpoint=%s/chat/completions\n", model, baseURL)
	if czdata.DryRun {
		return `{"candidates":[{"type":"chore","scope":"","subject":"dry run preview","body":"","breaking":"","footer":""}]}`, nil
	}

	apiKey, apiKeySource := resolveLLMValue(cfg.APIKey, "OPENAI_API_KEY", cwdEnv, aiwEnv, exeEnv, "")
	if shouldDebugSource(cfg) {
		printLLMDebugSource("OPENAI_MODEL", model, modelSource, false)
		printLLMDebugSource("OPENAI_BASE_URL", baseURL, baseURLSource, false)
		printLLMDebugSource("OPENAI_API_KEY", apiKey, apiKeySource, true)
	}
	if apiKey == "" {
		return "", errors.New("OPENAI_API_KEY is required for --llm mode")
	}

	out, status, err := callOpenAIChat(prompt, model, baseURL, apiKey, true)
	if err == nil {
		return strings.TrimSpace(out), nil
	}
	if status == http.StatusBadRequest {
		fallbackOut, _, fallbackErr := callOpenAIChat(prompt, model, baseURL, apiKey, false)
		if fallbackErr == nil {
			return strings.TrimSpace(fallbackOut), nil
		}
		return "", fallbackErr
	}
	return "", err
}

func callOpenAIChat(prompt, model, baseURL, apiKey string, useSchema bool) (string, int, error) {
	reqBody := openAIChatRequest{
		Model: model,
		Messages: []openAIChatMessage{
			{
				Role:    "system",
				Content: "You generate Conventional Commit candidates. Return JSON only.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.2,
	}
	if useSchema {
		reqBody.ResponseFormat = &openAIResponseShape{
			Type: "json_schema",
			JSONSchema: &openAIJSONSchema{
				Name:   "cz_candidates",
				Strict: true,
				Schema: czCandidatesSchema(),
			},
		}
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	endpoint := baseURL + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 1200 {
			msg = msg[:1200] + "..."
		}
		return "", resp.StatusCode, fmt.Errorf("openai api failed (%d): %s", resp.StatusCode, msg)
	}

	content, err := extractOpenAIContent(body)
	if err != nil {
		return "", resp.StatusCode, err
	}
	return content, resp.StatusCode, nil
}

func extractOpenAIContent(raw []byte) (string, error) {
	var resp openAIChatResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("decode openai response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("openai response has no choices")
	}
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	if content == "" {
		return "", errors.New("openai response content is empty")
	}
	return content, nil
}

func czCandidatesSchema() map[string]any {
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type":     map[string]any{"type": "string"},
			"scope":    map[string]any{"type": "string"},
			"subject":  map[string]any{"type": "string"},
			"body":     map[string]any{"type": "string"},
			"breaking": map[string]any{"type": "string"},
			"footer":   map[string]any{"type": "string"},
		},
		"required":             []string{"type", "subject", "scope", "body", "breaking", "footer"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"candidates": map[string]any{
				"type":  "array",
				"items": item,
			},
		},
		"required":             []string{"candidates"},
		"additionalProperties": false,
	}
}
