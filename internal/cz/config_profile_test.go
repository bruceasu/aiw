package cz

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCzConfigUsesFastProfileAsAnAtomicSelection(t *testing.T) {
	root := isolatedConfigRoot(t)
	config := `[ai]
provider = "openai"
model = "global-model"
base_url = "https://global.example/v1"
api_key = "global-key"
openai_model = "global-openai-model"
openai_base_url = "https://global-openai.example/v1"
openai_api_key = "global-openai-key"

[ai.profiles.fast]
provider = "gemini"
model = "fast-gemini-model"
base_url = "https://fast-gemini.example/v1"
api_key = "fast-gemini-key"
gemini_base_url = "https://fast-gemini-specific.example/v1"
gemini_api_key = "fast-gemini-specific-key"
`
	writeConfig(t, root, config)

	cfg, err := loadCzConfig(czOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMProvider != "gemini" || cfg.LLMModel != "fast-gemini-model" {
		t.Fatalf("profile provider/model = %q/%q, want gemini/fast-gemini-model", cfg.LLMProvider, cfg.LLMModel)
	}
	if cfg.APIBaseURL != "https://fast-gemini-specific.example/v1" || cfg.APIKey != "fast-gemini-specific-key" {
		t.Fatalf("profile connection = %q/%q", cfg.APIBaseURL, cfg.APIKey)
	}
}

func TestLoadCzConfigFallsBackToTopLevelWhenFastProfileIsAbsent(t *testing.T) {
	root := isolatedConfigRoot(t)
	config := `[ai]
provider = "openai"
model = "global-model"
base_url = "https://global.example/v1"
api_key = "global-key"
openai_model = "global-openai-model"
openai_base_url = "https://global-openai.example/v1"
openai_api_key = "global-openai-key"
`
	writeConfig(t, root, config)

	cfg, err := loadCzConfig(czOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMProvider != "openai" || cfg.LLMModel != "global-openai-model" {
		t.Fatalf("global provider/model = %q/%q, want openai/global-openai-model", cfg.LLMProvider, cfg.LLMModel)
	}
	if cfg.APIBaseURL != "https://global-openai.example/v1" || cfg.APIKey != "global-openai-key" {
		t.Fatalf("global connection = %q/%q", cfg.APIBaseURL, cfg.APIKey)
	}
}

func isolatedConfigRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	t.Setenv("AIW_ROOT", root)
	for _, key := range []string{
		"AIW_LLM_PROVIDER", "AIW_LLM_MODEL", "AIW_LLM_BASE_URL", "AIW_LLM_API_KEY",
		"OPENAI_MODEL", "OPENAI_BASE_URL", "OPENAI_API_KEY",
		"GEMINI_MODEL", "GEMINI_BASE_URL", "GEMINI_API_KEY",
		"OLLAMA_MODEL", "OLLAMA_BASE_URL", "OLLAMA_API_KEY",
		"LLAMACPP_MODEL", "LLAMACPP_BASE_URL", "LLAMACPP_API_KEY",
	} {
		t.Setenv(key, "")
	}
	return root
}

func writeConfig(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "aiw.toml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
