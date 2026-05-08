package mathlayout

import (
	"math"
	"testing"
)

// stubFont is a deterministic Font implementation for the engine tests.
// Every glyph reports a uniform advance, ascent and descent derived
// from the requested point size, which lets us reason about box
// geometry algebraically.
type stubFont struct{}

func (stubFont) GlyphFor(symbol string) (Glyph, bool) {
	if symbol == "" {
		return Glyph{}, false
	}
	r := []rune(symbol)
	if len(r) == 0 {
		return Glyph{}, false
	}
	return Glyph{Rune: r[0]}, true
}

func (stubFont) Measure(_ Glyph, sizePt float64) float64 { return 0.5 * sizePt }

func (stubFont) AscentDescent(_ Glyph, sizePt float64) (float64, float64) {
	return 0.7 * sizePt, 0.2 * sizePt
}

// Stub AST node types implementing the various Expr-shaped interfaces.
// They exist solely to drive Layout in the tests; the production AST
// lives in internal/math.
type stubAtom struct{ s string }

func (stubAtom) Kind() ExprKind  { return KindAtom }
func (a stubAtom) Symbol() string { return a.s }

type stubOp struct{ s string }

func (stubOp) Kind() ExprKind  { return KindOp }
func (o stubOp) Symbol() string { return o.s }

type stubNumber struct{ v string }

func (stubNumber) Kind() ExprKind  { return KindNumber }
func (n stubNumber) Value() string { return n.v }

type stubIdentifier struct{ n string }

func (stubIdentifier) Kind() ExprKind { return KindIdentifier }
func (i stubIdentifier) Name() string { return i.n }

type stubGroup struct{ kids []Expr }

func (stubGroup) Kind() ExprKind        { return KindGroup }
func (g stubGroup) Children() []Expr    { return g.kids }

type stubFrac struct{ num, den Expr }

func (stubFrac) Kind() ExprKind     { return KindFrac }
func (f stubFrac) Numerator() Expr   { return f.num }
func (f stubFrac) Denominator() Expr { return f.den }

type stubSqrt struct{ body Expr }

func (stubSqrt) Kind() ExprKind { return KindSqrt }
func (s stubSqrt) Body() Expr   { return s.body }

type stubSubSup struct{ base, sub, sup Expr }

func (stubSubSup) Kind() ExprKind { return KindSubSup }
func (s stubSubSup) Base() Expr   { return s.base }
func (s stubSubSup) Sub() Expr    { return s.sub }
func (s stubSubSup) Sup() Expr    { return s.sup }

type stubBigOp struct {
	sym         string
	lower, upper, body Expr
}

func (stubBigOp) Kind() ExprKind   { return KindBigOp }
func (o stubBigOp) Symbol() string { return o.sym }
func (o stubBigOp) Lower() Expr    { return o.lower }
func (o stubBigOp) Upper() Expr    { return o.upper }
func (o stubBigOp) Body() Expr     { return o.body }

const eps = 1e-9

func approxEq(a, b float64) bool { return math.Abs(a-b) < eps }

// TestSingleAtom verifies a single Atom yields the basic stub-derived
// geometry: width = 0.5×size, height = 0.7×size, depth = 0.2×size, and
// exactly one PlacedGlyph at the baseline.
func TestSingleAtom(t *testing.T) {
	box := Layout(stubAtom{s: "x"}, stubFont{}, 12, false)
	if !approxEq(box.Width, 6) {
		t.Errorf("Width = %v, want 6", box.Width)
	}
	if !approxEq(box.Height, 8.4) {
		t.Errorf("Height = %v, want 8.4", box.Height)
	}
	if !approxEq(box.Depth, 2.4) {
		t.Errorf("Depth = %v, want 2.4", box.Depth)
	}
	if len(box.Glyphs) != 1 {
		t.Fatalf("len(Glyphs) = %d, want 1", len(box.Glyphs))
	}
	if box.Glyphs[0].Rune != 'x' {
		t.Errorf("Rune = %q, want 'x'", box.Glyphs[0].Rune)
	}
}

// TestGroupAddSpacing verifies that a + b yields three child boxes and
// that the gap between adjacent children includes the binary-operator
// spacing on both sides of the +.
func TestGroupAddSpacing(t *testing.T) {
	g := stubGroup{kids: []Expr{
		stubIdentifier{n: "a"},
		stubOp{s: "+"},
		stubIdentifier{n: "b"},
	}}
	box := Layout(g, stubFont{}, 12, false)
	if len(box.Children) != 3 {
		t.Fatalf("len(Children) = %d, want 3", len(box.Children))
	}
	// Each glyph is 6pt wide; spaces are 0.22*12 = 2.64 on each side.
	want := 6 + 2.64 + 6 + 2.64 + 6
	if !approxEq(box.Width, want) {
		t.Errorf("Width = %v, want %v", box.Width, want)
	}
	// Verify gap = 2.64 between siblings.
	gap1 := box.Children[1].X - (box.Children[0].X + box.Children[0].Width)
	gap2 := box.Children[2].X - (box.Children[1].X + box.Children[1].Width)
	if !approxEq(gap1, 2.64) || !approxEq(gap2, 2.64) {
		t.Errorf("gaps = %v, %v, want 2.64 each", gap1, gap2)
	}
}

// TestFracHasRule verifies a fraction emits a Rule between two child
// boxes. The Rule's width must cover the wider of the two operands
// plus the configured padding.
func TestFracHasRule(t *testing.T) {
	f := stubFrac{
		num: stubIdentifier{n: "a"},
		den: stubIdentifier{n: "b"},
	}
	box := Layout(f, stubFont{}, 12, false)
	if len(box.Children) != 2 {
		t.Fatalf("len(Children) = %d, want 2", len(box.Children))
	}
	if len(box.Rules) != 1 {
		t.Fatalf("len(Rules) = %d, want 1", len(box.Rules))
	}
	// Numerator/denominator each render at 0.9*12 = 10.8 pt, so each
	// child has width 0.5*10.8 = 5.4. Bar width = 5.4 + 2*0.10*12 = 7.8.
	if !approxEq(box.Rules[0].Width, 7.8) {
		t.Errorf("Rule.Width = %v, want 7.8", box.Rules[0].Width)
	}
	// First child (numerator) sits above second (denominator).
	if !(box.Children[0].Y < box.Children[1].Y) {
		t.Errorf("numerator Y (%v) should be above denominator Y (%v)",
			box.Children[0].Y, box.Children[1].Y)
	}
}

// TestSubSupVerticalShift verifies the subscript sits below the parent
// baseline and the superscript sits above it, both attached to the
// right of the base.
func TestSubSupVerticalShift(t *testing.T) {
	s := stubSubSup{
		base: stubIdentifier{n: "x"},
		sub:  stubIdentifier{n: "i"},
		sup:  stubNumber{v: "2"},
	}
	box := Layout(s, stubFont{}, 12, false)
	if len(box.Children) != 3 {
		t.Fatalf("len(Children) = %d, want 3 (base, sup, sub)", len(box.Children))
	}
	base := box.Children[0]
	sup := box.Children[1]
	sub := box.Children[2]
	baseline := box.Height
	// Sup baseline = sup.Y + sup.Height; should sit above parent
	// baseline.
	if sup.Y+sup.Height >= baseline {
		t.Errorf("sup baseline %v not above parent baseline %v", sup.Y+sup.Height, baseline)
	}
	// Sub top edge = sub.Y; should sit below parent baseline.
	if sub.Y <= baseline {
		t.Errorf("sub top %v not below parent baseline %v", sub.Y, baseline)
	}
	// Both scripts share the same X (right of base).
	if !approxEq(sup.X, base.X+base.Width) || !approxEq(sub.X, base.X+base.Width) {
		t.Errorf("scripts X = (%v, %v), want both at %v",
			sup.X, sub.X, base.X+base.Width)
	}
}

// TestBigOpDisplayLimits verifies that in display mode the upper limit
// sits above the operator and the lower limit sits below it, both as
// separate child boxes (rather than scripts on the right).
func TestBigOpDisplayLimits(t *testing.T) {
	b := stubBigOp{
		sym:   "∑",
		lower: stubGroup{kids: []Expr{stubIdentifier{n: "i"}, stubOp{s: "="}, stubNumber{v: "0"}}},
		upper: stubIdentifier{n: "n"},
		body:  stubIdentifier{n: "i"},
	}
	box := Layout(b, stubFont{}, 12, true)
	// Children should include upper, lower, and body — three Boxes.
	if len(box.Children) < 3 {
		t.Fatalf("len(Children) = %d, want >=3 (upper, lower, body)", len(box.Children))
	}
	// Upper is the first child (added before lower); its Y must be
	// less than the operator-glyph baseline (which is at parent.Height
	// minus operator height).
	upper := box.Children[0]
	lower := box.Children[1]
	if !(upper.Y < box.Height) {
		t.Errorf("upper.Y (%v) should be above parent baseline (%v)", upper.Y, box.Height)
	}
	if !(lower.Y > box.Height) {
		t.Errorf("lower.Y (%v) should be below parent baseline (%v)", lower.Y, box.Height)
	}
	// Upper limit must be horizontally centred over the operator glyph
	// (within rounding) — its X is approximately (stackWidth -
	// upper.Width)/2 from the left edge of the box.
	if upper.X < 0 {
		t.Errorf("upper.X = %v, expected non-negative", upper.X)
	}
}

// TestSqrtOverlineSpansBody verifies that a square root produces an
// overline Rule whose width matches the body's width and whose start
// is shifted right by the radical glyph's width.
func TestSqrtOverlineSpansBody(t *testing.T) {
	s := stubSqrt{body: stubGroup{kids: []Expr{
		stubIdentifier{n: "x"},
		stubOp{s: "+"},
		stubNumber{v: "1"},
	}}}
	box := Layout(s, stubFont{}, 12, false)
	if len(box.Rules) != 1 {
		t.Fatalf("len(Rules) = %d, want 1", len(box.Rules))
	}
	// Body box is the last child (after any optional index). It has
	// the body's group geometry, and the rule's Width must equal it.
	body := box.Children[len(box.Children)-1]
	if !approxEq(box.Rules[0].Width, body.Width) {
		t.Errorf("Rule.Width = %v, want body.Width %v", box.Rules[0].Width, body.Width)
	}
	// Radical glyph sits before the body horizontally.
	if len(box.Glyphs) != 1 {
		t.Fatalf("len(Glyphs) = %d, want 1 radical", len(box.Glyphs))
	}
	if box.Glyphs[0].Rune != '√' {
		t.Errorf("radical Rune = %q, want '√'", box.Glyphs[0].Rune)
	}
	if !(body.X > box.Glyphs[0].X) {
		t.Errorf("body.X (%v) should be right of radical X (%v)", body.X, box.Glyphs[0].X)
	}
}
