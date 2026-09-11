package cz

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeLocale(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trimmed-fr", input: " fr_FR.UTF-8 ", want: "fr"},
		{name: "dash-zh", input: "zh-CN", want: "zh"},
		{name: "underscore-ja", input: "ja_JP", want: "ja"},
		{name: "posix-variant", input: "en@variant", want: "en"},
		{name: "empty", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeLocale(tt.input); got != tt.want {
				t.Fatalf("normalizeLocale(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectedLocale(t *testing.T) {
	t.Run("lc_all_wins", func(t *testing.T) {
		t.Setenv("LC_ALL", "ja_JP.UTF-8")
		t.Setenv("LC_MESSAGES", "zh_CN.UTF-8")
		t.Setenv("LANGUAGE", "fr:zh")
		t.Setenv("LANG", "en_US.UTF-8")

		if got := detectedLocale(&Config{}); got != "ja" {
			t.Fatalf("detectedLocale() = %q, want %q", got, "ja")
		}
	})

	t.Run("language_uses_configured_locale", func(t *testing.T) {
		t.Setenv("LC_ALL", "")
		t.Setenv("LC_MESSAGES", "")
		t.Setenv("LANGUAGE", "de:fr_FR:zh")
		t.Setenv("LANG", "")

		cfg := &Config{Locales: map[string]LocaleOverride{"fr": {}}}
		if got := detectedLocale(cfg); got != "fr" {
			t.Fatalf("detectedLocale() = %q, want %q", got, "fr")
		}
	})

	t.Run("unsupported_falls_back_to_en", func(t *testing.T) {
		t.Setenv("LC_ALL", "de_DE.UTF-8")
		t.Setenv("LC_MESSAGES", "")
		t.Setenv("LANGUAGE", "")
		t.Setenv("LANG", "")

		if got := detectedLocale(&Config{}); got != "en" {
			t.Fatalf("detectedLocale() = %q, want %q", got, "en")
		}
	})
}

func TestMergeCzConfigFromTomlFileNormalizesConfiguredRegionLocale(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aiw.toml")
	content := `[i18n]
default_language = "fr-FR"

[cz.locales.fr-FR.messages]
subject = "Sujet 1"
body = "Corps 1"

[[cz.locales.fr-FR.types]]
value = "feat"
name = "Fonction"

[[cz.locales.fr-FR.scopes]]
value = "core"
name = "Noyau"

[cz.locales.fr-CA.messages]
subject = "Sujet 2"

[[cz.locales.fr-CA.types]]
value = "fix"
name = "Correction"

[[cz.locales.fr-CA.scopes]]
value = "docs"
name = "Docs"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	if err := mergeCzConfigFromTomlFile(&cfg, path); err != nil {
		t.Fatalf("mergeCzConfigFromTomlFile() error = %v", err)
	}

	if _, ok := cfg.Locales["fr"]; !ok {
		t.Fatalf("mergeCzConfigFromTomlFile() did not normalize locale key to %q", "fr")
	}
	if _, ok := cfg.Locales["fr-fr"]; ok {
		t.Fatalf("mergeCzConfigFromTomlFile() kept non-canonical locale key %q", "fr-fr")
	}

	if err := ApplyConfiguredLocale(&cfg, cfg.DefaultLanguage); err != nil {
		t.Fatalf("ApplyConfiguredLocale() error = %v", err)
	}

	if got, want := cfg.Language, "fr"; got != want {
		t.Fatalf("cfg.Language = %q, want %q", got, want)
	}
	if got, want := cfg.Messages.Subject, "Sujet 2"; got != want {
		t.Fatalf("cfg.Messages.Subject = %q, want %q", got, want)
	}
	if got, want := cfg.Messages.Body, "Corps 1"; got != want {
		t.Fatalf("cfg.Messages.Body = %q, want %q", got, want)
	}
	if got, want := len(cfg.Types), len(DefaultConfig().Types); got != want {
		t.Fatalf("len(cfg.Types) = %d, want %d", got, want)
	}
	if got, want := cfg.Types[0], (Type{Value: "feat", Name: "Fonction"}); got != want {
		t.Fatalf("cfg.Types[0] = %#v, want %#v", got, want)
	}
	if got, want := cfg.Types[1], (Type{Value: "fix", Name: "Correction"}); got != want {
		t.Fatalf("cfg.Types[1] = %#v, want %#v", got, want)
	}
	if got, want := len(cfg.Scopes), len(DefaultConfig().Scopes); got != want {
		t.Fatalf("len(cfg.Scopes) = %d, want %d", got, want)
	}
	if got, want := cfg.Scopes[0], (Scope{Value: "core", Name: "Noyau"}); got != want {
		t.Fatalf("cfg.Scopes[0] = %#v, want %#v", got, want)
	}
	if got, want := cfg.Scopes[2], (Scope{Value: "docs", Name: "Docs"}); got != want {
		t.Fatalf("cfg.Scopes[2] = %#v, want %#v", got, want)
	}
}

func TestMergeCzConfigFromTomlFileRetainsProjectDefaultLanguageOverLaterSource(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "aiw.toml")
	projectContent := `[i18n]
default_language = "fr-FR"

[cz.locales.fr.messages]
subject = "Sujet projet"
`
	if err := os.WriteFile(projectPath, []byte(projectContent), 0o600); err != nil {
		t.Fatal(err)
	}

	laterPath := filepath.Join(dir, "cz.toml")
	laterContent := `[i18n]
default_language = "ja@variant"

[cz.locales.ja.messages]
subject = "Sujet secondaire"

[[cz.locales.ja.types]]
value = "fix"
name = "Correction"
`
	if err := os.WriteFile(laterPath, []byte(laterContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	if err := mergeCzConfigFromTomlFile(&cfg, projectPath); err != nil {
		t.Fatalf("mergeCzConfigFromTomlFile(project) error = %v", err)
	}
	projectDefault := cfg.DefaultLanguage

	if err := mergeCzConfigFromTomlFile(&cfg, laterPath); err != nil {
		t.Fatalf("mergeCzConfigFromTomlFile(later) error = %v", err)
	}
	if got, want := cfg.DefaultLanguage, "ja@variant"; got != want {
		t.Fatalf("cfg.DefaultLanguage before restore = %q, want %q", got, want)
	}

	restoreProjectDefaultLanguage(&cfg, projectDefault)

	if got, want := cfg.DefaultLanguage, "fr-fr"; got != want {
		t.Fatalf("cfg.DefaultLanguage = %q, want %q", got, want)
	}
	if got, want := cfg.Locales["fr"].Messages["subject"], "Sujet projet"; got != want {
		t.Fatalf("cfg.Locales[\"fr\"].Messages[\"subject\"] = %q, want %q", got, want)
	}
	if got, want := cfg.Locales["ja"].Messages["subject"], "Sujet secondaire"; got != want {
		t.Fatalf("cfg.Locales[\"ja\"].Messages[\"subject\"] = %q, want %q", got, want)
	}
	if got, want := len(cfg.Locales["ja"].Types), 1; got != want {
		t.Fatalf("len(cfg.Locales[\"ja\"].Types) = %d, want %d", got, want)
	}
	if got, want := cfg.Locales["ja"].Types[0], (Type{Value: "fix", Name: "Correction"}); got != want {
		t.Fatalf("cfg.Locales[\"ja\"].Types[0] = %#v, want %#v", got, want)
	}
}

func TestMergeCzConfigFromTomlFileAccumulatesLocaleTypesAndScopesAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first.toml")
	firstContent := `[cz.locales.fr.messages]
subject = "Sujet 1"

[[cz.locales.fr.types]]
value = "feat"
name = "Fonction"

[[cz.locales.fr.scopes]]
value = "core"
name = "Noyau"
`
	if err := os.WriteFile(firstPath, []byte(firstContent), 0o600); err != nil {
		t.Fatal(err)
	}

	secondPath := filepath.Join(dir, "second.toml")
	secondContent := `[cz.locales.fr.messages]
body = "Corps 2"

[[cz.locales.fr.types]]
value = "fix"
name = "Correction"

[[cz.locales.fr.scopes]]
value = "docs"
name = "Docs"
`
	if err := os.WriteFile(secondPath, []byte(secondContent), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	if err := mergeCzConfigFromTomlFile(&cfg, firstPath); err != nil {
		t.Fatalf("mergeCzConfigFromTomlFile(first) error = %v", err)
	}
	if err := mergeCzConfigFromTomlFile(&cfg, secondPath); err != nil {
		t.Fatalf("mergeCzConfigFromTomlFile(second) error = %v", err)
	}

	if got, want := len(cfg.Locales["fr"].Types), 2; got != want {
		t.Fatalf("len(cfg.Locales[\"fr\"].Types) = %d, want %d", got, want)
	}
	if got, want := cfg.Locales["fr"].Types[0], (Type{Value: "feat", Name: "Fonction"}); got != want {
		t.Fatalf("cfg.Locales[\"fr\"].Types[0] = %#v, want %#v", got, want)
	}
	if got, want := cfg.Locales["fr"].Types[1], (Type{Value: "fix", Name: "Correction"}); got != want {
		t.Fatalf("cfg.Locales[\"fr\"].Types[1] = %#v, want %#v", got, want)
	}
	if got, want := len(cfg.Locales["fr"].Scopes), 2; got != want {
		t.Fatalf("len(cfg.Locales[\"fr\"].Scopes) = %d, want %d", got, want)
	}
	if got, want := cfg.Locales["fr"].Scopes[0], (Scope{Value: "core", Name: "Noyau"}); got != want {
		t.Fatalf("cfg.Locales[\"fr\"].Scopes[0] = %#v, want %#v", got, want)
	}
	if got, want := cfg.Locales["fr"].Scopes[1], (Scope{Value: "docs", Name: "Docs"}); got != want {
		t.Fatalf("cfg.Locales[\"fr\"].Scopes[1] = %#v, want %#v", got, want)
	}

	if err := ApplyConfiguredLocale(&cfg, "fr"); err != nil {
		t.Fatalf("ApplyConfiguredLocale() error = %v", err)
	}
	if got, want := cfg.Messages.Subject, "Sujet 1"; got != want {
		t.Fatalf("cfg.Messages.Subject = %q, want %q", got, want)
	}
	if got, want := cfg.Messages.Body, "Corps 2"; got != want {
		t.Fatalf("cfg.Messages.Body = %q, want %q", got, want)
	}
	if got, want := len(cfg.Types), len(DefaultConfig().Types); got != want {
		t.Fatalf("len(cfg.Types) = %d, want %d", got, want)
	}
	if got, want := cfg.Types[0], (Type{Value: "feat", Name: "Fonction"}); got != want {
		t.Fatalf("cfg.Types[0] = %#v, want %#v", got, want)
	}
	foundFix := false
	for _, typ := range cfg.Types {
		if typ.Value == "fix" {
			if typ.Name != "Correction" {
				t.Fatalf("cfg.Types[fix] = %#v, want %#v", typ, Type{Value: "fix", Name: "Correction"})
			}
			foundFix = true
			break
		}
	}
	if !foundFix {
		t.Fatalf("cfg.Types does not contain %q", "fix")
	}
	if got, want := len(cfg.Scopes), len(DefaultConfig().Scopes); got != want {
		t.Fatalf("len(cfg.Scopes) = %d, want %d", got, want)
	}
	if got, want := cfg.Scopes[0], (Scope{Value: "core", Name: "Noyau"}); got != want {
		t.Fatalf("cfg.Scopes[0] = %#v, want %#v", got, want)
	}
	foundDocs := false
	for _, scope := range cfg.Scopes {
		if scope.Value == "docs" {
			if scope.Name != "Docs" {
				t.Fatalf("cfg.Scopes[docs] = %#v, want %#v", scope, Scope{Value: "docs", Name: "Docs"})
			}
			foundDocs = true
			break
		}
	}
	if !foundDocs {
		t.Fatalf("cfg.Scopes does not contain %q", "docs")
	}
}

func TestApplyCzLocaleSelectionPrefersConfiguredDefaultOverDetectedLocale(t *testing.T) {
	t.Setenv("LC_ALL", "ja_JP.UTF-8")
	t.Setenv("LC_MESSAGES", "ja_JP.UTF-8")
	t.Setenv("LANGUAGE", "ja_JP.UTF-8")
	t.Setenv("LANG", "ja_JP.UTF-8")

	cfg := DefaultConfig()
	cfg.DefaultLanguage = "zh"

	if err := applyCzLocaleSelection(&cfg, false, ""); err != nil {
		t.Fatalf("applyCzLocaleSelection() error = %v", err)
	}
	if got, want := cfg.Language, "zh"; got != want {
		t.Fatalf("cfg.Language = %q, want %q", got, want)
	}
}

func TestApplyCzLocaleSelectionUsesDetectedLocaleWhenDefaultMissing(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANGUAGE", "")
	t.Setenv("LANG", "zh_CN.UTF-8")

	cfg := DefaultConfig()

	if err := applyCzLocaleSelection(&cfg, false, ""); err != nil {
		t.Fatalf("applyCzLocaleSelection() error = %v", err)
	}
	if got, want := cfg.Language, "zh"; got != want {
		t.Fatalf("cfg.Language = %q, want %q", got, want)
	}
}

func TestApplyCzLocaleSelectionAppliesConfiguredCustomLocaleOverEnglish(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANGUAGE", "")
	t.Setenv("LANG", "fr_FR.UTF-8")

	dir := t.TempDir()
	path := filepath.Join(dir, "aiw.toml")
	content := `[i18n]
default_language = "fr-FR"

[cz.locales.fr-FR.messages]
subject = "Sujet personnalisé"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	if err := mergeCzConfigFromTomlFile(&cfg, path); err != nil {
		t.Fatalf("mergeCzConfigFromTomlFile() error = %v", err)
	}

	if err := applyCzLocaleSelection(&cfg, false, ""); err != nil {
		t.Fatalf("applyCzLocaleSelection() error = %v", err)
	}
	if got, want := cfg.Language, "fr"; got != want {
		t.Fatalf("cfg.Language = %q, want %q", got, want)
	}
	if got, want := cfg.Messages.Subject, "Sujet personnalisé"; got != want {
		t.Fatalf("cfg.Messages.Subject = %q, want %q", got, want)
	}
	if got, want := cfg.Messages.Body, "Body (optional)"; got != want {
		t.Fatalf("cfg.Messages.Body = %q, want %q", got, want)
	}
}

func TestApplyConfiguredLocaleMergesCustomLocaleTypesAndScopesWithEnglishBase(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Locales = map[string]LocaleOverride{
		"fr": {
			Messages: map[string]string{
				"subject": "Sujet personnalisé",
			},
			Types: []Type{
				{Value: "feat", Name: "Fonctionnalité"},
			},
			HasTypes: true,
			Scopes: []Scope{
				{Value: "core", Name: "Noyau"},
			},
			HasScopes: true,
		},
	}

	if err := ApplyConfiguredLocale(&cfg, "fr"); err != nil {
		t.Fatalf("ApplyConfiguredLocale() error = %v", err)
	}

	if got, want := cfg.Language, "fr"; got != want {
		t.Fatalf("cfg.Language = %q, want %q", got, want)
	}
	if got, want := cfg.Messages.Subject, "Sujet personnalisé"; got != want {
		t.Fatalf("cfg.Messages.Subject = %q, want %q", got, want)
	}
	if got, want := len(cfg.Types), 11; got != want {
		t.Fatalf("len(cfg.Types) = %d, want %d", got, want)
	}
	if got, want := cfg.Types[0], (Type{Value: "feat", Name: "Fonctionnalité"}); got != want {
		t.Fatalf("cfg.Types[0] = %#v, want %#v", got, want)
	}
	if got, want := cfg.Types[1], (Type{Value: "fix", Name: "A bug fix"}); got != want {
		t.Fatalf("cfg.Types[1] = %#v, want %#v", got, want)
	}
	if got, want := len(cfg.Scopes), 5; got != want {
		t.Fatalf("len(cfg.Scopes) = %d, want %d", got, want)
	}
	if got, want := cfg.Scopes[0], (Scope{Value: "core", Name: "Noyau"}); got != want {
		t.Fatalf("cfg.Scopes[0] = %#v, want %#v", got, want)
	}
	if got, want := cfg.Scopes[1], (Scope{Value: "api", Name: "api"}); got != want {
		t.Fatalf("cfg.Scopes[1] = %#v, want %#v", got, want)
	}
}

func TestApplyConfiguredLocaleAcceptsCamelCaseMessageAliases(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Locales = map[string]LocaleOverride{
		"fr": {
			Messages: map[string]string{
				"customScope":          "Champ personnalisé",
				"footerPrefixesSelect": "Sélectionner le préfixe",
				"customFooterPrefix":   "Préfixe personnalisé",
				"confirmCommit":        "Confirmer le commit",
			},
		},
	}

	if err := ApplyConfiguredLocale(&cfg, "fr"); err != nil {
		t.Fatalf("ApplyConfiguredLocale() error = %v", err)
	}

	if got, want := cfg.Messages.CustomScope, "Champ personnalisé"; got != want {
		t.Fatalf("cfg.Messages.CustomScope = %q, want %q", got, want)
	}
	if got, want := cfg.Messages.FooterPrefixes, "Sélectionner le préfixe"; got != want {
		t.Fatalf("cfg.Messages.FooterPrefixes = %q, want %q", got, want)
	}
	if got, want := cfg.Messages.CustomFooterPrefix, "Préfixe personnalisé"; got != want {
		t.Fatalf("cfg.Messages.CustomFooterPrefix = %q, want %q", got, want)
	}
	if got, want := cfg.Messages.ConfirmCommit, "Confirmer le commit"; got != want {
		t.Fatalf("cfg.Messages.ConfirmCommit = %q, want %q", got, want)
	}
}

func TestApplyConfiguredLocaleOverridesSelectedLocale(t *testing.T) {
	t.Setenv("LC_ALL", "zh_CN.UTF-8")
	t.Setenv("LC_MESSAGES", "zh_CN.UTF-8")
	t.Setenv("LANGUAGE", "zh_CN.UTF-8")
	t.Setenv("LANG", "zh_CN.UTF-8")

	cfg := DefaultConfig()
	cfg.DefaultLanguage = "zh"

	if err := applyCzLocaleSelection(&cfg, false, ""); err != nil {
		t.Fatalf("applyCzLocaleSelection() error = %v", err)
	}
	if err := ApplyConfiguredLocale(&cfg, "ja"); err != nil {
		t.Fatalf("ApplyConfiguredLocale() error = %v", err)
	}
	if got, want := cfg.Language, "ja"; got != want {
		t.Fatalf("cfg.Language = %q, want %q", got, want)
	}
}

func TestApplyConfiguredLocaleSkipsReloadForCurrentBuiltinLocaleButKeepsConfiguredEnglishOverrides(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Messages.Subject = "sentinel-subject"
	cfg.Messages.Body = "sentinel-body"
	cfg.Locales = map[string]LocaleOverride{
		"en": {
			Messages: map[string]string{
				"subject": "configured subject",
			},
		},
	}

	if err := ApplyConfiguredLocale(&cfg, "en"); err != nil {
		t.Fatalf("ApplyConfiguredLocale() error = %v", err)
	}

	if got, want := cfg.Language, "en"; got != want {
		t.Fatalf("cfg.Language = %q, want %q", got, want)
	}
	if got, want := cfg.Messages.Subject, "configured subject"; got != want {
		t.Fatalf("cfg.Messages.Subject = %q, want %q", got, want)
	}
	if got, want := cfg.Messages.Body, "sentinel-body"; got != want {
		t.Fatalf("cfg.Messages.Body = %q, want %q", got, want)
	}
}

func TestApplyConfiguredLocaleRejectsWhitespaceLocale(t *testing.T) {
	cfg := DefaultConfig()

	if err := ApplyConfiguredLocale(&cfg, "   "); err == nil {
		t.Fatal("ApplyConfiguredLocale() error = nil, want error")
	} else if got, want := err.Error(), "unsupported language \"   \"; use built-in en, zh, ja or configure [cz.locales.en]"; got != want {
		t.Fatalf("ApplyConfiguredLocale() error = %q, want %q", got, want)
	}
}

func TestApplyCzLocaleSelectionPrefersCliLocaleOverInvalidDefault(t *testing.T) {
	t.Setenv("LC_ALL", "ja_JP.UTF-8")
	t.Setenv("LC_MESSAGES", "ja_JP.UTF-8")
	t.Setenv("LANGUAGE", "ja_JP.UTF-8")
	t.Setenv("LANG", "ja_JP.UTF-8")

	cfg := DefaultConfig()
	cfg.DefaultLanguage = "de"

	if err := applyCzLocaleSelection(&cfg, true, "en"); err != nil {
		t.Fatalf("applyCzLocaleSelection() error = %v", err)
	}
	if got, want := cfg.Language, "en"; got != want {
		t.Fatalf("cfg.Language = %q, want %q", got, want)
	}
}

func TestApplyCzLocaleSelectionReturnsErrorForInvalidDefaultWhenCliLocaleEmpty(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANGUAGE", "")
	t.Setenv("LANG", "")

	cfg := DefaultConfig()
	cfg.DefaultLanguage = "de"

	if err := applyCzLocaleSelection(&cfg, false, ""); err == nil {
		t.Fatal("applyCzLocaleSelection() error = nil, want error")
	}
}

func TestParseCzOptionsKeepsExplicitEmptyLangAsProvided(t *testing.T) {
	opts, err := parseCzOptions([]string{"--lang", ""})
	if err != nil {
		t.Fatalf("parseCzOptions() error = %v", err)
	}
	if !opts.LanguageSet {
		t.Fatal("parseCzOptions() did not preserve explicit --lang presence")
	}
	if got, want := opts.Language, ""; got != want {
		t.Fatalf("opts.Language = %q, want %q", got, want)
	}

	t.Setenv("LC_ALL", "ja_JP.UTF-8")
	t.Setenv("LC_MESSAGES", "ja_JP.UTF-8")
	t.Setenv("LANGUAGE", "ja_JP.UTF-8")
	t.Setenv("LANG", "ja_JP.UTF-8")

	cfg := DefaultConfig()
	cfg.DefaultLanguage = "zh"

	if err := applyCzLocaleSelection(&cfg, opts.LanguageSet, opts.Language); err == nil {
		t.Fatal("applyCzLocaleSelection() error = nil, want unsupported-language error")
	}
	if got, want := cfg.Language, "en"; got != want {
		t.Fatalf("cfg.Language = %q, want %q", got, want)
	}
}
