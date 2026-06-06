package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var extPriority = map[string]int{
	".bat": 0,
	".cmd": 0,
	".sh":  0,
	".ps1": 0,
	".py":  1,
	"":     2,
	".exe": 3,
	".js":  4,
}

// DiscoverPlugin searches standard locations for a plugin named `name` and
// returns the absolute path to the executable/script to run, or an error if not found.
func DiscoverPlugin(name string) (string, error) {
	var paths []string
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		paths = append(paths, filepath.Join(exeDir, "plugins"))
	}
	println("searching plugins in paths:", paths[0])
	// Pass plugin directories; PATH entries are handled inside DiscoverPluginIn
	return DiscoverPluginIn(paths, name)
}

// DiscoverPluginIn searches the provided paths (in order) for a plugin named `name`.
// Each entry in paths may be a directory; `plugins` directories may be recursive by one level.
func DiscoverPluginIn(paths []string, name string) (string, error) {
	var candidates []string
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
		for _, e := range entries {
			// check file match
			if e.IsDir() {
				// search one level down
				sub := filepath.Join(base, e.Name())
				subEntries, _ := os.ReadDir(sub)
				for _, se := range subEntries {
					if matchPluginName(se.Name(), wantBase) {
						candidates = append(candidates, filepath.Join(sub, se.Name()))
					}
				}
				continue
			}
			if matchPluginName(e.Name(), wantBase) {
				candidates = append(candidates, filepath.Join(base, e.Name()))
			}
		}
	}

	// Also search PATH entries (files directly in PATH)
	// Also search PATH entries (files directly in PATH)
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
		return "", errors.New("plugin not found")
	}

	// choose best candidate by extension priority
	best := candidates[0]
	bestScore := scoreExt(best)
	for _, c := range candidates[1:] {
		s := scoreExt(c)
		if s < bestScore {
			best = c
			bestScore = s
		}
	}
	return best, nil
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
