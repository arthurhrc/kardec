package kardec

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// AppendMarkdown parses src as CommonMark and appends the resulting blocks
// to the current section. Headings, paragraphs, emphasis, strong emphasis
// and inline code are translated; horizontal rules become page breaks.
//
// v0.1 scope (intentional):
//
//   - Lists are flattened to plain paragraphs prefixed with "• " or "1. "
//     until the List block lands in v0.2.
//   - Tables, images and link URLs are rendered as inline text only.
//   - Code blocks become paragraphs styled through StyleCode.
//
// Errors during parsing are captured in the document's deferred-error
// chain and surfaced by Err / Render.
func (d *Document) AppendMarkdown(src string) *Document {
	if d.err != nil {
		return d
	}
	source := []byte(src)
	parser := goldmark.DefaultParser()
	root := parser.Parse(text.NewReader(source))
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		d.appendMarkdownNode(child, source)
	}
	return d
}

// appendMarkdownNode dispatches a single block-level AST node onto the
// document's builder. Unknown block kinds fall through silently so an
// unsupported construct never aborts the whole import.
func (d *Document) appendMarkdownNode(node ast.Node, source []byte) {
	switch n := node.(type) {
	case *ast.Heading:
		d.Heading(n.Level, runsFromInline(n, source)...)
	case *ast.Paragraph:
		d.Paragraph(runsFromInline(n, source)...)
	case *ast.ThematicBreak:
		d.PageBreak()
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		d.AddParagraph(Text(string(extractText(node, source)))).
			WithNamedStyle(StyleCode).
			Done()
	case *ast.List:
		d.appendMarkdownList(n, source, 0)
	case *ast.Blockquote:
		d.AddParagraph(runsFromInline(n, source)...).
			WithNamedStyle(StyleQuote).
			Done()
	}
}

// appendMarkdownList walks an ordered or unordered list and emits one
// paragraph per item with a leading bullet or index marker. Nested lists
// indent visually by repeating the marker prefix; real list nesting lands
// in v0.2.
func (d *Document) appendMarkdownList(list *ast.List, source []byte, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	idx := 1
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		item, ok := child.(*ast.ListItem)
		if !ok {
			continue
		}
		marker := "• "
		if list.IsOrdered() {
			marker = formatOrderedMarker(idx)
			idx++
		}
		runs := []Run{Text(indent + marker)}
		runs = append(runs, runsFromInline(item, source)...)
		d.Paragraph(runs...)
		// Recurse into nested lists.
		for sub := item.FirstChild(); sub != nil; sub = sub.NextSibling() {
			if nested, ok := sub.(*ast.List); ok {
				d.appendMarkdownList(nested, source, depth+1)
			}
		}
	}
}

// runsFromInline flattens an inline subtree (the children of a Heading,
// Paragraph or similar) into a slice of Runs. Bold, italic and inline
// code each map to their kardec helper; nested combinations like
// bold-italic resolve to BoldItalic when both flags are seen.
func runsFromInline(parent ast.Node, source []byte) []Run {
	var out []Run
	walkInline(parent, source, false, false, &out)
	return out
}

// walkInline recursively traverses inline nodes, carrying the bold/italic
// flags down so a Run inherits formatting from its enclosing Emphasis
// node. Plain text is the only leaf that produces a Run.
func walkInline(node ast.Node, source []byte, bold, italic bool, out *[]Run) {
	switch n := node.(type) {
	case *ast.Text:
		s := string(n.Segment.Value(source))
		if s == "" {
			return
		}
		*out = append(*out, makeRun(s, bold, italic))
	case *ast.Emphasis:
		newItalic, newBold := italic, bold
		if n.Level == 1 {
			newItalic = true
		}
		if n.Level == 2 {
			newBold = true
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			walkInline(c, source, newBold, newItalic, out)
		}
	case *ast.CodeSpan:
		// Inline code: keep the text but annotate via inline style override
		// once codespan styling lands; for now plain (monospace handled at
		// font level once StyleCode flows down to runs).
		*out = append(*out, Text(string(extractText(n, source))))
	case *ast.Link:
		// Treat as plain text in v0.1; URL rendering is a v0.2 feature.
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			walkInline(c, source, bold, italic, out)
		}
	default:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			walkInline(c, source, bold, italic, out)
		}
	}
}

// makeRun constructs a Run carrying the requested bold/italic flags.
func makeRun(s string, bold, italic bool) Run {
	switch {
	case bold && italic:
		return BoldItalic(s)
	case bold:
		return Bold(s)
	case italic:
		return Italic(s)
	default:
		return Text(s)
	}
}

// extractText concatenates the textual content of a subtree, used for
// constructs that don't preserve formatting (code blocks).
func extractText(node ast.Node, source []byte) string {
	var buf []byte
	var walk func(ast.Node)
	walk = func(n ast.Node) {
		if t, ok := n.(*ast.Text); ok {
			buf = append(buf, t.Segment.Value(source)...)
			return
		}
		// Text segments held in code-block lines: iterate concrete lines.
		if lines := n.Lines(); lines != nil && lines.Len() > 0 {
			for i := 0; i < lines.Len(); i++ {
				ln := lines.At(i)
				buf = append(buf, ln.Value(source)...)
			}
			return
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(node)
	return string(buf)
}

// formatOrderedMarker returns the leading marker for the i-th item of an
// ordered list (1-based). Goldmark normalises any starting number to 1.
func formatOrderedMarker(i int) string {
	digits := []byte{}
	if i == 0 {
		digits = []byte{'0'}
	}
	for n := i; n > 0; n /= 10 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
	}
	return string(digits) + ". "
}
