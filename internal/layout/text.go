package layout

import (
	"github.com/arthurhrc/kardec"
)

// placeTextBlock lays out a paragraph- or heading-shaped block. It runs
// the line breaker, reserves SpaceBefore/SpaceAfter, advances Y line by
// line and emits one PlacedItem per visible token.
//
// When the block does not fit on the current page, the breaker is
// re-invoked: lines that fit are emitted on the current page, then flush
// is called and the remainder continues on the fresh page. Headings that
// don't fit at the top of a page are still emitted (oversized headings
// degrade gracefully rather than loop forever).
func (e Engine) placeTextBlock(cur *pageCursor, flush func(), runs []kardec.Run, style blockStyle, fonts FontProvider) error {
	if len(runs) == 0 {
		// Empty paragraph still consumes its inter-block spacing so the
		// layout matches what the user authored.
		applySpaceBefore(cur, flush, style.spaceBeforePt)
		cur.cursorY += style.lineHeight * style.sizePt
		cur.cursorY += style.spaceAfterPt
		return nil
	}

	applySpaceBefore(cur, flush, style.spaceBeforePt)

	tokens := shapeRuns(runs, fonts, kardec.Pt(style.sizePt), style.color)
	if len(tokens) == 0 {
		cur.cursorY += style.spaceAfterPt
		return nil
	}

	lines := breakLines(tokens, cur.availableWidth())

	for i, ln := range lines {
		lineHeight := style.lineHeight * style.sizePt
		if cur.remainingHeight() < lineHeight {
			flush()
		}
		isLast := i == len(lines)-1
		emitLine(cur, ln, style, isLast)
	}
	cur.cursorY += style.spaceAfterPt
	return nil
}

// applySpaceBefore advances the cursor by spaceBefore, paginating if that
// would push past the bottom margin.
func applySpaceBefore(cur *pageCursor, flush func(), spaceBefore float64) {
	if spaceBefore <= 0 {
		return
	}
	// SpaceBefore at the very top of a page is conventionally swallowed,
	// matching Word's "remove space if at top of page" behaviour.
	if cur.cursorY <= cur.y0 {
		return
	}
	if cur.cursorY+spaceBefore > cur.y1 {
		flush()
		return
	}
	cur.cursorY += spaceBefore
}

// emitLine places one fully-broken line on the page. The baseline is
// computed from the cursor's current top + the line's ascent so glyphs
// sit visually correct relative to neighbour blocks.
//
// Justification (AlignJustify) distributes extra space across whitespace
// gaps proportionally; the very last line of a paragraph falls back to
// AlignLeft so the trailing words don't fan out awkwardly.
func emitLine(cur *pageCursor, ln line, style blockStyle, isLastLine bool) {
	available := cur.availableWidth()
	extra := available - ln.width
	if extra < 0 {
		extra = 0
	}

	x := cur.x0
	switch style.alignment {
	case kardec.AlignCenter:
		x = cur.x0 + extra/2
	case kardec.AlignRight:
		x = cur.x0 + extra
	case kardec.AlignJustify:
		if isLastLine {
			// Last line of a justified paragraph stays left-aligned.
			break
		}
		emitJustifiedLine(cur, ln, style, available)
		cur.cursorY += style.lineHeight * style.sizePt
		return
	}

	baselineY := cur.cursorY
	for _, t := range ln.tokens {
		if t.isSpace {
			x += t.width
			continue
		}
		cur.items = append(cur.items, PlacedItem{
			X:     kardec.Pt(x),
			Y:     kardec.Pt(baselineY),
			Text:  t.text,
			Font:  t.font,
			Size:  kardec.Pt(t.sizePt),
			Color: t.color,
		})
		x += t.width
	}
	cur.cursorY += style.lineHeight * style.sizePt
}

// emitJustifiedLine spreads the surplus available width across whitespace
// tokens. Each whitespace gap receives an equal share of the extra width.
// If the line has no whitespace gaps, it falls back to left alignment.
func emitJustifiedLine(cur *pageCursor, ln line, style blockStyle, available float64) {
	var spaceCount int
	for _, t := range ln.tokens {
		if t.isSpace {
			spaceCount++
		}
	}
	extraPerSpace := 0.0
	if spaceCount > 0 {
		extraPerSpace = (available - ln.width) / float64(spaceCount)
		if extraPerSpace < 0 {
			extraPerSpace = 0
		}
	}
	x := cur.x0
	baselineY := cur.cursorY
	for _, t := range ln.tokens {
		if t.isSpace {
			x += t.width + extraPerSpace
			continue
		}
		cur.items = append(cur.items, PlacedItem{
			X:     kardec.Pt(x),
			Y:     kardec.Pt(baselineY),
			Text:  t.text,
			Font:  t.font,
			Size:  kardec.Pt(t.sizePt),
			Color: t.color,
		})
		x += t.width
	}
}
