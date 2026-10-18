package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

type options struct {
	action, format, encoding, output string
	inputs []string
	help, overwrite, stdout, deleteSource, links, follow bool
}

var extensions = map[string]string{
	"zip": ".zip", "tar": ".tar", "tar.gz": ".tar.gz",
	"tar.bz2": ".tar.bz2", "gz": ".gz", "bz2": ".bz2",
}

func parseOptions(args []string) (options, error) {
	o := options{action: "pack", encoding: "utf-8"}
	var positional, named []string
	end := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !end && a == "--" { end = true; continue }
		if end || a == "-" || !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
			continue
		}
		key, value, hasValue := strings.Cut(a, "=")
		switch key {
		case "-i", "--input", "-o", "--output", "-f", "--format", "-e", "--encoding":
			if !hasValue {
				i++
				if i == len(args) { return o, fmt.Errorf("%s needs a value", key) }
				value = args[i]
			}
			if value == "" { return o, fmt.Errorf("%s cannot be empty", key) }
			switch key {
			case "-i", "--input": named = append(named, value)
			case "-o", "--output": o.output = value
			case "-f", "--format": o.format = strings.ToLower(value)
			case "-e", "--encoding": o.encoding = strings.ToLower(value)
			}
		case "-h", "--help", "-O", "--overwrite", "-c", "-d", "--delete", "-l", "-L":
			if hasValue { return o, fmt.Errorf("%s does not accept a value", key) }
			switch key {
			case "-h", "--help": o.help = true
			case "-O", "--overwrite": o.overwrite = true
			case "-c": o.stdout = true
			case "-d", "--delete": o.deleteSource = true
			case "-l": o.links = true
			case "-L": o.follow = true
			}
		default: return o, fmt.Errorf("unknown option %s (use --help)", key)
		}
	}
	if o.help { return o, nil }
	if len(positional) > 0 {
		o.action = positional[0]
		if o.action != "pack" && o.action != "unpack" { return o, fmt.Errorf("first positional argument must be pack or unpack") }
		positional = positional[1:]
	}
	if len(named) > 0 { o.inputs = named } else { o.inputs = positional }
	if len(o.inputs) == 0 { return o, fmt.Errorf("input is required (-i or positional input)") }
	if o.action == "unpack" {
		if len(named) > 1 { return o, fmt.Errorf("unpack accepts only one -i") }
		o.inputs = o.inputs[:1]
		if o.follow { return o, fmt.Errorf("-L is supported only for pack") }
	}
	if o.links && o.follow { return o, fmt.Errorf("-l and -L are mutually exclusive") }
	if o.follow && o.deleteSource { return o, fmt.Errorf("-L and -d are mutually exclusive") }
	stdin := false
	for _, input := range o.inputs {
		if input == "-" { stdin = true }
	}
	if stdin && len(o.inputs) != 1 { return o, fmt.Errorf("stdin cannot be repeated or mixed with file inputs") }
	if stdin && o.output == "" && !o.stdout { return o, fmt.Errorf("stdin requires explicit -o or -c") }
	if o.stdout { o.output = "-" }
	o.stdout = o.output == "-"
	if o.deleteSource && (stdin || o.stdout) { return o, fmt.Errorf("-d cannot be used with stdin or stdout") }
	if o.format == "tgz" { o.format = "tar.gz" }
	if o.format == "" {
		if o.action == "pack" { o.format = "zip" } else { o.format = "auto" }
	}
	if _, ok := extensions[o.format]; !ok && !(o.action == "unpack" && o.format == "auto") {
		return o, fmt.Errorf("unsupported format %q", o.format)
	}
	if o.output == "" {
		if o.action == "pack" {
			if len(o.inputs) == 1 { o.output = filepath.Clean(o.inputs[0]) + extensions[o.format] } else { o.output = "archive" + extensions[o.format] }
		} else {
			o.output = filepath.Join(filepath.Dir(o.inputs[0]), streamName(o.inputs[0]))
		}
	}
	return o, nil
}

func streamName(input string) string {
	if input == "-" { return "stdin" }
	name := filepath.Base(input)
	lower := strings.ToLower(name)
	for _, ext := range []string{".tar.bz2", ".tar.gz", ".tgz", ".zip", ".tar", ".bz2", ".gz"} {
		if strings.HasSuffix(lower, ext) { name = name[:len(name)-len(ext)]; break }
	}
	if name == "" || name == "." || name == ".." { return "unpacked" }
	return name
}

const helpText = `Usage:
  aiw-packer [options] [pack|unpack] [inputs...]
  aiw packer [options] [pack|unpack] [inputs...]

Options:
  -h, --help             Show help.
  -f, --format FORMAT    zip, tar, tar.gz/tgz, tar.bz2, gz, bz2.
                        Default: zip for pack; content detection for unpack.
  -e, --encoding NAME    Filename encoding (default utf-8).
                        gb18030, gbk, cp437, shift_jis, big5.
                        Full Big5-HKSCS is not provided; select big5 explicitly.
  -i, --input PATH       Input; - reads stdin. Repeat for pack only.
  -o, --output PATH      Output file or unpack directory; - writes stdout.
  -O, --overwrite        Overwrite ordinary files. Never replace directories
                        with files or overwrite existing symbolic links.
  -c                    Force stdout, overriding -o. Unpack requires exactly
                        one ordinary file, optional directories, and no links.
  -d, --delete          Delete sources only after the entire operation succeeds.
                        Not with stdin, stdout or -L. Delete failures keep the
                        result, report remaining sources and return nonzero.
  -l                    Preserve/restore safe relative symbolic links.
  -L                    Pack link target content, using the link's logical name.
                        Mutually exclusive with -l and -d; pack only.

The first positional argument is ACTION; without positionals ACTION is pack.
Named inputs override positional inputs; they are never merged. Unpack uses
only the first positional input and ignores the rest. Options may precede or
follow ACTION. -- ends option parsing. The misspellings --ouput/--overwirte
are not aliases.

Without -o, pack appends the format extension to a single input, or writes
archive.<extension> for multiple inputs in the current directory. Unpack
removes the complete format suffix to name its output directory. Stdin needs
explicit -o or -c; its archive entry is named stdin. Directories retain their
top-level input basename. Colliding names and overlapping inputs are rejected.

GZ/BZ2 are single-file byte streams: directories and multiple inputs need
TAR.GZ/TAR.BZ2. GZIP FNAME is not written or used for output paths. Stream
output filenames come from the input basename (stdin for stdin); -e does not
transcode stream contents or these generated names. -e never changes contents.
ZIP UTF-8 flags and TAR PAX Unicode paths take precedence over legacy -e.

Links are rejected by default; dangling, cyclic or out-of-scope links fail.
-l requires targets included in the archive; Windows link creation can fail
with a permission/filesystem error. No automatic privilege changes occur.
-L follows an explicit top-level link, confines nested targets to the resolved
input tree, and does not preserve hard-link identity. Hard-link archive
entries and special files are rejected. Older ZIP readers may not restore links.

Input/output identity and output inside a source directory are rejected.
Unpack validates all content and checksums before stdout or target publication.
Staging limits: 4 GiB input, 4 GiB expanded data, 100000 entries, 4096 bytes/path,
64 MiB ZIP directory metadata. Multi-disk/self-extracting ZIP is not supported.
Existing directories merge; default file conflicts fail. -O permits ordinary
file replacement only. Target publication and source deletion are not multi-
file transactions: a failure can leave completed outputs or deletions.

Examples:
  aiw-packer -i report.txt
  aiw-packer pack -i a.txt -i b.txt -o bundle.zip
  aiw-packer unpack -e gb18030 -i old.zip -o restored
  aiw-packer pack -f tar.gz -i folder -o folder.tar.gz
  aiw-packer pack -f gz -i - -c
  aiw-packer unpack -i one.zip -c
`
