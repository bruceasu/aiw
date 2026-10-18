package markdown

import (
    "bufio"
    "bytes"
    "html"
    "strings"
    "github.com/yuin/goldmark"
    "github.com/yuin/goldmark/ast"
    "github.com/yuin/goldmark/extension"
    goldmarkhtml "github.com/yuin/goldmark/renderer/html"
    "github.com/yuin/goldmark/text"
)

// Plain uses Markdown syntax nodes, preserving punctuation in the actual text.
func Plain(source []byte) string {
    md := goldmark.New(goldmark.WithExtensions(extension.GFM))
    doc := md.Parser().Parse(text.NewReader(source))
    var out strings.Builder
    writeText := func(value []byte, literal bool) {
        if literal { out.Write(value); return }
        // Reuse Goldmark's single-pass Markdown escaping/entity rules. Decode
        // its escaped HTML text once, so escaped ampersands and &amp;#65; do
        // not get incorrectly expanded a second time.
        var encoded bytes.Buffer
        writer := bufio.NewWriter(&encoded)
        goldmarkhtml.DefaultWriter.Write(writer, value)
        _ = writer.Flush()
        out.WriteString(html.UnescapeString(encoded.String()))
    }
    _ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
        if !entering {
            switch n.Kind() {
            case ast.KindParagraph, ast.KindHeading, ast.KindListItem, ast.KindBlockquote:
                out.WriteString("\n")
            }
            if n.Kind().String() == "TableCell" { out.WriteString("\t") }
            if n.Kind().String() == "TableRow" || n.Kind().String() == "TableHeader" { out.WriteString("\n") }
            return ast.WalkContinue, nil
        }
        switch node := n.(type) {
        case *ast.Text:
            code := node.Parent() != nil && node.Parent().Kind() == ast.KindCodeSpan
            writeText(node.Segment.Value(source), node.IsRaw() || code)
            if node.SoftLineBreak() || node.HardLineBreak() {
                if code { out.WriteString(" ") } else { out.WriteString("\n") }
            }
        case *ast.String:
            writeText(node.Value, node.IsCode() || node.IsRaw())
        case *ast.AutoLink:
            out.Write(node.Label(source))
        case *ast.FencedCodeBlock:
            for i := 0; i < node.Lines().Len(); i++ { line := node.Lines().At(i); out.Write(line.Value(source)) }
            out.WriteString("\n")
            return ast.WalkSkipChildren, nil
        case *ast.CodeBlock:
            for i := 0; i < node.Lines().Len(); i++ { line := node.Lines().At(i); out.Write(line.Value(source)) }
            out.WriteString("\n")
            return ast.WalkSkipChildren, nil
        }
        return ast.WalkContinue, nil
    })
    return strings.TrimSpace(out.String()) + "\n"
}
