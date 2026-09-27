package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aiw/internal/ai"
)

func TestSaveAndReadTurnUsagePreservesProviderEvidenceAndExistingOutputs(t *testing.T) {
	store := NewStore(t.TempDir())
	status, err := store.Create("S-usage", "usage", t.TempDir(), "codex", "session-model", "instructions")
	if err != nil {
		t.Fatal(err)
	}
	input, output, total := int64(14), int64(6), int64(20)
	amount := json.Number("0.12")
	currency := "USD"
	started := time.Date(2026, 9, 27, 8, 10, 0, 0, time.UTC)
	completed := started.Add(time.Second)
	result := TurnResult{
		FinalOutput: "visible output", Events: []byte("{\"type\":\"completed\"}\n"), Stderr: []byte("provider diagnostic"),
		StartedAt: started, CompletedAt: completed,
		Usage: &ai.UsageEnvelope{
			Version: ai.UsageEnvelopeVersion, Availability: ai.UsageFieldKnown,
			Provider: "codex", Model: "model-1", StartedAt: started, CompletedAt: completed,
			InputTokens: ai.UsageField[int64]{State: ai.UsageFieldKnown, Value: &input},
			OutputTokens: ai.UsageField[int64]{State: ai.UsageFieldKnown, Value: &output},
			TotalTokens: ai.UsageField[int64]{State: ai.UsageFieldKnown, Value: &total},
			CostAmount: ai.UsageField[json.Number]{State: ai.UsageFieldKnown, Value: &amount},
			CostCurrency: ai.UsageField[string]{State: ai.UsageFieldKnown, Value: &currency},
			RawResponse: json.RawMessage(`{"input_tokens":14,"output_tokens":6,"total_tokens":20,"cost_amount":0.12,"cost_currency":"USD"}`),
		},
	}
	if err := SaveTurnResult(store, status, result); err != nil {
		t.Fatal(err)
	}

	usage, err := store.ReadTurnUsage("S-usage", 1, "fallback", "fallback-model")
	if err != nil {
		t.Fatal(err)
	}
	if usage.Version != ai.UsageEnvelopeVersion || usage.Provider != "codex" || usage.Model != "model-1" || usage.TotalTokens.Value == nil || *usage.TotalTokens.Value != total || usage.CostAmount.Value == nil || *usage.CostAmount.Value != amount || usage.CostCurrency.Value == nil || *usage.CostCurrency.Value != currency {
		t.Fatalf("persisted usage = %+v", usage)
	}
	if string(usage.RawResponse) != string(result.Usage.RawResponse) {
		t.Fatalf("raw Provider evidence = %s, want %s", usage.RawResponse, result.Usage.RawResponse)
	}
	assertFileContent(t, filepath.Join(store.sessionDir("S-usage"), "outputs", "0001-final.txt"), "visible output")
	assertFileContent(t, filepath.Join(store.sessionDir("S-usage"), "outputs", "0001-events.jsonl"), string(result.Events))
	assertFileContent(t, filepath.Join(store.sessionDir("S-usage"), "outputs", "0001-stderr.log"), string(result.Stderr))
}

func TestReadLegacyTurnWithoutUsageReturnsUnknownWithoutRewriting(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Create("S-legacy", "legacy", t.TempDir(), "codex", "old-model", "instructions"); err != nil {
		t.Fatal(err)
	}
	usagePath := filepath.Join(store.sessionDir("S-legacy"), "outputs", "0001-usage.json")
	usage, err := store.ReadTurnUsage("S-legacy", 1, "copilot", "override-model")
	if err != nil {
		t.Fatal(err)
	}
	if usage.Version != ai.UsageEnvelopeVersion || usage.Availability != ai.UsageFieldUnknown || usage.Provider != "copilot" || usage.Model != "override-model" || usage.InputTokens.State != ai.UsageFieldUnknown || usage.OutputTokens.State != ai.UsageFieldUnknown || usage.TotalTokens.State != ai.UsageFieldUnknown || usage.CostAmount.State != ai.UsageFieldUnknown || usage.CostCurrency.State != ai.UsageFieldUnknown {
		t.Fatalf("legacy turn usage = %+v", usage)
	}
	if _, err := os.Stat(usagePath); !os.IsNotExist(err) {
		t.Fatalf("legacy usage read rewrote or unexpectedly found sidecar (stat error %v)", err)
	}
}

func TestReadExpiredTurnUsageRemovesOnlySanitizedRawEvidence(t *testing.T) {
	store := NewStore(t.TempDir())
	status, err := store.Create("S-expiry", "expiry", t.TempDir(), "codex", "session-model", "instructions")
	if err != nil {
		t.Fatal(err)
	}
	input, output, total := int64(31), int64(12), int64(43)
	started := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	completed := started.Add(time.Second)
	result := TurnResult{
		FinalOutput: "completed response",
		StartedAt: started, CompletedAt: completed,
		Usage: &ai.UsageEnvelope{
			Version: ai.UsageEnvelopeVersion, Availability: ai.UsageFieldKnown,
			Provider: "codex", Model: "model-expiry", StartedAt: started, CompletedAt: completed,
			InputTokens: ai.UsageField[int64]{State: ai.UsageFieldKnown, Value: &input},
			OutputTokens: ai.UsageField[int64]{State: ai.UsageFieldKnown, Value: &output},
			TotalTokens: ai.UsageField[int64]{State: ai.UsageFieldKnown, Value: &total},
			CostAmount: ai.UsageField[json.Number]{State: ai.UsageFieldUnknown},
			CostCurrency: ai.UsageField[string]{State: ai.UsageFieldUnknown},
			RawResponse: json.RawMessage(`{"input_tokens":31,"output_tokens":12,"total_tokens":43,"api_key":"secret","prompt":"private prompt"}`),
		},
	}
	if err := SaveTurnResult(store, status, result); err != nil {
		t.Fatal(err)
	}

	usagePath := filepath.Join(store.sessionDir("S-expiry"), "outputs", "0001-usage.json")
	var persisted ai.UsageEnvelope
	content, err := os.ReadFile(usagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &persisted); err != nil {
		t.Fatal(err)
	}
	if string(persisted.RawResponse) != `{"input_tokens":31,"output_tokens":12,"total_tokens":43}` {
		t.Fatalf("sanitized raw evidence = %s", persisted.RawResponse)
	}
	wantDigest := sha256.Sum256(persisted.RawResponse)
	if persisted.RawResponseDigest != hex.EncodeToString(wantDigest[:]) {
		t.Fatalf("raw evidence digest = %q, want digest of sanitized fragment", persisted.RawResponseDigest)
	}
	if !persisted.RawResponseExpiresAt.Equal(completed.Add(ai.UsageRawEvidenceRetention)) {
		t.Fatalf("raw evidence expiry = %s, want %s", persisted.RawResponseExpiresAt, completed.Add(ai.UsageRawEvidenceRetention))
	}
	wantDigestText := persisted.RawResponseDigest
	persisted.RawResponseExpiresAt = time.Now().UTC().Add(-time.Second)
	expiredContent, err := json.Marshal(persisted)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(usagePath, expiredContent, 0o600); err != nil {
		t.Fatal(err)
	}

	usage, err := store.ReadTurnUsage("S-expiry", 1, "fallback", "fallback-model")
	if err != nil {
		t.Fatal(err)
	}
	if len(usage.RawResponse) != 0 || usage.RawResponseDigest != wantDigestText {
		t.Fatalf("expired evidence raw=%s digest=%q, want raw removed and digest %q", usage.RawResponse, usage.RawResponseDigest, wantDigestText)
	}
	if usage.Provider != "codex" || usage.Model != "model-expiry" || usage.Availability != ai.UsageFieldKnown || usage.InputTokens.Value == nil || *usage.InputTokens.Value != input || usage.OutputTokens.Value == nil || *usage.OutputTokens.Value != output || usage.TotalTokens.Value == nil || *usage.TotalTokens.Value != total || usage.CostAmount.State != ai.UsageFieldUnknown || usage.CostCurrency.State != ai.UsageFieldUnknown || !usage.CompletedAt.Equal(completed) {
		t.Fatalf("normalized usage changed after expiry: %+v", usage)
	}

	content, err = os.ReadFile(usagePath)
	if err != nil {
		t.Fatal(err)
	}
	var stored ai.UsageEnvelope
	if err := json.Unmarshal(content, &stored); err != nil {
		t.Fatal(err)
	}
	if len(stored.RawResponse) != 0 || stored.RawResponseDigest != wantDigestText || stored.TotalTokens.Value == nil || *stored.TotalTokens.Value != total || !stored.CompletedAt.Equal(completed) {
		t.Fatalf("persisted post-expiry usage lost normalized history or digest: %+v", stored)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf("%s = %q, want %q", path, content, want)
	}
}
