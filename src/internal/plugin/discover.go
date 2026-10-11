package plugin

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var ErrPluginNotFound = errors.New("plugin not found")

var extPriority = map[string]int{
	".bat": 0,
	".cmd": 0,
	".sh":  0,
	".ps1": 0,
	".py":  1,
	"":     2,
	".exe": 3,
	".js":  4,
	".mjs": 5,
	".cjs": 6,
	".ts":  7,
	".mts": 8,
	".cts": 9,
}

// DiscoverPlugin searches standard locations for a plugin named `name` and
// returns the absolute path to the executable/script to run, or an error if not found.
func DiscoverPlugin(name string) (string, error) {
	plugin, err := DiscoverPluginInfo(name)
	if err != nil {
		return "", err
	}
	return plugin.Path, nil
}

// DiscoverPluginInfo searches the standard locations and returns the complete
// launch descriptor for a plugin.
func DiscoverPluginInfo(name string) (PluginInfo, error) {
	var paths []string
	if exePath, err := os.Executable(); err == nil {
		// println("exePath:",exePath)
		exeDir := filepath.Dir(exePath)
		paths = append(paths, filepath.Join(exeDir, "plugins"))
	}
	// also check current working directory's plugins (useful in development)
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, "plugins"))
	}
	// Repository source plugins are available during development. Installed
	// plugins beside the executable and workspace-local plugins keep priority.
	if root := checkoutRoot(); root != "" {
		paths = append(paths, filepath.Join(root, "src", "plugins"))
	}
	// println("searching plugins in paths:", paths[0])
	// Pass plugin directories; PATH entries are handled inside DiscoverPluginIn
	return DiscoverPluginInfoIn(paths, name)
}

func checkoutRoot() string {
	command := exec.Command("git", "rev-parse", "--show-toplevel")
	if output, err := command.Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	return ""
}

// DiscoverPluginIn searches the provided paths (in order) for a plugin named `name`.
// Each entry in paths may be a directory; `plugins` directories may be recursive by one level.
func DiscoverPluginIn(paths []string, name string) (string, error) {
	plugin, err := DiscoverPluginInfoIn(paths, name)
	if err != nil {
		return "", err
	}
	return plugin.Path, nil
}

// DiscoverPluginInfoIn searches plugin directories in order, then PATH, for a
// plugin named `name`. A directory manifest is authoritative for that package.
func DiscoverPluginInfoIn(paths []string, name string) (PluginInfo, error) {
	wantBase := "aiw-" + name

	for _, base := range paths {
		fi, err := os.Stat(base)
		if err != nil {
			continue
		}
		if !fi.IsDir() {
			// PATH entries: only check files directly matching aiw-<name>.* or aiw-<name>
			// skip non-directory entries here; PATH entries handled later
			continue
		}

		// If the path looks like a plugins directory, we'll search top-level files
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		var candidates []string
		for _, e := range entries {
			if e.IsDir() {
				sub := filepath.Join(base, e.Name())
				manifestPath := filepath.Join(sub, manifestFileName)
				if _, err := os.Stat(manifestPath); err == nil {
					manifest, err := readPluginManifest(sub)
					if err != nil {
						return PluginInfo{}, err
					}
					for _, definition := range manifest {
						if definition.Name != name {
							continue
						}
						return resolveManifestPlugin(sub, definition)
					}
					continue
				} else if !errors.Is(err, os.ErrNotExist) {
					return PluginInfo{}, fmt.Errorf("inspect plugin manifest %s: %w", manifestPath, err)
				}
				subEntries, err := os.ReadDir(sub)
				if err != nil {
					continue
				}
				for _, se := range subEntries {
					if !se.IsDir() && matchPluginName(se.Name(), wantBase) {
						candidates = append(candidates, filepath.Join(sub, se.Name()))
					}
				}
				continue
			}
			if matchPluginName(e.Name(), wantBase) {
				candidates = append(candidates, filepath.Join(base, e.Name()))
			}
		}
		if len(candidates) > 0 {
			return PluginInfo{Name: name, Path: bestCandidate(candidates), Startup: StartupAuto}, nil
		}
	}

	// Also search PATH entries (files directly in PATH)
	var candidates []string
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == "" {
			continue
		}
		file := filepath.Join(p, wantBase)
		// check base name and with known extensions
		if existsAndExecutable(file) {
			candidates = append(candidates, file)
		}
		for ext := range extPriority {
			if ext == "" {
				continue
			}
			fileExt := file + ext
			if existsAndExecutable(fileExt) {
				candidates = append(candidates, fileExt)
			}
		}
	}

	if len(candidates) == 0 {
		return PluginInfo{}, ErrPluginNotFound
	}

	return PluginInfo{Name: name, Path: bestCandidate(candidates), Startup: StartupAuto}, nil
}

func resolveManifestPlugin(directory string, definition pluginManifestEntry) (PluginInfo, error) {
	entrypoint := definition.Entrypoint
	if entrypoint == "" {
		wantBase := "aiw-" + definition.Name
		entries, err := os.ReadDir(directory)
		if err != nil {
			return PluginInfo{}, fmt.Errorf("read plugin directory %s: %w", directory, err)
		}
		var candidates []string
		for _, entry := range entries {
			if !entry.IsDir() && matchPluginName(entry.Name(), wantBase) {
				candidates = append(candidates, filepath.Join(directory, entry.Name()))
			}
		}
		if len(candidates) == 0 {
			return PluginInfo{}, fmt.Errorf("plugin %q in %s has no entrypoint and no legacy aiw-%s file", definition.Name, filepath.Join(directory, manifestFileName), definition.Name)
		}
		entrypoint = bestCandidate(candidates)
	} else {
		resolved, err := resolveManifestEntrypoint(directory, entrypoint)
		if err != nil {
			return PluginInfo{}, fmt.Errorf("plugin %q in %s: %w", definition.Name, filepath.Join(directory, manifestFileName), err)
		}
		entrypoint = resolved
	}
	return PluginInfo{
		Name:        definition.Name,
		Description: definition.Description,
		Help:        definition.Help,
		Path:        entrypoint,
		Startup:     definition.Startup,
	}, nil
}

func bestCandidate(candidates []string) string {
	best := candidates[0]
	bestScore := scoreExt(best)
	for _, c := range candidates[1:] {
		s := scoreExt(c)
		if s < bestScore {
			best = c
			bestScore = s
		}
	}
	return best
}

func matchPluginName(filename, wantBase string) bool {
	// exact match
	if filename == wantBase {
		return true
	}
	// check known extensions
	lower := strings.ToLower(filename)
	for ext := range extPriority {
		if ext == "" {
			continue
		}
		if lower == strings.ToLower(wantBase+ext) {
			return true
		}
	}
	// fallback: strip last dot segment
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		if filename[:idx] == wantBase {
			return true
		}
	}
	return false
}

func scoreExt(path string) int {
	ext := strings.ToLower(filepath.Ext(path))
	// Prefer the Windows executable when a plugin ships both platform binaries.
	if runtime.GOOS == "windows" {
		if ext == ".exe" {
			return extPriority[""]
		}
		if ext == "" {
			return extPriority[".exe"]
		}
	}
	if ext == "" {
		// if file is text and has shebang, treat as empty ext priority
		if hasShebang(path) {
			return extPriority[""]
		}
	}
	if v, ok := extPriority[ext]; ok {
		return v
	}
	// fallback
	return 100
}

func hasShebang(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	// read a small prefix to detect shebang without relying on newline
	buf := make([]byte, 64)
	n, err := f.Read(buf)
	if err != nil {
		// reading may fail for binary files, treat as no shebang
		return false
	}
	return strings.HasPrefix(string(buf[:n]), "#!")
}

func existsAndExecutable(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		if fi.IsDir() {
			return false
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == "" {
			return false
		}
		if _, ok := extPriority[ext]; ok {
			return true
		}
		// also respect PATHEXT if set
		pathext := os.Getenv("PATHEXT")
		for _, e := range filepath.SplitList(pathext) {
			if strings.EqualFold(e, ext) {
				return true
			}
		}
		return false
	}
	if fi.IsDir() {
		return false
	}
	mode := fi.Mode()
	return mode&0111 != 0 || hasShebang(path)
}
