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

	// markdownBaseDir resolves relative image paths in source passed
	// to AppendMarkdown. Empty (the default) keeps the bridge
	// network- and disk-free: inline images stay a warning rather
	// than load arbitrary bytes off the filesystem.
	markdownBaseDir string

	// subsetFonts opts the renderer into glyf-table subsetting:
	// every glyph not referenced by a PlacedItem is zeroed out
	// before the FontFile2 stream is compressed. Off by default to
	// preserve byte-equivalence with v0.4 outputs; turn it on for
	// a 5-10x reduction in font payload at the cost of dropping
	// the full glyph table from each embedded face.
	subsetFonts bool

	// pdfa toggles PDF/A-2b compliance markers — XMP metadata
	// stream, /Metadata catalog entry, /ID in trailer. Strict
	// validators still need an OutputIntent with an embedded
	// sRGB ICC profile (deferred to v0.6); the lite output
	// produced here is what Acrobat accepts as PDF/A.
	pdfa bool

	// figureCounter / tableCounter assign the auto-numbers behind
	// LabeledFigure / LabeledTable. They are 1-based and never reset
	// across sections.
	figureCounter int
	tableCounter  int

	// clauseCounters tracks the per-level counter stack used by
	// Document.Clause to compose hierarchical numbers (1, 1.1,
	// 1.2, 2, 2.1, ...). Index 0 is the top level. Calling
	// Clause(N) increments counters[N-1] and truncates anything
	// deeper than N so the next call at N+1 starts at 1.
	clauseCounters []int

	// citationOrder records the BibEntry keys in the order they were
	// first cited. citations maps a key to its 1-based citation
	// number. Cite consults / mutates both; Bibliography uses
	// citationOrder to emit entries in citation order.
	citationOrder []string
	citations     map[string]int

	// labels maps a user-supplied label (the "growth-2024" in
	// LabeledFigure("growth-2024", img)) to the resolved kind plus
	// number. Ref / RefPage consult this map to compose a cross-
	// reference run; layout's post-pass consults the engine's
	// AnchorMark slice for the page-number side of the answer.
	labels map[string]labelInfo

	err error // first error encountered during builder usage; surfaced by Err and Render
}

// labelKind discriminates between the cross-reference families.
type labelKind uint8

const (
	labelFigure labelKind = iota + 1
	labelTable
)

// labelInfo is the resolved metadata behind a single registered
// label: which family it belongs to plus the auto-number assigned
// at registration time.
type labelInfo struct {
	kind   labelKind
	number int
}

// PDFA opts the document into PDF/A-2b conformance markers (XMP
// metadata + /ID + /Metadata catalog entry). Default off; calling
// with no arguments turns it on for fluent chaining.
//
// "Lite" caveat: without an OutputIntent referencing an embedded
// sRGB ICC profile (a v0.6 feature) the document is not strictly
// PDF/A-2b — the markers are present but veraPDF flags the
// missing OutputIntent. Most consumer readers (Acrobat, Foxit,
// Chrome) honor the marker regardless.
// EnablePDFA opts the document into PDF/A-2b conformance markers.
// Replaces the variadic-bool PDFA(on ...bool) form for an idiomatic
// Go API. Pair with DisablePDFA when conditional code paths need to
// turn the flag back off.
func (d *Document) EnablePDFA() *Document {
	if d.err != nil {
		return d
	}
	d.pdfa = true
	return d
}

// DisablePDFA clears the PDF/A flag set by EnablePDFA. Useful when a
// conditional template flips the conformance choice between renders.
func (d *Document) DisablePDFA() *Document {
	if d.err != nil {
		return d
	}
	d.pdfa = false
	return d
}

// PDFA opts the document into PDF/A markers.
//
// Deprecated: use EnablePDFA / DisablePDFA. Variadic-bool toggles are
// unidiomatic Go and the form ships only for the v0.x line. Removed
// at v1.0.
func (d *Document) PDFA(on ...bool) *Document {
	if d.err != nil {
		return d
	}
	if len(on) == 0 {
		d.pdfa = true
	} else {
		d.pdfa = on[0]
	}
	return d
}

// PDFAEnabled reports whether PDF/A markers will be emitted.
func (d *Document) PDFAEnabled() bool { return d.pdfa }

// EnableFontSubsetting opts the document into glyf-table subsetting:
// every glyph not referenced by a PlacedItem is wiped from the
// embedded TTF before the FontFile2 stream is FlateDecode-compressed.
// Real documents drop ~70% of their PDF size with this on.
func (d *Document) EnableFontSubsetting() *Document {
	if d.err != nil {
		return d
	}
	d.subsetFonts = true
	return d
}

// DisableFontSubsetting clears the subset flag set by
// EnableFontSubsetting, restoring the full-font embed path.
func (d *Document) DisableFontSubsetting() *Document {
	if d.err != nil {
		return d
	}
	d.subsetFonts = false
	return d
}

// SubsetFonts toggles glyf-table subsetting for embedded fonts.
//
// Deprecated: use EnableFontSubsetting / DisableFontSubsetting.
// Variadic-bool is unidiomatic; this form ships only for the v0.x
// line. Removed at v1.0.
func (d *Document) SubsetFonts(on ...bool) *Document {
	if d.err != nil {
		return d
	}
	if len(on) == 0 {
		d.subsetFonts = true
	} else {
		d.subsetFonts = on[0]
	}
	return d
}

// FontSubsetEnabled reports whether the next render will subset
// embedded fonts.
func (d *Document) FontSubsetEnabled() bool { return d.subsetFonts }

// SetMarkdownBaseDir configures the directory inline `![alt](path)`
// links resolve against during AppendMarkdown. When unset, the
// bridge keeps its conservative default of warning + dropping the
// image — the document never reaches the filesystem on its own.
func (d *Document) SetMarkdownBaseDir(dir string) *Document {
	if d.err != nil {
		return d
	}
	d.markdownBaseDir = dir
	return d
}

// MarkdownBaseDir returns the configured directory or "" when none.
func (d *Document) MarkdownBaseDir() string { return d.markdownBaseDir }

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
	return NewWithSetup(PageSetup{
		Size:        size,
		Orientation: Portrait,
		Margins:     margins,
	})
}

// NewWithSetup is the explicit-PageSetup form of New. Pass a fully
// populated PageSetup to set the orientation, column count or
// inter-column gap from the start; everything else is identical to
// New, including the bundled-font registration.
func NewWithSetup(setup PageSetup) *Document {
	first := &Section{Setup: setup}
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

// Paragraph appends a body paragraph composed of inline runs and returns
// a *ParagraphRef the caller can use to layer style overrides on top of
// the just-appended block. The ref embeds *Document so unrelated chained
// methods (Heading, Image, etc.) keep working unchanged:
//
//	doc.Paragraph(kardec.Text("body"))
//	doc.Paragraph(kardec.Text("body")).WithStyle(myStyle).LineHeight(1.4)
//	doc.Paragraph(kardec.Text("a")).Heading(2, kardec.Text("next"))
//
// The latter two forms used to require AddParagraph + Done; both are
// now redundant on *ParagraphRef.
func (d *Document) Paragraph(runs ...Run) *ParagraphRef {
	if d.err != nil {
		return &ParagraphRef{Document: d, blockIdx: -1}
	}
	d.cur.Blocks = append(d.cur.Blocks, Paragraph{runs: runs})
	return &ParagraphRef{Document: d, blockIdx: len(d.cur.Blocks) - 1, sec: d.cur}
}

// ParagraphRef is the return type of Document.Paragraph: a thin wrapper
// over the just-appended Paragraph block plus the *Document the caller
// continues to chain off of. The Document is embedded so doc methods
// (Heading, Image, Table, ...) flow through field promotion without
// the caller ever having to drop back to a *Document value.
//
// Style overrides on the ref mutate the appended block in place by
// reading the interface entry from the section's slice, mutating, and
// writing back. This keeps Block as a value type while still allowing
// retroactive configuration of the most-recently-added paragraph.
type ParagraphRef struct {
	*Document
	sec      *Section
	blockIdx int
}

// patch reads the appended Paragraph back from the slice, applies fn,
// and writes the result. No-op when the document is in an error
// state (blockIdx == -1).
func (r *ParagraphRef) patch(fn func(*Paragraph)) *ParagraphRef {
	if r == nil || r.blockIdx < 0 || r.sec == nil {
		return r
	}
	p := r.sec.Blocks[r.blockIdx].(Paragraph)
	fn(&p)
	r.sec.Blocks[r.blockIdx] = p
	return r
}

// WithStyle attaches an inline Style override to the appended paragraph.
func (r *ParagraphRef) WithStyle(s Style) *ParagraphRef {
	return r.patch(func(p *Paragraph) {
		p.style = s
		p.hasStyle = true
	})
}

// WithNamedStyle selects a named style for the appended paragraph.
func (r *ParagraphRef) WithNamedStyle(name string) *ParagraphRef {
	return r.patch(func(p *Paragraph) { p.styleName = name })
}

// Align sets the paragraph's horizontal alignment.
func (r *ParagraphRef) Align(a Alignment) *ParagraphRef {
	return r.patch(func(p *Paragraph) { p.alignment = a })
}

// Justify shortcut for AlignJustify.
func (r *ParagraphRef) Justify() *ParagraphRef { return r.Align(AlignJustify) }

// LineHeight sets the paragraph's line-height multiplier (e.g. 1.4 for
// 140% leading). Zero clears the override and falls back to the
// resolved style's lineHeight.
func (r *ParagraphRef) LineHeight(v float64) *ParagraphRef {
	return r.patch(func(p *Paragraph) { p.lineHeight = v })
}

// Done returns the underlying *Document. Retained for source
// compatibility with the deprecated AddParagraph chain — call sites
// updated to the new ref API don't need it because the embedded
// *Document is reachable directly.
func (r *ParagraphRef) Done() *Document {
	if r == nil {
		return nil
	}
	return r.Document
}

// ParagraphBuilder is the legacy alias for *ParagraphRef.
//
// Deprecated: use the *ParagraphRef returned by Document.Paragraph.
type ParagraphBuilder = ParagraphRef

// AddParagraph is the legacy entry point.
//
// Deprecated: use Document.Paragraph, which now returns the same ref
// type and exposes the same WithStyle / WithNamedStyle / Align /
// Justify / LineHeight methods.
func (d *Document) AddParagraph(runs ...Run) *ParagraphRef {
	return d.Paragraph(runs...)
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

// KeepTogether appends a group of inner blocks bound to the same page.
// The engine guarantees the supplied blocks land on the same page, or
// flushes once and starts the group on a fresh page when the current
// remaining space cannot hold them. Groups taller than a full page
// overflow naturally onto further pages — KeepTogether does not loop
// trying to find a page tall enough for an oversized group.
func (d *Document) KeepTogether(blocks ...Block) *Document {
	return d.append(NewKeepTogether(blocks...))
}

// HorizontalRule appends a horizontal divider line spanning the content
// width. Calling without arguments produces the default (0.5pt gray
// line, 6pt padding above and below); pass a populated HorizontalRule
// to override thickness, color, or padding.
func (d *Document) HorizontalRule(rule ...HorizontalRule) *Document {
	var r HorizontalRule
	if len(rule) > 0 {
		r = rule[0]
	}
	return d.append(r)
}

// Leader appends a left-and-right block with a dotted fill between
// the two sides. Convenience wrapper over NewLeader for fluent
// chaining: doc.Leader([]Run{Text("Skill")}, []Run{Text("80%")}).
func (d *Document) Leader(left, right []Run) *Document {
	return d.append(NewLeader(left, right))
}

// Anchor appends a named destination at the current flow position.
// The name is matched against the "#<name>" target of internal Link
// runs — see Link / SetLink — and is also exposed in the PDF's named
// destinations table so external tools can jump directly via URL
// fragments.
//
// Names should be ASCII identifiers without spaces; the writer does
// not escape them. An empty name is a no-op so callers can guard
// programmatic anchor generation without special-casing.
func (d *Document) Anchor(name string) *Document {
	if d.err != nil || name == "" {
		return d
	}
	return d.append(Anchor{name: name})
}

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

// SetRenderImpl wires a render implementation. The render package
// calls it from init(); user code should not invoke it directly.
// Calling SetRenderImpl with a nil function clears the registration.
//
// Deprecated: this is an internal seam exposed only because Go has
// no friend-package mechanism for the renderer-injection pattern.
// User code must never call it; the render package's init() is the
// only legitimate consumer. The function may move behind an
// internal helper at v1.0 without further notice.
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
