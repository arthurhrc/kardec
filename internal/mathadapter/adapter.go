// Package mathadapter wraps the parser's concrete AST nodes
// (internal/math) into the interface shape the layout engine
// (internal/mathlayout) expects, and bridges typography.MathFont onto
// mathlayout.Font. The two upstream tracks evolved independently with
// equivalent but non-identical Go shapes; this package is the seam.
package mathadapter

import (
	mathast "github.com/arthurhrc/kardec/internal/math"
	"github.com/arthurhrc/kardec/internal/mathlayout"
	"github.com/arthurhrc/kardec/internal/typography"
)

// WrapExpr converts a parser AST node into the matching mathlayout
// Expr interface. Children are wrapped recursively so the engine never
// touches the underlying parser types directly.
//
// A nil input returns nil so optional fields (SubSup.Sub, BigOp.Lower
// and friends) round-trip cleanly.
func WrapExpr(e mathast.Expr) mathlayout.Expr {
	if e == nil {
		return nil
	}
	switch v := e.(type) {
	case mathast.Atom:
		return atomAdapter{symbol: v.Symbol}
	case mathast.Op:
		return opAdapter{symbol: v.Symbol}
	case mathast.Number:
		return numberAdapter{value: v.Value}
	case mathast.Identifier:
		return identifierAdapter{name: v.Name}
	case mathast.Group:
		children := make([]mathlayout.Expr, 0, len(v.Children))
		for _, c := range v.Children {
			children = append(children, WrapExpr(c))
		}
		return groupAdapter{children: children}
	case mathast.Frac:
		return fracAdapter{
			numerator:   WrapExpr(v.Numerator),
			denominator: WrapExpr(v.Denominator),
		}
	case mathast.Sqrt:
		return sqrtAdapter{body: WrapExpr(v.Body)}
	case mathast.NthRoot:
		return nthRootAdapter{
			index: WrapExpr(v.Index),
			body:  WrapExpr(v.Body),
		}
	case mathast.SubSup:
		return subSupAdapter{
			base: WrapExpr(v.Base),
			sub:  WrapExpr(v.Sub),
			sup:  WrapExpr(v.Sup),
		}
	case mathast.BigOp:
		return bigOpAdapter{
			symbol: v.Symbol,
			lower:  WrapExpr(v.Lower),
			upper:  WrapExpr(v.Upper),
			body:   WrapExpr(v.Body),
		}
	default:
		// Unknown future node kinds are surfaced as an empty atom so
		// layout still completes without panicking.
		return atomAdapter{symbol: ""}
	}
}

type atomAdapter struct{ symbol string }

func (atomAdapter) Kind() mathlayout.ExprKind { return mathlayout.KindAtom }
func (a atomAdapter) Symbol() string          { return a.symbol }

type opAdapter struct{ symbol string }

func (opAdapter) Kind() mathlayout.ExprKind { return mathlayout.KindOp }
func (a opAdapter) Symbol() string          { return a.symbol }

type numberAdapter struct{ value string }

func (numberAdapter) Kind() mathlayout.ExprKind { return mathlayout.KindNumber }
func (a numberAdapter) Value() string           { return a.value }

type identifierAdapter struct{ name string }

func (identifierAdapter) Kind() mathlayout.ExprKind { return mathlayout.KindIdentifier }
func (a identifierAdapter) Name() string            { return a.name }

type groupAdapter struct{ children []mathlayout.Expr }

func (groupAdapter) Kind() mathlayout.ExprKind { return mathlayout.KindGroup }
func (a groupAdapter) Children() []mathlayout.Expr {
	return a.children
}

type fracAdapter struct {
	numerator, denominator mathlayout.Expr
}

func (fracAdapter) Kind() mathlayout.ExprKind { return mathlayout.KindFrac }
func (a fracAdapter) Numerator() mathlayout.Expr {
	return a.numerator
}
func (a fracAdapter) Denominator() mathlayout.Expr { return a.denominator }

type sqrtAdapter struct{ body mathlayout.Expr }

func (sqrtAdapter) Kind() mathlayout.ExprKind { return mathlayout.KindSqrt }
func (a sqrtAdapter) Body() mathlayout.Expr   { return a.body }

type nthRootAdapter struct {
	index, body mathlayout.Expr
}

func (nthRootAdapter) Kind() mathlayout.ExprKind { return mathlayout.KindNthRoot }
func (a nthRootAdapter) Index() mathlayout.Expr  { return a.index }
func (a nthRootAdapter) Body() mathlayout.Expr   { return a.body }

type subSupAdapter struct {
	base, sub, sup mathlayout.Expr
}

func (subSupAdapter) Kind() mathlayout.ExprKind { return mathlayout.KindSubSup }
func (a subSupAdapter) Base() mathlayout.Expr   { return a.base }
func (a subSupAdapter) Sub() mathlayout.Expr    { return a.sub }
func (a subSupAdapter) Sup() mathlayout.Expr    { return a.sup }

type bigOpAdapter struct {
	symbol            string
	lower, upper, body mathlayout.Expr
}

func (bigOpAdapter) Kind() mathlayout.ExprKind { return mathlayout.KindBigOp }
func (a bigOpAdapter) Symbol() string          { return a.symbol }
func (a bigOpAdapter) Lower() mathlayout.Expr  { return a.lower }
func (a bigOpAdapter) Upper() mathlayout.Expr  { return a.upper }
func (a bigOpAdapter) Body() mathlayout.Expr   { return a.body }

// WrapFont converts a typography.MathFont into the mathlayout.Font
// shape. The two interfaces differ only in their Glyph type — the
// adapter swaps unexported metric fields by re-asking the source font
// at call time, which today re-queries the underlying *canvas.FontFace
// per call.
func WrapFont(f typography.MathFont) mathlayout.Font {
	return fontAdapter{inner: f}
}

type fontAdapter struct{ inner typography.MathFont }

func (a fontAdapter) GlyphFor(symbol string) (mathlayout.Glyph, bool) {
	g, ok := a.inner.GlyphFor(symbol)
	if !ok {
		return mathlayout.Glyph{}, false
	}
	return mathlayout.Glyph{Rune: g.Rune}, true
}

func (a fontAdapter) Measure(g mathlayout.Glyph, sizePt float64) float64 {
	return a.inner.Measure(typography.MathGlyph{Rune: g.Rune}, sizePt)
}

func (a fontAdapter) AscentDescent(g mathlayout.Glyph, sizePt float64) (float64, float64) {
	return a.inner.AscentDescent(typography.MathGlyph{Rune: g.Rune}, sizePt)
}
