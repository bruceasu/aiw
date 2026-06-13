package llm

import (
	czdata "aiw-cz/internal/cz"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func RunOllama(prompt string, cfg czdata.Config) (string, error) {
	cwdEnv, aiwEnv, exeEnv, err := loadLLMEnvFromDotEnv()
	if err != nil {
		return "", err
	}

	// Ollama defaults
	model, modelSource := resolveLLMValue(cfg.LLMModel, "OLLAMA_MODEL", cwdEnv, aiwEnv, exeEnv, "llama3")
	baseURL, baseURLSource := resolveLLMValue(cfg.APIBaseURL, "OLLAMA_BASE_URL", cwdEnv, aiwEnv, exeEnv, "http://localhost:11434")
	baseURL = strings.TrimRight(baseURL, "/")

	fmt.Fprintf(os.Stderr, "+ ollama chat model=%s endpoint=%s/api/chat\n", model, baseURL)
	if czdata.DryRun {
		return `{"candidates":[{"type":"chore","scope":"","subject":"dry run preview","body":"","breaking":"","footer":""}]}`, nil
	}

	if shouldDebugSource(cfg) {
		printLLMDebugSource("OLLAMA_MODEL", model, modelSource, false)
		printLLMDebugSource("OLLAMA_BASE_URL", baseURL, baseURLSource, false)
	}

	// Ollama natively supports a simple chat API
	out, err := callOllamaChat(prompt, model, baseURL)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	Format   string              `json:"format,omitempty"`
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

func callOllamaChat(prompt, model, baseURL string) (string, error) {
	reqBody := ollamaChatRequest{
		Model: model,
		Messages: []ollamaChatMessage{
			{Role: "system", Content: "You generate Conventional Commit candidates. Return JSON only."},
			{Role: "user", Content: prompt},
		},
		Stream: false,
		Format: "json",
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	endpoint := baseURL + "/api/chat"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama api failed (%d): %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaChatResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("decode ollama response: %w", err)
	}

	return ollamaResp.Message.Content, nil
}
