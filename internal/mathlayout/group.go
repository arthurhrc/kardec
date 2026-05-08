package mathlayout

// Spacing constants, expressed as a fraction of the current font size.
// They are intentionally smaller than TeX's classical \thinmuskip /
// \medmuskip / \thickmuskip ratios so the resulting text-mode formulae
// look at home in a paragraph; refinement against the TeXbook tables is
// a follow-up once the typography track ships real metrics.
const (
	// spaceRelOp is inserted between a relation/operator atom and a
	// neighbouring atom (a x b style). 0.16 of the font size matches a
	// "thin" muskip in textstyle.
	spaceRelOp = 0.16

	// spaceBinOp is inserted around binary operators like + and -. It
	// is slightly wider than spaceRelOp so a + b looks visually open
	// without the extra padding a relation would carry.
	spaceBinOp = 0.22
)

// layoutGroup concatenates a Group's children left-to-right, inserting
// thin spacing between adjacent atoms when one of them is a binary or
// relational operator. Each child is laid out independently, then placed
// as a child Box at the running x-cursor; the parent's Width is the sum
// of child widths plus the inter-child spacing, and the parent's Height
// and Depth are the maxima across children so the bounding box is tight.
func layoutGroup(g Group, font Font, sizePt float64, display bool) Box {
	children := g.Children()
	if len(children) == 0 {
		return Box{}
	}

	boxes := make([]Box, len(children))
	for i, c := range children {
		boxes[i] = layoutNode(c, font, sizePt, display)
	}

	var parent Box
	x := 0.0
	for i, child := range boxes {
		if i > 0 {
			x += spacingBetween(children[i-1], children[i], sizePt)
		}
		child.X = x
		// Align all child baselines: position the child so its top edge
		// sits at (parent.maxAscent - child.Height). We discover the
		// parent's max ascent in a second pass below; for now record
		// the per-child ascent in Y and rewrite it once known.
		child.Y = child.Height
		parent.Children = append(parent.Children, child)
		x += child.Width
		if child.Height > parent.Height {
			parent.Height = child.Height
		}
		if child.Depth > parent.Depth {
			parent.Depth = child.Depth
		}
	}
	for i := range parent.Children {
		ascent := parent.Children[i].Y
		parent.Children[i].Y = parent.Height - ascent
	}
	parent.Width = x
	return parent
}

// spacingBetween returns the gap inserted between two adjacent siblings
// in a Group. The classification is intentionally simple: a binary
// operator (+, -) on either side gets the wider spaceBinOp; any other
// op or relation gets spaceRelOp; two ordinary atoms touch directly.
//
// More nuanced TeXbook-style classification (Bin, Rel, Op, Punct, ...)
// is deferred until the parser surfaces the additional categories;
// today the engine only sees KindOp without a sub-category.
func spacingBetween(left, right Expr, sizePt float64) float64 {
	if left == nil || right == nil {
		return 0
	}
	if isBinaryOp(left) || isBinaryOp(right) {
		return spaceBinOp * sizePt
	}
	if left.Kind() == KindOp || right.Kind() == KindOp {
		return spaceRelOp * sizePt
	}
	return 0
}

// isBinaryOp reports whether e is an Op carrying a + or - symbol. These
// are the two operators where TeX widens the surrounding spacing; other
// op-like glyphs (=, <, ...) keep the thinner relation spacing.
func isBinaryOp(e Expr) bool {
	if e.Kind() != KindOp {
		return false
	}
	op, ok := e.(Op)
	if !ok {
		return false
	}
	switch op.Symbol() {
	case "+", "-":
		return true
	}
	return false
}
