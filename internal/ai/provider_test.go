package ai

import "testing"

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
