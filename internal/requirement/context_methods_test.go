package requirement

import (
	"strings"
	"testing"
)

func methodSuggestion(input string) *ConversationMethodSuggestion {
	return &ConversationMethodSuggestion{
		Domain: "finance", Gap: "metric", Reason: "The metric definition is disputed",
		Source: "user-input", SourceDigest: digest([]byte(input)), Quote: input,
	}
}

func TestConversationContextGenericMethod(t *testing.T) {
	inTempDir(t)
	snapshot, err := LoadConversationContext("", "Notify me when a task completes", nil)
	if err != nil || len(snapshot.Methods) != 1 || snapshot.MethodSelection.Primary != "generic-discovery" {
		t.Fatalf("generic routing failed: %#v, %v", snapshot, err)
	}
	method := snapshot.Methods[0]
	if method.Content != discoveryBaseline || method.Digest != digest([]byte(method.Content)) || method.Status != "loaded" {
		t.Fatalf("baseline provenance missing: %#v", method)
	}
	if snapshot.UsedBytes != len(snapshot.UserInput)+len(discoveryBaseline) {
		t.Fatalf("baseline not counted: %#v", snapshot)
	}
}

func TestConversationContextLoadsSelectedMethods(t *testing.T) {
	inTempDir(t)
	input := "Our financial settlement metric has two conflicting definitions"
	suggestion := methodSuggestion(input)
	suggestion.TermsUnclear = true
	writeContextDocument(t, ".agents/skills/finance-metric-brief/SKILL.md", "Discuss metric definitions and cut-off rules")
	writeContextDocument(t, ".agents/skills/domain-modeling/SKILL.md", "Clarify terms in Requirement mode")
	writeContextDocument(t, "docs/background.md", strings.Repeat("x", 4096))
	snapshot, err := LoadConversationContextWithOptions("", input, nil, ConversationContextOptions{
		MaxBytes: 2048, Method: suggestion, BackgroundPaths: []string{"docs/background.md"},
	})
	if err != nil || len(snapshot.Methods) != 3 || snapshot.MethodSelection.Status != "loaded" {
		t.Fatalf("selected methods not loaded: %#v, %v", snapshot, err)
	}
	if snapshot.MethodSelection.Primary != "finance-metric-brief" || snapshot.MethodSelection.Supplement != "domain-modeling" {
		t.Fatalf("wrong selection: %#v", snapshot.MethodSelection)
	}
	for _, source := range snapshot.Methods {
		if source.Status != "loaded" || source.Digest != digest([]byte(source.Content)) {
			t.Fatalf("missing actual content or version: %#v", source)
		}
	}
	if snapshot.Sources[0].Status != "omitted" || snapshot.UsedBytes > snapshot.BudgetBytes {
		t.Fatal("optional background took the method budget")
	}
	suggestion.Domain = "changed"
	if snapshot.MethodSelection.Suggestion.Domain != "finance" {
		t.Fatal("snapshot aliases caller suggestion")
	}
	prompt, err := snapshot.Prompt()
	if err != nil || !strings.Contains(prompt, "Discuss metric definitions") {
		t.Fatalf("actual method missing from prompt: %v", err)
	}
}

func TestConversationContextRejectsInvalidMethodEvidence(t *testing.T) {
	for _, name := range []string{"unknown-gap", "unknown-domain", "stale", "missing-source", "false-quote", "oversized"} {
		t.Run(name, func(t *testing.T) {
			inTempDir(t)
			input := "Discuss the financial metric"
			suggestion := methodSuggestion(input)
			switch name {
			case "unknown-gap":
				suggestion.Gap = "../../release"
			case "unknown-domain":
				suggestion.Domain = "invented"
			case "stale":
				suggestion.SourceDigest = "old"
			case "missing-source":
				suggestion.Source = "missing.md"
			case "false-quote":
				suggestion.Quote = "not in input"
			case "oversized":
				suggestion.Reason = strings.Repeat("x", 4097)
			}
			snapshot, err := LoadConversationContextWithOptions("", input, nil, ConversationContextOptions{Method: suggestion})
			if err != nil || len(snapshot.Methods) != 1 || snapshot.MethodSelection.Status != "degraded" {
				t.Fatalf("invalid recommendation accepted: %#v, %v", snapshot, err)
			}
		})
	}
}

func TestConversationContextUnavailableMethodBlocks(t *testing.T) {
	for _, name := range []string{"missing", "empty", "over-budget"} {
		t.Run(name, func(t *testing.T) {
			inTempDir(t)
			input := "Discuss the financial metric"
			if name != "missing" {
				content := " "
				if name == "over-budget" {
					content = strings.Repeat("x", 4096)
				}
				writeContextDocument(t, ".agents/skills/finance-metric-brief/SKILL.md", content)
			}
			snapshot, err := LoadConversationContextWithOptions("", input, nil, ConversationContextOptions{
				MaxBytes: 2048, Method: methodSuggestion(input),
			})
			if err == nil || snapshot.SourcesLoaded || snapshot.MethodSelection.Status != "blocked" {
				t.Fatalf("unavailable method accepted: %#v, %v", snapshot, err)
			}
			if _, err := snapshot.Prompt(); err == nil {
				t.Fatal("missing method still allowed a usable prompt")
			}
		})
	}
}

func TestConversationContextNonFinanceAndBaselineBudget(t *testing.T) {
	inTempDir(t)
	input := "Notify on task completion"
	for _, domain := range []string{"generic", "unknown"} {
		suggestion := methodSuggestion(input)
		suggestion.Domain = domain
		snapshot, err := LoadConversationContextWithOptions("", input, nil, ConversationContextOptions{Method: suggestion})
		if err != nil || len(snapshot.Methods) != 1 || snapshot.MethodSelection.Primary != "generic-discovery" {
			t.Fatalf("non-finance input required finance skill: %#v, %v", snapshot, err)
		}
	}
	snapshot, err := LoadConversationContextWithOptions("", input, nil, ConversationContextOptions{MaxBytes: len(input)})
	if err == nil || snapshot.SourcesLoaded || snapshot.Methods[0].Status != "omitted" {
		t.Fatalf("baseline silently omitted: %#v, %v", snapshot, err)
	}
}
