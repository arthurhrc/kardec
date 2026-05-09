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
	kindTOC
	kindHorizontalRule
	kindKeepTogether
	kindLeader
)

// NewParagraph constructs a standalone Paragraph block. Callers that
// build blocks outside the fluent Document chain — most commonly when
// supplying children to KeepTogether — use this. WithStyle and
// WithNamedStyle compose on the returned value.
func NewParagraph(runs ...Run) Paragraph {
	return Paragraph{runs: runs}
}

// NewHeading constructs a standalone Heading block at the given level.
// Levels are clamped to the 1..6 range to mirror HTML conventions.
func NewHeading(level int, runs ...Run) Heading {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	return Heading{level: level, runs: runs}
}

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

// HorizontalRule is a thin filled line stretched across the content area,
// used to separate sections of body text. Defaults to a 0.5pt gray line
// with 6pt of vertical padding above and below; the zero value renders
// without explicit configuration.
type HorizontalRule struct {
	Thickness Length // 0 means the layout default (0.5pt)
	Color     Color  // zero value (black) means the layout default (gray)
	Padding   Length // 0 means the layout default (6pt)
}

func (HorizontalRule) blockKind() blockKind { return kindHorizontalRule }

// Anchor is a named position inside the document. It occupies no
// vertical space on its own; the layout engine simply records the
// page and Y coordinate at which it was inserted, and the PDF writer
// turns the result into a named destination linkable from anywhere
// else in the document via a "#<name>" URL.
//
// Pair Anchor with Link("text", "#name") to build clickable cross-
// references — the canonical use case is a TOC at the top of the
// document linking to each section heading.
type Anchor struct {
	name string
}

func (Anchor) blockKind() blockKind { return kindAnchor }

// Name returns the anchor's identifier. Read-only access for layout
// and renderer integrations.
func (a Anchor) Name() string { return a.name }

// Leader is a one-line block that places left runs at the left margin
// and right runs at the right margin, filling the gap with a row of
// dots. The canonical use case is a "Skill........80%" or
// "Senator (R-CA)......$1,200,000" row in a CV / financial layout.
//
// Construct via NewLeader; the type's fields stay unexported so future
// versions can add fill characters or alignment knobs without breaking
// callers. The block resolves its style through StyleDefault unless a
// named or explicit style is attached on top via WithStyle.
type Leader struct {
	left  []Run
	right []Run
}

// NewLeader returns a Leader block with the given left and right run
// sequences. Pass either side empty for a one-sided dotted line.
func NewLeader(left, right []Run) Leader {
	return Leader{left: left, right: right}
}

func (Leader) blockKind() blockKind { return kindLeader }

// Left returns the left-aligned runs of the leader. Read-only access
// for layout integrations.
func (l Leader) Left() []Run { return l.left }

// Right returns the right-aligned runs of the leader.
func (l Leader) Right() []Run { return l.right }

// KeepTogether wraps a slice of inner blocks so the layout engine
// guarantees they all land on the same page. The canonical use case
// is binding a heading to the first paragraph that follows it: the
// engine never produces a page that ends with the heading and starts
// the next page with the paragraph.
//
// When the group is taller than a single page, KeepTogether degrades
// gracefully: the engine flushes once to start the group on a fresh
// page, then allows the inner blocks to overflow naturally onto
// further pages. This avoids an infinite-flush loop on oversized
// groups.
type KeepTogether struct {
	blocks []Block
}

// NewKeepTogether returns a KeepTogether group containing the supplied
// blocks in document order. The slice is copied so further mutation by
// the caller does not affect the document.
func NewKeepTogether(blocks ...Block) KeepTogether {
	cp := make([]Block, len(blocks))
	copy(cp, blocks)
	return KeepTogether{blocks: cp}
}

func (KeepTogether) blockKind() blockKind { return kindKeepTogether }

// Blocks returns the inner blocks of the group. The returned slice is
// the group's own backing storage; callers must not mutate it.
func (k KeepTogether) Blocks() []Block { return k.blocks }

// Alignment controls the horizontal arrangement of inline content within a
// paragraph or table cell.
type Alignment uint8

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
	AlignJustify
	// AlignDecimal aligns content on the decimal point. Only
	// meaningful inside table cells driven by AlignDecimalCol; using
	// it on a Paragraph degrades to AlignRight at layout time. Cells
	// without a "." pivot in their text fall back to right alignment
	// so an integer mixed in with decimals still rests at the same
	// vertical baseline as the dotted neighbours.
	AlignDecimal
)
