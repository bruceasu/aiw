package plugin

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var pluginExecutablePathFn = os.Executable
var lookPathFn = exec.LookPath

// ExecPlugin starts the plugin executable/script at path with provided args and env overrides.
// Returns the exit code (or -1 if execution failed before process start) and error.
func ExecPlugin(path string, args []string, env map[string]string) (int, error) {
	cmd, err := buildPluginCommand(path, args)
	if err != nil {
		return -1, err
	}
	if cmd == nil {
		return -1, fmt.Errorf("unsupported plugin execution for %s", path)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// build environment
	cmd.Env = mergeEnvironment(os.Environ(), env)

	err = cmd.Run()
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(interface{ ExitStatus() int }); ok {
			return status.ExitStatus(), nil
		}
		return -1, nil
	}
	return -1, err
}

// ExecPluginWithInput invokes a Plugin using the same interpreter resolution
// as the CLI fallback while exchanging a bounded JSON-style request/response.
// It is used by adapters that must persist the request before dispatching it.
func ExecPluginWithInput(path string, args []string, env map[string]string, input []byte) ([]byte, int, error) {
	cmd, err := buildPluginCommand(path, args)
	if err != nil {
		return nil, -1, err
	}
	if cmd == nil {
		return nil, -1, fmt.Errorf("unsupported plugin execution for %s", path)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = mergeEnvironment(os.Environ(), env)
	if err := cmd.Run(); err == nil {
		return stdout.Bytes(), 0, nil
	} else if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(interface{ ExitStatus() int }); ok {
			return stdout.Bytes(), status.ExitStatus(), nil
		}
		return stdout.Bytes(), -1, nil
	} else {
		return stdout.Bytes(), -1, fmt.Errorf("run plugin: %w: %s", err, stderr.String())
	}
}

// mergeEnvironment preserves the parent environment while making explicit
// plugin variables authoritative. Duplicate keys are avoided for predictable
// behavior across Windows and Unix process launchers.
func mergeEnvironment(base []string, overrides map[string]string) []string {
	result := append([]string(nil), base...)
	positions := make(map[string]int, len(result)+len(overrides))
	for i, entry := range result {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			positions[key] = i
		}
	}
	for key, value := range overrides {
		entry := fmt.Sprintf("%s=%s", key, value)
		if i, ok := positions[key]; ok {
			result[i] = entry
			continue
		}
		positions[key] = len(result)
		result = append(result, entry)
	}
	return result
}

func buildPluginCommand(path string, args []string) (*exec.Cmd, error) {
	ext := strings.ToLower(filepath.Ext(path))
	shebang := getShebangInterpreter(path)

	switch ext {
	case ".py", ".pl", ".jar", ".sh":
		return buildCommand(path, args, ext, shebang)
	case ".bat", ".cmd":
		if runtime.GOOS == "windows" {
			return exec.Command("cmd", append([]string{"/C", path}, args...)...), nil
		}
		return exec.Command(path, args...), nil
	case ".ps1":
		if runtime.GOOS == "windows" {
			return exec.Command("powershell", append([]string{"-File", path}, args...)...), nil
		}
		return exec.Command("pwsh", append([]string{"-File", path}, args...)...), nil
	case ".js", ".mjs", ".cjs":
		return buildNodeCommand(path, args, false)
	case ".ts", ".mts", ".cts":
		return buildNodeCommand(path, args, true)
	default:
		if shebang != "" {
			return buildCommand(path, args, ext, shebang)
		}
		return exec.Command(path, args...), nil
	}
}

func buildNodeCommand(path string, args []string, stripTypes bool) (*exec.Cmd, error) {
	exeDir, err := pluginExecutablePathFn()
	if err != nil {
		exeDir = ""
	} else {
		exeDir = filepath.Dir(exeDir)
	}
	node, err := configuredNodeInterpreter(exeDir)
	if err != nil {
		return nil, err
	}
	commandArgs := []string{}
	if stripTypes {
		commandArgs = append(commandArgs, "--experimental-strip-types")
	}
	commandArgs = append(commandArgs, path)
	commandArgs = append(commandArgs, args...)
	return exec.Command(node, commandArgs...), nil
}

func buildCommand(path string, args []string, ext, shebang string) (*exec.Cmd, error) {
	prefix, err := resolveInterpreterCommand(ext, shebang)
	if err != nil {
		return nil, err
	}
	cmdArgs := append(append([]string{}, prefix[1:]...), path)
	cmdArgs = append(cmdArgs, args...)
	return exec.Command(prefix[0], cmdArgs...), nil
}

func getShebangInterpreter(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	r := bufio.NewReader(f)
	line, err := r.ReadString('\n')
	if err != nil {
		return ""
	}
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "#!") {
		return ""
	}
	// remove #!
	fields := strings.Fields(strings.TrimPrefix(line, "#!"))
	if len(fields) == 0 {
		return ""
	}
	// if interpreter is /usr/bin/env node style, take last field
	if strings.HasSuffix(fields[0], "env") && len(fields) > 1 {
		return fields[1]
	}
	return fields[0]
}

func resolveInterpreterCommand(ext, shebang string) ([]string, error) {
	family, commands := interpreterCandidates(ext, shebang)
	if len(commands) == 0 {
		return nil, errors.New("no interpreter candidates")
	}

	localBaseDir, localSubDir := interpreterLocalDir(family)
	if family == "python" {
		configured, err := configuredPythonInterpreter(localBaseDir)
		if err != nil {
			return nil, err
		}
		if configured.path != "" {
			return []string{configured.path}, nil
		}
	}
	if localBaseDir != "" {
		if resolved := findLocalInterpreter(localBaseDir, localSubDir, commands); resolved != "" {
			if family == "java" && ext == ".jar" {
				return []string{resolved, "-jar"}, nil
			}
			return []string{resolved}, nil
		}
	}

	for _, candidate := range commands {
		if resolved, err := lookPathFn(candidate); err == nil && resolved != "" {
			if family == "java" && ext == ".jar" {
				return []string{resolved, "-jar"}, nil
			}
			return []string{resolved}, nil
		}
	}
	return nil, fmt.Errorf("interpreter not found for ext=%s shebang=%s", ext, shebang)
}

func interpreterCandidates(ext, shebang string) (string, []string) {
	switch ext {
	case ".py":
		return "python", []string{"python", "python3"}
	case ".pl":
		return "perl", []string{"perl"}
	case ".jar":
		return "java", []string{"java"}
	case ".sh":
		return "bash", []string{"bash", "sh"}
	}

	switch normalizeInterpreterName(shebang) {
	case "python", "python3":
		return "python", []string{"python", "python3"}
	case "perl":
		return "perl", []string{"perl"}
	case "java":
		return "java", []string{"java"}
	case "bash", "sh":
		return "bash", []string{"bash", "sh"}
	default:
		if shebang == "" {
			return "", nil
		}
		return "", []string{shebang}
	}
}

func interpreterLocalDir(family string) (string, string) {
	exePath, err := pluginExecutablePathFn()
	if err != nil {
		return "", ""
	}
	if resolvedPath, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolvedPath
	}
	exeDir := filepath.Dir(exePath)
	switch family {
	case "python":
		return exeDir, "python"
	case "perl":
		return exeDir, "perl"
	case "java":
		return exeDir, filepath.Join("java", "bin")
	case "bash":
		return exeDir, "bash"
	default:
		return "", ""
	}
}

func findLocalInterpreter(exeDir, subDir string, candidates []string) string {
	baseDir := filepath.Join(exeDir, subDir)
	for _, candidate := range candidates {
		full := filepath.Join(baseDir, executableName(candidate))
		if fileExists(full) {
			return full
		}
		raw := filepath.Join(baseDir, candidate)
		if raw != full && fileExists(raw) {
			return raw
		}
	}
	return ""
}

func normalizeInterpreterName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = filepath.Base(name)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return strings.ToLower(name)
}

func executableName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
