package ai

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfilesParseAndResolveFallback(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(root); err != nil { t.Fatal(err) }
	t.Setenv("AIW_ROOT", root)
	for _, key := range []string{"AIW_LLM_PROVIDER", "AIW_LLM_MODEL", "OPENAI_MODEL", "GEMINI_MODEL"} { t.Setenv(key, "") }
	config := `[ai]
provider = "openai"
model = "global-model"
[ai.profiles.fast]
provider = "gemini"
model = "fast-model"
[ai.profiles.balanced]
provider = "openai"
[ai.profiles.reasoning]
provider = ""
model = "reasoning-model"
`
	if err := os.WriteFile(filepath.Join(root, "aiw.toml"), []byte(config), 0600); err != nil { t.Fatal(err) }
	profiles, err := LoadProfiles()
	if err != nil { t.Fatal(err) }
	if len(profiles) != 1 || profiles["fast"] != (Profile{Name: "fast", Provider: "gemini", Model: "fast-model"}) {
		t.Fatalf("parsed profiles = %+v", profiles)
	}
	for _, tc := range []struct { actor, name, provider, model string }{
		{"analysis", "fast", "gemini", "fast-model"},
		{"coder", "balanced", "openai", "global-model"},
		{"tester", "balanced", "openai", "global-model"},
		{"verifier", "reasoning", "openai", "global-model"},
	} {
		t.Run(tc.actor, func(t *testing.T) {
			profile, cfg, err := ResolveActorProfile(tc.actor)
			if err != nil { t.Fatal(err) }
			if profile != (Profile{Name: tc.name, Provider: tc.provider, Model: tc.model}) || cfg.Name != tc.provider || cfg.Model != tc.model {
				t.Fatalf("selection = %+v, config provider/model = %s/%s", profile, cfg.Name, cfg.Model)
			}
		})
	}
	profile, _, err := ResolveProfile("missing")
	if err != nil || profile != (Profile{Name: "missing", Provider: "openai", Model: "global-model"}) { t.Fatalf("missing profile = %+v, %v", profile, err) }
	if DefaultProfileForActor("compiler") != "" { t.Fatal("compiler received an AI profile") }
	global, err := LoadConfig()
	if err != nil || global.Name != "openai" || global.Model != "global-model" { t.Fatalf("global resolution changed: %s/%s, %v", global.Name, global.Model, err) }
}
