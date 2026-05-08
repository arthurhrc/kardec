// Package mathlayout positions a math expression tree into a tree of
// boxes ready for the renderer. It owns vertical metrics, sub/superscript
// stacking, fraction bars, radicals and big-operator limits; it does not
// own LaTeX parsing (that lives in internal/math) nor real font metrics
// (that lives in internal/typography). Both dependencies are expressed as
// local interfaces so this package compiles in isolation.
//
// The contract with the parser track is the Expr interface defined here;
// the contract with the typography track is the Font interface defined in
// font.go. The contract with the renderer is the Box tree returned by
// Layout. Coordinates use a top-left origin within each Box, in PDF
// points, and are reported relative to the parent Box; the renderer
// composes absolute coordinates by accumulating offsets along the path.
package mathlayout

// Expr is the minimum surface this layout package needs from a math AST.
// Concrete AST node types from internal/math implement this interface
// (directly or via thin adapter types) so the integration step does not
// need a wire format between the two packages.
type Expr interface {
	// Kind reports the node's discriminator. Layout dispatches on Kind
	// rather than performing type switches against concrete types so the
	// parser is free to evolve its representation without touching the
	// layout engine.
	Kind() ExprKind
}

// ExprKind is the discriminator returned by Expr.Kind. New kinds are
// appended; existing values are stable so binary representations of the
// AST (should they ever exist) survive parser refactors.
type ExprKind uint8

// Recognised expression kinds. The names mirror the AST nodes the parser
// track ships.
const (
	// KindAtom is a single, non-numeric, non-letter symbol such as a
	// punctuation glyph or a delimiter. Atoms render as one glyph at the
	// baseline.
	KindAtom ExprKind = iota + 1

	// KindOp is a binary or relational operator (+, -, =, ...). Layout
	// inserts thin spacing around operators when surrounded by atoms.
	KindOp

	// KindNumber is a literal numeric token. Numbers render the same as
	// atoms but are surfaced as their own kind so future work can apply
	// digit-specific kerning without re-typing the parser.
	KindNumber

	// KindIdentifier is a single italic letter or named identifier. It
	// renders as one glyph; multi-letter identifiers are produced by the
	// parser as a Group.
	KindIdentifier

	// KindGroup is an ordered sequence of child expressions concatenated
	// horizontally. Braces in the source disappear once parsed; the group
	// records the resulting list.
	KindGroup

	// KindFrac is a built-up fraction: numerator stacked above
	// denominator with a horizontal rule between them.
	KindFrac

	// KindSqrt is a square root: a radical glyph followed by an overline
	// that spans the body.
	KindSqrt

	// KindNthRoot is a generalised root with an explicit small index
	// drawn at the top-left of the radical glyph.
	KindNthRoot

	// KindSubSup carries optional subscript and superscript expressions
	// attached to a base. Either may be nil but not both, and the
	// combined form stacks scripts vertically next to the base.
	KindSubSup

	// KindBigOp is a large operator (sum, integral, product, ...). In
	// inline mode its limits attach as ordinary scripts; in display mode
	// they centre above and below the operator.
	KindBigOp
)

// Atom is a single non-numeric, non-letter symbol expression.
type Atom interface {
	Expr
	// Symbol returns the printable representation, typically a single
	// rune. Layout passes the value through to Font.GlyphFor unchanged.
	Symbol() string
}

// Op is a binary or relational operator expression.
type Op interface {
	Expr
	// Symbol returns the operator glyph (for example "+" or "=").
	Symbol() string
}

// Number is a numeric literal expression.
type Number interface {
	Expr
	// Value returns the literal source text of the number ("3.14",
	// "42"). Layout treats it as an opaque string and looks each rune up
	// individually.
	Value() string
}

// Identifier is a letter or named identifier expression.
type Identifier interface {
	Expr
	// Name returns the identifier's printable name. Single-letter
	// identifiers are common ("x"); multi-letter names are also valid
	// and render as a sequence of glyphs.
	Name() string
}

// Group is an ordered horizontal list of child expressions.
type Group interface {
	Expr
	// Children returns the ordered list. The slice is borrowed by the
	// caller and must not be modified.
	Children() []Expr
}

// Frac is a built-up fraction expression.
type Frac interface {
	Expr
	// Numerator returns the expression typeset above the bar.
	Numerator() Expr
	// Denominator returns the expression typeset below the bar.
	Denominator() Expr
}

// Sqrt is a square-root expression.
type Sqrt interface {
	Expr
	// Body returns the expression under the radical.
	Body() Expr
}

// NthRoot is a generalised root with an explicit index.
type NthRoot interface {
	Expr
	// Index returns the small index drawn at the top-left of the radical.
	Index() Expr
	// Body returns the expression under the radical.
	Body() Expr
}

// SubSup carries optional sub- and super-scripts attached to a base.
type SubSup interface {
	Expr
	// Base returns the expression the scripts attach to. It is never nil.
	Base() Expr
	// Sub returns the subscript expression, or nil if absent.
	Sub() Expr
	// Sup returns the superscript expression, or nil if absent.
	Sup() Expr
}

// BigOp is a large operator (sum, integral, product, ...) carrying
// optional lower and upper limits and an optional body that follows the
// operator on the same line.
type BigOp interface {
	Expr
	// Symbol returns the operator glyph (typically the LaTeX symbol
	// resolved to its Unicode code point).
	Symbol() string
	// Lower returns the lower-limit expression, or nil if absent.
	Lower() Expr
	// Upper returns the upper-limit expression, or nil if absent.
	Upper() Expr
	// Body returns the expression that follows the operator (the
	// summand, integrand, ...) or nil if absent.
	Body() Expr
}
