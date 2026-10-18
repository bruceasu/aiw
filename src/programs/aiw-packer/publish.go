package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Open each directory separately and pin its identity before descending.
// Existing symbolic-link components are never accepted as output parents.
func descend(parent *os.Root, name string, create bool) (*os.Root, error) {
	info, err := parent.Lstat(name)
	if errors.Is(err, os.ErrNotExist) && create {
		err = parent.Mkdir(name, 0755)
		if err != nil && !errors.Is(err, os.ErrExist) { return nil, err }
		info, err = parent.Lstat(name)
	}
	if err != nil { return nil, err }
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 { return nil, fmt.Errorf("output parent is not an ordinary directory: %s", name) }
	next, err := parent.OpenRoot(name)
	if err != nil { return nil, err }
	opened, err := next.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		next.Close()
		return nil, fmt.Errorf("output directory changed: %s", name)
	}
	return next, nil
}

func outputRoot(directory string) (*os.Root, error) {
	abs, err := filepath.Abs(directory)
	if err != nil { return nil, err }
	volume := filepath.VolumeName(abs)
	anchor := volume + string(filepath.Separator)
	root, err := os.OpenRoot(anchor)
	if err != nil { return nil, err }
	rel, err := filepath.Rel(anchor, abs)
	if err != nil { root.Close(); return nil, err }
	if rel == "." { return root, nil }
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		next, openErr := descend(root, component, true)
		closeErr := root.Close()
		if openErr != nil { return nil, errors.Join(openErr, closeErr) }
		if closeErr != nil { next.Close(); return nil, closeErr }
		root = next
	}
	return root, nil
}

func entryParent(root *os.Root, name string, create bool) (*os.Root, string, error) {
	parts := strings.Split(name, "/")
	current, err := root.OpenRoot(".")
	if err != nil { return nil, "", err }
	for _, part := range parts[:len(parts)-1] {
		next, openErr := descend(current, part, create)
		closeErr := current.Close()
		if errors.Is(openErr, os.ErrNotExist) && !create { return nil, parts[len(parts)-1], closeErr }
		if openErr != nil { return nil, "", errors.Join(openErr, closeErr) }
		if closeErr != nil { next.Close(); return nil, "", closeErr }
		current = next
	}
	return current, parts[len(parts)-1], nil
}

func (op *operation) checkTarget(parent *os.Root, base string, e entry, overwrite bool) error {
	info, err := parent.Lstat(base)
	if errors.Is(err, os.ErrNotExist) { return nil }
	if err != nil { return err }
	if info.Mode()&os.ModeSymlink != 0 { return fmt.Errorf("output target is a symbolic link: %s", e.name) }
	for _, source := range op.protected {
		if os.SameFile(source, info) { return fmt.Errorf("output aliases an input file: %s", e.name) }
	}
	if e.kind == 'd' && info.IsDir() { return nil }
	if e.kind == 'f' && info.Mode().IsRegular() && overwrite { return nil }
	return fmt.Errorf("output conflict at %s (only ordinary files can be overwritten with -O)", e.name)
}

func (op *operation) preflight(root *os.Root, entries []entry, overwrite bool) error {
	for _, e := range entries {
		parent, base, err := entryParent(root, e.name, false)
		if err != nil { return err }
		if parent == nil { continue }
		err = errors.Join(op.checkTarget(parent, base, e, overwrite), parent.Close())
		if err != nil { return err }
	}
	return nil
}

func (op *operation) publishFile(root *os.Root, e entry, overwrite bool) (result error) {
	parent, base, err := entryParent(root, e.name, true)
	if err != nil { return err }
	defer func() { result = errors.Join(result, parent.Close()) }()
	if err := op.checkTarget(parent, base, e, overwrite); err != nil { return err }
	name := base
	if overwrite {
		var token [16]byte
		if _, err := rand.Read(token[:]); err != nil { return err }
		name = fmt.Sprintf(".aiw-packer-%x", token[:])
	}
	f, err := parent.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, e.mode.Perm())
	if err != nil { return err }
	created, err := f.Stat()
	if err != nil { f.Close(); return err }
	owned := true
	defer func() {
		if owned {
			current, err := parent.Lstat(name)
			if err == nil && os.SameFile(created, current) { result = errors.Join(result, parent.Remove(name)) }
		}
	}()
	err = errors.Join(copyData(f, e), f.Close())
	if err != nil { return err }
	if overwrite {
		if err := op.checkTarget(parent, base, e, true); err != nil { return err }
		if err := parent.Rename(name, base); err != nil { return err }
	}
	owned = false
	return nil
}

func (op *operation) publish(o options, stdout io.Writer) (result error) {
	byName, links, err := validateEntries(op.entries, o.links)
	if err != nil { return err }
	if o.stdout {
		var only *entry
		for i := range op.entries {
			e := &op.entries[i]
			if e.kind == 'd' { continue }
			if e.kind != 'f' || only != nil { return fmt.Errorf("stdout unpack requires exactly one ordinary file and no links/special entries") }
			only = e
		}
		if only == nil { return fmt.Errorf("stdout unpack requires one ordinary file") }
		return copyData(stdout, *only)
	}
	root, err := outputRoot(o.output)
	if err != nil { return err }
	defer func() { result = errors.Join(result, root.Close()) }()
	all := make([]entry, 0, len(byName))
	for _, e := range byName { all = append(all, e) }
	if err := op.preflight(root, all, o.overwrite); err != nil { return err }
	sort.Slice(all, func(i, j int) bool { return all[i].name < all[j].name })
	for _, e := range all {
		if e.kind != 'd' { continue }
		parent, base, err := entryParent(root, e.name, true)
		if err != nil { return err }
		child, openErr := descend(parent, base, true)
		if child != nil { openErr = errors.Join(openErr, child.Close()) }
		err = errors.Join(openErr, parent.Close())
		if err != nil { return err }
	}
	for _, e := range all { if e.kind == 'f' { if err := op.publishFile(root, e, o.overwrite); err != nil { return err } } }
	for _, e := range links {
		parent, base, err := entryParent(root, e.name, false)
		if err != nil { return err }
		if parent == nil { return fmt.Errorf("missing link parent %s", e.name) }
		err = errors.Join(op.checkTarget(parent, base, e, false), parent.Close())
		if err != nil { return err }
		// Use the full destination root: on Windows a sibling directory target
		// must be visible to Root.Symlink when it chooses the link type.
		if err := root.Symlink(filepath.FromSlash(e.target), filepath.FromSlash(e.name)); err != nil { return fmt.Errorf("restore link %s: %w", e.name, err) }
	}
	return nil
}

func execute(o options, stdin io.Reader, stdout io.Writer) (result error) {
	codec, err := codecFor(o.encoding)
	if err != nil { return err }
	dir, err := os.MkdirTemp("", "aiw-packer-")
	if err != nil { return err }
	op := &operation{dir:dir}
	defer func() { result = errors.Join(result, op.close()) }()
	if o.action == "pack" {
		packed, err := op.pack(o, stdin, codec)
		if err != nil { return err }
		if o.stdout {
			err = copyData(stdout, packed)
		} else {
			packed.name = filepath.Base(o.output)
			if _, err := safeName(packed.name); err != nil { return err }
			root, openErr := outputRoot(filepath.Dir(o.output))
			if openErr != nil { return openErr }
			err = errors.Join(op.publishFile(root, packed, o.overwrite), root.Close())
		}
		if err != nil { return err }
	} else {
		if err := op.unpack(o, stdin, codec); err != nil { return err }
		if err := op.publish(o, stdout); err != nil { return err }
	}
	if o.deleteSource { return op.deleteSources() }
	return nil
}
