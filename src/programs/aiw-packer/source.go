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
)

func sameSnapshot(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Mode() == b.Mode() && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

func (op *operation) collect(o options, stdin io.Reader, codec nameCodec) error {
	if (o.format == "gz" || o.format == "bz2") && len(o.inputs) != 1 { return fmt.Errorf("%s needs one file; use tar.%s for multiple inputs", o.format, o.format) }
	var selected []string
	output := ""
	if !o.stdout {
		var err error
		output, err = canonicalOutput(o.output)
		if err != nil { return err }
	}
	for _, input := range o.inputs {
		if input == "-" {
			data, n, digest, err := op.stage(stdin)
			if err != nil { return err }
			op.entries = append(op.entries, entry{name:"stdin", kind:'f', mode:0644, data:data, size:n, digest:digest})
			continue
		}
		abs, err := filepath.Abs(input)
		if err != nil { return err }
		info, err := os.Lstat(abs)
		if err != nil { return err }
		parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
		if err != nil { return err }
		lexical := filepath.Join(parent, filepath.Base(abs))
		actual := lexical
		if o.follow {
			actual, err = filepath.EvalSymlinks(abs)
			if err != nil { return fmt.Errorf("cannot follow input %s: %w", input, err) }
			info, err = os.Stat(actual)
			if err != nil { return err }
		}
		for _, previous := range selected {
			if within(previous, actual) || within(actual, previous) { return fmt.Errorf("overlapping or repeated inputs: %s", input) }
		}
		selected = append(selected, actual)
		if info.IsDir() && within(actual, op.dir) { return fmt.Errorf("temporary staging directory is inside input; choose a TMPDIR outside %s", input) }
		if output != "" {
			if output == actual || (info.IsDir() && within(actual, output)) { return fmt.Errorf("output is identical to or inside input: %s", input) }
			if outInfo, err := os.Stat(output); err == nil && os.SameFile(info, outInfo) { return fmt.Errorf("output aliases input: %s", input) }
		}
		logical, err := safeName(filepath.Base(filepath.Clean(input)))
		if err != nil { return err }
		anchor := filepath.Dir(actual)
		rel := filepath.Base(actual)
		if info.IsDir() { anchor, rel = actual, "." }
		root, err := op.root(anchor)
		if err != nil { return err }
		var parentRoot *os.Root
		if info.IsDir() && o.deleteSource {
			parentRoot, err = op.root(filepath.Dir(actual))
			if err != nil { return err }
		}
		if err := op.walkSource(root, anchor, rel, logical, o, nil, parentRoot, filepath.Base(actual)); err != nil { return err }
	}
	if o.format == "gz" || o.format == "bz2" {
		if len(op.entries) != 1 || op.entries[0].kind != 'f' { return fmt.Errorf("%s needs one ordinary file or stdin; use tar.%s for directories/multiple inputs", o.format, o.format) }
	}
	if _, _, err := validateEntries(op.entries, o.links); err != nil { return err }
	for _, e := range op.entries {
		if _, err := codec.encode(e.name); err != nil && o.format != "gz" && o.format != "bz2" { return err }
		if e.kind == 'l' { if _, err := codec.encode(e.target); err != nil { return err } }
	}
	return nil
}

func (op *operation) walkSource(root *os.Root, anchor, rel, logical string, o options, ancestors []os.FileInfo, parentRoot *os.Root, base string) error {
	if len(op.entries) >= maxEntries { return fmt.Errorf("input exceeds %d entries", maxEntries) }
	if len(ancestors) > 128 { return fmt.Errorf("directory depth exceeds 128") }
	if _, err := safeName(logical); err != nil { return err }
	info, err := root.Lstat(rel)
	if err != nil { return err }
	if info.Mode()&os.ModeSymlink != 0 && o.follow {
		real, err := filepath.EvalSymlinks(filepath.Join(anchor, rel))
		if err != nil { return fmt.Errorf("dangling/cyclic link %s: %w", logical, err) }
		if !within(anchor, real) { return fmt.Errorf("link escapes input tree: %s", logical) }
		targetRel, err := filepath.Rel(anchor, real)
		if err != nil { return err }
		return op.walkSource(root, anchor, targetRel, logical, o, ancestors, nil, "")
	}
	e := entry{name:logical, mode:info.Mode(), modified:info.ModTime()}
	snap := snapshot{root:root, removeRoot:root, rel:rel, removeRel:rel, display:filepath.Join(anchor, rel), info:info}
	if rel == "." && parentRoot != nil { snap.removeRoot, snap.removeRel = parentRoot, base }
	switch {
	case info.Mode().IsRegular():
		op.protected = append(op.protected, info)
		f, err := root.Open(rel)
		if err != nil { return err }
		opened, statErr := f.Stat()
		if statErr != nil || !sameSnapshot(info, opened) { f.Close(); return fmt.Errorf("input changed before reading %s", logical) }
		e.data, e.size, e.digest, err = op.stage(f)
		after, statErr := f.Stat()
		err = errors.Join(err, statErr, f.Close())
		if err != nil { return err }
		if !sameSnapshot(info, after) { return fmt.Errorf("input changed while reading %s", logical) }
		e.kind = 'f'
		snap.digest, snap.hasDigest = e.digest, true
	case info.IsDir():
		for _, ancestor := range ancestors { if os.SameFile(info, ancestor) { return fmt.Errorf("directory link cycle at %s", logical) } }
		f, err := root.Open(rel)
		if err != nil { return err }
		children, readErr := f.ReadDir(-1)
		err = errors.Join(readErr, f.Close())
		if err != nil { return err }
		e.kind = 'd'
		op.entries = append(op.entries, e)
		if o.deleteSource { op.sources = append(op.sources, snap) }
		ancestors = append(ancestors, info)
		for _, child := range children {
			if err := op.walkSource(root, anchor, filepath.Join(rel, child.Name()), path.Join(logical, child.Name()), o, ancestors, nil, ""); err != nil { return err }
		}
		current, err := root.Lstat(rel)
		if err != nil || !sameSnapshot(info, current) { return fmt.Errorf("directory changed during snapshot: %s", logical) }
		return nil
	case info.Mode()&os.ModeSymlink != 0:
		if !o.links { return fmt.Errorf("symbolic link %s requires -l or -L", logical) }
		e.kind = 'l'
		e.target, err = root.Readlink(rel)
		if err != nil { return err }
		// The archive graph is validated after collection. Existence here is
		// checked in the bounded source tree, not by reading target contents.
		if _, err := root.Stat(rel); err != nil { return fmt.Errorf("dangling or out-of-scope link %s: %w", logical, err) }
		e.target = filepath.ToSlash(e.target)
		snap.target = e.target
	default: return fmt.Errorf("unsupported special input file: %s", logical)
	}
	op.entries = append(op.entries, e)
	if o.deleteSource { op.sources = append(op.sources, snap) }
	return nil
}

func (s snapshot) unchanged(full bool) error {
	current, err := s.root.Lstat(s.rel)
	if err != nil { return err }
	if !os.SameFile(s.info, current) || s.info.Mode() != current.Mode() { return fmt.Errorf("source identity changed: %s", s.display) }
	removal, err := s.removeRoot.Lstat(s.removeRel)
	if err != nil || !os.SameFile(s.info, removal) { return fmt.Errorf("source removal path changed: %s", s.display) }
	if full && !sameSnapshot(s.info, current) { return fmt.Errorf("source changed: %s", s.display) }
	if s.target != "" {
		target, err := s.root.Readlink(s.rel)
		if err != nil || filepath.ToSlash(target) != s.target { return fmt.Errorf("source link changed: %s", s.display) }
	}
	if s.hasDigest {
		f, err := s.root.Open(s.rel)
		if err != nil { return err }
		h := sha256.New()
		_, readErr := io.Copy(h, io.LimitReader(f, maxBytes+1))
		err = errors.Join(readErr, f.Close())
		if err != nil { return err }
		if string(h.Sum(nil)) != string(s.digest[:]) { return fmt.Errorf("source contents changed: %s", s.display) }
	}
	return nil
}

func (op *operation) deleteSources() error {
	// Validate the entire snapshot before deleting any source. Removing only
	// known entries prevents recursive cleanup from deleting newly added files.
	for _, s := range op.sources {
		if err := s.unchanged(true); err != nil {
			var retained []string
			for _, original := range op.sources { retained = append(retained, original.display) }
			return fmt.Errorf("result saved; no deletion started: %w\nsources retained:\n%s", err, strings.Join(retained, "\n"))
		}
	}
	var failures []string
	for i := len(op.sources)-1; i >= 0; i-- {
		s := op.sources[i]
		err := s.unchanged(!s.info.IsDir())
		if err == nil { err = s.removeRoot.Remove(s.removeRel) }
		if err != nil { failures = append(failures, s.display+": "+err.Error()) }
	}
	if len(failures) > 0 { return fmt.Errorf("result saved; sources not deleted:\n%s", strings.Join(failures, "\n")) }
	return nil
}
