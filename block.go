package kardec

// Block is the unit of vertical flow inside a section. Implementations carry
// their own content (runs, rows, image ref) and are positioned by the layout
// engine in document order.
//
// The interface is intentionally minimal at the public surface; concrete
// methods needed by the layout engine live on the unexported types.
type Block interface {
	blockKind() blockKind
}

type blockKind uint8

const (
	kindParagraph blockKind = iota + 1
	kindHeading
	kindPageBreak
	kindSpacer
	kindTable
	kindImage
	kindMath
	kindList
	kindAnchor
)

// Paragraph is a body-text block of one or more inline Runs.
type Paragraph struct {
	runs       []Run
	styleName  string  // optional named style; "" defers to block kind default
	style      Style   // optional explicit override; merged on top of styleName
	hasStyle   bool    // true if style was set explicitly via WithStyle
	alignment  Alignment
	lineHeight float64 // 0 means "use style default"
}

func (Paragraph) blockKind() blockKind { return kindParagraph }

// WithStyle returns a copy of p carrying s as an inline style override.
// Style resolution merges s on top of any named style and the block kind's
// default during layout.
func (p Paragraph) WithStyle(s Style) Paragraph {
	p.style = s
	p.hasStyle = true
	return p
}

// WithNamedStyle returns a copy of p that resolves through the given named
// style before falling back to the block kind's default. Passing "" clears
// the named style.
func (p Paragraph) WithNamedStyle(name string) Paragraph {
	p.styleName = name
	return p
}

// Heading carries a level (1–6) and inline runs. Levels above 6 are clamped
// at construction time to 6, mirroring HTML conventions.
type Heading struct {
	level     int
	runs      []Run
	styleName string
	style     Style
	hasStyle  bool
}

func (Heading) blockKind() blockKind { return kindHeading }

// WithStyle returns a copy of h carrying s as an inline style override.
// See Paragraph.WithStyle for the resolution rules.
func (h Heading) WithStyle(s Style) Heading {
	h.style = s
	h.hasStyle = true
	return h
}

// WithNamedStyle returns a copy of h that resolves through the given named
// style instead of the level-derived default (H1..H6).
func (h Heading) WithNamedStyle(name string) Heading {
	h.styleName = name
	return h
}

// PageBreak forces the layout engine to start the next block on a new page.
type PageBreak struct{}

func (PageBreak) blockKind() blockKind { return kindPageBreak }

// Spacer reserves vertical whitespace of the given height.
type Spacer struct {
	Height Length
}

func (Spacer) blockKind() blockKind { return kindSpacer }

// Alignment controls the horizontal arrangement of inline content within a
// paragraph or table cell.
type Alignment uint8

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
	AlignJustify
)
