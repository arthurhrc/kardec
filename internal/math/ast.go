package math

// ExprKind tags concrete Expr implementations so downstream code can dispatch
// without a type switch when that is more convenient. The zero value is
// reserved and never returned by a well-formed node.
type ExprKind uint8

// Concrete kinds returned by exprKind. The list is closed: adding a new kind
// requires updating downstream layout code.
const (
	// KindAtom marks a literal symbol (e.g. a greek letter or named symbol).
	KindAtom ExprKind = iota + 1
	// KindOp marks a relational or binary operator like + - = <.
	KindOp
	// KindNumber marks a digit run, possibly with a single decimal point.
	KindNumber
	// KindIdentifier marks a single-letter variable.
	KindIdentifier
	// KindGroup marks a brace-delimited sequence of children.
	KindGroup
	// KindFrac marks a numerator-over-denominator construct.
	KindFrac
	// KindSqrt marks a square root with a single body.
	KindSqrt
	// KindNthRoot marks a root with an explicit index (\sqrt[n]{...}).
	KindNthRoot
	// KindSubSup marks a base carrying an optional subscript and/or superscript.
	KindSubSup
	// KindBigOp marks a large operator (\sum, \int, \prod) with optional
	// lower and upper limits and an optional body.
	KindBigOp
)

// Expr is the sealed interface implemented by every node in the math AST.
// The single sentinel method is unexported so external packages cannot add
// new kinds; downstream consumers must assert against the concrete types
// exported by this package.
type Expr interface {
	// exprKind returns the discriminator for this node. The method is
	// unexported by design; layout code switches on the public ExprKind
	// returned by Kind below.
	exprKind() ExprKind
}

// Kind returns the ExprKind discriminator for e. It is the public bridge to
// the unexported sentinel method on Expr and exists so downstream packages
// can switch on ExprKind without performing a type assertion first.
func Kind(e Expr) ExprKind {
	if e == nil {
		return 0
	}
	return e.exprKind()
}

// Atom is a literal symbol carried verbatim through the pipeline. The parser
// emits an Atom for every greek letter, named symbol, and unknown command;
// the layout track resolves Symbol against the typography glyph table.
type Atom struct {
	// Symbol is the canonical command form including the leading backslash
	// (e.g. "\\alpha", "\\infty"). Unknown commands are preserved verbatim
	// so renderers can fall back to plain-text output.
	Symbol string
}

func (Atom) exprKind() ExprKind { return KindAtom }

// Op is a relational or binary operator such as +, -, =, <, >, * or /. The
// Symbol is the literal source character; conversion to a typographic glyph
// (e.g. an en-dash minus) happens later in the pipeline.
type Op struct {
	// Symbol is the operator's source character.
	Symbol string
}

func (Op) exprKind() ExprKind { return KindOp }

// Number is a contiguous digit run, optionally containing one decimal point.
// The parser preserves the source representation as-is; no numeric parsing
// happens here so trailing zeros and locale-style separators stay intact.
type Number struct {
	// Value is the source representation of the number.
	Value string
}

func (Number) exprKind() ExprKind { return KindNumber }

// Identifier is a single-letter variable. Multi-letter variable names (such
// as "sin") are not produced by the parser; LaTeX models those as commands.
type Identifier struct {
	// Name is the single-rune identifier.
	Name string
}

func (Identifier) exprKind() ExprKind { return KindIdentifier }

// Group is a brace-delimited sequence of children. It exists primarily so
// the layout engine can treat {ab} differently from "a b" when it matters.
type Group struct {
	// Children are the in-order children of the group. May be empty.
	Children []Expr
}

func (Group) exprKind() ExprKind { return KindGroup }

// Frac is a fraction with a numerator and a denominator. Both \frac and
// \dfrac collapse to this kind; the display-style preference is recovered
// later from the surrounding context, not from the AST.
type Frac struct {
	// Numerator is the expression placed above the fraction bar.
	Numerator Expr
	// Denominator is the expression placed below the fraction bar.
	Denominator Expr
}

func (Frac) exprKind() ExprKind { return KindFrac }

// Sqrt is a square root containing a single body expression.
type Sqrt struct {
	// Body is the radicand placed under the radical sign.
	Body Expr
}

func (Sqrt) exprKind() ExprKind { return KindSqrt }

// NthRoot is a root with an explicit Index (the source's optional argument
// in \sqrt[n]{x}).
type NthRoot struct {
	// Index is the small expression in the radical's notch.
	Index Expr
	// Body is the radicand placed under the radical sign.
	Body Expr
}

func (NthRoot) exprKind() ExprKind { return KindNthRoot }

// SubSup attaches a subscript and/or a superscript to a base expression.
// Either Sub or Sup may be nil; both being nil is illegal output from the
// parser but layout code must still treat nil fields as valid input.
type SubSup struct {
	// Base is the expression the scripts attach to.
	Base Expr
	// Sub is the subscript expression, or nil when absent.
	Sub Expr
	// Sup is the superscript expression, or nil when absent.
	Sup Expr
}

func (SubSup) exprKind() ExprKind { return KindSubSup }

// BigOp is a large operator (sum, integral, product) carrying optional
// lower and upper limits plus an optional body. The body is the integrand
// or summand; layout decides whether limits sit beside or above the symbol
// based on display style and the operator's typographic class.
type BigOp struct {
	// Symbol is the canonical command form including the leading backslash
	// (e.g. "\\sum", "\\int", "\\prod").
	Symbol string
	// Lower is the limit attached to the operator's "_" position, or nil.
	Lower Expr
	// Upper is the limit attached to the operator's "^" position, or nil.
	Upper Expr
	// Body is the integrand or summand expression, or nil.
	Body Expr
}

func (BigOp) exprKind() ExprKind { return KindBigOp }
