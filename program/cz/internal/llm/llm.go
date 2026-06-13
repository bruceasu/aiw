package llm

import (
	czdata "aiw-cz/internal/cz"
	"aiw-cz/internal/envx"
	"aiw-cz/internal/fsx"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func RunLLM(prompt string, cfg czdata.Config) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.LLMProvider))
	switch provider {
	case "gemini":
		return RunGemini(prompt, cfg)
	case "openai":
		return RunOpenAI(prompt, cfg)
	case "ollama":
		return RunOllama(prompt, cfg)
	default:
		// Default to ollama if not specified
		return RunOllama(prompt, cfg)
	}
}

func resolveLLMValue(configValue, envKey string, cwdEnv, aiwEnv, exeEnv map[string]string, defaultValue string) (string, string) {
	if v := strings.TrimSpace(configValue); v != "" {
		return v, "config"
	}
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v, "env"
	}
	if v := strings.TrimSpace(cwdEnv[envKey]); v != "" {
		return v, "cwd .env"
	}
	if v := strings.TrimSpace(aiwEnv[envKey]); v != "" {
		return v, "aiw_root .env"
	}
	if v := strings.TrimSpace(exeEnv[envKey]); v != "" {
		return v, "exe .env"
	}
	if defaultValue != "" {
		return defaultValue, "default"
	}
	return "", "missing"
}

func loadLLMEnvFromDotEnv() (map[string]string, map[string]string, map[string]string, error) {
	exeLoader := &envx.Loader{Env: map[string]string{}}
	aiwLoader := &envx.Loader{Env: map[string]string{}}
	cwdLoader := &envx.Loader{Env: map[string]string{}}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		exeEnv := filepath.Join(exeDir, ".env")
		if fsx.Exists(exeEnv) {
			if err := exeLoader.ParseFile(exeEnv); err != nil {
				return nil, nil, nil, fmt.Errorf("parse %s: %w", exeEnv, err)
			}
		}
	}

	if aiwRoot := os.Getenv("AIW_ROOT"); aiwRoot != "" {
		aiwEnv := filepath.Join(aiwRoot, ".env")
		if fsx.Exists(aiwEnv) {
			if err := aiwLoader.ParseFile(aiwEnv); err != nil {
				return nil, nil, nil, fmt.Errorf("parse %s: %w", aiwEnv, err)
			}
		}
	}

	if wd, err := os.Getwd(); err == nil {
		wdEnv := filepath.Join(wd, ".env")
		if fsx.Exists(wdEnv) {
			if err := cwdLoader.ParseFile(wdEnv); err != nil {
				return nil, nil, nil, fmt.Errorf("parse %s: %w", wdEnv, err)
			}
		}
	}

	return cwdLoader.Env, aiwLoader.Env, exeLoader.Env, nil
}

func shouldDebugSource(cfg czdata.Config) bool {
	if cfg.DebugSource {
		return true
	}
	v := strings.ToLower(strings.TrimSpace(os.Getenv("AIW_CZ_DEBUG")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func printLLMDebugSource(key, value, source string, secret bool) {
	display := value
	if secret {
		display = maskSecret(value)
	}
	fmt.Fprintf(os.Stderr, "[cz llm debug] %s=%s (source=%s)\n", key, display, source)
}

func maskSecret(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "<empty>"
	}
	if len(v) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(v)-4) + v[len(v)-4:]
}

func ParseLLMCandidates(out string) ([]czdata.Draft, error) {
	if cands, ok := parseCandidatesJSON(strings.TrimSpace(out)); ok {
		return cands, nil
	}

	chunks := strings.Split(out, "```")
	for i := 1; i < len(chunks); i += 2 {
		block := strings.TrimSpace(chunks[i])
		block = strings.TrimPrefix(block, "json")
		if cands, ok := parseCandidatesJSON(strings.TrimSpace(block)); ok {
			return cands, nil
		}
	}

	start := strings.Index(out, "{")
	end := strings.LastIndex(out, "}")
	if start >= 0 && end > start {
		candidate := out[start : end+1]
		if cands, ok := parseCandidatesJSON(candidate); ok {
			return cands, nil
		}
	}

	lines := strings.Split(out, "\n")
	var cands []czdata.Draft
	for _, ln := range lines {
		ln = cleanCandidateLine(ln)
		if ln == "" {
			continue
		}
		if d, ok := ParseConventionalHeader(ln); ok {
			cands = append(cands, d)
		}
	}
	if len(cands) == 0 {
		return nil, errors.New("invalid output")
	}
	return cands, nil
}

func parseCandidatesJSON(raw string) ([]czdata.Draft, bool) {
	if raw == "" {
		return nil, false
	}
	var resp czdata.LLMResponse
	if err := json.Unmarshal([]byte(raw), &resp); err == nil && len(resp.Candidates) > 0 {
		return resp.Candidates, true
	}

	var list []czdata.Draft
	if err := json.Unmarshal([]byte(raw), &list); err == nil && len(list) > 0 {
		return list, true
	}
	return nil, false
}

func cleanCandidateLine(line string) string {
	v := strings.TrimSpace(line)
	v = strings.TrimPrefix(v, "-")
	v = strings.TrimPrefix(v, "*")
	v = strings.TrimPrefix(v, "•")
	v = strings.TrimSpace(v)

	if idx := strings.Index(v, ")"); idx > 0 {
		prefix := strings.TrimSpace(v[:idx])
		if _, err := strconv.Atoi(prefix); err == nil {
			v = strings.TrimSpace(v[idx+1:])
		}
	}
	if idx := strings.Index(v, "."); idx > 0 {
		prefix := strings.TrimSpace(v[:idx])
		if _, err := strconv.Atoi(prefix); err == nil {
			v = strings.TrimSpace(v[idx+1:])
		}
	}
	return v
}

func ParseConventionalHeader(line string) (czdata.Draft, bool) {
	i := strings.Index(line, ":")
	if i <= 0 || i+1 >= len(line) {
		return czdata.Draft{}, false
	}

	left := strings.TrimSpace(line[:i])
	subject := strings.TrimSpace(line[i+1:])
	if left == "" || subject == "" {
		return czdata.Draft{}, false
	}

	left = strings.TrimSuffix(left, "!")
	typePart := left
	scope := ""
	if l := strings.Index(left, "("); l >= 0 {
		r := strings.LastIndex(left, ")")
		if r <= l || r != len(left)-1 {
			return czdata.Draft{}, false
		}
		typePart = strings.TrimSpace(left[:l])
		scope = strings.TrimSpace(left[l+1 : r])
		if scope == "" {
			return czdata.Draft{}, false
		}
	}

	typePart = strings.ToLower(strings.TrimSpace(typePart))
	if typePart == "" || strings.ContainsAny(typePart, " `\t") {
		return czdata.Draft{}, false
	}
	if _, ok := conventionalTypeSet()[typePart]; !ok {
		return czdata.Draft{}, false
	}

	return czdata.Draft{Type: typePart, Scope: scope, Subject: subject}, true
}

func conventionalTypeSet() map[string]struct{} {
	set := map[string]struct{}{}
	for _, t := range czdata.DefaultConfig().Types {
		set[t.Value] = struct{}{}
	}
	return set
}
