package openspecgen

import (
	"context"
	"errors"
	"strings"
	"testing"

	"aiw/internal/ai"
	"aiw/internal/requirement"
)

type generationFakeProvider struct { generate func(context.Context, ai.Request) (ai.Response, error) }
func (p generationFakeProvider) Name() string { return "fake" }
func (p generationFakeProvider) Generate(ctx context.Context, request ai.Request) (ai.Response, error) { return p.generate(ctx, request) }
func (p generationFakeProvider) Interactive(ctx context.Context, request ai.Request) (ai.Response, error) { return p.Generate(ctx, request) }

func TestGenerationFailureBudgetSurvivesResume(t *testing.T) {
	request := requirement.GenerationRequest{SchemaVersion: 1, RequestID: "generation-1", InputDigest: "digest"}
	record := requirement.GenerationRecord{SchemaVersion: 1, RequestID: request.RequestID, InputDigest: request.InputDigest, State: requirement.GenerationPrepared}
	profiles := []ai.ArtifactProfile{
		{ID: "first", Provider: "openai", Model: "one", Config: ai.Config{Name: "openai", APIKey: "test-secret"}},
		{ID: "second", Provider: "gemini", Model: "two", Config: ai.Config{Name: "gemini"}},
	}
	profiles = append(profiles, profiles[0])
	calls, checkpoints := 0, 0
	var persisted requirement.GenerationRecord
	hooks := GenerationHooks{
		Save: func(value requirement.GenerationRecord) error { persisted = value; checkpoints++; return nil },
		CheckContext: func() error { return nil },
		NewProvider: func(cfg ai.Config) (ai.Provider, error) {
			return generationFakeProvider{generate: func(ctx context.Context, req ai.Request) (ai.Response, error) {
				calls++
				if !req.ReadOnly || req.OutputDir != "" || req.Timeout <= 0 { return ai.Response{}, errors.New("unsafe request") }
				if cfg.Name == "openai" { return ai.Response{}, errors.New("provider failed with test-secret") }
				return ai.Response{FinalOutput: "invalid JSON"}, nil
			}}, nil
		},
	}
	candidate, err := GenerateCandidate(context.Background(), request, &record, profiles, hooks)
	if err != nil || candidate != nil || calls != 2 || checkpoints < 5 || record.State != requirement.GenerationAwaitingAgent { t.Fatalf("calls=%d state=%s err=%v", calls, record.State, err) }
	if strings.Contains(record.ModelAttempts[0].Diagnostic, "test-secret") { t.Fatal("credential leaked") }
	if record.ModelAttempts[1].Status != "invalid-candidate" { t.Fatal("missing invalid candidate diagnostic") }
	record = persisted
	profiles = append(profiles, ai.ArtifactProfile{ID: "new-profile"})
	_, err = GenerateCandidate(context.Background(), request, &record, profiles, hooks)
	if err != nil || calls != 2 || len(record.ModelAttempts) != 2 { t.Fatal("resume replenished consumed budget") }
}

func TestGenerationCheckpointAndContextFailurePreventCalls(t *testing.T) {
	for _, failure := range []string{"checkpoint", "context"} {
		t.Run(failure, func(t *testing.T) {
			request := requirement.GenerationRequest{SchemaVersion: 1, RequestID: "generation-1", InputDigest: "digest"}
			record := requirement.GenerationRecord{SchemaVersion: 1, RequestID: request.RequestID, InputDigest: request.InputDigest}
			calls := 0
			_, err := GenerateCandidate(context.Background(), request, &record, []ai.ArtifactProfile{{ID: "first"}}, GenerationHooks{
				Save: func(requirement.GenerationRecord) error { if failure == "checkpoint" { return errors.New("disk failure") }; return nil },
				CheckContext: func() error { if failure == "context" { return errors.New("stale source") }; return nil },
				NewProvider: func(ai.Config) (ai.Provider, error) { calls++; return nil, errors.New("unexpected call") },
			})
			if err == nil || calls != 0 { t.Fatal("failure did not stop model calls") }
		})
	}
}

func TestBoundedGenerationCancelsUncooperativeProvider(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	entered, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	provider := generationFakeProvider{generate: func(context.Context, ai.Request) (ai.Response, error) {
		close(entered)
		<-release
		return ai.Response{}, nil
	}}
	go func() { <-entered; cancel() }()
	_, err := boundedGeneration(ctx, provider, ai.Request{})
	if !errors.Is(err, context.Canceled) { t.Fatalf("cancel result: %v", err) }
}
