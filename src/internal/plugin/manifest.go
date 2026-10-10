package plugin

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const manifestFileName = "plugin.toml"

type StartupMode string

const (
	StartupAuto       StartupMode = "auto"
	StartupExec       StartupMode = "exec"
	StartupPython     StartupMode = "python"
	StartupNode       StartupMode = "node"
	StartupTypeScript StartupMode = "typescript"
	StartupBash       StartupMode = "bash"
	StartupPowerShell StartupMode = "powershell"
)

// PluginInfo is the resolved metadata and launch configuration for one plugin.
type PluginInfo struct {
	Name        string
	Description string
	Help        string
	Path        string
	Startup     StartupMode
}

type pluginManifest struct {
	Schema  int                    `toml:"schema"`
	Plugins []pluginManifestEntry `toml:"plugins"`
}

type pluginManifestEntry struct {
	Name        string      `toml:"name"`
	Description string      `toml:"description"`
	Help        string      `toml:"help"`
	Entrypoint  string      `toml:"entrypoint"`
	Startup     StartupMode `toml:"startup"`
}

func readPluginManifest(directory string) ([]pluginManifestEntry, error) {
	manifestPath := filepath.Join(directory, manifestFileName)
	var manifest pluginManifest
	metadata, err := toml.DecodeFile(manifestPath, &manifest)
	if err != nil {
		return nil, fmt.Errorf("read plugin manifest %s: %w", manifestPath, err)
	}
	if keys := metadata.Undecoded(); len(keys) > 0 {
		return nil, fmt.Errorf("read plugin manifest %s: unknown field %q", manifestPath, keys[0].String())
	}
	if manifest.Schema != 1 {
		return nil, fmt.Errorf("read plugin manifest %s: schema must be 1", manifestPath)
	}
	if len(manifest.Plugins) == 0 {
		return nil, fmt.Errorf("read plugin manifest %s: at least one [[plugins]] entry is required", manifestPath)
	}

	seen := make(map[string]bool, len(manifest.Plugins))
	for i := range manifest.Plugins {
		entry := &manifest.Plugins[i]
		if !validPluginName(entry.Name) {
			return nil, fmt.Errorf("read plugin manifest %s: plugins[%d].name must use lowercase letters, digits, and single hyphens", manifestPath, i)
		}
		if seen[entry.Name] {
			return nil, fmt.Errorf("read plugin manifest %s: duplicate plugin name %q", manifestPath, entry.Name)
		}
		seen[entry.Name] = true
		if strings.TrimSpace(entry.Description) == "" {
			return nil, fmt.Errorf("read plugin manifest %s: plugins[%d].description is required", manifestPath, i)
		}
		if entry.Startup == "" {
			entry.Startup = StartupAuto
		}
		if !validStartupMode(entry.Startup) {
			return nil, fmt.Errorf("read plugin manifest %s: unsupported startup mode %q", manifestPath, entry.Startup)
		}
	}
	return manifest.Plugins, nil
}

func resolveManifestEntrypoint(directory, entrypoint string) (string, error) {
	if strings.Contains(entrypoint, "\\") || path.IsAbs(entrypoint) || filepath.VolumeName(filepath.FromSlash(entrypoint)) != "" {
		return "", errors.New("entrypoint must be relative to the plugin directory")
	}
	clean := path.Clean(entrypoint)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("entrypoint must stay inside the plugin directory")
	}

	root, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return "", fmt.Errorf("resolve plugin directory: %w", err)
	}
	fullPath := filepath.Join(directory, filepath.FromSlash(clean))
	resolved, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve entrypoint %q: %w", entrypoint, err)
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("entrypoint resolves outside the plugin directory")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("inspect entrypoint %q: %w", entrypoint, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("entrypoint %q is a directory", entrypoint)
	}
	return resolved, nil
}

func validPluginName(name string) bool {
	if name == "" {
		return false
	}
	first := name[0]
	if (first < 'a' || first > 'z') && (first < '0' || first > '9') {
		return false
	}
	previousHyphen := false
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '-' {
			if i == 0 || i == len(name)-1 || previousHyphen {
				return false
			}
			previousHyphen = true
			continue
		}
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
		previousHyphen = false
	}
	return true
}

// ListPluginsIn returns discoverable plugins from the given plugin directories.
// The first occurrence of a command name in path order wins.
func ListPluginsIn(paths []string) ([]PluginInfo, error) {
	var plugins []PluginInfo
	seen := make(map[string]bool)
	for _, base := range paths {
		entries, err := os.ReadDir(base)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read plugin directory %s: %w", base, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				directory := filepath.Join(base, entry.Name())
				manifestPath := filepath.Join(directory, manifestFileName)
				if _, err := os.Stat(manifestPath); err == nil {
					definitions, err := readPluginManifest(directory)
					if err != nil {
						return nil, err
					}
					for _, definition := range definitions {
						if seen[definition.Name] {
							continue
						}
						plugin, err := resolveManifestPlugin(directory, definition)
						if err != nil {
							return nil, err
						}
						plugins = append(plugins, plugin)
						seen[plugin.Name] = true
					}
					continue
				} else if !errors.Is(err, os.ErrNotExist) {
					return nil, fmt.Errorf("inspect plugin manifest %s: %w", manifestPath, err)
				}
				files, err := os.ReadDir(directory)
				if err != nil {
					continue
				}
				for _, file := range files {
					if name, ok := legacyPluginName(file.Name()); ok && !seen[name] && !file.IsDir() {
						plugins = append(plugins, PluginInfo{Name: name, Path: filepath.Join(directory, file.Name()), Startup: StartupAuto})
						seen[name] = true
					}
				}
				continue
			}
			if name, ok := legacyPluginName(entry.Name()); ok && !seen[name] {
				plugins = append(plugins, PluginInfo{Name: name, Path: filepath.Join(base, entry.Name()), Startup: StartupAuto})
				seen[name] = true
			}
		}
	}
	return plugins, nil
}

func legacyPluginName(filename string) (string, bool) {
	if !strings.HasPrefix(filename, "aiw-") {
		return "", false
	}
	extension := filepath.Ext(filename)
	switch strings.ToLower(extension) {
	case "", ".py", ".exe":
		name := strings.TrimSuffix(strings.TrimPrefix(filename, "aiw-"), extension)
		return name, name != ""
	default:
		return "", false
	}
}

func validStartupMode(mode StartupMode) bool {
	switch mode {
	case StartupAuto, StartupExec, StartupPython, StartupNode, StartupTypeScript, StartupBash, StartupPowerShell:
		return true
	default:
		return false
	}
}
