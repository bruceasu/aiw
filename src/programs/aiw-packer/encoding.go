package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

type nameCodec struct { legacy encoding.Encoding }

func codecFor(name string) (nameCodec, error) {
	var e encoding.Encoding
	switch strings.ToLower(name) {
	case "utf-8", "utf8":
	case "gb18030": e = simplifiedchinese.GB18030
	case "gbk": e = simplifiedchinese.GBK
	case "cp437", "ibm437": e = charmap.CodePage437
	case "shift_jis", "shift-jis", "sjis": e = japanese.ShiftJIS
	case "big5", "big-5": e = traditionalchinese.Big5
	case "big5-hkscs", "big5-hkcsc": return nameCodec{}, fmt.Errorf("full Big5-HKSCS is not supported; use -e big5 explicitly")
	default: return nameCodec{}, fmt.Errorf("unsupported filename encoding %q", name)
	}
	return nameCodec{e}, nil
}

func (c nameCodec) encode(s string) (string, error) {
	if !utf8.ValidString(s) { return "", fmt.Errorf("filename is not valid UTF-8") }
	if c.legacy == nil { return s, nil }
	raw, err := c.legacy.NewEncoder().String(s)
	if err != nil { return "", fmt.Errorf("filename cannot be represented in selected encoding: %w", err) }
	decoded, err := c.legacy.NewDecoder().String(raw)
	if err != nil || decoded != s { return "", fmt.Errorf("filename cannot be represented losslessly in selected encoding: %q", s) }
	return raw, nil
}

func (c nameCodec) decode(raw string, unicodeName bool) (string, error) {
	if c.legacy == nil || unicodeName {
		if !utf8.ValidString(raw) { return "", fmt.Errorf("invalid UTF-8 filename; specify its legacy -e encoding") }
		return raw, nil
	}
	decoded, err := c.legacy.NewDecoder().String(raw)
	if err != nil { return "", fmt.Errorf("invalid legacy filename: %w", err) }
	// x/text decoders can replace malformed bytes without returning an error.
	// Re-encoding makes such loss visible instead of publishing a damaged name.
	roundtrip, err := c.legacy.NewEncoder().String(decoded)
	if err != nil || roundtrip != raw { return "", fmt.Errorf("filename decoding is not lossless under selected encoding") }
	return decoded, nil
}
