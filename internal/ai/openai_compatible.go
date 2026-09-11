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

type openAICompatibleProvider struct {
	name       string
	model      string
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func (p openAICompatibleProvider) Interactive(context.Context, Request) (Response, error) {
	return unsupportedInteractiveProvider(p.name)
}

func NewOpenAICompatibleProvider(name string, cfg Config) Provider {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		switch name {
		case "ollama":
			baseURL = "http://localhost:11434"
		case "llama.cpp", "llamacpp":
			baseURL = "http://localhost:8080/v1"
		}
	}
	if name == "ollama" && !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}
	return openAICompatibleProvider{name: name, model: cfg.Model, apiKey: cfg.APIKey, baseURL: baseURL, httpClient: cfg.HTTPClient}
}

func (p openAICompatibleProvider) Name() string { return p.name }

func (p openAICompatibleProvider) Generate(ctx context.Context, request Request) (Response, error) {
	started := time.Now().UTC()
	model := request.Model
	if model == "" {
		model = p.model
	}
	messages := []map[string]string{}
	if request.SystemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": request.SystemPrompt})
	}
	messages = append(messages, map[string]string{"role": "user", "content": request.Prompt})
	body := map[string]any{"model": model, "messages": messages, "stream": false}
	if request.OutputSchema != nil {
		body["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "aiw_response", "strict": true, "schema": request.OutputSchema}}
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	endpoint := strings.TrimRight(p.baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	client := p.httpClient
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("%s request failed: %w", p.name, err)
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return Response{}, readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("%s API failed (%d): %s", p.name, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var decoded struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return Response{}, fmt.Errorf("decode %s response: %w", p.name, err)
	}
	if len(decoded.Choices) == 0 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
		return Response{}, fmt.Errorf("%s response has no content", p.name)
	}
	return Response{ThreadID: decoded.ID, FinalOutput: strings.TrimSpace(decoded.Choices[0].Message.Content), Events: raw,
		ExitCode: 0, StartedAt: started, CompletedAt: time.Now().UTC(),
		Metadata: map[string]string{"provider": p.name, "model": model}}, nil
}
