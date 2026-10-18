package markdown

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark/ast"
)

func EmbedImages(root string, document ast.Node, source []byte) error {
	base, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	base, err = filepath.EvalSymlinks(base)
	if err != nil {
		return err
	}
	return ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node.Kind() != ast.KindImage {
			return ast.WalkContinue, nil
		}
		image := node.(*ast.Image)
		raw := string(image.Destination)
		u, err := url.Parse(raw)
		if err != nil || u.IsAbs() || strings.HasPrefix(raw, "//") {
			return ast.WalkStop, fmt.Errorf("local image path is not allowed: %s", raw)
		}
		path := filepath.Join(base, filepath.FromSlash(u.Path))
		path, err = filepath.Abs(path)
		if err != nil {
			return ast.WalkStop, err
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return ast.WalkStop, fmt.Errorf("local image not found: %s", raw)
		}
		rel, err := filepath.Rel(base, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return ast.WalkStop, fmt.Errorf("local image path escapes input directory: %s", raw)
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			return ast.WalkStop, fmt.Errorf("unable to read local image %s: %w", raw, err)
		}
		mime := mimeType(filepath.Ext(resolved))
		if mime == "" {
			return ast.WalkStop, fmt.Errorf("unsupported local image type: %s", raw)
		}
		image.Destination = []byte("data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data))
		return ast.WalkSkipChildren, nil
	})
}

func mimeType(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	}
	return ""
}
