package kardec

// Run is an inline fragment of text with optional style overrides. Runs are
// the leaf content nodes that paragraphs and headings carry. Construct them
// via Text, Bold, Italic and similar helpers; do not instantiate the struct
// directly so future fields stay backward-compatible.
type Run struct {
	text          string
	bold          bool
	italic        bool
	underline     bool
	strikethrough bool
	link          string // empty when this run is not a hyperlink
	footnoteRef   int    // 1-based footnote number; 0 for non-footnote runs
	mathSource    string // non-empty when this run is an inline math expression
	override      styleOverride
}

// Text returns a plain Run carrying no inline overrides.
func Text(s string) Run { return Run{text: s} }

// Bold returns a Run rendered in bold weight.
func Bold(s string) Run { return Run{text: s, bold: true} }

// Italic returns a Run rendered in italic style.
func Italic(s string) Run { return Run{text: s, italic: true} }

// BoldItalic returns a Run rendered in both bold weight and italic style.
func BoldItalic(s string) Run { return Run{text: s, bold: true, italic: true} }

// Underline returns a Run drawn with a thin line under its glyphs.
// Compose with Bold / Italic via the Decorate helpers when more than
// one inline attribute is needed.
func Underline(s string) Run { return Run{text: s, underline: true} }

// Strikethrough returns a Run drawn with a thin line through its
// glyphs, the typographic convention for retracted text.
func Strikethrough(s string) Run { return Run{text: s, strikethrough: true} }

// WithUnderline returns a copy of r with the underline decoration set.
// Lets callers stack decorations on a base run without re-typing the
// text.
func WithUnderline(r Run) Run { r.underline = true; return r }

// WithStrikethrough returns a copy of r with the strikethrough
// decoration set.
func WithStrikethrough(r Run) Run { r.strikethrough = true; return r }

// Colored wraps a Run, attaching an explicit color override.
func Colored(r Run, c Color) Run {
	r.override.color = &c
	return r
}

// InlineMath returns a Run carrying a LaTeX math expression that
// will render at the surrounding paragraph's font size, inline with
// the body text. The supported subset matches Document.Math:
// greek letters, fractions, square roots, sub/superscripts, sums
// and integrals (the latter two laid out in inline style — small
// operators with limits at the side rather than above/below).
//
// Use it to mix math into ordinary prose:
//
//	doc.Paragraph(
//	    Text("By Pythagoras, "),
//	    InlineMath("a^2 + b^2 = c^2"),
//	    Text(" for any right triangle."),
//	)
//
// Parsing failures degrade to a "[math: <error>]" plain-text fallback
// at render time so a typo in one expression never aborts the
// surrounding paragraph.
func InlineMath(src string) Run {
	return Run{mathSource: src}
}

// Link returns a Run that, in addition to carrying the visible text,
// becomes a clickable hyperlink in the rendered PDF. The url is
// emitted as an external `/URI` action; relative URLs are passed
// through unchanged so callers may target intra-document anchors via
// "#anchor" once the outline track exposes them.
//
// The returned Run is plain text by default; callers may further
// decorate it (e.g. wrap with Colored) before adding to a paragraph.
func Link(text, url string) Run {
	return Run{text: text, link: url}
}

// styleOverride captures the optional inline style fields a Run may carry.
// Fields are pointers so "absent" remains distinguishable from "zero value".
type styleOverride struct {
	color *Color
	size  *Length
}
