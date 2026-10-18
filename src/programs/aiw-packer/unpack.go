package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (op *operation) cacheInput(input string, stdin io.Reader, deleting bool) (name string, result error) {
	var r io.Reader = stdin
	var source *os.File
	var snap snapshot
	if input != "-" {
		abs, err := filepath.Abs(input)
		if err != nil { return "", err }
		parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
		if err != nil { return "", err }
		root, err := op.root(parent)
		if err != nil { return "", err }
		rel := filepath.Base(abs)
		info, err := root.Lstat(rel)
		if err != nil { return "", err }
		if !info.Mode().IsRegular() { return "", fmt.Errorf("compressed input must be an ordinary file: %s", input) }
		op.protected = append(op.protected, info)
		source, err = root.Open(rel)
		if err != nil { return "", err }
		defer func() { result = errors.Join(result, source.Close()) }()
		opened, err := source.Stat()
		if err != nil || !sameSnapshot(info, opened) { return "", fmt.Errorf("compressed input changed before reading") }
		snap = snapshot{root:root, removeRoot:root, rel:rel, removeRel:rel, display:abs, info:info, hasDigest:true}
		r = source
	}
	f, err := os.CreateTemp(op.dir, "input-")
	if err != nil { return "", err }
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h), io.LimitReader(r, maxBytes+1))
	err = errors.Join(copyErr, f.Close())
	if err != nil { return "", err }
	if n > maxBytes { return "", fmt.Errorf("compressed input exceeds 4 GiB limit") }
	if source != nil {
		after, err := source.Stat()
		if err != nil || !sameSnapshot(snap.info, after) { return "", fmt.Errorf("compressed input changed while reading") }
		if deleting {
			copy(snap.digest[:], h.Sum(nil))
			op.sources = append(op.sources, snap)
		}
	}
	return f.Name(), nil
}

func looksTar(header []byte) bool {
	if len(header) < 512 { return false }
	if bytes.Equal(header[:512], make([]byte, 512)) { return true }
	check, err := strconv.ParseInt(strings.Trim(string(header[148:156]), " \x00"), 8, 64)
	if err != nil { return false }
	var unsigned, signed int64
	for i, b := range header[:512] {
		if i >= 148 && i < 156 { b = ' ' }
		unsigned += int64(b)
		signed += int64(int8(b))
	}
	return check == unsigned || check == signed
}

func (op *operation) readZip(filename string, codec nameCodec) (result error) {
	if err := checkZipDirectory(filename); err != nil { return err }
	z, err := zip.OpenReader(filename)
	if err != nil { return err }
	defer func() { result = errors.Join(result, z.Close()) }()
	if len(z.File) > maxEntries { return fmt.Errorf("archive exceeds %d entries", maxEntries) }
	for _, file := range z.File {
		if file.Mode().IsDir() && (file.Name == "." || file.Name == "./") {
			if file.UncompressedSize64 != 0 { return fmt.Errorf("root directory has unexpected data") }
			r, err := file.Open()
			if err != nil { return err }
			n, readErr := io.Copy(io.Discard, io.LimitReader(r, 1))
			err = errors.Join(readErr, r.Close())
			if err != nil { return err }
			if n != 0 { return fmt.Errorf("root directory has unexpected content") }
			continue
		}
		name, err := codec.decode(file.Name, file.Flags&0x800 != 0)
		if err != nil { return err }
		name, err = safeName(name)
		if err != nil { return err }
		mode := file.Mode()
		e := entry{name:name, mode:mode, modified:file.Modified}
		r, err := file.Open()
		if err != nil { return err }
		switch mode.Type() {
		case 0:
			e.kind = 'f'
			e.data, e.size, e.digest, err = op.stage(r)
		case os.ModeDir:
			e.kind = 'd'
			var n int64
			n, err = io.Copy(io.Discard, io.LimitReader(r, 1))
			if err == nil && n != 0 { err = fmt.Errorf("directory has content: %s", name) }
		case os.ModeSymlink:
			e.kind = 'l'
			var target []byte
			target, err = io.ReadAll(io.LimitReader(r, maxPathBytes+1))
			if err == nil && len(target) > maxPathBytes { err = fmt.Errorf("link target is too long") }
			if err == nil { e.target, err = codec.decode(string(target), file.Flags&0x800 != 0) }
		default: err = fmt.Errorf("unsupported ZIP special entry %s", name)
		}
		err = errors.Join(err, r.Close())
		if err != nil { return fmt.Errorf("read %s: %w", name, err) }
		op.entries = append(op.entries, e)
	}
	return nil
}

func (op *operation) readTar(r io.Reader, codec nameCodec) error {
	t := tar.NewReader(r)
	count := 0
	for {
		h, err := t.Next()
		if err == io.EOF { break }
		if err != nil { return err }
		count++
		if count > maxEntries { return fmt.Errorf("archive exceeds %d entries", maxEntries) }
		if h.Typeflag == tar.TypeDir && (h.Name == "." || h.Name == "./") {
			if h.Size != 0 { return fmt.Errorf("root directory has unexpected data") }
			continue
		}
		name, err := codec.decode(h.Name, h.PAXRecords["path"] != "")
		if err != nil { return err }
		name, err = safeName(name)
		if err != nil { return err }
		e := entry{name:name, mode:os.FileMode(h.Mode)&0777, modified:h.ModTime}
		switch h.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			e.kind = 'f'
			e.data, e.size, e.digest, err = op.stage(t)
		case tar.TypeDir:
			e.kind = 'd'
			if h.Size != 0 { err = fmt.Errorf("directory has content: %s", name) }
		case tar.TypeSymlink:
			e.kind = 'l'
			if h.Size != 0 { err = fmt.Errorf("link header has unexpected content") } else { e.target, err = codec.decode(h.Linkname, h.PAXRecords["linkpath"] != "") }
		default: return fmt.Errorf("unsupported TAR link/special entry %s", name)
		}
		if err != nil { return err }
		op.entries = append(op.entries, e)
	}
	// Consume all padding and the compression trailer. Otherwise a successful
	// TAR EOF could hide a bad GZIP/BZIP2 checksum until after stdout is written.
	buffer := make([]byte, 32768)
	for {
		n, err := r.Read(buffer)
		for _, b := range buffer[:n] { if b != 0 { return fmt.Errorf("nonzero data after TAR end marker") } }
		if err == io.EOF { return nil }
		if err != nil { return err }
	}
}

func (op *operation) unpack(o options, stdin io.Reader, codec nameCodec) (result error) {
	filename, err := op.cacheInput(o.inputs[0], stdin, o.deleteSource)
	if err != nil { return err }
	f, err := os.Open(filename)
	if err != nil { return err }
	defer func() { result = errors.Join(result, f.Close()) }()
	buffered := bufio.NewReader(f)
	header, _ := buffered.Peek(512)
	format := o.format
	if format == "auto" {
		switch {
		case bytes.HasPrefix(header, []byte("PK\x03\x04")), bytes.HasPrefix(header, []byte("PK\x05\x06")): format = "zip"
		case bytes.HasPrefix(header, []byte{0x1f, 0x8b}): format = "gz"
		case bytes.HasPrefix(header, []byte("BZh")): format = "bz2"
		case looksTar(header): format = "tar"
		default: return fmt.Errorf("cannot detect archive format; use -f for an explicit format")
		}
	}
	if format == "zip" { return op.readZip(filename, codec) }
	var r io.Reader = buffered
	var gz *gzip.Reader
	if format == "gz" || format == "tar.gz" {
		gz, err = gzip.NewReader(buffered)
		if err != nil { return err }
		defer func() { result = errors.Join(result, gz.Close()) }()
		r = gz
	} else if format == "bz2" || format == "tar.bz2" { r = bzip2.NewReader(buffered) }
	limit := &io.LimitedReader{R:r, N:maxBytes+1}
	content := bufio.NewReader(limit)
	if o.format == "auto" && (format == "gz" || format == "bz2") {
		prefix, _ := content.Peek(512)
		if looksTar(prefix) { format = "tar."+format }
	}
	if format == "tar" || format == "tar.gz" || format == "tar.bz2" {
		err = op.readTar(content, codec)
	} else {
		name, nameErr := safeName(streamName(o.inputs[0]))
		if nameErr != nil { return nameErr }
		e := entry{name:name, kind:'f', mode:0644}
		e.data, e.size, e.digest, err = op.stage(content)
		if err == nil { op.entries = append(op.entries, e) }
	}
	if err != nil { return err }
	if limit.N <= 0 { return fmt.Errorf("expanded stream exceeds 4 GiB limit") }
	return nil
}
