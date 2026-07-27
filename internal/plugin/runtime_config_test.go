package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveInterpreterCommandUsesProgramConfigDefault(t *testing.T) {
	td := t.TempDir()
	exeDir := filepath.Join(td, "bin")
	python := writeTestFile(t, filepath.Join(td, "python", executableName("program-python")), "")
	writeTestFile(t, filepath.Join(exeDir, "aiw.toml"),
		"[runtime]\npython = \""+filepath.ToSlash(python)+"\"\n")
	stubInterpreterEnvironment(t, exeDir, filepath.Join(td, "config"), filepath.Join(td, "home"))

	got, err := resolveInterpreterCommand(".py", "")
	if err != nil {
		t.Fatalf("resolveInterpreterCommand returned error: %v", err)
	}
	if len(got) != 1 || got[0] != python {
		t.Fatalf("resolveInterpreterCommand() = %v, want [%q]", got, python)
	}
}

func TestResolveInterpreterCommandUsesHomeConfigFallback(t *testing.T) {
	td := t.TempDir()
	exeDir := filepath.Join(td, "bin")
	homeDir := filepath.Join(td, "home")
	python := writeTestFile(t, filepath.Join(td, "python", executableName("home-python")), "")
	writeTestFile(t, filepath.Join(homeDir, ".config", "aiw", "aiw.toml"),
		"[runtime]\npython = \""+filepath.ToSlash(python)+"\"\n")
	stubInterpreterEnvironment(t, exeDir, filepath.Join(td, "canonical-config"), homeDir)

	got, err := resolveInterpreterCommand(".py", "")
	if err != nil {
		t.Fatalf("resolveInterpreterCommand returned error: %v", err)
	}
	if len(got) != 1 || got[0] != python {
		t.Fatalf("resolveInterpreterCommand() = %v, want [%q]", got, python)
	}
}

func TestFindUserConfigPathUsesPlatformConfigDirectory(t *testing.T) {
	for _, platformPath := range []struct {
		name string
		dir  string
	}{
		{name: "Windows APPDATA", dir: filepath.Join("AppData", "Roaming")},
		{name: "XDG_CONFIG_HOME", dir: filepath.Join(".xdg", "config")},
		{name: "macOS user config", dir: filepath.Join("Library", "Application Support")},
	} {
		t.Run(platformPath.name, func(t *testing.T) {
			td := t.TempDir()
			configDir := filepath.Join(td, platformPath.dir)
			want := writeTestFile(t, filepath.Join(configDir, "aiw", "aiw.toml"), "")
			stubInterpreterEnvironment(t, filepath.Join(td, "bin"), configDir, filepath.Join(td, "home"))

			got, err := findUserConfigPath()
			if err != nil {
				t.Fatalf("findUserConfigPath returned error: %v", err)
			}
			if got != want {
				t.Fatalf("findUserConfigPath() = %q, want %q", got, want)
			}
		})
	}
}

func TestResolveInterpreterCommandDoesNotMergeHomeFallback(t *testing.T) {
	td := t.TempDir()
	exeDir := filepath.Join(td, "bin")
	configRoot := filepath.Join(td, "canonical-config")
	homeDir := filepath.Join(td, "home")
	bundledPython := writeTestFile(t,
		filepath.Join(exeDir, "python", executableName("python")), "")
	ignoredPython := writeTestFile(t,
		filepath.Join(td, "python", executableName("ignored-home-python")), "")
	writeTestFile(t, filepath.Join(configRoot, "aiw", "aiw.toml"),
		"[runtime]\npython = \"\"\n")
	writeTestFile(t, filepath.Join(homeDir, ".config", "aiw", "aiw.toml"),
		"[runtime]\npython = \""+filepath.ToSlash(ignoredPython)+"\"\n")
	stubInterpreterEnvironment(t, exeDir, configRoot, homeDir)

	got, err := resolveInterpreterCommand(".py", "")
	if err != nil {
		t.Fatalf("resolveInterpreterCommand returned error: %v", err)
	}
	if len(got) != 1 || got[0] != bundledPython {
		t.Fatalf("resolveInterpreterCommand() = %v, want bundled [%q]", got, bundledPython)
	}
}

func TestResolveInterpreterCommandDoesNotCreateUserConfig(t *testing.T) {
	td := t.TempDir()
	exeDir := filepath.Join(td, "bin")
	configRoot := filepath.Join(td, "missing-config")
	homeDir := filepath.Join(td, "missing-home")
	bundledPython := writeTestFile(t,
		filepath.Join(exeDir, "python", executableName("python")), "")
	stubInterpreterEnvironment(t, exeDir, configRoot, homeDir)

	got, err := resolveInterpreterCommand(".py", "")
	if err != nil {
		t.Fatalf("resolveInterpreterCommand returned error: %v", err)
	}
	if len(got) != 1 || got[0] != bundledPython {
		t.Fatalf("resolveInterpreterCommand() = %v, want bundled [%q]", got, bundledPython)
	}
	for _, path := range []string{configRoot, homeDir} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("resolution created %q or returned unexpected stat error: %v", path, err)
		}
	}
}

func TestResolveInterpreterCommandRejectsInvalidExplicitPython(t *testing.T) {
	t.Run("relative environment path", func(t *testing.T) {
		td := t.TempDir()
		stubInterpreterEnvironment(t, filepath.Join(td, "bin"),
			filepath.Join(td, "config"), filepath.Join(td, "home"))
		t.Setenv("AIW_PYTHON", "relative/python")

		_, err := resolveInterpreterCommand(".py", "")
		assertErrorContains(t, err, "AIW_PYTHON", "absolute")
	})

	t.Run("missing program path", func(t *testing.T) {
		td := t.TempDir()
		exeDir := filepath.Join(td, "bin")
		missing := filepath.Join(td, "missing", executableName("python"))
		writeTestFile(t, filepath.Join(exeDir, "aiw.toml"),
			"[runtime]\npython = \""+filepath.ToSlash(missing)+"\"\n")
		stubInterpreterEnvironment(t, exeDir,
			filepath.Join(td, "config"), filepath.Join(td, "home"))

		_, err := resolveInterpreterCommand(".py", "")
		assertErrorContains(t, err, "program config", "not an existing file")
	})

	t.Run("user path is a directory", func(t *testing.T) {
		td := t.TempDir()
		exeDir := filepath.Join(td, "bin")
		configRoot := filepath.Join(td, "config")
		directory := filepath.Join(td, "python-directory")
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(configRoot, "aiw", "aiw.toml"),
			"[runtime]\npython = \""+filepath.ToSlash(directory)+"\"\n")
		stubInterpreterEnvironment(t, exeDir, configRoot, filepath.Join(td, "home"))

		_, err := resolveInterpreterCommand(".py", "")
		assertErrorContains(t, err, "user config", "not an existing file")
	})
}

func TestResolveInterpreterCommandReportsConfigDiscoveryErrors(t *testing.T) {
	t.Run("user config directory", func(t *testing.T) {
		td := t.TempDir()
		stubInterpreterEnvironment(t, filepath.Join(td, "bin"),
			filepath.Join(td, "config"), filepath.Join(td, "home"))
		userConfigDirFn = func() (string, error) {
			return "", errors.New("config unavailable")
		}

		_, err := resolveInterpreterCommand(".py", "")
		assertErrorContains(t, err, "resolve user config directory", "config unavailable")
	})

	t.Run("user config inspection", func(t *testing.T) {
		td := t.TempDir()
		configDir := filepath.Join(td, "config")
		canonicalPath := filepath.Join(configDir, "aiw", "aiw.toml")
		stubInterpreterEnvironment(t, filepath.Join(td, "bin"),
			configDir, filepath.Join(td, "home"))
		statPathFn = func(path string) (os.FileInfo, error) {
			if path == canonicalPath {
				return nil, os.ErrPermission
			}
			return os.Stat(path)
		}

		_, err := resolveInterpreterCommand(".py", "")
		assertErrorContains(t, err, "inspect user config", "permission denied")
	})
}

func TestResolveInterpreterCommandParsesFocusedRuntimeConfig(t *testing.T) {
	t.Run("literal string comment and unrelated section", func(t *testing.T) {
		td := t.TempDir()
		exeDir := filepath.Join(td, "bin")
		python := writeTestFile(t, filepath.Join(td, "python", executableName("configured-python")), "")
		writeTestFile(t, filepath.Join(exeDir, "aiw.toml"),
			"[unrelated]\ninvalid syntax is ignored\n\n[runtime]\npython = '"+
				filepath.ToSlash(python)+"' # selected runtime\nother = \"ignored\"\n")
		stubInterpreterEnvironment(t, exeDir,
			filepath.Join(td, "config"), filepath.Join(td, "home"))

		got, err := resolveInterpreterCommand(".py", "")
		if err != nil {
			t.Fatalf("resolveInterpreterCommand returned error: %v", err)
		}
		if len(got) != 1 || got[0] != python {
			t.Fatalf("resolveInterpreterCommand() = %v, want [%q]", got, python)
		}
	})

	t.Run("empty value falls back", func(t *testing.T) {
		td := t.TempDir()
		exeDir := filepath.Join(td, "bin")
		bundledPython := writeTestFile(t,
			filepath.Join(exeDir, "python", executableName("python")), "")
		writeTestFile(t, filepath.Join(exeDir, "aiw.toml"),
			"[runtime]\npython = \"   \"\n")
		stubInterpreterEnvironment(t, exeDir,
			filepath.Join(td, "config"), filepath.Join(td, "home"))

		got, err := resolveInterpreterCommand(".py", "")
		if err != nil {
			t.Fatalf("resolveInterpreterCommand returned error: %v", err)
		}
		if len(got) != 1 || got[0] != bundledPython {
			t.Fatalf("resolveInterpreterCommand() = %v, want bundled [%q]", got, bundledPython)
		}
	})

	t.Run("malformed runtime value reports source", func(t *testing.T) {
		td := t.TempDir()
		exeDir := filepath.Join(td, "bin")
		writeTestFile(t, filepath.Join(exeDir, "aiw.toml"),
			"[runtime]\npython = not-quoted\n")
		stubInterpreterEnvironment(t, exeDir,
			filepath.Join(td, "config"), filepath.Join(td, "home"))

		_, err := resolveInterpreterCommand(".py", "")
		assertErrorContains(t, err, "program config", "quoted string")
	})

	t.Run("duplicate runtime value is rejected", func(t *testing.T) {
		td := t.TempDir()
		exeDir := filepath.Join(td, "bin")
		writeTestFile(t, filepath.Join(exeDir, "aiw.toml"),
			"[runtime]\npython = \"\"\npython = \"\"\n")
		stubInterpreterEnvironment(t, exeDir,
			filepath.Join(td, "config"), filepath.Join(td, "home"))

		_, err := resolveInterpreterCommand(".py", "")
		assertErrorContains(t, err, "program config", "duplicate runtime.python")
	})

	t.Run("commented table header leaves runtime section", func(t *testing.T) {
		td := t.TempDir()
		exeDir := filepath.Join(td, "bin")
		bundledPython := writeTestFile(t,
			filepath.Join(exeDir, "python", executableName("python")), "")
		writeTestFile(t, filepath.Join(exeDir, "aiw.toml"),
			"[runtime]\nother = \"ignored\"\n[other] # leave runtime\npython = \"relative/ignored\"\n")
		stubInterpreterEnvironment(t, exeDir,
			filepath.Join(td, "config"), filepath.Join(td, "home"))

		got, err := resolveInterpreterCommand(".py", "")
		if err != nil {
			t.Fatalf("resolveInterpreterCommand returned error: %v", err)
		}
		if len(got) != 1 || got[0] != bundledPython {
			t.Fatalf("resolveInterpreterCommand() = %v, want bundled [%q]", got, bundledPython)
		}
	})

	t.Run("malformed runtime header reports source", func(t *testing.T) {
		td := t.TempDir()
		exeDir := filepath.Join(td, "bin")
		writeTestFile(t, filepath.Join(exeDir, "aiw.toml"),
			"[runtime\npython = \"\"\n")
		stubInterpreterEnvironment(t, exeDir,
			filepath.Join(td, "config"), filepath.Join(td, "home"))

		_, err := resolveInterpreterCommand(".py", "")
		assertErrorContains(t, err, "program config", "malformed table header")
	})

	t.Run("malformed unrelated header is ignored", func(t *testing.T) {
		td := t.TempDir()
		exeDir := filepath.Join(td, "bin")
		python := writeTestFile(t,
			filepath.Join(td, "python", executableName("configured-python")), "")
		writeTestFile(t, filepath.Join(exeDir, "aiw.toml"),
			"[notruntime\ninvalid syntax is ignored\n[runtime]\npython = '"+
				filepath.ToSlash(python)+"'\n")
		stubInterpreterEnvironment(t, exeDir,
			filepath.Join(td, "config"), filepath.Join(td, "home"))

		got, err := resolveInterpreterCommand(".py", "")
		if err != nil {
			t.Fatalf("resolveInterpreterCommand returned error: %v", err)
		}
		if len(got) != 1 || got[0] != python {
			t.Fatalf("resolveInterpreterCommand() = %v, want [%q]", got, python)
		}
	})
}

func TestResolveInterpreterCommandFallsBackToPython3(t *testing.T) {
	td := t.TempDir()
	exeDir := filepath.Join(td, "bin")
	stubInterpreterEnvironment(t, exeDir,
		filepath.Join(td, "config"), filepath.Join(td, "home"))

	oldLookPathFn := lookPathFn
	lookPathFn = func(file string) (string, error) {
		if file == "python3" {
			return "/usr/bin/python3", nil
		}
		return "", os.ErrNotExist
	}
	t.Cleanup(func() {
		lookPathFn = oldLookPathFn
	})

	got, err := resolveInterpreterCommand(".py", "")
	if err != nil {
		t.Fatalf("resolveInterpreterCommand returned error: %v", err)
	}
	if len(got) != 1 || got[0] != "/usr/bin/python3" {
		t.Fatalf("resolveInterpreterCommand() = %v, want [/usr/bin/python3]", got)
	}
}

func TestPythonConfigurationDoesNotAffectOtherInterpreters(t *testing.T) {
	for _, runtimeCase := range []struct {
		name     string
		ext      string
		subdir   string
		command  string
		extraArg string
	}{
		{name: "Perl", ext: ".pl", subdir: "perl", command: "perl"},
		{name: "Bash", ext: ".sh", subdir: "bash", command: "bash"},
		{name: "Java", ext: ".jar", subdir: filepath.Join("java", "bin"), command: "java", extraArg: "-jar"},
	} {
		t.Run(runtimeCase.name, func(t *testing.T) {
			td := t.TempDir()
			exeDir := filepath.Join(td, "bin")
			interpreter := writeTestFile(t,
				filepath.Join(exeDir, runtimeCase.subdir, executableName(runtimeCase.command)), "")
			writeTestFile(t, filepath.Join(exeDir, "aiw.toml"),
				"[runtime]\npython = \"relative/python\"\n")
			stubInterpreterEnvironment(t, exeDir,
				filepath.Join(td, "config"), filepath.Join(td, "home"))
			t.Setenv("AIW_PYTHON", "also/relative")

			got, err := resolveInterpreterCommand(runtimeCase.ext, "")
			if err != nil {
				t.Fatalf("resolveInterpreterCommand returned error: %v", err)
			}
			want := []string{interpreter}
			if runtimeCase.extraArg != "" {
				want = append(want, runtimeCase.extraArg)
			}
			if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
				t.Fatalf("resolveInterpreterCommand() = %v, want %v", got, want)
			}
		})
	}
}

func TestPythonConfigurationDoesNotAffectDirectCommandRuntimes(t *testing.T) {
	t.Setenv("AIW_PYTHON", "relative/python")
	oldLookPathFn := lookPathFn
	lookPathFn = func(file string) (string, error) {
		return "", os.ErrNotExist
	}
	t.Cleanup(func() {
		lookPathFn = oldLookPathFn
	})

	for _, runtimeCase := range []struct {
		name        string
		pluginPath  string
		wantCommand string
	}{
		{name: "JavaScript", pluginPath: "plugin.js", wantCommand: "node"},
		{name: "PowerShell", pluginPath: "plugin.ps1", wantCommand: powerShellCommand()},
	} {
		t.Run(runtimeCase.name, func(t *testing.T) {
			cmd, err := buildPluginCommand(runtimeCase.pluginPath, []string{"arg"})
			if err != nil {
				t.Fatalf("buildPluginCommand returned error: %v", err)
			}
			if filepath.Base(cmd.Path) != executableName(runtimeCase.wantCommand) &&
				filepath.Base(cmd.Path) != runtimeCase.wantCommand {
				t.Fatalf("buildPluginCommand path = %q, want %q", cmd.Path, runtimeCase.wantCommand)
			}
		})
	}
}

func powerShellCommand() string {
	if filepath.Separator == '\\' {
		return "powershell"
	}
	return "pwsh"
}

func stubInterpreterEnvironment(t *testing.T, exeDir, configDir, homeDir string) {
	t.Helper()
	t.Setenv("AIW_PYTHON", "")

	oldExecutablePathFn := pluginExecutablePathFn
	oldUserConfigDirFn := userConfigDirFn
	oldUserHomeDirFn := userHomeDirFn
	oldStatPathFn := statPathFn
	pluginExecutablePathFn = func() (string, error) {
		return filepath.Join(exeDir, "aiw.exe"), nil
	}
	userConfigDirFn = func() (string, error) {
		return configDir, nil
	}
	userHomeDirFn = func() (string, error) {
		return homeDir, nil
	}
	statPathFn = os.Stat
	t.Cleanup(func() {
		pluginExecutablePathFn = oldExecutablePathFn
		userConfigDirFn = oldUserConfigDirFn
		userHomeDirFn = oldUserHomeDirFn
		statPathFn = oldStatPathFn
	})
}

func assertErrorContains(t *testing.T, err error, parts ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, part := range parts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error %q does not contain %q", err, part)
		}
	}
}
