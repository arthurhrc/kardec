package mathlayout

const (
	// bigOpLimitScale is the size factor applied to upper and lower
	// limits. Matching scriptScale keeps the visual weight close to
	// TeX's textstyle rendering of the same formula.
	bigOpLimitScale = 0.70
)

// layoutBigOp typesets a large operator (sum, integral, product, ...)
// with optional lower and upper limits and an optional body that
// follows the operator on the same line.
//
// In inline (text) mode, the operator renders at the surrounding size
// and limits attach as ordinary subscript/superscript on the right.
// Display mode uses larger operator glyphs and stacks limits above and
// below; that branch lives in layoutBigOpDisplay.
func layoutBigOp(b BigOp, font Font, sizePt float64, display bool) Box {
	if display {
		return layoutBigOpDisplay(b, font, sizePt)
	}
	return layoutBigOpInline(b, font, sizePt)
}

// layoutBigOpInline renders a big operator in textstyle. The operator
// glyph sits at the baseline; lower/upper limits attach as ordinary
// scripts to the right; the body follows after the script column with
// the same medium spacing TeX inserts between an Op and its operand.
func layoutBigOpInline(b BigOp, font Font, sizePt float64) Box {
	op := layoutGlyphRun(b.Symbol(), font, sizePt)
	scriptSize := sizePt * scriptScale
	hasSub := b.Lower() != nil
	hasSup := b.Upper() != nil

	parent := op
	scriptX := op.Width

	if hasSub {
		sub := layoutNode(b.Lower(), font, scriptSize, false)
		subTop := sizePt * subscriptDrop
		subBottom := subTop + sub.Height + sub.Depth
		if subBottom > parent.Depth {
			parent.Depth = subBottom
		}
		sub.X = scriptX
		sub.Y = parent.Height + subTop
		parent.Children = append(parent.Children, sub)
		w := scriptX + sub.Width
		if w > parent.Width {
			parent.Width = w
		}
	}
	if hasSup {
		sup := layoutNode(b.Upper(), font, scriptSize, false)
		supBaselineUp := sizePt * superscriptShift
		supTotalAscent := supBaselineUp + sup.Height
		if supTotalAscent > parent.Height {
			delta := supTotalAscent - parent.Height
			rebaseDown(&parent, delta)
			parent.Height = supTotalAscent
		}
		sup.X = scriptX
		sup.Y = parent.Height - supBaselineUp - sup.Height
		parent.Children = append(parent.Children, sup)
		w := scriptX + sup.Width
		if w > parent.Width {
			parent.Width = w
		}
	}

	if body := b.Body(); body != nil {
		bodyBox := layoutNode(body, font, sizePt, false)
		gap := spaceRelOp * sizePt
		bodyBox.X = parent.Width + gap
		if bodyBox.Height > parent.Height {
			delta := bodyBox.Height - parent.Height
			rebaseDown(&parent, delta)
			parent.Height = bodyBox.Height
		}
		if bodyBox.Depth > parent.Depth {
			parent.Depth = bodyBox.Depth
		}
		bodyBox.Y = parent.Height - bodyBox.Height
		parent.Width = bodyBox.X + bodyBox.Width
		parent.Children = append(parent.Children, bodyBox)
	}

	return parent
}

// layoutBigOpDisplay is the display-mode entry point. It is filled in
// by a subsequent commit; for now it falls back to inline rendering so
// the engine produces a sensible (if visually less impressive) result
// either way.
func layoutBigOpDisplay(b BigOp, font Font, sizePt float64) Box {
	return layoutBigOpInline(b, font, sizePt)
}

// rebaseDown shifts every child and glyph in box down by delta. Used
// when the parent's ascent grows after children have already been
// placed, so existing content stays anchored to the baseline.
func rebaseDown(box *Box, delta float64) {
	for i := range box.Children {
		box.Children[i].Y += delta
	}
	for i := range box.Glyphs {
		box.Glyphs[i].Y += delta
	}
}
