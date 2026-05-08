package mathlayout

// Fraction-related constants. The shifts mirror TeX's textstyle and
// displaystyle defaults closely enough to look right at 12pt; refinement
// against MATH-table FractionNumeratorShiftUp etc. is a follow-up.
const (
	// fracTextScale is the size factor applied to numerator and
	// denominator in textstyle. Display-mode fractions render at full
	// size.
	fracTextScale = 0.90

	// fracNumShiftUp is the distance the numerator's baseline sits
	// above the parent baseline, expressed as a fraction of the
	// surrounding (un-scaled) size. Numerator height extends further
	// up by its own ascent.
	fracNumShiftUp = 0.50

	// fracDenomShiftDown is the distance the denominator's top edge
	// sits below the parent baseline, expressed as a fraction of the
	// surrounding size.
	fracDenomShiftDown = 0.40

	// fracAxis is the height of the math axis above the baseline. The
	// fraction bar centres on this axis.
	fracAxis = 0.25

	// fracBarPadding is the horizontal slack added to each side of the
	// fraction bar so it slightly overhangs the wider of the two
	// stacked boxes.
	fracBarPadding = 0.10

	// fracRuleThickness is the thickness of the fraction bar relative
	// to the surrounding font size.
	fracRuleThickness = 0.06
)

// layoutFrac typesets a built-up fraction. Numerator and denominator are
// laid out at the appropriate scaled size, centred horizontally, and
// stacked above and below a horizontal Rule that sits on the math axis.
//
// Display mode keeps the children at full size; text mode shrinks them
// to fracTextScale × the surrounding size so the formula does not
// dwarf the running text.
func layoutFrac(f Frac, font Font, sizePt float64, display bool) Box {
	childSize := sizePt
	if !display {
		childSize = sizePt * fracTextScale
	}

	num := layoutNode(f.Numerator(), font, childSize, display)
	den := layoutNode(f.Denominator(), font, childSize, display)

	// The bar's width is max(num, den) plus a small overhang. The
	// parent's Width is the bar's width.
	maxChildWidth := num.Width
	if den.Width > maxChildWidth {
		maxChildWidth = den.Width
	}
	barOverhang := fracBarPadding * sizePt
	barWidth := maxChildWidth + 2*barOverhang
	barThickness := fracRuleThickness * sizePt

	// Vertical layout, working in baseline-relative coordinates and
	// then translating to box-top-relative at the end.
	axis := fracAxis * sizePt

	// Numerator: baseline is fracNumShiftUp × sizePt above the parent
	// baseline. Top edge is that plus the numerator's own height.
	numBaselineUp := fracNumShiftUp * sizePt
	numTotalAscent := numBaselineUp + num.Height

	// Denominator: top edge is fracDenomShiftDown × sizePt below the
	// parent baseline. Bottom edge is that plus depth + height of the
	// denominator (height = above its own baseline, depth below it).
	denTopDown := fracDenomShiftDown * sizePt
	denTotalDescent := denTopDown + den.Height + den.Depth

	parent := Box{
		Width:  barWidth,
		Height: numTotalAscent,
		Depth:  denTotalDescent,
	}

	// Numerator placement: centred over the bar.
	num.X = (barWidth - num.Width) / 2
	num.Y = parent.Height - numTotalAscent
	parent.Children = append(parent.Children, num)

	// Denominator placement: centred under the bar. Its top edge sits
	// at parent.Height + denTopDown (Y grows downward from parent
	// top, parent baseline is at parent.Height).
	den.X = (barWidth - den.Width) / 2
	den.Y = parent.Height + denTopDown
	parent.Children = append(parent.Children, den)

	// Fraction bar: vertically centred on the math axis. The bar's
	// midline sits axis above the baseline; convert to top-relative.
	parent.Rules = append(parent.Rules, Rule{
		X:         0,
		Y:         parent.Height - axis - barThickness/2,
		Width:     barWidth,
		Thickness: barThickness,
	})

	return parent
}
