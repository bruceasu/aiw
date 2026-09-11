package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

type openAIResponsesProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *openai.Client
}

func (p openAIResponsesProvider) Interactive(context.Context, Request) (Response, error) {
	return unsupportedInteractiveProvider("openai")
}

func NewOpenAIResponsesProvider(cfg Config) Provider {
	return openAIResponsesProvider{apiKey: cfg.APIKey, baseURL: cfg.BaseURL, model: cfg.Model}
}

func (p openAIResponsesProvider) Name() string { return "openai" }

func (p openAIResponsesProvider) Generate(ctx context.Context, request Request) (Response, error) {
	started := time.Now().UTC()
	apiKey := p.apiKey
	if apiKey == "" {
		return Response{}, errors.New("OPENAI_API_KEY is required")
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
		Store: openai.Bool(true),
	}
	if request.ThreadID != "" && !request.ForceNewThread {
		params.PreviousResponseID = openai.String(request.ThreadID)
	}
	response, err := client.Responses.New(ctx, params)
	if err != nil {
		return Response{ExitCode: 1, StartedAt: started, CompletedAt: time.Now().UTC()}, fmt.Errorf("openai responses provider: %w", err)
	}
	events, err := json.Marshal(response)
	if err != nil {
		return Response{}, fmt.Errorf("encode OpenAI response: %w", err)
	}
	return Response{ThreadID: response.ID, FinalOutput: response.OutputText(), Events: events,
		ExitCode: 0, StartedAt: started, CompletedAt: time.Now().UTC(),
		Metadata: map[string]string{"provider": "openai", "model": model}}, nil
}
