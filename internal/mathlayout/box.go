package mathlayout

// Box is a positioned, sized math sub-expression. Layout produces a
// single root Box containing nested children. The renderer walks the
// tree in document order and emits glyphs / rules at the absolute
// coordinates derived by adding the box's offsets along the path.
//
// All coordinates are in PDF points with a top-left origin within the
// parent Box. Width is the horizontal extent; Height is the rise above
// the box's baseline; Depth is the drop below the baseline reported as
// a positive number, mirroring TeX's convention.
type Box struct {
	// X and Y are the top-left offsets relative to the parent Box.
	// The root Box uses (0, 0) — callers translate it into page
	// coordinates when embedding the formula in a paragraph.
	X, Y float64

	// Width is the horizontal extent of the box.
	Width float64

	// Height is the distance from the baseline up to the top edge of
	// the box. It is always non-negative.
	Height float64

	// Depth is the distance from the baseline down to the bottom edge
	// of the box, reported as a positive number. Combined with Height
	// it yields the box's total vertical extent.
	Depth float64

	// Glyphs holds the directly-emitted glyphs for this box. Each
	// PlacedGlyph's coordinates are local to this Box.
	Glyphs []PlacedGlyph

	// Rules holds the horizontal lines drawn by this box, used for
	// fraction bars and the overline of square roots.
	Rules []Rule

	// Children carries nested boxes whose coordinates are local to this
	// Box. Layout never mutates a parent's Glyphs/Rules to express a
	// child's content; nested structure is preserved so the renderer
	// (and tests) can reason about the tree.
	Children []Box
}

// PlacedGlyph is one renderable glyph at a fixed position within its
// containing Box.
type PlacedGlyph struct {
	// X and Y are the top-left offsets within the parent Box, in PDF
	// points.
	X, Y float64

	// Rune is the Unicode code point the renderer emits.
	Rune rune

	// SizePt is the point size at which the glyph is drawn. Sub- and
	// super-script glyphs carry their reduced size here so the renderer
	// can pick the right font scale without extra context.
	SizePt float64
}

// Rule is a horizontal line drawn at a fixed offset within its
// containing Box. Vertical rules are not currently produced; if matrix
// or array support is added later this struct will gain an Orientation
// field rather than a parallel VerticalRule type.
type Rule struct {
	// X and Y are the top-left offsets within the parent Box, in PDF
	// points. The rule extends right by Width and down by Thickness.
	X, Y float64

	// Width is the horizontal extent of the rule.
	Width float64

	// Thickness is the vertical extent of the rule. Layout uses a
	// fraction of the current font size for fraction bars and the
	// square-root overline.
	Thickness float64
}
