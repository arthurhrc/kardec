package mathlayout

const (
	// bigOpDisplayScale is the size factor applied to a big operator's
	// glyph in display mode. TeX swaps in a larger glyph variant from
	// the MATH table; until that's wired through we approximate by
	// rendering the same glyph at a larger point size.
	bigOpDisplayScale = 1.50

	// bigOpLimitScale is the size factor applied to upper and lower
	// limits. Matching scriptScale keeps the visual weight close to
	// TeX's textstyle rendering of the same formula.
	bigOpLimitScale = 0.70

	// bigOpLimitGap is the vertical gap between the operator's edge
	// and the limit's near edge in display mode, expressed as a
	// fraction of the operator size. It mirrors TeX's BigOpSpacing
	// parameters loosely.
	bigOpLimitGap = 0.10
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

// layoutBigOpDisplay renders a big operator in displaystyle: the
// operator glyph is enlarged by bigOpDisplayScale, the upper limit
// centres above it and the lower limit centres below it. The optional
// body follows the resulting stack on the right.
func layoutBigOpDisplay(b BigOp, font Font, sizePt float64) Box {
	opSize := sizePt * bigOpDisplayScale
	op := layoutGlyphRun(b.Symbol(), font, opSize)

	limitSize := sizePt * bigOpLimitScale
	gap := opSize * bigOpLimitGap

	var upper, lower Box
	hasUpper := b.Upper() != nil
	hasLower := b.Lower() != nil
	if hasUpper {
		upper = layoutNode(b.Upper(), font, limitSize, false)
	}
	if hasLower {
		lower = layoutNode(b.Lower(), font, limitSize, false)
	}

	// Stack width: max(op, upper, lower) so limits centre over the
	// widest contributor.
	stackWidth := op.Width
	if upper.Width > stackWidth {
		stackWidth = upper.Width
	}
	if lower.Width > stackWidth {
		stackWidth = lower.Width
	}

	parentHeight := op.Height
	parentDepth := op.Depth
	if hasUpper {
		parentHeight += gap + upper.Height + upper.Depth
	}
	if hasLower {
		parentDepth += gap + lower.Height + lower.Depth
	}

	parent := Box{
		Width:  stackWidth,
		Height: parentHeight,
		Depth:  parentDepth,
	}

	// Operator glyph: centred horizontally within stackWidth, baseline
	// aligned to the parent baseline. Promote glyphs from the op Box
	// into the parent so subsequent body re-baselining can shift them
	// uniformly.
	opXShift := (stackWidth - op.Width) / 2
	opYShift := parent.Height - op.Height
	for _, g := range op.Glyphs {
		parent.Glyphs = append(parent.Glyphs, PlacedGlyph{
			X:      g.X + opXShift,
			Y:      g.Y + opYShift,
			Rune:   g.Rune,
			SizePt: g.SizePt,
		})
	}

	if hasUpper {
		upper.X = (stackWidth - upper.Width) / 2
		upper.Y = parent.Height - op.Height - gap - upper.Depth - upper.Height
		if upper.Y < 0 {
			upper.Y = 0
		}
		parent.Children = append(parent.Children, upper)
	}
	if hasLower {
		lower.X = (stackWidth - lower.Width) / 2
		lower.Y = parent.Height + op.Depth + gap
		parent.Children = append(parent.Children, lower)
	}

	if body := b.Body(); body != nil {
		bodyBox := layoutNode(body, font, sizePt, false)
		bodyGap := spaceRelOp * sizePt
		bodyBox.X = parent.Width + bodyGap
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
