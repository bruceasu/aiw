package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type geminiProvider struct {
	model      string
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewGeminiProvider(cfg Config) Provider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	return geminiProvider{model: cfg.Model, apiKey: cfg.APIKey, baseURL: strings.TrimRight(baseURL, "/"), httpClient: cfg.HTTPClient}
}

func (p geminiProvider) Name() string { return "gemini" }

func (p geminiProvider) Generate(ctx context.Context, request Request) (Response, error) {
	model := request.Model
	if model == "" {
		model = p.model
	}
	body := map[string]any{"contents": []any{map[string]any{"role": "user", "parts": []any{map[string]string{"text": request.Prompt}}}}}
	if request.SystemPrompt != "" {
		body["system_instruction"] = map[string]any{"parts": []any{map[string]string{"text": request.SystemPrompt}}}
	}
	if request.OutputSchema != nil {
		body["generationConfig"] = map[string]any{"responseMimeType": "application/json", "responseSchema": request.OutputSchema}
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", p.baseURL, model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := p.httpClient
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	started := time.Now().UTC()
	resp, err := client.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("gemini API failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var decoded struct {
		Candidates []struct {
			Content struct {
				Parts []struct { Text string `json:"text"` } `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return Response{}, fmt.Errorf("decode gemini response: %w", err)
	}
	if len(decoded.Candidates) == 0 || len(decoded.Candidates[0].Content.Parts) == 0 {
		return Response{}, fmt.Errorf("gemini response has no content")
	}
	return Response{FinalOutput: strings.TrimSpace(decoded.Candidates[0].Content.Parts[0].Text), Events: raw,
		ExitCode: 0, StartedAt: started, CompletedAt: time.Now().UTC(),
		Metadata: map[string]string{"provider": "gemini", "model": model}}, nil
}
