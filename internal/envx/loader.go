package envx

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"aiw/internal/fsx"
)

var varPattern = regexp.MustCompile(`(?s)(\\)?\$\{([^}]+)\}`)

type Loader struct {
	Env map[string]string
}

func New() *Loader {
	env := map[string]string{}
	for _, item := range os.Environ() {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return &Loader{Env: env}
}

func (l *Loader) Load(dir string, profile string) error {
	base := filepath.Join(dir, ".env")
	if fsx.Exists(base) {
		if err := l.ParseFile(base); err != nil {
			return err
		}
	}
	if profile != "" {
		profileFile := filepath.Join(dir, ".env."+profile)
		if !fsx.Exists(profileFile) {
			return fmt.Errorf("missing env profile file: %s", profileFile)
		}
		if err := l.ParseFile(profileFile); err != nil {
			return err
		}
	}
	return nil
}

func (l *Loader) ParseFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var (
		lineNo        int
		multilineKey  string
		multilineBuff strings.Builder
	)

	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		if multilineKey != "" {
			multilineBuff.WriteString("\n")
			multilineBuff.WriteString(raw)
			if strings.HasSuffix(strings.TrimSpace(raw), `"`) {
				value := multilineBuff.String()
				value = strings.TrimPrefix(value, `"`)
				value = strings.TrimSuffix(value, `"`)
				resolved, err := l.expand(value, map[string]bool{})
				if err != nil {
					return fmt.Errorf("%s:%d: %w", path, lineNo, err)
				}
				l.Env[multilineKey] = resolved
				multilineKey = ""
				multilineBuff.Reset()
			}
			continue
		}

		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if strings.HasPrefix(value, `"`) && !strings.HasSuffix(value, `"`) {
			multilineKey = key
			multilineBuff.WriteString(value)
			continue
		}
		if strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`) {
			value = strings.TrimPrefix(value, `'`)
			value = strings.TrimSuffix(value, `'`)
			l.Env[key] = value
			continue
		}
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			value = strings.TrimPrefix(value, `"`)
			value = strings.TrimSuffix(value, `"`)
		}
		resolved, err := l.expand(value, map[string]bool{})
		if err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
		l.Env[key] = resolved
	}
	return scanner.Err()
}

func (l *Loader) expand(input string, visiting map[string]bool) (string, error) {
	result := varPattern.ReplaceAllStringFunc(input, func(match string) string {
		sub := varPattern.FindStringSubmatch(match)
		if sub[1] == `\` {
			return "${" + sub[2] + "}"
		}
		expr := sub[2]
		var (
			name string
			def  string
		)
		if strings.Contains(expr, ":-") {
			parts := strings.SplitN(expr, ":-", 2)
			name = parts[0]
			def = parts[1]
		} else {
			name = expr
		}
		if visiting[name] {
			panic("cyclic env reference: " + name)
		}
		visiting[name] = true
		defer delete(visiting, name)
		val, ok := l.Env[name]
		if !ok {
			val = os.Getenv(name)
		}
		if val == "" {
			val = def
		}
		expanded, err := l.expand(val, visiting)
		if err != nil {
			panic(err)
		}
		return expanded
	})
	return result, nil
}

func (l *Loader) MustGet(key string) string {
	v, ok := l.Env[key]
	if !ok || v == "" {
		panic("missing env: " + key)
	}
	return v
}

func (l *Loader) Get(key string) string {
	return l.Env[key]
}

func (l *Loader) SetSystemEnv() error {
	for k, v := range l.Env {
		if err := os.Setenv(k, v); err != nil {
			return err
		}
	}
	return nil
}
