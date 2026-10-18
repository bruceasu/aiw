package markdown

import (
	"bytes"
	"fmt"
	"html"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type Options struct {
	AllowHTML bool
	ImageRoot string
}

func Render(source []byte, options Options) ([]byte, error) {
	var out bytes.Buffer
	rendererOptions := []renderer.Option{
		renderer.WithNodeRenderers(util.Prioritized(&mermaidRenderer{}, 500)),
	}
	if options.AllowHTML {
		rendererOptions = append(rendererOptions, goldmarkhtml.WithUnsafe())
	}
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(rendererOptions...),
	)
	reader := text.NewReader(source)
	document := md.Parser().Parse(reader)
	if options.ImageRoot != "" {
		if err := EmbedImages(options.ImageRoot, document, source); err != nil {
			return nil, err
		}
	}
	if err := md.Renderer().Render(&out, source, document); err != nil {
		return nil, fmt.Errorf("render markdown: %w", err)
	}
	return out.Bytes(), nil
}

type mermaidRenderer struct{}

func (r *mermaidRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.render)
}

func (r *mermaidRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.FencedCodeBlock)
	if !strings.EqualFold(strings.TrimSpace(string(n.Language(source))), "mermaid") {
		_, _ = w.WriteString("<pre><code")
		if language := n.Language(source); language != nil {
			_, _ = w.WriteString(` class="language-`)
			_, _ = w.WriteString(html.EscapeString(string(language)))
			_ = w.WriteByte('"')
		}
		_ = w.WriteByte('>')
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			_, _ = w.WriteString(html.EscapeString(string(line.Value(source))))
		}
		_, _ = w.WriteString("</code></pre>\n")
		return ast.WalkSkipChildren, nil
	}
	var b strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		line := n.Lines().At(i)
		b.Write(line.Value(source))
	}
	_, _ = w.WriteString("<pre class=\"mermaid\">")
	_, _ = w.WriteString(html.EscapeString(b.String()))
	_, _ = w.WriteString("</pre>\n")
	return ast.WalkSkipChildren, nil
}
