package ask

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateAnswer(t *testing.T) {
	valid := Answer{SchemaVersion: "1.0", Status: "ok", Summary: "done", Capability: map[string]any{}, Safety: map[string]any{}}
	if err := validateAnswer(valid); err != nil {
		t.Fatalf("valid answer rejected: %v", err)
	}
	for _, answer := range []Answer{
		{SchemaVersion: "2.0", Status: "ok", Capability: map[string]any{}, Safety: map[string]any{}},
		{SchemaVersion: "1.0", Status: "unknown", Capability: map[string]any{}, Safety: map[string]any{}},
		{SchemaVersion: "1.0", Status: "ok", Safety: map[string]any{}},
	} {
		if err := validateAnswer(answer); err == nil {
			t.Fatalf("invalid answer accepted: %#v", answer)
		}
	}
}

func TestAnswerSchemaDisallowsAdditionalObjectProperties(t *testing.T) {
	schema := answerSchema()
	if schema["additionalProperties"] != false {
		t.Fatalf("answer schema must disallow additional root properties: %#v", schema)
	}
	properties := schema["properties"].(map[string]any)
	for _, name := range []string{"schema_version", "status"} {
		property := properties[name].(map[string]any)
		if property["type"] != "string" {
			t.Fatalf("%s schema must declare string type: %#v", name, property)
		}
	}
	for _, name := range []string{"capability", "safety"} {
		property := properties[name].(map[string]any)
		if property["additionalProperties"] != false {
			t.Fatalf("%s schema must disallow additional properties: %#v", name, property)
		}
		if _, ok := property["properties"].(map[string]any); !ok {
			t.Fatalf("%s schema must declare its empty property set: %#v", name, property)
		}
		if required, ok := property["required"].([]string); !ok || len(required) != 0 {
			t.Fatalf("%s schema must require no nested properties: %#v", name, property)
		}
	}
}

func TestInvalidStructuredResponseErrorIncludesReasonAndOutput(t *testing.T) {
	err := invalidStructuredResponseError("invalid answer", errors.New("missing required answer object"), `{"status":"ok"}`)
	if !strings.Contains(err, "missing required answer object") || !strings.Contains(err, `{"status":"ok"}`) {
		t.Fatalf("invalid structured response error = %q", err)
	}
}

func TestParseChatOptionsAndAllowPaths(t *testing.T) {
	opts, err := parse([]string{"--chat", "--resume", "--allow-path", "one", "--allow-path", "two", "hello", "world"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.chat || !opts.resume || opts.prompt != "hello world" || strings.Join(opts.allowPaths, ",") != "one,two" {
		t.Fatalf("unexpected options: %#v", opts)
	}
	if _, err := parse([]string{"--allow-path"}); err == nil {
		t.Fatal("missing allow path was accepted")
	}
}

func TestConfiguredExternalPromptRequiresAllowedPath(t *testing.T) {
	root := t.TempDir()
	external := filepath.Join(root, "external.txt")
	if err := os.WriteFile(external, []byte("prompt"), 0o600); err != nil {
		t.Fatal(err)
	}
	policy := &readPolicy{workspace: filepath.Join(root, "workspace"), private: filepath.Join(root, "private")}
	if _, err := policy.authorizeSystemPromptFile(external, false, false); err == nil {
		t.Fatal("unapproved external configured prompt was accepted")
	}
	policy.allowed = []string{root}
	if got, err := policy.authorizeSystemPromptFile(external, false, false); err != nil || got != external {
		t.Fatalf("allowed prompt = %q, %v", got, err)
	}
}

func TestPersistSeparatesSuccessfulAndFailedTurns(t *testing.T) {
	setAskHome(t)
	success := persist("question", "ok", "", `{"schema_version":"1.0"}`, "")
	contents, err := os.ReadFile(success)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "## answer") || !strings.Contains(string(contents), "## data") {
		t.Fatalf("successful turn missing payload: %s", contents)
	}
	failed := persist("failed question", "error", "bad response", "", "")
	contents, err = os.ReadFile(failed)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contents), "## answer") || strings.Contains(string(contents), "## data") || !strings.Contains(string(contents), "## error") {
		t.Fatalf("failed turn payload mismatch: %s", contents)
	}
}

func TestLatestSessionUsesMostRecentlyUpdatedFile(t *testing.T) {
	home := setAskHome(t)
	root := filepath.Join(home, ".aiw", "ask")
	older := filepath.Join(root, "2026-01-01", "older.md")
	newer := filepath.Join(root, "2026-01-02", "newer.md")
	for _, path := range []string{older, newer} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("# session\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	if err := os.Chtimes(older, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, now, now); err != nil {
		t.Fatal(err)
	}
	got, err := latestSession()
	if err != nil || got != newer {
		t.Fatalf("latest session = %q, %v; want %q", got, err, newer)
	}
}

func setAskHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}
