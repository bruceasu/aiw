package llm

import (
	czdata "aiw-cz/internal/cz"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type geminiRequest struct {
	Contents         []geminiContent   `json:"contents"`
	GenerationConfig *geminiGenConfig  `json:"generationConfig,omitempty"`
	SystemInstruction *geminiContent   `json:"system_instruction,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	Temperature      float64         `json:"temperature,omitempty"`
	ResponseMimeType string          `json:"responseMimeType,omitempty"`
	ResponseSchema   map[string]any `json:"responseSchema,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
}

func callGeminiChat(prompt, model, baseURL, apiKey string, useSchema bool) (string, int, error) {
	reqBody := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: "You generate Conventional Commit candidates. Return JSON only."}},
		},
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: prompt}},
			},
		},
		GenerationConfig: &geminiGenConfig{
			Temperature: 0.2,
		},
	}

	if useSchema {
		reqBody.GenerationConfig.ResponseMimeType = "application/json"
		reqBody.GenerationConfig.ResponseSchema = geminiCandidatesSchema()
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	// Default baseURL for Gemini is https://generativelanguage.googleapis.com
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	// Gemini API endpoint: /v1beta/models/{model}:generateContent?key={apiKey}
	endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", baseURL, model, apiKey)
	
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return "", 0, err
	}
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
		return "", resp.StatusCode, fmt.Errorf("gemini api failed (%d): %s", resp.StatusCode, msg)
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", resp.StatusCode, fmt.Errorf("decode gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", resp.StatusCode, errors.New("gemini response has no content")
	}

	content := geminiResp.Candidates[0].Content.Parts[0].Text
	return strings.TrimSpace(content), resp.StatusCode, nil
}

func geminiCandidatesSchema() map[string]any {
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
		"required": []string{"type", "subject", "scope", "body", "breaking", "footer"},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"candidates": map[string]any{
				"type":  "array",
				"items": item,
			},
		},
		"required": []string{"candidates"},
	}
}

func RunGemini(prompt string, cfg czdata.Config) (string, error) {
	cwdEnv, aiwEnv, exeEnv, err := loadLLMEnvFromDotEnv()
	if err != nil {
		return "", err
	}

	model, modelSource := resolveLLMValue(cfg.LLMModel, "GEMINI_MODEL", cwdEnv, aiwEnv, exeEnv, "gemini-1.5-flash")
	baseURL, baseURLSource := resolveLLMValue(cfg.APIBaseURL, "GEMINI_BASE_URL", cwdEnv, aiwEnv, exeEnv, "https://generativelanguage.googleapis.com")
	apiKey, apiKeySource := resolveLLMValue(cfg.APIKey, "GEMINI_API_KEY", cwdEnv, aiwEnv, exeEnv, "")

	fmt.Fprintf(os.Stderr, "+ gemini generateContent model=%s endpoint=%s\n", model, baseURL)
	if czdata.DryRun {
		return `{"candidates":[{"type":"chore","scope":"","subject":"dry run preview","body":"","breaking":"","footer":""}]}`, nil
	}

	if shouldDebugSource(cfg) {
		printLLMDebugSource("GEMINI_MODEL", model, modelSource, false)
		printLLMDebugSource("GEMINI_BASE_URL", baseURL, baseURLSource, false)
		printLLMDebugSource("GEMINI_API_KEY", apiKey, apiKeySource, true)
	}
	if apiKey == "" {
		return "", errors.New("GEMINI_API_KEY is required for gemini provider")
	}

	out, status, err := callGeminiChat(prompt, model, baseURL, apiKey, true)
	if err == nil {
		return strings.TrimSpace(out), nil
	}
	if status == http.StatusBadRequest {
		// Fallback without schema if needed, but Gemini 1.5 usually supports it
		fallbackOut, _, fallbackErr := callGeminiChat(prompt, model, baseURL, apiKey, false)
		if fallbackErr == nil {
			return strings.TrimSpace(fallbackOut), nil
		}
		return "", fallbackErr
	}
	return "", err
}
