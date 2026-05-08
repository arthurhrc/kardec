package mathlayout

// Script-related constants. The shifts are reported as fractions of the
// base size; TeX uses sigma_5/sigma_6 super- and subscript drops from a
// font's MATH table, but until the typography track surfaces those we
// fall back to plausible constants that match a 12pt textstyle layout.
const (
	// scriptScale is the size factor applied to subscripts and
	// superscripts. TeX uses 0.7 for textstyle, which is what we adopt.
	scriptScale = 0.70

	// subscriptDrop is the distance the subscript's top edge sits below
	// the parent's baseline, expressed as a fraction of the base size.
	subscriptDrop = 0.30

	// superscriptShift is the distance the superscript's baseline sits
	// above the parent's baseline, expressed as a fraction of the base
	// size.
	superscriptShift = 0.45
)

// layoutSubSup typesets a base expression with optional subscript and
// superscript expressions attached to its right-hand side. Scripts are
// laid out at scriptScale × the surrounding size; their vertical shifts
// are derived from subscriptDrop and superscriptShift.
//
// When both scripts are present they share the same x-offset (just to
// the right of the base) so they stack vertically rather than fan out
// diagonally — this matches TeX's default placement for scripts on
// ordinary atoms.
func layoutSubSup(s SubSup, font Font, sizePt float64, display bool) Box {
	base := layoutNode(s.Base(), font, sizePt, display)

	var sub, sup Box
	hasSub := s.Sub() != nil
	hasSup := s.Sup() != nil
	scriptSize := sizePt * scriptScale
	if hasSub {
		sub = layoutNode(s.Sub(), font, scriptSize, false)
	}
	if hasSup {
		sup = layoutNode(s.Sup(), font, scriptSize, false)
	}

	// All three boxes share the parent's baseline. Compute parent
	// ascent/descent from each contribution.
	parentHeight := base.Height
	parentDepth := base.Depth

	scriptX := base.Width
	subTopOffset := 0.0   // Y of subscript top edge, relative to parent baseline
	supTopOffset := 0.0   // Y of superscript top edge, relative to parent baseline

	if hasSup {
		// Superscript baseline sits superscriptShift × sizePt above the
		// parent baseline. Top edge is that shift plus the script's
		// own ascent.
		supBaselineUp := sizePt * superscriptShift
		supTotalAscent := supBaselineUp + sup.Height
		if supTotalAscent > parentHeight {
			parentHeight = supTotalAscent
		}
		supTopOffset = supBaselineUp + sup.Height
	}
	if hasSub {
		// Subscript top edge sits subscriptDrop × sizePt below the
		// parent baseline. Bottom edge is that drop plus script depth.
		subTop := sizePt * subscriptDrop
		subBottom := subTop + sub.Height + sub.Depth
		if subBottom > parentDepth {
			parentDepth = subBottom
		}
		subTopOffset = subTop
	}

	parent := Box{
		Width:  base.Width,
		Height: parentHeight,
		Depth:  parentDepth,
	}

	// Place base with its baseline aligned to the parent baseline.
	base.X = 0
	base.Y = parentHeight - base.Height
	parent.Children = append(parent.Children, base)

	if hasSup {
		sup.X = scriptX
		// supTopOffset is measured upward from the parent baseline; the
		// parent baseline sits at Y = parentHeight; the script's top
		// edge is supTopOffset above the baseline, hence parentHeight -
		// supTopOffset.
		sup.Y = parentHeight - supTopOffset
		parent.Children = append(parent.Children, sup)
		w := scriptX + sup.Width
		if w > parent.Width {
			parent.Width = w
		}
	}
	if hasSub {
		sub.X = scriptX
		// subTopOffset is measured downward from the parent baseline;
		// the script's top edge is below the baseline, so its Y from
		// the parent's top edge is parentHeight + subTopOffset.
		sub.Y = parentHeight + subTopOffset
		parent.Children = append(parent.Children, sub)
		w := scriptX + sub.Width
		if w > parent.Width {
			parent.Width = w
		}
	}
	return parent
}
