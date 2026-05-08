// Package math models a small, well-defined subset of LaTeX-style math source
// as a typed expression tree. The package owns three concerns:
//
//   - The Expr AST that downstream layout, glyph and rendering tracks consume.
//   - A hand-rolled lexer and recursive-descent parser that produces an Expr
//     tree from a UTF-8 source string.
//   - A canonical symbol table that maps known LaTeX command names to their
//     Unicode runes and a coarse typographic category.
//
// The package is intentionally framework-agnostic: it has no dependency on
// the rest of kardec, allocates no global state, and never performs IO. Box
// metrics, glyph positioning and font lookup are out of scope and live in
// sibling packages (internal/math/layout, internal/math/typography).
//
// Stability: the exported types in this package form the contract that the
// math layout track depends on. Adding new kinds is backward-compatible;
// renaming or removing existing ones is not.
package math
