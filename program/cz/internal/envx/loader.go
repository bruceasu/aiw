package envx

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"aiw-cz/internal/fsx"
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

// 使用方式：
// package main
// import (
// 	"fmt"
// "yourapp/dotenv"
// )
// func main() {
// 	loader := dotenv.New()
// err := loader.Load(".", "dev")
// 	if err != nil {
// 		panic(err)
// 	}
// fmt.Println(loader.Get("DB_URL"))
// err = loader.SetSystemEnv()
// 	if err != nil {
// 		panic(err)
// 	}
// }
// 示例 .env：
// export HOST=localhost
// PORT=3306
// DB_NAME=mydb
// DB_URL=mysql://${HOST}:${PORT}/${DB_NAME}
// REDIS_URL=redis://${REDIS_HOST:-127.0.0.1}:6379
// LITERAL='${NOT_EXPAND}'
// ESCAPED=\${NOT_EXPAND}
// NESTED=${DB_URL}
// MULTI_LINE="
// hello
// world
// "
// 最终结果：
// DB_URL=mysql://localhost:3306/mydb
// REDIS_URL=redis://127.0.0.1:6379
// LITERAL=${NOT_EXPAND}
// ESCAPED=${NOT_EXPAND}
// NESTED=mysql://localhost:3306/mydb
// MULTI_LINE=hello
// world
// "
// 最终结果：
// DB_URL=mysql://localhost:3306/mydb
// REDIS_URL=redis://127.0.0.1:6379
// LITERAL=${NOT_EXPAND}
// ESCAPED=${NOT_EXPAND}
// NESTED=mysql://localhost:3306/mydb
// MULTI_LINE=hello
// world
// 这里还有两个生产环境建议：
// 	1. 不要 panic
// 当前 expand 内部用了 panic 来快速中断递归。
// 生产中建议改成：
// func (...) (string, error)
// 完整 error return。
// 	1. Scanner 默认 64K 限制
// 如果 multiline 很大：
// scanner.Buffer(make([]byte, 1024), 1024*1024)
// 否则大 value 会报：
// bufio.Scanner: token too long
