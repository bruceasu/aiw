package llm

import (
	czdata "aiw-cz/internal/cz"
	"aiw-cz/internal/envx"
	"aiw-cz/internal/fsx"
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

func RunLLM(prompt string, cfg czdata.Config) (string, error) {
	cwdEnv, exeEnv, err := loadOpenAIEnvFromDotEnv()
	if err != nil {
		return "", err
	}

	model, modelSource := resolveOpenAIValue(cfg.LLMModel, "OPENAI_MODEL", cwdEnv, exeEnv, "gpt-4o-mini")
	baseURL, baseURLSource := resolveOpenAIValue(cfg.APIBaseURL, "OPENAI_BASE_URL", cwdEnv, exeEnv, "https://api.openai.com/v1")
	baseURL = strings.TrimRight(baseURL, "/")

	fmt.Fprintf(os.Stderr, "+ openai chat.completions model=%s endpoint=%s/chat/completions\n", model, baseURL)
	if czdata.DryRun {
		return `{"candidates":[{"type":"chore","scope":"","subject":"dry run preview","body":"","breaking":"","footer":""}]}`, nil
	}

	apiKey, apiKeySource := resolveOpenAIValue(cfg.APIKey, "OPENAI_API_KEY", cwdEnv, exeEnv, "")
	if shouldDebugSource(cfg) {
		printOpenAIDebugSource("OPENAI_MODEL", model, modelSource, false)
		printOpenAIDebugSource("OPENAI_BASE_URL", baseURL, baseURLSource, false)
		printOpenAIDebugSource("OPENAI_API_KEY", apiKey, apiKeySource, true)
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

func resolveOpenAIValue(configValue, envKey string, cwdEnv, exeEnv map[string]string, defaultValue string) (string, string) {
	if v := strings.TrimSpace(configValue); v != "" {
		return v, "config"
	}
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v, "env"
	}
	if v := strings.TrimSpace(cwdEnv[envKey]); v != "" {
		return v, "cwd .env"
	}
	if v := strings.TrimSpace(exeEnv[envKey]); v != "" {
		return v, "exe .env"
	}
	if defaultValue != "" {
		return defaultValue, "default"
	}
	return "", "missing"
}

func loadOpenAIEnvFromDotEnv() (map[string]string, map[string]string, error) {
	exeLoader := &envx.Loader{Env: map[string]string{}}
	cwdLoader := &envx.Loader{Env: map[string]string{}}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		exeEnv := filepath.Join(exeDir, ".env")
		if fsx.Exists(exeEnv) {
			if err := exeLoader.ParseFile(exeEnv); err != nil {
				return nil, nil, fmt.Errorf("parse %s: %w", exeEnv, err)
			}
		}
	}

	if wd, err := os.Getwd(); err == nil {
		wdEnv := filepath.Join(wd, ".env")
		if fsx.Exists(wdEnv) {
			if err := cwdLoader.ParseFile(wdEnv); err != nil {
				return nil, nil, fmt.Errorf("parse %s: %w", wdEnv, err)
			}
		}
	}

	return cwdLoader.Env, exeLoader.Env, nil
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

func shouldDebugSource(cfg czdata.Config) bool {
	if cfg.DebugSource {
		return true
	}
	v := strings.ToLower(strings.TrimSpace(os.Getenv("AIW_CZ_DEBUG")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func printOpenAIDebugSource(key, value, source string, secret bool) {
	display := value
	if secret {
		display = maskSecret(value)
	}
	fmt.Fprintf(os.Stderr, "[cz llm debug] %s=%s (source=%s)\n", key, display, source)
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

func ParseLLMCandidates(out string) ([]czdata.Draft, error) {
	if cands, ok := parseCandidatesJSON(strings.TrimSpace(out)); ok {
		return cands, nil
	}

	chunks := strings.Split(out, "```")
	for i := 1; i < len(chunks); i += 2 {
		block := strings.TrimSpace(chunks[i])
		block = strings.TrimPrefix(block, "json")
		if cands, ok := parseCandidatesJSON(strings.TrimSpace(block)); ok {
			return cands, nil
		}
	}

	start := strings.Index(out, "{")
	end := strings.LastIndex(out, "}")
	if start >= 0 && end > start {
		candidate := out[start : end+1]
		if cands, ok := parseCandidatesJSON(candidate); ok {
			return cands, nil
		}
	}

	lines := strings.Split(out, "\n")
	var cands []czdata.Draft
	for _, ln := range lines {
		ln = cleanCandidateLine(ln)
		if ln == "" {
			continue
		}
		if d, ok := ParseConventionalHeader(ln); ok {
			cands = append(cands, d)
		}
	}
	if len(cands) == 0 {
		return nil, errors.New("invalid output")
	}
	return cands, nil
}

func parseCandidatesJSON(raw string) ([]czdata.Draft, bool) {
	if raw == "" {
		return nil, false
	}
	var resp czdata.LLMResponse
	if err := json.Unmarshal([]byte(raw), &resp); err == nil && len(resp.Candidates) > 0 {
		return resp.Candidates, true
	}

	var list []czdata.Draft
	if err := json.Unmarshal([]byte(raw), &list); err == nil && len(list) > 0 {
		return list, true
	}
	return nil, false
}

func maskSecret(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "<empty>"
	}
	if len(v) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(v)-4) + v[len(v)-4:]
}

func cleanCandidateLine(line string) string {
	v := strings.TrimSpace(line)
	v = strings.TrimPrefix(v, "-")
	v = strings.TrimPrefix(v, "*")
	v = strings.TrimPrefix(v, "•")
	v = strings.TrimSpace(v)

	if idx := strings.Index(v, ")"); idx > 0 {
		prefix := strings.TrimSpace(v[:idx])
		if _, err := strconv.Atoi(prefix); err == nil {
			v = strings.TrimSpace(v[idx+1:])
		}
	}
	if idx := strings.Index(v, "."); idx > 0 {
		prefix := strings.TrimSpace(v[:idx])
		if _, err := strconv.Atoi(prefix); err == nil {
			v = strings.TrimSpace(v[idx+1:])
		}
	}
	return v
}

func ParseConventionalHeader(line string) (czdata.Draft, bool) {
	i := strings.Index(line, ":")
	if i <= 0 || i+1 >= len(line) {
		return czdata.Draft{}, false
	}

	left := strings.TrimSpace(line[:i])
	subject := strings.TrimSpace(line[i+1:])
	if left == "" || subject == "" {
		return czdata.Draft{}, false
	}

	left = strings.TrimSuffix(left, "!")
	typePart := left
	scope := ""
	if l := strings.Index(left, "("); l >= 0 {
		r := strings.LastIndex(left, ")")
		if r <= l || r != len(left)-1 {
			return czdata.Draft{}, false
		}
		typePart = strings.TrimSpace(left[:l])
		scope = strings.TrimSpace(left[l+1 : r])
		if scope == "" {
			return czdata.Draft{}, false
		}
	}

	typePart = strings.ToLower(strings.TrimSpace(typePart))
	if typePart == "" || strings.ContainsAny(typePart, " `\t") {
		return czdata.Draft{}, false
	}
	if _, ok := conventionalTypeSet()[typePart]; !ok {
		return czdata.Draft{}, false
	}

	return czdata.Draft{Type: typePart, Scope: scope, Subject: subject}, true
}

func conventionalTypeSet() map[string]struct{} {
	set := map[string]struct{}{}
	for _, t := range czdata.DefaultConfig().Types {
		set[t.Value] = struct{}{}
	}
	return set
}
