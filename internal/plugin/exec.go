package plugin

import (
	"bufio"
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
	var cmd *exec.Cmd
	var err error
	ext := strings.ToLower(filepath.Ext(path))

	// helper to read shebang interpreter
	shebang := getShebangInterpreter(path)

	switch ext {
	case ".py":
		cmd, err = buildCommand(path, args, ext, shebang)
	case ".pl":
		cmd, err = buildCommand(path, args, ext, shebang)
	case ".jar":
		cmd, err = buildCommand(path, args, ext, shebang)
	case ".sh":
		cmd, err = buildCommand(path, args, ext, shebang)
	case ".bat", ".cmd":
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", append([]string{"/C", path}, args...)...)
		} else {
			cmd = exec.Command(path, args...)
		}
	case ".ps1":
		if runtime.GOOS == "windows" {
			cmd = exec.Command("powershell", append([]string{"-File", path}, args...)...)
		} else {
			cmd = exec.Command("pwsh", append([]string{"-File", path}, args...)...)
		}
	case ".js":
		// prefer bun then node if available
		if exePath, _ := lookPathFn("bun"); exePath != "" {
			cmd = exec.Command("bun", append([]string{path}, args...)...)
		} else {
			cmd = exec.Command("node", append([]string{path}, args...)...)
		}
	default:
		// no ext: if shebang present, use it; otherwise try to execute directly
		if shebang != "" {
			cmd, err = buildCommand(path, args, ext, shebang)
		} else {
			cmd = exec.Command(path, args...)
		}
	}
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
	finalEnv := os.Environ()
	for k, v := range env {
		finalEnv = append(finalEnv, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = finalEnv

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
