package math

import (
	"unicode"
	"unicode/utf8"
)

// tokenKind classifies a token produced by the lexer.
type tokenKind uint8

const (
	tokEOF tokenKind = iota
	// tokCommand is a backslash-prefixed LaTeX command name (e.g. "\alpha").
	// Single-character commands like "\\" are also reported here.
	tokCommand
	// tokLBrace is "{".
	tokLBrace
	// tokRBrace is "}".
	tokRBrace
	// tokLBracket is "[".
	tokLBracket
	// tokRBracket is "]".
	tokRBracket
	// tokCaret is "^".
	tokCaret
	// tokUnderscore is "_".
	tokUnderscore
	// tokIdent is a single Latin letter used as an identifier.
	tokIdent
	// tokNumber is a contiguous digit run, possibly with one decimal point.
	tokNumber
	// tokOp is a single-character operator (+, -, =, <, >, *, /).
	tokOp
	// tokOther is any other single-character literal not covered above. The
	// parser may surface or skip it depending on context.
	tokOther
)

// token is the unit produced by the lexer. Value carries the source text
// covered by the token; Offset is the byte offset of the first rune.
type token struct {
	kind   tokenKind
	value  string
	offset int
}

// lexer is a tiny stateful tokenizer over a UTF-8 source string. It exposes
// peek and next semantics so the parser can implement single-token lookahead
// without buffering more than necessary.
type lexer struct {
	src string
	pos int // byte offset of the next rune to consume
}

// newLexer returns a lexer positioned at the start of src.
func newLexer(src string) *lexer {
	return &lexer{src: src}
}

// next returns the next token, advancing the position past it. Whitespace
// runs are silently skipped. At the end of input, a tokEOF with an empty
// Value is returned repeatedly.
func (l *lexer) next() token {
	l.skipWhitespace()
	if l.pos >= len(l.src) {
		return token{kind: tokEOF, offset: l.pos}
	}
	start := l.pos
	r, size := utf8.DecodeRuneInString(l.src[l.pos:])
	switch {
	case r == '\\':
		return l.lexCommand(start)
	case r == '{':
		l.pos += size
		return token{kind: tokLBrace, value: "{", offset: start}
	case r == '}':
		l.pos += size
		return token{kind: tokRBrace, value: "}", offset: start}
	case r == '[':
		l.pos += size
		return token{kind: tokLBracket, value: "[", offset: start}
	case r == ']':
		l.pos += size
		return token{kind: tokRBracket, value: "]", offset: start}
	case r == '^':
		l.pos += size
		return token{kind: tokCaret, value: "^", offset: start}
	case r == '_':
		l.pos += size
		return token{kind: tokUnderscore, value: "_", offset: start}
	case isOperatorRune(r):
		l.pos += size
		return token{kind: tokOp, value: string(r), offset: start}
	case unicode.IsDigit(r):
		return l.lexNumber(start)
	case isIdentRune(r):
		l.pos += size
		return token{kind: tokIdent, value: string(r), offset: start}
	default:
		l.pos += size
		return token{kind: tokOther, value: string(r), offset: start}
	}
}

// skipWhitespace advances past any whitespace runes.
func (l *lexer) skipWhitespace() {
	for l.pos < len(l.src) {
		r, size := utf8.DecodeRuneInString(l.src[l.pos:])
		if !unicode.IsSpace(r) {
			return
		}
		l.pos += size
	}
}

// lexCommand consumes a "\name" sequence. Names are runs of Latin letters;
// when no letters follow the backslash, a single-character command (e.g.
// "\,") is returned instead so unknown punctuation does not stall the lexer.
func (l *lexer) lexCommand(start int) token {
	// Skip the backslash itself.
	l.pos++
	if l.pos >= len(l.src) {
		return token{kind: tokCommand, value: "\\", offset: start}
	}
	r, size := utf8.DecodeRuneInString(l.src[l.pos:])
	if !isCommandLetter(r) {
		// Single-character command like "\\" or "\,".
		l.pos += size
		return token{kind: tokCommand, value: l.src[start : start+1+size], offset: start}
	}
	for l.pos < len(l.src) {
		rr, sz := utf8.DecodeRuneInString(l.src[l.pos:])
		if !isCommandLetter(rr) {
			break
		}
		l.pos += sz
	}
	return token{kind: tokCommand, value: l.src[start:l.pos], offset: start}
}

// lexNumber consumes a digit run, optionally including one decimal point.
func (l *lexer) lexNumber(start int) token {
	dotSeen := false
	for l.pos < len(l.src) {
		r, size := utf8.DecodeRuneInString(l.src[l.pos:])
		switch {
		case unicode.IsDigit(r):
			l.pos += size
		case r == '.' && !dotSeen:
			// Look ahead: only treat as decimal if a digit follows.
			next, _ := utf8.DecodeRuneInString(l.src[l.pos+size:])
			if !unicode.IsDigit(next) {
				return token{kind: tokNumber, value: l.src[start:l.pos], offset: start}
			}
			dotSeen = true
			l.pos += size
		default:
			return token{kind: tokNumber, value: l.src[start:l.pos], offset: start}
		}
	}
	return token{kind: tokNumber, value: l.src[start:l.pos], offset: start}
}

// isCommandLetter reports whether r may appear in a multi-letter LaTeX
// command name (the body after the leading backslash).
func isCommandLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isIdentRune reports whether r may appear as a single-letter identifier in
// math source. Only Latin letters qualify; greek letters are commands.
func isIdentRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isOperatorRune reports whether r should be emitted as a tokOp.
func isOperatorRune(r rune) bool {
	switch r {
	case '+', '-', '=', '<', '>', '*', '/':
		return true
	default:
		return false
	}
}
