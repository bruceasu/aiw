package plugin

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ExecPlugin starts the plugin executable/script at path with provided args and env overrides.
// Returns the exit code (or -1 if execution failed before process start) and error.
func ExecPlugin(path string, args []string, env map[string]string) (int, error) {
	var cmd *exec.Cmd
	ext := strings.ToLower(filepath.Ext(path))

	// helper to read shebang interpreter
	shebang := getShebangInterpreter(path)

	switch ext {
	case ".py":
		cmd = exec.Command("python", append([]string{path}, args...)...)
	case ".sh":
		if shebang != "" {
			cmd = exec.Command(shebang, append([]string{path}, args...)...)
		} else {
			shell := "bash"
			cmd = exec.Command(shell, append([]string{path}, args...)...)
		}
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
		if exePath, _ := exec.LookPath("bun"); exePath != "" {
			cmd = exec.Command("bun", append([]string{path}, args...)...)
		} else {
			cmd = exec.Command("node", append([]string{path}, args...)...)
		}
	default:
		// no ext: if shebang present, use it; otherwise try to execute directly
		if shebang != "" {
			cmd = exec.Command(shebang, append([]string{path}, args...)...)
		} else {
			cmd = exec.Command(path, args...)
		}
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

	err := cmd.Run()
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
