package kardec

import (
	"bytes"
	"errors"
	"io"
	"os"
	"time"

	"github.com/arthurhrc/kardec/internal/typography"
)

// Document is the central builder a caller composes content onto. It holds
// one or more sections (a section owns a PageSetup, an optional header /
// footer, and a list of blocks).
//
// A *Document is not safe for concurrent use; see the package overview.
type Document struct {
	sections []*Section
	cur      *Section // pointer into sections; the section currently receiving blocks

	styles map[string]Style     // named style table; pre-populated from BuiltinStyles
	fonts  *typography.Registry // registry of registered + bundled font faces

	// mathFont memoises the lazily loaded Latin Modern Math face served
	// by Document.MathFont, so each AST atom resolved by the layout
	// engine reuses a single parsed *canvas.Font.
	mathFont typography.MathFont

	// creationDate fixes the timestamp written to the rendered PDF's
	// /Info /CreationDate. nil means "use time.Now() at render time".
	// Callers that want byte-reproducible output pin this via
	// SetCreationDate.
	creationDate *time.Time

	// warnings accumulates non-fatal advisories — Markdown nodes that
	// were dropped silently, link URLs that arrived empty, etc.
	// Surfaced via Document.Warnings; callers that opt into noise
	// log them or fail their CI on non-empty output.
	warnings []string

	// footnotes carries every FootnoteRef the document has registered
	// via the Footnote helper, in declaration order. Layout looks up
	// the matching body when a Run with a non-zero FootnoteRef is
	// emitted on a page.
	footnotes       []FootnoteRef
	footnoteCounter int

	err error // first error encountered during builder usage; surfaced by Err and Render
}

// Footnotes returns every footnote registered on the document in
// declaration order. The slice is the document's own backing store;
// callers must not mutate it.
func (d *Document) Footnotes() []FootnoteRef { return d.footnotes }

// Section couples a PageSetup with the ordered list of blocks that flow
// inside it. Most documents use a single Section; multi-section support
// (different page setups within one document) lands later in v0.x.
//
// Header and Footer are inline runs reprinted at the top and bottom of
// every page in the section. They support the token substitutions
// documented on Document.Header / Document.Footer.
type Section struct {
	Setup  PageSetup
	Blocks []Block
	Header []Run
	Footer []Run
}

// New creates an empty Document with a single Section configured from the
// supplied page size and margins. Default orientation is Portrait.
func New(size PageSize, margins Margins) *Document {
	first := &Section{
		Setup: PageSetup{
			Size:        size,
			Orientation: Portrait,
			Margins:     margins,
		},
	}
	d := &Document{
		sections: []*Section{first},
		cur:      first,
		styles:   BuiltinStyles(),
		fonts:    typography.NewRegistry(),
	}
	// Best-effort: register the bundled OFL families so MeasureText works
	// out of the box. A failure here is captured in the deferred error
	// chain so Render surfaces it instead of panicking.
	if err := typography.LoadBuiltinFonts(d.fonts); err != nil {
		d.err = err
	}
	return d
}

// DefineStyle adds or overrides a named style on the document. Subsequent
// blocks that resolve through name will see the new definition; existing
// blocks already laid out are unaffected (layout consumes resolved values
// at render time). Returns d for fluent chaining.
func (d *Document) DefineStyle(name string, s Style) *Document {
	if d.err != nil {
		return d
	}
	if d.styles == nil {
		d.styles = BuiltinStyles()
	}
	d.styles[name] = s
	return d
}

// ResolveStyle returns the fully merged Style identified by name. The walk
// order is: the named style → its ParentStyle (recursively) → DefaultStyle
// at the root. Unknown names resolve through DefaultStyle alone, and a cycle
// in the ParentStyle chain is captured via Document.fail.
//
// Resolution is read-only with respect to the style table; callers may
// invoke ResolveStyle freely without invalidating builder state.
func (d *Document) ResolveStyle(name string) Style {
	if d.styles == nil {
		d.styles = BuiltinStyles()
	}

	// Collect the chain top-down: chain[0] is the named style, chain[n]
	// the most distant ancestor before Default. Detect cycles through a
	// visited set keyed by name; depth is also bounded as a belt-and-
	// braces guard against pathological tables.
	const maxDepth = 32
	chain := make([]Style, 0, 4)
	visited := make(map[string]struct{}, 4)

	current := name
	for i := 0; current != "" && i < maxDepth; i++ {
		if _, seen := visited[current]; seen {
			d.fail(errors.New("kardec: style cycle detected at " + current))
			break
		}
		visited[current] = struct{}{}

		s, ok := d.styles[current]
		if !ok {
			break
		}
		chain = append(chain, s)
		// Default has no parent; stop explicitly so an empty ParentStyle
		// on Default does not redirect the walk back into itself.
		if current == StyleDefault {
			break
		}
		current = s.ParentStyle
	}

	// Fold from root downward. Start with DefaultStyle as the absolute
	// floor, then layer the document's Default entry (so users can
	// override even Default), then everything from the chain in
	// ancestor→descendant order.
	out := DefaultStyle
	if defStyle, ok := d.styles[StyleDefault]; ok && name != StyleDefault {
		out = mergeStyle(defStyle, out)
	}
	for i := len(chain) - 1; i >= 0; i-- {
		out = mergeStyle(chain[i], out)
	}
	return out
}

// Err returns the first error captured during builder usage, or nil. Render
// returns the same error before attempting any I/O, so most callers can
// wait until Render to handle failures.
func (d *Document) Err() error { return d.err }

// fail records err as the document's first error if no error was set yet,
// then returns d for fluent chaining. Subsequent calls remain inert until
// the document is recreated.
func (d *Document) fail(err error) *Document {
	if d.err == nil {
		d.err = err
	}
	return d
}

// append adds a block to the current section, unless a prior error has
// already invalidated the document.
func (d *Document) append(b Block) *Document {
	if d.err != nil {
		return d
	}
	d.cur.Blocks = append(d.cur.Blocks, b)
	return d
}

// Heading appends a heading block at the requested level. Levels outside
// 1..6 are clamped to the nearest valid value.
func (d *Document) Heading(level int, runs ...Run) *Document {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	return d.append(Heading{level: level, runs: runs})
}

// Paragraph appends a body paragraph composed of inline runs.
func (d *Document) Paragraph(runs ...Run) *Document {
	return d.append(Paragraph{runs: runs})
}

// Builder pattern note
// --------------------
// The bare Paragraph / Heading methods above return *Document so existing
// chains like doc.Heading(...).Paragraph(...).PageBreak() keep working
// without changes. Style-aware fluent construction requires a different
// return type, so AddParagraph / AddHeading expose dedicated builders that
// commit back to the document via Done(). This split keeps the pre-style
// builder backward compatible while still allowing
//
//     doc.AddParagraph(kardec.Text("Body")).WithStyle(myStyle).LineHeight(1.4).Done()
//
// to flow naturally.

// ParagraphBuilder accumulates customization for a Paragraph and commits it
// to the document on Done. It is intentionally not safe for concurrent use.
type ParagraphBuilder struct {
	doc *Document
	p   Paragraph
}

// AddParagraph starts a fluent Paragraph builder pre-loaded with the given
// runs. Customize via WithStyle / WithNamedStyle / Justify / LineHeight,
// then call Done to commit and rejoin the *Document chain.
func (d *Document) AddParagraph(runs ...Run) *ParagraphBuilder {
	return &ParagraphBuilder{doc: d, p: Paragraph{runs: runs}}
}

// WithStyle attaches an inline Style override.
func (b *ParagraphBuilder) WithStyle(s Style) *ParagraphBuilder {
	b.p = b.p.WithStyle(s)
	return b
}

// WithNamedStyle selects a named style for resolution.
func (b *ParagraphBuilder) WithNamedStyle(name string) *ParagraphBuilder {
	b.p = b.p.WithNamedStyle(name)
	return b
}

// Justify shorthand for setting Alignment to AlignJustify.
func (b *ParagraphBuilder) Justify() *ParagraphBuilder {
	b.p.alignment = AlignJustify
	return b
}

// Align sets the paragraph's horizontal alignment explicitly.
func (b *ParagraphBuilder) Align(a Alignment) *ParagraphBuilder {
	b.p.alignment = a
	return b
}

// LineHeight sets the paragraph's line-height multiplier.
func (b *ParagraphBuilder) LineHeight(v float64) *ParagraphBuilder {
	b.p.lineHeight = v
	return b
}

// Done commits the accumulated Paragraph onto the document and returns the
// underlying *Document so the caller can resume the top-level builder
// chain.
func (b *ParagraphBuilder) Done() *Document {
	return b.doc.append(b.p)
}

// HeadingBuilder is the fluent counterpart for Heading blocks.
type HeadingBuilder struct {
	doc *Document
	h   Heading
}

// AddHeading starts a fluent Heading builder. Levels outside 1..6 are
// clamped, mirroring Heading.
func (d *Document) AddHeading(level int, runs ...Run) *HeadingBuilder {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	return &HeadingBuilder{doc: d, h: Heading{level: level, runs: runs}}
}

// WithStyle attaches an inline Style override to the heading.
func (b *HeadingBuilder) WithStyle(s Style) *HeadingBuilder {
	b.h = b.h.WithStyle(s)
	return b
}

// WithNamedStyle selects a named style for resolution.
func (b *HeadingBuilder) WithNamedStyle(name string) *HeadingBuilder {
	b.h = b.h.WithNamedStyle(name)
	return b
}

// Done commits the heading and returns the underlying *Document.
func (b *HeadingBuilder) Done() *Document {
	return b.doc.append(b.h)
}

// ResolveBlockStyle returns the effective Style for a Block, applying the
// priority chain documented in RFC-001 §6:
//
//  1. Block-level WithStyle override (Paragraph.style / Heading.style)
//  2. Block-level WithNamedStyle (Paragraph.styleName / Heading.styleName)
//  3. The block kind's default named style
//     (H1..H6 for Heading by level, Default for Paragraph)
//  4. ParentStyle chain of whichever named style was selected
//  5. DefaultStyle as the absolute floor
//
// Run-level inline overrides apply during typography (per-glyph) and are
// therefore out of scope here.
func (d *Document) ResolveBlockStyle(b Block) Style {
	switch v := b.(type) {
	case Paragraph:
		name := v.styleName
		if name == "" {
			name = StyleDefault
		}
		base := d.ResolveStyle(name)
		if v.hasStyle {
			base = mergeStyle(v.style, base)
		}
		// Paragraph builder convenience overrides applied last.
		if v.alignment != AlignLeft {
			base.Alignment = v.alignment
		}
		if v.lineHeight != 0 {
			base.LineHeight = v.lineHeight
		}
		return base
	case Heading:
		name := v.styleName
		if name == "" {
			name = HeadingStyleName(v.level)
		}
		base := d.ResolveStyle(name)
		if v.hasStyle {
			base = mergeStyle(v.style, base)
		}
		return base
	default:
		return d.ResolveStyle(StyleDefault)
	}
}

// PageBreak appends a forced page break.
func (d *Document) PageBreak() *Document { return d.append(PageBreak{}) }

// Spacer appends vertical whitespace of the given height.
func (d *Document) Spacer(h Length) *Document { return d.append(Spacer{Height: h}) }

// ErrRendererUnregistered is returned by Render / RenderTo / Bytes when no
// rendering implementation has been wired in. Importing the public render
// package — github.com/arthurhrc/kardec/render — installs the default
// implementation via init(), so this error typically signals a missing
// blank import:
//
//	import _ "github.com/arthurhrc/kardec/render"
var ErrRendererUnregistered = errors.New("kardec: no render implementation registered (import github.com/arthurhrc/kardec/render)")

// renderImpl is the registered render function, set at init time by the
// render package. The indirection avoids an import cycle between kardec
// and the orchestrator that combines layout, typography and pdf.
var renderImpl func(*Document, io.Writer) error

// SetRenderImpl wires a render implementation. The render package calls it
// from init(); user code should not invoke it directly. Calling SetRenderImpl
// with a nil function clears the registration.
func SetRenderImpl(f func(*Document, io.Writer) error) {
	renderImpl = f
}

// Render produces a PDF and writes it to the named file. The file is
// created (or truncated) with default permissions and closed before
// Render returns; callers don't manage the handle.
func (d *Document) Render(path string) error {
	if d.err != nil {
		return d.err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return d.RenderTo(f)
}

// RenderTo produces a PDF and writes it to the supplied io.Writer.
func (d *Document) RenderTo(w io.Writer) error {
	if d.err != nil {
		return d.err
	}
	if renderImpl == nil {
		return ErrRendererUnregistered
	}
	return renderImpl(d, w)
}

// Bytes returns the rendered PDF as a byte slice. Convenient for tests and
// for HTTP handlers that buffer responses; for large documents callers
// should prefer RenderTo + io.Pipe.
func (d *Document) Bytes() ([]byte, error) {
	if d.err != nil {
		return nil, d.err
	}
	var buf bytes.Buffer
	if err := d.RenderTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
