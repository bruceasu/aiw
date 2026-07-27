package plugin

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var userConfigDirFn = os.UserConfigDir
var userHomeDirFn = os.UserHomeDir
var statPathFn = os.Stat

type pythonInterpreterSetting struct {
	path string
}

func configuredPythonInterpreter(exeDir string) (pythonInterpreterSetting, error) {
	if value := strings.TrimSpace(os.Getenv("AIW_PYTHON")); value != "" {
		return validatePythonInterpreter(value, "AIW_PYTHON")
	}

	userConfigPath, err := findUserConfigPath()
	if err != nil {
		return pythonInterpreterSetting{}, err
	}
	if userConfigPath != "" {
		configured, err := configuredPythonFromFile(userConfigPath, "user config")
		if err != nil || configured.path != "" {
			return configured, err
		}
	}

	if exeDir != "" {
		programConfigPath := filepath.Join(exeDir, "aiw.toml")
		exists, err := configFileExists(programConfigPath)
		if err != nil {
			return pythonInterpreterSetting{}, fmt.Errorf("inspect program config %s: %w", programConfigPath, err)
		}
		if exists {
			return configuredPythonFromFile(programConfigPath, "program config")
		}
	}

	return pythonInterpreterSetting{}, nil
}

func configuredPythonFromFile(path, source string) (pythonInterpreterSetting, error) {
	value, err := readRuntimePython(path)
	if err != nil {
		return pythonInterpreterSetting{}, fmt.Errorf("read %s %s: %w", source, path, err)
	}
	if strings.TrimSpace(value) == "" {
		return pythonInterpreterSetting{}, nil
	}
	return validatePythonInterpreter(value, source+" "+path)
}

func findUserConfigPath() (string, error) {
	configDir, err := userConfigDirFn()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	canonicalPath := ""
	if configDir != "" {
		canonicalPath = filepath.Join(configDir, "aiw", "aiw.toml")
		exists, err := configFileExists(canonicalPath)
		if err != nil {
			return "", fmt.Errorf("inspect user config %s: %w", canonicalPath, err)
		}
		if exists {
			return canonicalPath, nil
		}
	}

	homeDir, err := userHomeDirFn()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	if homeDir == "" {
		return "", nil
	}
	compatibilityPath := filepath.Join(homeDir, ".config", "aiw", "aiw.toml")
	if canonicalPath != "" && samePath(canonicalPath, compatibilityPath) {
		return "", nil
	}
	exists, err := configFileExists(compatibilityPath)
	if err != nil {
		return "", fmt.Errorf("inspect user config %s: %w", compatibilityPath, err)
	}
	if exists {
		return compatibilityPath, nil
	}
	return "", nil
}

func samePath(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

func validatePythonInterpreter(path, source string) (pythonInterpreterSetting, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if !filepath.IsAbs(path) {
		return pythonInterpreterSetting{}, fmt.Errorf(
			"invalid Python interpreter from %s: path must be absolute: %s", source, path,
		)
	}
	info, err := statPathFn(path)
	if err != nil {
		if os.IsNotExist(err) {
			return pythonInterpreterSetting{}, fmt.Errorf(
				"invalid Python interpreter from %s: not an existing file: %s", source, path,
			)
		}
		return pythonInterpreterSetting{}, fmt.Errorf(
			"inspect Python interpreter from %s at %s: %w", source, path, err,
		)
	}
	if info.IsDir() {
		return pythonInterpreterSetting{}, fmt.Errorf(
			"invalid Python interpreter from %s: not an existing file: %s", source, path,
		)
	}
	return pythonInterpreterSetting{path: path}, nil
}

func configFileExists(path string) (bool, error) {
	info, err := statPathFn(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("path is a directory")
	}
	return true, nil
}

func readRuntimePython(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	inRuntime := false
	pythonSeen := false
	pythonValue := ""
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\uFEFF"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			section, err := parseConfigSection(line, inRuntime)
			if err != nil {
				return "", fmt.Errorf("line %d: %w", lineNumber, err)
			}
			inRuntime = section == "runtime"
			continue
		}
		if !inRuntime {
			continue
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return "", fmt.Errorf("line %d: expected key = value in [runtime]", lineNumber)
		}
		if strings.TrimSpace(key) != "python" {
			continue
		}
		if pythonSeen {
			return "", fmt.Errorf("line %d: duplicate runtime.python", lineNumber)
		}
		value, err := parseConfigString(rawValue)
		if err != nil {
			return "", fmt.Errorf("line %d: runtime.python: %w", lineNumber, err)
		}
		pythonValue = value
		pythonSeen = true
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return pythonValue, nil
}

func parseConfigSection(line string, inRuntime bool) (string, error) {
	end := strings.IndexByte(line, ']')
	if end < 0 {
		section := strings.TrimSpace(strings.TrimPrefix(line, "["))
		if inRuntime || section == "runtime" || strings.HasPrefix(section, "runtime.") {
			return "", fmt.Errorf("malformed table header")
		}
		return "", nil
	}
	section := strings.TrimSpace(line[1:end])
	if err := validateConfigRemainder(line[end+1:]); err != nil {
		if inRuntime || section == "runtime" {
			return "", fmt.Errorf("malformed table header: %w", err)
		}
		return section, nil
	}
	return section, nil
}

func parseConfigString(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("expected a quoted string")
	}

	switch value[0] {
	case '"':
		end := closingDoubleQuote(value)
		if end < 0 {
			return "", fmt.Errorf("unterminated quoted string")
		}
		parsed, err := strconv.Unquote(value[:end+1])
		if err != nil {
			return "", fmt.Errorf("invalid quoted string: %w", err)
		}
		if err := validateConfigRemainder(value[end+1:]); err != nil {
			return "", err
		}
		return parsed, nil
	case '\'':
		end := strings.IndexByte(value[1:], '\'')
		if end < 0 {
			return "", fmt.Errorf("unterminated literal string")
		}
		end++
		if err := validateConfigRemainder(value[end+1:]); err != nil {
			return "", err
		}
		return value[1:end], nil
	default:
		return "", fmt.Errorf("expected a quoted string")
	}
}

func closingDoubleQuote(value string) int {
	escaped := false
	for i := 1; i < len(value); i++ {
		if escaped {
			escaped = false
			continue
		}
		switch value[i] {
		case '\\':
			escaped = true
		case '"':
			return i
		}
	}
	return -1
}

func validateConfigRemainder(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "#") {
		return nil
	}
	return fmt.Errorf("unexpected content after string")
}
