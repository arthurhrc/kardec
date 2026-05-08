package math

import (
	"fmt"
)

// Parse converts LaTeX-style math source into an Expr tree. The grammar is a
// pragmatic subset documented in package math; unknown commands degrade to
// Atom nodes carrying the literal source so callers can fall back to plain
// text rendering. An error is returned only for structural problems such as
// unbalanced braces or a missing fraction argument.
//
// The empty source returns a Group with no children and a nil error.
func Parse(src string) (Expr, error) {
	p := &parser{lex: newLexer(src)}
	p.advance() // prime cur
	exprs, err := p.parseSequence(tokEOF)
	if err != nil {
		return nil, err
	}
	if len(exprs) == 1 {
		return exprs[0], nil
	}
	return Group{Children: exprs}, nil
}

// parser is the recursive-descent driver. It carries one-token lookahead in
// cur; advance() refreshes it from the underlying lexer.
type parser struct {
	lex *lexer
	cur token
}

// advance refreshes cur with the next token from the lexer.
func (p *parser) advance() {
	p.cur = p.lex.next()
}

// errorf builds a descriptive error tagged with the current byte offset.
func (p *parser) errorf(offset int, format string, args ...any) error {
	return fmt.Errorf("math: parse error at offset %d: %s", offset, fmt.Sprintf(format, args...))
}

// parseSequence consumes expressions until it sees a terminator token. The
// terminator itself is left in cur for the caller to consume.
func (p *parser) parseSequence(terminator tokenKind) ([]Expr, error) {
	var out []Expr
	for p.cur.kind != terminator && p.cur.kind != tokEOF {
		e, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		if e == nil {
			// Defensive: parseTerm should always return a node or an error
			// on a non-terminator token.
			return nil, p.errorf(p.cur.offset, "unexpected token %q", p.cur.value)
		}
		out = append(out, e)
	}
	return out, nil
}

// parseTerm parses one logical term: an atom (with optional scripts) or a
// big operator with its limits and body. parseTerm consumes exactly one
// term from the input.
func (p *parser) parseTerm() (Expr, error) {
	base, err := p.parseAtom()
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, nil
	}

	// Big operators absorb their _/^ limits and an optional body before
	// scripts can attach to them.
	if bigSym, ok := bigOpSymbol(base); ok {
		return p.parseBigOp(bigSym)
	}

	return p.parseScripts(base)
}

// bigOpSymbol returns the command symbol of e if e is an Atom referring to
// a big operator command (\sum, \int, \prod). Otherwise it returns "" and
// false.
func bigOpSymbol(e Expr) (string, bool) {
	a, ok := e.(Atom)
	if !ok {
		return "", false
	}
	if !IsBigOpCommand(a.Symbol) {
		return "", false
	}
	return a.Symbol, true
}

// parseBigOp builds a BigOp node. Lower and Upper are picked up from the
// immediate _/^ pair (in either order); Body is the next single term, if
// present and not interrupted by a closing delimiter.
func (p *parser) parseBigOp(symbol string) (Expr, error) {
	node := BigOp{Symbol: symbol}

	// Limits: accept _ and ^ in any order, at most one of each.
	for i := 0; i < 2; i++ {
		switch p.cur.kind {
		case tokUnderscore:
			if node.Lower != nil {
				return nil, p.errorf(p.cur.offset, "duplicate subscript on %s", symbol)
			}
			p.advance()
			arg, err := p.parseScriptArgument()
			if err != nil {
				return nil, err
			}
			node.Lower = arg
		case tokCaret:
			if node.Upper != nil {
				return nil, p.errorf(p.cur.offset, "duplicate superscript on %s", symbol)
			}
			p.advance()
			arg, err := p.parseScriptArgument()
			if err != nil {
				return nil, err
			}
			node.Upper = arg
		default:
			i = 2 // break loop
		}
	}

	// Body: a single following term, if any, parsed without picking up
	// subsequent terms. Stop at structural boundaries.
	if p.canStartTerm() {
		body, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		node.Body = body
	}
	return node, nil
}

// canStartTerm reports whether the current token can begin a new term.
func (p *parser) canStartTerm() bool {
	switch p.cur.kind {
	case tokEOF, tokRBrace, tokRBracket:
		return false
	}
	return true
}

// parseScripts attaches an optional subscript and/or superscript to base.
// _ and ^ can appear in either order; each may appear at most once. When
// neither is present, base is returned as-is.
func (p *parser) parseScripts(base Expr) (Expr, error) {
	var sub, sup Expr
	for {
		switch p.cur.kind {
		case tokUnderscore:
			if sub != nil {
				return nil, p.errorf(p.cur.offset, "duplicate subscript")
			}
			p.advance()
			arg, err := p.parseScriptArgument()
			if err != nil {
				return nil, err
			}
			sub = arg
		case tokCaret:
			if sup != nil {
				return nil, p.errorf(p.cur.offset, "duplicate superscript")
			}
			p.advance()
			arg, err := p.parseScriptArgument()
			if err != nil {
				return nil, err
			}
			sup = arg
		default:
			if sub == nil && sup == nil {
				return base, nil
			}
			return SubSup{Base: base, Sub: sub, Sup: sup}, nil
		}
	}
}

// parseScriptArgument parses the argument of a _ or ^ operator. It is a
// single atom or a brace group; brace groups are flattened so a single-child
// group {n} resolves to the bare child for ergonomic AST shapes.
func (p *parser) parseScriptArgument() (Expr, error) {
	if p.cur.kind == tokEOF {
		return nil, p.errorf(p.cur.offset, "missing script argument")
	}
	if p.cur.kind == tokLBrace {
		openOffset := p.cur.offset
		p.advance()
		children, err := p.parseSequence(tokRBrace)
		if err != nil {
			return nil, err
		}
		if p.cur.kind != tokRBrace {
			return nil, p.errorf(openOffset, "unbalanced '{' in script argument")
		}
		p.advance()
		return flatten(children), nil
	}
	return p.parseAtom()
}

// parseAtom parses a single atomic expression: a number, identifier,
// operator, group, command-driven construct, or unknown command Atom.
// It does NOT consume trailing _/^ scripts; that belongs to parseScripts.
func (p *parser) parseAtom() (Expr, error) {
	switch p.cur.kind {
	case tokEOF:
		return nil, nil
	case tokRBrace, tokRBracket:
		return nil, nil
	case tokLBrace:
		return p.parseGroup()
	case tokNumber:
		t := p.cur
		p.advance()
		return Number{Value: t.value}, nil
	case tokIdent:
		t := p.cur
		p.advance()
		return Identifier{Name: t.value}, nil
	case tokOp:
		t := p.cur
		p.advance()
		return Op{Symbol: t.value}, nil
	case tokCommand:
		return p.parseCommand()
	case tokOther:
		// Pass through as Atom so the source character is preserved.
		t := p.cur
		p.advance()
		return Atom{Symbol: t.value}, nil
	default:
		t := p.cur
		p.advance()
		return nil, p.errorf(t.offset, "unexpected token %q", t.value)
	}
}

// parseGroup parses { ... } into a Group. The opening brace is current.
func (p *parser) parseGroup() (Expr, error) {
	openOffset := p.cur.offset
	p.advance() // consume "{"
	children, err := p.parseSequence(tokRBrace)
	if err != nil {
		return nil, err
	}
	if p.cur.kind != tokRBrace {
		return nil, p.errorf(openOffset, "unbalanced '{' (no matching '}')")
	}
	p.advance() // consume "}"
	if len(children) == 1 {
		return Group{Children: children}, nil
	}
	return Group{Children: children}, nil
}

// parseCommand handles a tokCommand. It dispatches to special-case handlers
// for fractions and roots; known atomic commands resolve via the symbol
// table; unknown commands fall through as Atom nodes.
func (p *parser) parseCommand() (Expr, error) {
	cmd := p.cur.value
	cmdOffset := p.cur.offset
	switch cmd {
	case "\\frac", "\\dfrac":
		p.advance()
		return p.parseFrac(cmd, cmdOffset)
	case "\\sqrt":
		p.advance()
		return p.parseSqrt(cmdOffset)
	}
	// Generic command: emit Atom regardless of whether it is in the symbol
	// table. The layout track resolves rendering via LookupSymbol; unknown
	// commands fall through as plain-text fallback.
	p.advance()
	return Atom{Symbol: cmd}, nil
}

// parseFrac parses \frac{num}{den} or \dfrac{num}{den}. The leading command
// has already been consumed by the caller.
func (p *parser) parseFrac(cmd string, offset int) (Expr, error) {
	num, err := p.parseRequiredGroup(cmd, offset, "numerator")
	if err != nil {
		return nil, err
	}
	den, err := p.parseRequiredGroup(cmd, offset, "denominator")
	if err != nil {
		return nil, err
	}
	return Frac{Numerator: num, Denominator: den}, nil
}

// parseSqrt parses \sqrt{body} or \sqrt[index]{body}. The leading command
// has already been consumed.
func (p *parser) parseSqrt(offset int) (Expr, error) {
	if p.cur.kind == tokLBracket {
		// Optional index argument.
		p.advance()
		idxChildren, err := p.parseSequence(tokRBracket)
		if err != nil {
			return nil, err
		}
		if p.cur.kind != tokRBracket {
			return nil, p.errorf(offset, "\\sqrt: missing ']' for index argument")
		}
		p.advance()
		index := flatten(idxChildren)
		body, err := p.parseRequiredGroup("\\sqrt", offset, "body")
		if err != nil {
			return nil, err
		}
		return NthRoot{Index: index, Body: body}, nil
	}
	body, err := p.parseRequiredGroup("\\sqrt", offset, "body")
	if err != nil {
		return nil, err
	}
	return Sqrt{Body: body}, nil
}

// parseRequiredGroup parses a brace-delimited argument to a command. cmd and
// offset are used only for error messages; role names the argument slot.
func (p *parser) parseRequiredGroup(cmd string, offset int, role string) (Expr, error) {
	if p.cur.kind != tokLBrace {
		return nil, p.errorf(p.cur.offset, "%s: expected '{' for %s argument (after offset %d)", cmd, role, offset)
	}
	p.advance()
	children, err := p.parseSequence(tokRBrace)
	if err != nil {
		return nil, err
	}
	if p.cur.kind != tokRBrace {
		return nil, p.errorf(offset, "%s: unbalanced '{' in %s argument", cmd, role)
	}
	p.advance()
	return flatten(children), nil
}

// flatten collapses a child list into a single Expr: empty becomes an empty
// Group, length-one returns the single element directly, longer lists are
// wrapped in a Group. Used to keep argument shapes ergonomic for callers.
func flatten(children []Expr) Expr {
	switch len(children) {
	case 0:
		return Group{}
	case 1:
		return children[0]
	default:
		return Group{Children: children}
	}
}
