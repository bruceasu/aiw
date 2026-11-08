package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

type openAIResponsesProvider struct {
	apiKey    string
	baseURL   string
	model     string
	name      string
	resumable bool
	client    *openai.Client
}

func (p openAIResponsesProvider) Interactive(context.Context, Request) (Response, error) {
	return unsupportedInteractiveProvider(p.Name())
}

func NewOpenAIResponsesProvider(cfg Config) Provider {
	return openAIResponsesProvider{apiKey: cfg.APIKey, baseURL: cfg.BaseURL, model: cfg.Model, name: "openai", resumable: true}
}

func NewDeepSeekResponsesProvider(cfg Config) Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com"
	}
	if cfg.Model == "" {
		cfg.Model = "deepseek-flash"
	}
	return openAIResponsesProvider{apiKey: cfg.APIKey, baseURL: cfg.BaseURL, model: cfg.Model, name: "deepseek"}
}

func (p openAIResponsesProvider) Name() string {
	if p.name == "" {
		return "openai"
	}
	return p.name
}

func (p openAIResponsesProvider) Generate(ctx context.Context, request Request) (Response, error) {
	started := time.Now().UTC()
	apiKey := p.apiKey
	if apiKey == "" {
		return Response{}, fmt.Errorf("%s_API_KEY is required", strings.ToUpper(p.Name()))
	}
	client := p.client
	if client == nil {
		options := []option.RequestOption{option.WithAPIKey(apiKey)}
		if p.baseURL != "" {
			options = append(options, option.WithBaseURL(strings.TrimRight(p.baseURL, "/")))
		}
		created := openai.NewClient(options...)
		client = &created
	}
	model := request.Model
	if model == "" {
		model = p.model
	}
	params := responses.ResponseNewParams{
		Model: shared.ResponsesModel(model),
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(request.Prompt)},
	}
	if p.resumable {
		params.Store = openai.Bool(true)
	}
	if request.SystemPrompt != "" {
		params.Instructions = openai.String(request.SystemPrompt)
	}
	if p.resumable && request.ThreadID != "" && !request.ForceNewThread {
		params.PreviousResponseID = openai.String(request.ThreadID)
	}
	response, err := client.Responses.New(ctx, params)
	if err != nil {
		return Response{ExitCode: 1, StartedAt: started, CompletedAt: time.Now().UTC()}, fmt.Errorf("%s responses provider: %w", p.Name(), err)
	}
	events, err := json.Marshal(response)
	if err != nil {
		return Response{}, fmt.Errorf("encode %s response: %w", p.Name(), err)
	}
	completed := time.Now().UTC()
	threadID := response.ID
	if !p.resumable {
		threadID = ""
	}
	return Response{ThreadID: threadID, FinalOutput: response.OutputText(), Events: events,
		ExitCode: 0, StartedAt: started, CompletedAt: completed,
		Metadata: map[string]string{"provider": p.Name(), "model": model},
		Usage: usageFromRaw(p.Name(), model, started, completed, events)}, nil
}
