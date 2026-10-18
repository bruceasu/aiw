package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"
)

const maxBytes int64 = 4 << 30
const maxEntries = 100000
const maxPathBytes = 4096

type entry struct {
	name, data, target string
	kind byte // f: ordinary file, d: directory, l: symbolic link
	mode os.FileMode
	modified time.Time
	size int64
	digest [32]byte
}

type snapshot struct {
	root, removeRoot *os.Root
	rel, removeRel, display, target string
	info os.FileInfo
	digest [32]byte
	hasDigest bool
}

type operation struct {
	dir string
	total int64
	entries []entry
	sources []snapshot
	roots []*os.Root
	protected []os.FileInfo
}

func (op *operation) close() error {
	var result error
	for _, r := range op.roots { result = errors.Join(result, r.Close()) }
	// dir is the exact path returned by MkdirTemp, never a user-supplied path.
	return errors.Join(result, os.RemoveAll(op.dir))
}

func (op *operation) root(name string) (*os.Root, error) {
	r, err := os.OpenRoot(name)
	if err == nil { op.roots = append(op.roots, r) }
	return r, err
}

func (op *operation) stage(r io.Reader) (name string, n int64, digest [32]byte, err error) {
	f, err := os.CreateTemp(op.dir, "data-")
	if err != nil { return "", 0, digest, err }
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h), io.LimitReader(r, maxBytes-op.total+1))
	err = errors.Join(copyErr, f.Close())
	if err == nil && n > maxBytes-op.total { err = fmt.Errorf("expanded data exceeds 4 GiB limit") }
	if err != nil { return "", n, digest, err }
	op.total += n
	copy(digest[:], h.Sum(nil))
	return f.Name(), n, digest, nil
}

func safeName(name string) (string, error) {
	name = strings.TrimSuffix(name, "/")
	for strings.HasPrefix(name, "./") { name = strings.TrimPrefix(name, "./") }
	if name == "" || len(name) > maxPathBytes || strings.ContainsAny(name, "\\\x00:") || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	parts := strings.Split(name, "/")
	if len(parts) > 128 { return "", fmt.Errorf("archive path exceeds 128 levels") }
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return "", fmt.Errorf("unsafe archive path %q", name)
		}
		for _, c := range part { if c < 32 || strings.ContainsRune("<>\"|?*", c) { return "", fmt.Errorf("nonportable archive path %q", name) } }
		stem := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" || stem == "CONIN$" || stem == "CONOUT$" ||
			(len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) && stem[3] >= '0' && stem[3] <= '9') {
			return "", fmt.Errorf("reserved device name %q", name)
		}
	}
	return name, nil
}

func nameKey(name string) string { return strings.ToLower(norm.NFC.String(name)) }

func validateEntries(entries []entry, links bool) (map[string]entry, []entry, error) {
	if len(entries) > maxEntries { return nil, nil, fmt.Errorf("archive exceeds %d entries", maxEntries) }
	byName := make(map[string]entry)
	spellings := make(map[string]string)
	for _, e := range entries {
		if _, err := safeName(e.name); err != nil { return nil, nil, err }
		if _, ok := byName[e.name]; ok { return nil, nil, fmt.Errorf("duplicate archive entry %q", e.name) }
		byName[e.name] = e
		for p := e.name; p != "."; p = path.Dir(p) {
			key := nameKey(p)
			if old, ok := spellings[key]; ok && old != p { return nil, nil, fmt.Errorf("colliding archive names %q and %q", old, p) }
			spellings[key] = p
		}
	}
	for _, e := range entries {
		for p := path.Dir(e.name); p != "."; p = path.Dir(p) {
			if ancestor, ok := byName[p]; ok && ancestor.kind != 'd' { return nil, nil, fmt.Errorf("non-directory ancestor %q", p) }
		}
	}
	// Add implied directories to the target namespace so a link can reference
	// a valid parent even when an archive omits explicit directory headers.
	for _, e := range entries {
		for p := path.Dir(e.name); p != "."; p = path.Dir(p) {
			if _, ok := byName[p]; !ok { byName[p] = entry{name:p, kind:'d', mode:0755} }
		}
	}
	active := make(map[string]bool)
	orderedSet := make(map[string]bool)
	var ordered []entry
	isDirectory := func(stack []string) bool {
		return len(stack) == 0 || byName[strings.Join(stack, "/")].kind == 'd'
	}
	// Expand links before interpreting subsequent dot/dot-dot components.
	// Cleaning the whole target first does not match filesystem semantics.
	var resolve func([]string, []string, int) ([]string, error)
	resolve = func(parts, parent []string, depth int) ([]string, error) {
		if depth > 128 { return nil, fmt.Errorf("link chain exceeds 128 levels") }
		stack := append([]string(nil), parent...)
		for i, part := range parts {
			switch part {
			case "", ".":
				if !isDirectory(stack) { return nil, fmt.Errorf("link target traverses a file") }
				continue
			case "..":
				if !isDirectory(stack) { return nil, fmt.Errorf("link target traverses a file") }
				if len(stack) == 0 { return nil, fmt.Errorf("link target escapes archive") }
				stack = stack[:len(stack)-1]
				continue
			}
			if _, err := safeName(part); err != nil { return nil, err }
			candidate := append(append([]string(nil), stack...), part)
			prefix := strings.Join(candidate, "/")
			e, exists := byName[prefix]
			if !exists { return nil, fmt.Errorf("dangling archive link target %q", prefix) }
			if e.kind == 'l' {
				if !links { return nil, fmt.Errorf("symbolic links require -l") }
				if active[prefix] { return nil, fmt.Errorf("cyclic link %q", prefix) }
				if e.target == "" || len(e.target) > maxPathBytes || strings.HasPrefix(e.target, "/") || strings.ContainsAny(e.target, "\\\x00:") {
					return nil, fmt.Errorf("unsafe link target %q", e.target)
				}
				active[prefix] = true
				resolved, err := resolve(strings.Split(e.target, "/"), stack, depth+1)
				if err != nil { return nil, err }
				active[prefix] = false
				stack = resolved
				if !orderedSet[prefix] { ordered = append(ordered, e); orderedSet[prefix] = true }
			} else { stack = candidate }
			if i < len(parts)-1 && !isDirectory(stack) { return nil, fmt.Errorf("link target traverses a file %q", prefix) }
		}
		return stack, nil
	}
	directoryEdges := make(map[string][]string)
	for name, e := range byName { if e.kind == 'd' { directoryEdges[path.Dir(name)] = append(directoryEdges[path.Dir(name)], name) } }
	for _, e := range entries {
		if e.kind == 'l' {
			if !links { return nil, nil, fmt.Errorf("symbolic links require -l: %q", e.name) }
			components, err := resolve(strings.Split(e.name, "/"), nil, 0)
			resolved := strings.Join(components, "/")
			if resolved == "" { resolved = "." }
			if err != nil { return nil, nil, err }
			if target := byName[resolved]; target.kind == 'd' && strings.HasPrefix(e.name, resolved+"/") {
				return nil, nil, fmt.Errorf("directory link cycle at %q", e.name)
			}
			if resolved == "." || byName[resolved].kind == 'd' { directoryEdges[path.Dir(e.name)] = append(directoryEdges[path.Dir(e.name)], resolved) }
		}
	}
	directoryState := make(map[string]byte)
	var visit func(string, int) error
	visit = func(name string, depth int) error {
		if depth > 128 || directoryState[name] == 1 { return fmt.Errorf("cyclic or too-deep directory link graph at %q", name) }
		if directoryState[name] == 2 { return nil }
		directoryState[name] = 1
		for _, child := range directoryEdges[name] { if err := visit(child, depth+1); err != nil { return err } }
		directoryState[name] = 2
		return nil
	}
	if err := visit(".", 0); err != nil { return nil, nil, err }
	return byName, ordered, nil
}

func within(base, candidate string) bool {
	rel, err := filepath.Rel(base, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// Resolve existing path components without requiring the final output to exist.
func canonicalOutput(name string) (string, error) {
	abs, err := filepath.Abs(name)
	if err != nil { return "", err }
	var suffix []string
	p := abs
	for {
		resolved, err := filepath.EvalSymlinks(p)
		if err == nil {
			for i := len(suffix)-1; i >= 0; i-- { resolved = filepath.Join(resolved, suffix[i]) }
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) { return "", err }
		parent := filepath.Dir(p)
		if parent == p { return "", err }
		suffix = append(suffix, filepath.Base(p))
		p = parent
	}
}
