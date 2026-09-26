package cz

import (
	"embed"
	"fmt"
	"os"
	"strings"
)

//go:embed locales/*.toml
var localeFiles embed.FS

func ParseLocale(value string) (string, error) {
	value = normalizeLocale(value)
	switch value {
	case "en", "zh", "ja":
		return value, nil
	}
	return "", fmt.Errorf("unsupported built-in language %q", value)
}

func ApplyConfiguredLocale(cfg *Config, locale string) error {
	requested := locale
	locale = normalizeLocale(locale)
	if locale == "" {
		return fmt.Errorf("unsupported language %q; use built-in en, zh, ja or configure [cz.locales.en]", requested)
	}
	if _, err := ParseLocale(locale); err == nil {
		if cfg == nil || cfg.Language != locale {
			ApplyLocale(cfg, locale)
		}
	} else {
		if !isConfiguredLocale(cfg, locale) {
			return fmt.Errorf("unsupported language %q; use built-in en, zh, ja or configure [cz.locales.%s]", locale, locale)
		}
		ApplyLocale(cfg, "en")
	}
	override, ok := cfg.Locales[locale]
	if !ok {
		return nil
	}
	for key, value := range override.Messages {
		setMessage(&cfg.Messages, key, value)
	}
	if override.HasTypes && len(override.Types) > 0 {
		cfg.Types = mergeTypesByValue(cfg.Types, override.Types)
	}
	if override.HasScopes && len(override.Scopes) > 0 {
		cfg.Scopes = mergeScopesByValue(cfg.Scopes, override.Scopes)
	}
	cfg.Language = locale
	return nil
}

func normalizeLocale(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	if i := strings.IndexByte(value, '.'); i >= 0 {
		value = value[:i]
	}
	if i := strings.IndexAny(value, "_-@"); i >= 0 {
		value = value[:i]
	}
	return value
}

func isConfiguredLocale(cfg *Config, locale string) bool {
	if cfg == nil {
		return false
	}
	locale = normalizeLocale(locale)
	if locale == "" {
		return false
	}
	_, ok := cfg.Locales[locale]
	return ok
}

func detectedLocale(cfg *Config) string {
	for _, envName := range []string{"LC_ALL", "LC_MESSAGES", "LANGUAGE", "LANG"} {
		value := os.Getenv(envName)
		if value == "" {
			continue
		}
		if envName == "LANGUAGE" {
			for _, candidate := range strings.Split(value, ":") {
				if locale, ok := selectableLocale(cfg, candidate); ok {
					return locale
				}
			}
			continue
		}
		if locale, ok := selectableLocale(cfg, value); ok {
			return locale
		}
	}
	return "en"
}

func selectableLocale(cfg *Config, value string) (string, bool) {
	locale := normalizeLocale(value)
	if locale == "" {
		return "", false
	}
	if _, err := ParseLocale(locale); err == nil {
		return locale, true
	}
	if isConfiguredLocale(cfg, locale) {
		return locale, true
	}
	return "", false
}

func ApplyLocale(cfg *Config, locale string) {
	data, err := localeFiles.ReadFile("locales/" + locale + ".toml")
	if err != nil { return }
	cfg.Language = locale
	cfg.Types = nil
	section, current := "", -1
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") { continue }
		if line == "[messages]" { section="messages"; continue }
		if line == "[[types]]" { section="types"; cfg.Types=append(cfg.Types, Type{}); current=len(cfg.Types)-1; continue }
		parts:=strings.SplitN(line,"=",2); if len(parts)!=2 { continue }
		key:=strings.TrimSpace(parts[0]); value:=strings.Trim(strings.TrimSpace(parts[1]), `"`)
		if section=="messages" { setMessage(&cfg.Messages,key,value) }
		if section=="types" && current>=0 { if key=="value" { cfg.Types[current].Value=value } else if key=="name" { cfg.Types[current].Name=value } }
	}
}
func setMessage(m *Messages, key, value string) {
	switch key {
	case "type": m.Type=value; case "scope": m.Scope=value; case "custom_scope", "customScope": m.CustomScope=value; case "subject": m.Subject=value; case "body": m.Body=value; case "breaking": m.Breaking=value; case "footer_prefixes", "footerPrefixesSelect": m.FooterPrefixes=value; case "custom_footer_prefix", "customFooterPrefix": m.CustomFooterPrefix=value; case "footer": m.Footer=value; case "confirm_commit", "confirmCommit": m.ConfirmCommit=value
	case "select_type": m.SelectType=value; case "select_prefix": m.SelectPrefix=value; case "subject_required": m.SubjectRequired=value; case "subject_too_long": m.SubjectTooLong=value; case "invalid_type":m.InvalidType=value; case "invalid_selection":m.InvalidSelection=value; case "preview":m.Preview=value; case "action":m.Action=value; case "candidate":m.Candidate=value; case "regenerate":m.Regenerate=value; case "commit":m.Commit=value; case "edit":m.Edit=value; case "cancel":m.Cancel=value; case "aborted":m.Aborted=value; case "editing":m.Editing=value; case "done_editing":m.DoneEditing=value; case "empty_scope":m.EmptyScope=value; case "custom_scope_option":m.CustomScopeOption=value; case "footer_edit":m.FooterEdit=value; case "editor_hint":m.EditorHint=value
	}
}

func mergeTypesByValue(base, override []Type) []Type {
	merged := append([]Type(nil), base...)
	for _, entry := range override {
		replaced := false
		for i := range merged {
			if merged[i].Value == entry.Value {
				merged[i] = entry
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, entry)
		}
	}
	return merged
}

func mergeScopesByValue(base, override []Scope) []Scope {
	merged := append([]Scope(nil), base...)
	for _, entry := range override {
		replaced := false
		for i := range merged {
			if merged[i].Value == entry.Value {
				merged[i] = entry
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, entry)
		}
	}
	return merged
}
