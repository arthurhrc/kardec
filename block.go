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
)

// Paragraph is a body-text block of one or more inline Runs.
type Paragraph struct {
	runs       []Run
	styleName  string
	alignment  Alignment
	lineHeight float64 // 0 means "use style default"
}

func (Paragraph) blockKind() blockKind { return kindParagraph }

// Heading carries a level (1–6) and inline runs. Levels above 6 are clamped
// at construction time to 6, mirroring HTML conventions.
type Heading struct {
	level int
	runs  []Run
}

func (Heading) blockKind() blockKind { return kindHeading }

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
