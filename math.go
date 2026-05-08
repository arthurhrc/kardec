package kardec

// Math is the block carrying a LaTeX math expression. Display-style
// math centers the formula on its own line at the surrounding font
// size; inline math (added through MathInline once that lands) flows
// at the surrounding x-height.
//
// v0.3 ships display math only. Inline-flow math (mixing math runs
// inside a paragraph) is queued for v0.4 alongside Markdown's `$...$`
// parsing.
type Math struct {
	source  string
	display bool
}

// blockKind implements Block.
func (Math) blockKind() blockKind { return kindMath }

// Source returns the raw LaTeX text the parser will consume.
// Read-only access for layout / rendering integrations.
func (m Math) Source() string { return m.source }

// Display reports whether this Math block should be typeset in
// display style (large operators, limits above/below) versus
// inline style.
func (m Math) Display() bool { return m.display }

// Math appends a display-style math block parsed from src. The source
// is the LaTeX subset documented in internal/math: greek letters,
// fractions, square roots, sub/superscripts, sums and integrals.
//
// Parsing errors do not fail this call — they propagate through the
// document's deferred-error chain when Render is invoked, mirroring
// AppendMarkdown.
func (d *Document) Math(src string) *Document {
	return d.append(Math{source: src, display: true})
}

// MathInline appends a single-paragraph math block typeset in inline
// style (small operators, limits as side-scripts). Until v0.4 wires
// genuine inline math into Run, this is the way to add math without
// the display-mode visual emphasis.
func (d *Document) MathInline(src string) *Document {
	return d.append(Math{source: src, display: false})
}
