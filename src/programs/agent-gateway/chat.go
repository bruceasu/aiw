package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type chatMessage struct {
	Role string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model string `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream bool `json:"stream,omitempty"`
	N *int `json:"n,omitempty"`
}

func decodeChatRequest(data []byte) (Request, error) {
	var chat chatRequest
	if err := strictDecode(data, &chat); err != nil {
		return Request{}, errors.New("invalid or unsupported chat request fields")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return Request{}, errors.New("invalid chat request")
	}
	for key, value := range fields {
		switch key {
		case "model", "messages", "stream", "n":
		default:
			return Request{}, errors.New("unsupported chat request field")
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return Request{}, errors.New("null chat request fields are unsupported")
		}
	}
	if strings.TrimSpace(chat.Model) == "" || len(chat.Messages) == 0 || chat.Stream || (chat.N != nil && *chat.N != 1) {
		return Request{}, errors.New("unsupported chat model or execution options")
	}
	instructions := []string{}
	dialogue := []chatMessage{}
	hasUser := false
	var messageFields []map[string]json.RawMessage
	if err := json.Unmarshal(fields["messages"], &messageFields); err != nil {
		return Request{}, errors.New("invalid chat messages")
	}
	for index, message := range chat.Messages {
		for key, value := range messageFields[index] {
			if key != "role" && key != "content" {
				return Request{}, errors.New("unsupported chat message field")
			}
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return Request{}, errors.New("null chat message fields are unsupported")
			}
		}
		if strings.TrimSpace(message.Content) == "" {
			return Request{}, errors.New("chat messages must contain nonempty text")
		}
		switch message.Role {
		case "system", "developer":
			if len(dialogue) != 0 {
				return Request{}, errors.New("chat instructions must precede the conversation")
			}
			instructions = append(instructions, message.Content)
		case "user", "assistant":
			dialogue = append(dialogue, message)
			if message.Role == "user" { hasUser = true }
		default:
			return Request{}, errors.New("unsupported chat message role")
		}
	}
	if !hasUser {
		return Request{}, errors.New("chat messages require user text")
	}
	// Use the same conversation encoding and size checks as Responses.
	mapped, err := json.Marshal(map[string]any{
		"model": chat.Model, "input": dialogue, "instructions": strings.Join(instructions, "\n\n"),
	})
	if err != nil { return Request{}, errors.New("cannot encode chat input") }
	return decodeRequest(mapped)
}

func chatCompletionObject(id string, created time.Time, request Request, texts []string, usage *Usage) map[string]any {
	var chatUsage any
	if usage != nil {
		chatUsage = map[string]any{
			"prompt_tokens": usage.InputTokens, "completion_tokens": usage.OutputTokens, "total_tokens": usage.TotalTokens,
		}
	}
	return map[string]any{
		"id": "chatcmpl_" + strings.TrimPrefix(id, "resp_"), "object": "chat.completion",
		"created": created.Unix(), "model": request.Model,
		"choices": []any{map[string]any{
			"index": 0, "message": map[string]any{"role": "assistant", "content": strings.Join(texts, "\n")},
			"finish_reason": "stop", "logprobs": nil,
		}},
		"usage": chatUsage,
	}
}
