package layout

import "github.com/arthurhrc/kardec"

// blockStyle is the resolved style the engine consumes for a single
// block. It is intentionally tiny: only the fields that influence
// placement live here. Once feat/dsl-style ships, this struct will be
// produced by Style.Resolve rather than the heuristics below.
type blockStyle struct {
	sizePt       float64
	bold         bool
	italic       bool
	color        kardec.Color
	lineHeight   float64 // multiplier; 1.2 == 120% of size
	spaceBeforePt float64
	spaceAfterPt  float64
	alignment    kardec.Alignment
}

// defaultParagraphStyle is the fallback style used for body paragraphs
// when no Style.Resolve hook is wired up (the v0.1 case).
func defaultParagraphStyle() blockStyle {
	return blockStyle{
		sizePt:        11,
		color:         kardec.ColorBlack,
		lineHeight:    1.2,
		spaceBeforePt: 0,
		spaceAfterPt:  6,
		alignment:     kardec.AlignLeft,
	}
}

// headingStyle returns the heuristic style for a heading at the given
// level. Sizes follow Word's default H1..H6 scale loosely (24/18/14/12/11/10
// pt). Spacing-before grows with importance so H1 visually anchors a new
// section without manual spacers.
func headingStyle(level int) blockStyle {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	sizes := [6]float64{24, 18, 14, 12, 11, 10}
	spacesBefore := [6]float64{18, 14, 10, 8, 6, 6}
	spacesAfter := [6]float64{6, 6, 4, 4, 4, 4}
	return blockStyle{
		sizePt:        sizes[level-1],
		bold:          true,
		color:         kardec.ColorBlack,
		lineHeight:    1.15,
		spaceBeforePt: spacesBefore[level-1],
		spaceAfterPt:  spacesAfter[level-1],
		alignment:     kardec.AlignLeft,
	}
}
