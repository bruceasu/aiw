package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	codex "aiw/pkg/openai-codex"
)

func executeOpenAICodex(ctx context.Context, c Config, s *Store, p Principal, r Request, id string, created time.Time) RunResult {
	result := RunResult{Texts: []string{}, State: "failed", HTTP: 502, Code: "backend_error"}
	record := Record{SchemaVersion: 1, ID: id, Principal: p.ID, Model: r.Model, ReservedAt: created, State: "reserved"}
	content := Content{CreatedAt: created, ExpiresAt: created.Add(7 * 24 * time.Hour), Input: r.Input, Instructions: r.Instructions, Output: []string{}, State: "reserved"}
	if err := s.Reserve(record, p, c.zone); err != nil {
		result.Code = err.Error(); result.HTTP = 503
		if result.Code == "daily_limit" { result.HTTP = 429 }; if result.Code == "storage_error" { result.HTTP = 500 }
		return result
	}
	finish := func() {
		now := time.Now().UTC(); record.State = result.State
		if record.StartedAt == nil { record.State = "spawn_failed" }
		record.Usage = result.Usage; record.ErrorCode = result.Code; record.FinishedAt = &now
		content.Output = result.Texts; content.State = result.State
		if err := s.SaveContent(id, content); err != nil { result.HTTP = 500; result.Code = "storage_error"; record.ErrorCode = result.Code; if record.StartedAt != nil { result.State = "failed"; record.State = "failed" } }
		if err := s.Put(record); err != nil { result.HTTP = 500; result.Code = "storage_error"; if record.StartedAt != nil { result.State = "failed" } }
	}
	if err := s.SaveContent(id, content); err != nil { result.Code = "storage_error"; result.HTTP = 500; finish(); return result }
	if err := ctx.Err(); err != nil { result.Code = "cancelled_before_start"; result.State = "cancelled"; finish(); return result }
	model := c.Models[r.Model]
	toolData, _ := json.Marshal(r.Tools)
	startedErr := false
	response, err := codex.OpenAICodexResponses(ctx, codex.OpenAICodexResponsesRequest{
		Model: model, Prompt: r.prompt, Instructions: r.Instructions, Tools: toolData,
		ToolChoice: r.ToolChoice, JSONMode: r.jsonObject,
		OnStart: func() error {
			started := time.Now().UTC(); record.StartedAt = &started; record.State = "started"
			if err := s.Put(record); err != nil { startedErr = true; return err }
			result.Started = true
			return nil
		},
	})
	if err != nil {
		if startedErr { result.Code = "storage_error"; result.HTTP = 500
		} else if errors.Is(err, context.DeadlineExceeded) { result.Code = "timeout"; result.State = "timed_out"; result.HTTP = 504
		} else if errors.Is(err, context.Canceled) { result.Code = "cancelled"; result.State = "cancelled"
		} else { result.Code = "upstream_unavailable" }
		finish(); return result
	}
	if len(r.Tools) > 0 {
		calls := codex.OpenAICodexResponseToolCalls(response.Output)
		if len(calls) > 1 { result.Code = "invalid_tool_output"; finish(); return result }
		if len(calls) == 1 {
			call := calls[0]
			valid := false
			for _, tool := range r.Tools { if tool.Name == call.Name && validateToolArguments(tool, call.Arguments) { valid = true; break } }
			if !valid || r.ToolChoice == "none" { result.Code = "invalid_tool_output"; finish(); return result }
			result.ToolCall = &FunctionCall{ID: "fc_" + strings.TrimPrefix(newID(), "resp_"), Type: "function_call", Status: "completed", CallID: "call_" + strings.TrimPrefix(newID(), "resp_"), Name: call.Name, Arguments: call.Arguments}
		} else if r.ToolChoice == "required" { result.Code = "invalid_tool_output"; finish(); return result }
	}
	text := codex.OpenAICodexResponseText(response.Output)
	if len(text) > maxOutput { result.Code = "output_limit"; finish(); return result }
	if text != "" { result.Texts = []string{text} }
	if r.jsonObject && !validJSONObject([]byte(text)) { result.Code = "invalid_json_output"; finish(); return result }
	if len(r.Tools) == 0 && text == "" { result.Code = "empty_output"; finish(); return result }
	if response.Usage != nil { var usage Usage; if json.Unmarshal(response.Usage, &usage) == nil { result.Usage = &usage } }
	result.State = "succeeded"; result.Code = ""; result.HTTP = 200
	finish()
	return result
}
