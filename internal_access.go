package kardec

// This file exposes read-only accessors on the document tree so that the
// internal/layout package can walk the unexported fields of Run, Paragraph
// and Heading without breaking encapsulation. Public surface remains the
// fluent builder; these accessors are intentionally narrow.

// Sections returns the ordered sections that compose the document. The
// returned slice is the document's own backing storage; callers must not
// mutate it.
func (d *Document) Sections() []*Section { return d.sections }

// Runs returns the inline runs that make up the paragraph.
func (p Paragraph) Runs() []Run { return p.runs }

// Alignment returns the paragraph's alignment, or AlignLeft when the
// builder did not set one explicitly.
func (p Paragraph) Alignment() Alignment { return p.alignment }

// LineHeight returns the paragraph's line-height multiplier, or 0 when the
// style default should apply.
func (p Paragraph) LineHeight() float64 { return p.lineHeight }

// Level returns the heading level (1..6).
func (h Heading) Level() int { return h.level }

// Runs returns the inline runs that make up the heading.
func (h Heading) Runs() []Run { return h.runs }

// Text returns the textual payload of a Run.
func (r Run) Text() string { return r.text }

// Bold reports whether the run carries the bold weight flag.
func (r Run) Bold() bool { return r.bold }

// Italic reports whether the run carries the italic style flag.
func (r Run) Italic() bool { return r.italic }

// ColorOverride returns the run's explicit color override, if any.
func (r Run) ColorOverride() (Color, bool) {
	if r.override.color == nil {
		return Color{}, false
	}
	return *r.override.color, true
}

// SizeOverride returns the run's explicit size override, if any.
func (r Run) SizeOverride() (Length, bool) {
	if r.override.size == nil {
		return 0, false
	}
	return *r.override.size, true
}

// Link returns the run's hyperlink target (its `url` argument when
// constructed via Link), or the empty string when the run is plain.
func (r Run) Link() string { return r.link }
