package ai

import (
	"context"
	"errors"
	"time"

	codex "aiw/pkg/openai-codex"
)

type openAICodexResponsesProvider struct{ model string }

func NewOpenAICodexResponsesProvider(cfg Config) Provider {
	return openAICodexResponsesProvider{model: cfg.Model}
}

func (openAICodexResponsesProvider) Name() string { return "openai-codex" }

func (p openAICodexResponsesProvider) Interactive(context.Context, Request) (Response, error) {
	return unsupportedInteractiveProvider("openai-codex")
}

func (p openAICodexResponsesProvider) Generate(ctx context.Context, request Request) (Response, error) {
	started := time.Now().UTC()
	model := request.Model
	if model == "" { model = p.model }
	if model == "" { return Response{}, errors.New("OpenAI Codex model is required") }
	result, err := codex.OpenAICodexResponses(ctx, codex.OpenAICodexResponsesRequest{
		Model: model, Prompt: request.Prompt, Instructions: request.SystemPrompt,
		PreviousID: request.ThreadID,
	})
	if err != nil { return Response{ExitCode: 1, StartedAt: started, CompletedAt: time.Now().UTC()}, err }
	completed := time.Now().UTC()
	return Response{
		ThreadID: result.ID, FinalOutput: codex.OpenAICodexResponseText(result.Output), Events: result.Raw,
		ExitCode: 0, StartedAt: started, CompletedAt: completed,
		Metadata: map[string]string{"provider": "openai-codex", "model": model},
		Usage: usageFromRaw("openai-codex", model, started, completed, result.Raw),
	}, nil
}
