package layout

import "github.com/arthurhrc/kardec"

// placeLeader lays out a single Leader block: left runs flush against
// the left margin, right runs flush against the right margin, dot row
// between them. Reuses the TOC's emitDotLeader helper for the dotted
// fill so visual proportions stay consistent across the two consumers.
func (e Engine) placeLeader(cur *pageCursor, flush func(), l kardec.Leader, style blockStyle, fonts FontProvider) {
	applySpaceBefore(cur, flush, style.spaceBeforePt)
	lineHeight := style.lineHeight * style.sizePt
	if cur.remainingHeight() < lineHeight {
		flush()
	}

	available := cur.availableWidth()
	leftTokens := shapeRuns(l.Left(), fonts, style, kardec.Pt(style.sizePt), style.color, nil)
	rightTokens := shapeRuns(l.Right(), fonts, style, kardec.Pt(style.sizePt), style.color, nil)

	leftWidth := tokensWidth(leftTokens)
	rightWidth := tokensWidth(rightTokens)

	emitInlineTokens(cur, leftTokens, cur.x0, cur.cursorY)
	emitInlineTokens(cur, rightTokens, cur.x0+available-rightWidth, cur.cursorY)

	gap := style.sizePt * 0.5
	dotStart := cur.x0 + leftWidth + gap
	dotEnd := cur.x0 + available - rightWidth - gap
	if dotEnd > dotStart {
		emitDotLeader(cur, dotStart, cur.cursorY, dotEnd-dotStart, style)
	}

	cur.cursorY += lineHeight
	cur.cursorY += style.spaceAfterPt
}
