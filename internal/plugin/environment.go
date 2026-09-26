package plugin

import (
	"os"
	"path/filepath"
)

// InvocationEnvironment describes the stable process contract shared by all
// plugin launch paths, including normal dispatch and plugin help.
//
// The child inherits the complete parent environment through ExecPlugin. This
// function only supplies AIW-owned metadata and deliberately does not copy
// configuration-file contents or secrets into new variables.
func InvocationEnvironment(name, path, commandLine string) map[string]string {
	env := map[string]string{
		"AIW_PLUGIN_NAME": name,
		"AIW_PLUGIN_PATH": path,
		"AIW_CMDLINE":     commandLine,
	}
	if home := os.Getenv("HOME"); home != "" {
		env["AIW_HOME"] = home
	} else if home := os.Getenv("USERPROFILE"); home != "" {
		env["AIW_HOME"] = home
	}
	if exe, err := os.Executable(); err == nil {
		env["AIW_ROOT"] = filepath.Dir(exe)
	}
	if workspace, err := os.Getwd(); err == nil {
		env["AIW_WORKSPACE"] = workspace
	}
	return env
}
