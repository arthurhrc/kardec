package kardec

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// AppendMarkdown parses src as Markdown (CommonMark plus the GFM table
// extension) and appends the resulting blocks to the current section.
// Headings, paragraphs, emphasis, strong emphasis, inline code, code
// blocks, lists, blockquotes and tables are translated; horizontal
// rules become page breaks.
//
// Current scope:
//
//   - Lists are flattened to plain paragraphs prefixed with "• " or "1. "
//     until a real List block lands.
//   - Images and link URLs are rendered as inline text only — image
//     embedding ships with the dedicated Image block.
//   - Code blocks become paragraphs styled through StyleCode.
//   - GFM tables become real Table blocks with column alignment honoured;
//     the first row repeats on continuation pages.
//
// Errors during parsing are captured in the document's deferred-error
// chain and surfaced by Err / Render.
func (d *Document) AppendMarkdown(src string) *Document {
	if d.err != nil {
		return d
	}
	source := []byte(src)
	md := goldmark.New(goldmark.WithExtensions(extension.Table))
	root := md.Parser().Parse(text.NewReader(source))
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
		d.appendMarkdownList(n, source)
	case *ast.Blockquote:
		d.AddParagraph(runsFromInline(n, source)...).
			WithNamedStyle(StyleQuote).
			Done()
	case *extast.Table:
		d.appendMarkdownTable(n, source)
	}
}

// appendMarkdownTable converts a GFM table AST into a kardec.Table block.
// The header row drives the column descriptors (header text plus column
// alignment), and every data row becomes a Row of Cell values whose Runs
// preserve inline emphasis from the source.
//
// RepeatHeader is set so multi-page tables keep their column titles
// visible after every page break.
func (d *Document) appendMarkdownTable(t *extast.Table, source []byte) {
	tb := d.Table().RepeatHeader()

	var (
		headerCells []string
		alignments  []Alignment
	)
	for n := t.FirstChild(); n != nil; n = n.NextSibling() {
		header, ok := n.(*extast.TableHeader)
		if !ok {
			continue
		}
		for cell := header.FirstChild(); cell != nil; cell = cell.NextSibling() {
			tc, ok := cell.(*extast.TableCell)
			if !ok {
				continue
			}
			headerCells = append(headerCells, string(extractInlineText(tc, source)))
			alignments = append(alignments, alignmentFromGFM(tc.Alignment))
		}
		break
	}
	if len(headerCells) == 0 {
		// No header → no columns; skip silently.
		return
	}
	cols := make([]Column, len(headerCells))
	for i, h := range headerCells {
		cols[i] = Col(h)
		cols[i].Alignment = alignments[i]
	}
	tb.Columns(cols...).RowCells(headerRowCells(headerCells)...)

	for n := t.FirstChild(); n != nil; n = n.NextSibling() {
		row, ok := n.(*extast.TableRow)
		if !ok {
			continue
		}
		var cells []Cell
		for c := row.FirstChild(); c != nil; c = c.NextSibling() {
			tc, ok := c.(*extast.TableCell)
			if !ok {
				continue
			}
			cells = append(cells, Cell{Runs: runsFromInline(tc, source)})
		}
		// Pad missing trailing cells so layout sees a uniform shape.
		for len(cells) < len(cols) {
			cells = append(cells, Cell{})
		}
		tb.RowCells(cells...)
	}
	tb.Build()
}

// headerRowCells turns the plain header strings into a slice of bold
// Cells so the rendered table visually distinguishes the header line.
func headerRowCells(headers []string) []Cell {
	out := make([]Cell, len(headers))
	for i, h := range headers {
		out[i] = Cell{Runs: []Run{Bold(h)}}
	}
	return out
}

// alignmentFromGFM maps the goldmark GFM alignment enum to kardec's
// Alignment. Unspecified columns inherit AlignLeft.
func alignmentFromGFM(a extast.Alignment) Alignment {
	switch a {
	case extast.AlignCenter:
		return AlignCenter
	case extast.AlignRight:
		return AlignRight
	default:
		return AlignLeft
	}
}

// extractInlineText concatenates every inline Text leaf under node into
// a single string — used for header cells where the rich Run structure
// would be discarded by Bold-wrapping anyway.
func extractInlineText(node ast.Node, source []byte) []byte {
	var buf []byte
	var walk func(ast.Node)
	walk = func(n ast.Node) {
		if t, ok := n.(*ast.Text); ok {
			buf = append(buf, t.Segment.Value(source)...)
			return
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(node)
	return buf
}

// appendMarkdownList walks an ordered or unordered list and emits a
// real List block, with nested lists carried through ListItem.Children
// so the layout engine can indent properly. The legacy "flatten to
// bulleted paragraphs" behaviour ended once the List block landed.
func (d *Document) appendMarkdownList(list *ast.List, source []byte) {
	d.append(buildMarkdownList(list, source))
}

// buildMarkdownList recursively converts a goldmark List node into a
// kardec.List. Plain string-prefixed paragraphs are no longer emitted;
// the layout engine owns marker rendering.
func buildMarkdownList(list *ast.List, source []byte) List {
	style := ListUnordered
	if list.IsOrdered() {
		style = ListOrdered
	}
	out := List{style: style}
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		item, ok := child.(*ast.ListItem)
		if !ok {
			continue
		}
		entry := ListItem{Runs: runsFromInlineExcludingNested(item, source)}
		for sub := item.FirstChild(); sub != nil; sub = sub.NextSibling() {
			if nested, ok := sub.(*ast.List); ok {
				entry.Children = append(entry.Children, buildMarkdownList(nested, source))
			}
		}
		out.items = append(out.items, entry)
	}
	return out
}

// runsFromInlineExcludingNested gathers the inline runs of a list item
// while skipping any nested list children — those become Children of
// the produced ListItem and would otherwise be flattened twice.
func runsFromInlineExcludingNested(item ast.Node, source []byte) []Run {
	var out []Run
	for c := item.FirstChild(); c != nil; c = c.NextSibling() {
		if _, isList := c.(*ast.List); isList {
			continue
		}
		walkInline(c, source, false, false, &out)
	}
	return out
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
// constructs that don't preserve formatting (code blocks, code spans).
//
// goldmark exposes block-level lines via Node.Lines() but panics if
// that method is called on inline nodes. This function dispatches on
// the concrete type to avoid the trap: code blocks read from Lines,
// inline subtrees recurse to their *ast.Text leaves.
func extractText(node ast.Node, source []byte) string {
	var buf []byte
	switch n := node.(type) {
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		_ = n
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			buf = append(buf, seg.Value(source)...)
		}
	default:
		var walk func(ast.Node)
		walk = func(x ast.Node) {
			if t, ok := x.(*ast.Text); ok {
				buf = append(buf, t.Segment.Value(source)...)
				return
			}
			for c := x.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
		}
		walk(node)
	}
	return string(buf)
}

