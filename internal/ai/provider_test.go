package ai

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewProviderSelectsInteractiveCLIProviders(t *testing.T) {
	for _, name := range []string{"codex", "copilot"} {
		provider, err := NewProvider(Config{Name: name})
		if err != nil {
			t.Fatal(err)
		}
		if provider.Name() != name {
			t.Fatalf("provider name = %q, want %q", provider.Name(), name)
		}
	}
}

func TestInteractiveRejectsNonCLIProvider(t *testing.T) {
	provider, err := NewProvider(Config{Name: "openai"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Interactive(nil, Request{}); err == nil || err.Error() != "AI provider openai does not support interactive execution" {
		t.Fatalf("unexpected interactive error: %v", err)
	}
}

func TestUsageFromRawPreservesCompleteProviderUsage(t *testing.T) {
	started := time.Date(2026, 9, 27, 8, 0, 0, 0, time.UTC)
	completed := started.Add(2 * time.Second)
	usage := usageFromRaw("openai", "model-a", started, completed, []byte(`{"usage":{"input_tokens":12,"output_tokens":5,"total_tokens":17,"cost_amount":0.004,"cost_currency":"usd"},"output":"private response"}`))

	if usage.Version != UsageEnvelopeVersion || usage.Availability != UsageFieldKnown {
		t.Fatalf("usage envelope metadata = version %d, availability %q", usage.Version, usage.Availability)
	}
	if usage.Provider != "openai" || usage.Model != "model-a" || !usage.StartedAt.Equal(started) || !usage.CompletedAt.Equal(completed) {
		t.Fatalf("usage identity/timestamps = %+v", usage)
	}
	assertKnownToken(t, "input", usage.InputTokens, 12)
	assertKnownToken(t, "output", usage.OutputTokens, 5)
	assertKnownToken(t, "total", usage.TotalTokens, 17)
	if usage.CostAmount.State != UsageFieldKnown || usage.CostAmount.Value == nil || *usage.CostAmount.Value != json.Number("0.004") {
		t.Fatalf("cost amount = %+v", usage.CostAmount)
	}
	if usage.CostCurrency.State != UsageFieldKnown || usage.CostCurrency.Value == nil || *usage.CostCurrency.Value != "USD" {
		t.Fatalf("cost currency = %+v", usage.CostCurrency)
	}
	if string(usage.RawResponse) != `{"cost_amount":0.004,"cost_currency":"usd","input_tokens":12,"output_tokens":5,"total_tokens":17}` {
		t.Fatalf("raw evidence = %s", usage.RawResponse)
	}
	if string(usage.RawResponse) == "" || contains(string(usage.RawResponse), "private response") {
		t.Fatalf("raw evidence is empty or contains unrelated output: %s", usage.RawResponse)
	}
}

func TestUsageFromRawKeepsPartialFieldsAndUnknownCost(t *testing.T) {
	usage := usageFromRaw("copilot", "model-b", time.Time{}, time.Time{}, []byte(`{"usage":{"prompt_tokens":9,"total_tokens":9,"cost_amount":0.25}}`))
	if usage.Availability != UsageFieldKnown {
		t.Fatalf("availability = %q, want known", usage.Availability)
	}
	assertKnownToken(t, "input", usage.InputTokens, 9)
	if usage.OutputTokens.State != UsageFieldUnknown || usage.OutputTokens.Value != nil {
		t.Fatalf("missing output token field = %+v", usage.OutputTokens)
	}
	if usage.TotalTokens.State != UsageFieldKnown || usage.TotalTokens.Value == nil || *usage.TotalTokens.Value != 9 {
		t.Fatalf("total token field = %+v", usage.TotalTokens)
	}
	if usage.CostAmount.State != UsageFieldUnknown || usage.CostAmount.Value != nil || usage.CostCurrency.State != UsageFieldUnknown || usage.CostCurrency.Value != nil {
		t.Fatalf("amount without Provider currency must remain unknown: amount=%+v currency=%+v", usage.CostAmount, usage.CostCurrency)
	}
}

func TestUsageFromRawMarksInvalidAndAbsentUsage(t *testing.T) {
	invalid := usageFromRaw("gemini", "model-c", time.Time{}, time.Time{}, []byte(`{"usageMetadata":{"promptTokenCount":"bad","totalTokenCount":-1}}`))
	if invalid.InputTokens.State != UsageFieldInvalid || invalid.InputTokens.Value != nil || invalid.TotalTokens.State != UsageFieldInvalid || invalid.TotalTokens.Value != nil {
		t.Fatalf("malformed Provider fields = input %+v, total %+v", invalid.InputTokens, invalid.TotalTokens)
	}

	absent := usageFromRaw("codex", "model-d", time.Time{}, time.Time{}, []byte(`{"type":"turn.completed"}`))
	if absent.Availability != UsageFieldUnknown || absent.InputTokens.State != UsageFieldUnknown || absent.OutputTokens.State != UsageFieldUnknown || absent.TotalTokens.State != UsageFieldUnknown || absent.CostAmount.State != UsageFieldUnknown || absent.CostCurrency.State != UsageFieldUnknown {
		t.Fatalf("absent Provider usage must stay unknown: %+v", absent)
	}
}

func TestCodexCompletedTurnKeepsTokenSubsetsWithoutInventingCost(t *testing.T) {
	raw := []byte("{\"type\":\"thread.started\",\"thread_id\":\"private\"}\n" +
		"{\"type\":\"turn.completed\",\"usage\":{\"input_tokens\":120,\"cached_input_tokens\":80,\"output_tokens\":30,\"reasoning_output_tokens\":10}}\n")
	usage := usageFromRaw("codex", "gpt-6-luna", time.Time{}, time.Time{}, raw)
	assertKnownToken(t, "input", usage.InputTokens, 120)
	assertKnownToken(t, "cached input", usage.CachedInputTokens, 80)
	assertKnownToken(t, "output", usage.OutputTokens, 30)
	assertKnownToken(t, "reasoning output", usage.ReasoningOutputTokens, 10)
	if usage.TotalTokens.State != UsageFieldUnknown || usage.CostAmount.State != UsageFieldUnknown || usage.CostCurrency.State != UsageFieldUnknown {
		t.Fatalf("Codex totals or cost were invented: %+v", usage)
	}
	if string(usage.RawResponse) != `{"cached_input_tokens":80,"input_tokens":120,"output_tokens":30,"reasoning_output_tokens":10}` {
		t.Fatalf("bounded Codex usage evidence = %s", usage.RawResponse)
	}
	failed := append(raw, []byte("{\"type\":\"turn.failed\"}\n")...)
	if got := usageFromRaw("codex", "gpt-6-luna", time.Time{}, time.Time{}, failed); got.Availability != UsageFieldUnknown {
		t.Fatalf("failed final turn reused earlier completed usage: %+v", got)
	}
}

func assertKnownToken(t *testing.T, field string, usage UsageField[int64], want int64) {
	t.Helper()
	if usage.State != UsageFieldKnown || usage.Value == nil || *usage.Value != want {
		t.Fatalf("%s token field = %+v, want known %d", field, usage, want)
	}
}

func contains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
