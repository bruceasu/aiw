package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const openAICodexResponsesURL = "https://chatgpt.com/backend-api/codex/responses"

type OpenAICodexResponsesRequest struct {
	Model        string
	Prompt       string
	Instructions string
	PreviousID   string
	Tools        json.RawMessage
	ToolChoice   string
	JSONMode     bool
	OnStart      func() error
}

type OpenAICodexToolCall struct {
	ID        string `json:"id"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenAICodexResponsesResult struct {
	ID         string                 `json:"id"`
	Model      string                 `json:"model"`
	Status     string                 `json:"status"`
	Output     []json.RawMessage      `json:"output"`
	Usage      json.RawMessage        `json:"usage"`
	Raw        json.RawMessage        `json:"-"`
}

// OpenAICodexResponses sends a single Responses request through the
// user-approved local PI compatibility target. It never retries a request.
func OpenAICodexResponses(ctx context.Context, request OpenAICodexResponsesRequest) (OpenAICodexResponsesResult, error) {
	credential, err := ResolveOpenAICodexCredential(ctx)
	if err != nil { return OpenAICodexResponsesResult{}, err }
	if strings.TrimSpace(request.Model) == "" || strings.TrimSpace(request.Prompt) == "" {
		return OpenAICodexResponsesResult{}, errors.New("OpenAI Codex model and prompt are required")
	}
	body := map[string]any{
		"model": request.Model,
		"input": request.Prompt,
		"stream": true,
		"store": false,
	}
	if request.Instructions != "" { body["instructions"] = request.Instructions }
	if request.PreviousID != "" { body["previous_response_id"] = request.PreviousID }
	if len(request.Tools) > 0 {
		var tools []json.RawMessage
		if err := json.Unmarshal(request.Tools, &tools); err != nil || len(tools) == 0 {
			return OpenAICodexResponsesResult{}, errors.New("OpenAI Codex tools are invalid")
		}
		body["tools"] = tools
		if request.ToolChoice != "" { body["tool_choice"] = request.ToolChoice }
		body["parallel_tool_calls"] = false
	}
	if request.JSONMode { body["text"] = map[string]any{"format": map[string]string{"type": "json_object"}} }
	encoded, err := json.Marshal(body)
	if err != nil { return OpenAICodexResponsesResult{}, errors.New("cannot encode OpenAI Codex request") }
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, openAICodexResponsesURL, bytes.NewReader(encoded))
	if err != nil { return OpenAICodexResponsesResult{}, errors.New("cannot create OpenAI Codex request") }
	httpRequest.Header.Set("Authorization", "Bearer "+credential.AccessToken)
	httpRequest.Header.Set("chatgpt-account-id", credential.AccountID)
	httpRequest.Header.Set("originator", "aiw")
	httpRequest.Header.Set("OpenAI-Beta", "responses=experimental")
	httpRequest.Header.Set("Accept", "text/event-stream")
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("User-Agent", "aiw")
	if request.OnStart != nil {
		if err := request.OnStart(); err != nil { return OpenAICodexResponsesResult{}, errors.New("cannot record OpenAI Codex request start") }
	}
	client := &http.Client{Timeout: 0, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(httpRequest)
	if err != nil {
		if ctx.Err() != nil { return OpenAICodexResponsesResult{}, ctx.Err() }
		return OpenAICodexResponsesResult{}, errors.New("OpenAI Codex request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return OpenAICodexResponsesResult{}, fmt.Errorf("OpenAI Codex request failed (HTTP %d)", response.StatusCode)
	}
	return readOpenAICodexSSE(response.Body)
}

func readOpenAICodexSSE(body io.Reader) (OpenAICodexResponsesResult, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64<<10), 8<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") { continue }
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" { continue }
		var event struct {
			Type     string          `json:"type"`
			Response json.RawMessage `json:"response"`
			Message  string          `json:"message"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil { return OpenAICodexResponsesResult{}, errors.New("OpenAI Codex returned an invalid event") }
		switch event.Type {
		case "response.completed":
			var result OpenAICodexResponsesResult
			if err := json.Unmarshal(event.Response, &result); err != nil || result.ID == "" || result.Status != "completed" {
				return OpenAICodexResponsesResult{}, errors.New("OpenAI Codex returned an incomplete response")
			}
			result.Raw = append(json.RawMessage(nil), event.Response...)
			return result, nil
		case "response.failed", "response.incomplete", "error":
			return OpenAICodexResponsesResult{}, errors.New("OpenAI Codex could not complete the request")
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) { return OpenAICodexResponsesResult{}, errors.New("OpenAI Codex response exceeded 8 MiB") }
		return OpenAICodexResponsesResult{}, errors.New("cannot read OpenAI Codex response")
	}
	return OpenAICodexResponsesResult{}, errors.New("OpenAI Codex response ended without a completion event")
}

func OpenAICodexResponseText(output []json.RawMessage) string {
	var texts []string
	for _, raw := range output {
		var item struct {
			Type    string `json:"type"`
			Content []struct { Type string `json:"type"`; Text string `json:"text"` } `json:"content"`
		}
		if json.Unmarshal(raw, &item) != nil || item.Type != "message" { continue }
		for _, part := range item.Content { if part.Type == "output_text" { texts = append(texts, part.Text) } }
	}
	return strings.Join(texts, "")
}

func OpenAICodexResponseToolCalls(output []json.RawMessage) []OpenAICodexToolCall {
	var calls []OpenAICodexToolCall
	for _, raw := range output {
		var item struct {
			Type      string `json:"type"`
			ID        string `json:"id"`
			CallID    string `json:"call_id"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}
		if json.Unmarshal(raw, &item) == nil && item.Type == "function_call" {
			calls = append(calls, OpenAICodexToolCall{ID: item.ID, CallID: item.CallID, Name: item.Name, Arguments: item.Arguments})
		}
	}
	return calls
}


