package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactProfilesIsolationAndBudget(t *testing.T) {
	values := map[string]string{
		"ai.provider": "openai", "ai.model": "primary", "ai.base_url": "https://primary.example/v1", "ai.api_key": "primary-secret",
		"ai.command": "primary-command", "ai.gemini_api_key": "backup-secret",
		"ai.artifact_generation.profiles": `["first", "backup", "same"]`,
		"ai.profiles.first.provider": "openai", "ai.profiles.first.model": "primary",
		"ai.profiles.backup.provider": "gemini", "ai.profiles.backup.model": "backup",
		"ai.profiles.same.provider": "gemini", "ai.profiles.same.model": "backup",
	}
	profiles, err := artifactProfiles(values, func(string) string { return "" })
	if err != nil { t.Fatal(err) }
	if len(profiles) != 2 { t.Fatalf("expected deduplicated budget, got %d", len(profiles)) }
	backup := profiles[1]
	if backup.Config.APIKey != "backup-secret" || backup.Model != "backup" || backup.Endpoint != "https://generativelanguage.googleapis.com" || backup.Command != "" {
		t.Fatal("backup inherited primary provider configuration")
	}
	encoded, err := json.Marshal(profiles)
	if err != nil || strings.Contains(string(encoded), "secret") { t.Fatal("credentials entered audit") }
	values["ai.gemini_api_key"] = "rotated-secret"
	rotated, err := artifactProfiles(values, func(string) string { return "" })
	if err != nil || rotated[1].ID != backup.ID { t.Fatal("credential rotation replenished budget identity") }
}

func TestArtifactProfileConfigParsingAndFailClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aiw.toml")
	if err := os.WriteFile(path, []byte("[ai.artifact_generation]\nprofiles = [\n  'one', # comment\n  \"two\",\n]\n"), 0600); err != nil { t.Fatal(err) }
	values := map[string]string{}
	if err := mergeConfigFile(values, path); err != nil { t.Fatal(err) }
	names, err := artifactProfileNames(values["ai.artifact_generation.profiles"])
	if err != nil || len(names) != 2 || names[0] != "one" || names[1] != "two" { t.Fatalf("names = %v, err = %v", names, err) }
	for _, raw := range []string{`"one"`, `[one]`, `["one" "two"]`, `["one"`, `null`, `[""]`} {
		if _, err := artifactProfileNames(raw); err == nil { t.Fatalf("accepted invalid array: %s", raw) }
	}
	for _, values := range []map[string]string{
		{}, {"ai.provider": "auto"}, {"cz.provider": "openai"},
		{"ai.provider": "openai", "ai.artifact_generation.profiles": "[]"},
	} {
		profiles, err := artifactProfiles(values, func(string) string { return "" })
		if err != nil || len(profiles) != 0 { t.Fatalf("expected handoff, profiles = %d, err = %v", len(profiles), err) }
	}
}

func TestArtifactGlobalProviderOverrideDoesNotCarryFileCredentials(t *testing.T) {
	profiles, err := artifactProfiles(map[string]string{
		"ai.provider": "openai", "ai.api_key": "wrong-secret", "ai.base_url": "https://wrong.example", "ai.model": "wrong-model",
	}, func(key string) string { if key == "AIW_LLM_PROVIDER" { return "gemini" }; return "" })
	if err != nil { t.Fatal(err) }
	if len(profiles) != 1 || profiles[0].Config.APIKey != "" || profiles[0].Model == "wrong-model" || profiles[0].Endpoint == "https://wrong.example" { t.Fatal("provider override carried file settings") }
}

func TestArtifactCodexUsesResolvedModel(t *testing.T) {
	p := cliProvider{name: "codex"}
	args := p.args(Request{Phase: "artifact-generation", Model: "selected-model", ReadOnly: true})
	if !strings.Contains(strings.Join(args, " "), "--model selected-model") { t.Fatal("artifact model was not passed to Codex") }
	legacy := p.args(Request{Model: "selected-model"})
	if strings.Contains(strings.Join(legacy, " "), "--model") { t.Fatal("changed unrelated CLI model behavior") }
}
