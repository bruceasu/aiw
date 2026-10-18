package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"compress/gzip"
	writebzip "github.com/dsnet/compress/bzip2"
)

func copyData(w io.Writer, e entry) error {
	f, err := os.Open(e.data)
	if err != nil { return err }
	n, copyErr := io.Copy(w, f)
	err = errors.Join(copyErr, f.Close())
	if err == nil && n != e.size { err = fmt.Errorf("staged size changed for %s", e.name) }
	return err
}

func (op *operation) writeZip(w io.Writer, codec nameCodec) (result error) {
	z := zip.NewWriter(w)
	defer func() { result = errors.Join(result, z.Close()) }()
	for _, e := range op.entries {
		name, err := codec.encode(e.name)
		if err != nil { return err }
		if e.kind == 'd' { name += "/" }
		h := &zip.FileHeader{Name:name, Method:zip.Deflate, NonUTF8:codec.legacy != nil}
		h.SetMode(e.mode)
		if e.kind == 'd' { h.SetMode(os.ModeDir | e.mode.Perm()) }
		if e.kind == 'l' { h.SetMode(os.ModeSymlink | 0777) }
		h.SetModTime(e.modified)
		dst, err := z.CreateHeader(h)
		if err != nil { return err }
		switch e.kind {
		case 'f': if err := copyData(dst, e); err != nil { return err }
		case 'l':
			target, err := codec.encode(e.target)
			if err != nil { return err }
			if _, err := io.Copy(dst, bytes.NewBufferString(target)); err != nil { return err }
		}
	}
	return nil
}

func (op *operation) writeTar(w io.Writer, codec nameCodec) (result error) {
	t := tar.NewWriter(w)
	defer func() { result = errors.Join(result, t.Close()) }()
	for _, e := range op.entries {
		name, err := codec.encode(e.name)
		if err != nil { return err }
		h := &tar.Header{Name:name, Mode:int64(e.mode.Perm()), ModTime:e.modified, Typeflag:tar.TypeReg, Size:e.size, Format:tar.FormatPAX}
		if codec.legacy != nil { h.Format = tar.FormatGNU }
		switch e.kind {
		case 'd': h.Typeflag, h.Size = tar.TypeDir, 0
		case 'l':
			h.Typeflag, h.Size = tar.TypeSymlink, 0
			h.Linkname, err = codec.encode(e.target)
			if err != nil { return err }
		}
		if err := t.WriteHeader(h); err != nil { return err }
		if e.kind == 'f' { if err := copyData(t, e); err != nil { return err } }
	}
	return nil
}

func (op *operation) writePacked(w io.Writer, o options, codec nameCodec) (result error) {
	var stream io.Writer = w
	var closer io.Closer
	switch o.format {
	case "gz", "tar.gz":
		z := gzip.NewWriter(w) // Optional FNAME intentionally omitted.
		stream, closer = z, z
	case "bz2", "tar.bz2":
		z, err := writebzip.NewWriter(w, nil)
		if err != nil { return err }
		stream, closer = z, z
	}
	if closer != nil { defer func() { result = errors.Join(result, closer.Close()) }() }
	switch o.format {
	case "zip": return op.writeZip(stream, codec)
	case "tar", "tar.gz", "tar.bz2": return op.writeTar(stream, codec)
	case "gz", "bz2": return copyData(stream, op.entries[0])
	}
	return fmt.Errorf("unsupported pack format %s", o.format)
}

type cappedWriter struct { w io.Writer; n int64 }

func (w *cappedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.n { return 0, fmt.Errorf("compressed result exceeds 4 GiB limit") }
	n, err := w.w.Write(p)
	w.n -= int64(n)
	return n, err
}

func (op *operation) pack(o options, stdin io.Reader, codec nameCodec) (entry, error) {
	if err := op.collect(o, stdin, codec); err != nil { return entry{}, err }
	f, err := os.CreateTemp(op.dir, "archive-")
	if err != nil { return entry{}, err }
	err = op.writePacked(&cappedWriter{f, maxBytes}, o, codec)
	info, statErr := f.Stat()
	err = errors.Join(err, statErr, f.Close())
	if err != nil { return entry{}, err }
	return entry{data:f.Name(), size:info.Size(), kind:'f', mode:0644}, nil
}
