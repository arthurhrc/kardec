package typography

// latexMathSymbols maps a backslash-prefixed LaTeX command name to the
// Unicode rune that represents it in a math font (Latin Modern Math or any
// other STIX-style face). The set is deliberately scoped to what the math
// layout / parser tracks need on day one: lowercase + uppercase Greek, the
// big operators, the common relations, arrows, and a handful of misc.
// constants. Anything outside this set falls through GlyphFor as a literal
// rune (ASCII pass-through).
//
// Redundancy note
// ---------------
// A richer symbol table is being produced in parallel under
// `internal/math/symbols.go` by the math-parser track. This worktree was
// branched before that work landed, so the local map is duplicated to
// keep `internal/typography` self-contained. When the integration step
// merges `internal/math/symbols.go` into the tree, this table SHOULD be
// dropped and `GlyphFor` reworked to call into the canonical map.
var latexMathSymbols = map[string]rune{
	// --- lowercase Greek -------------------------------------------------
	"\\alpha":      'α',
	"\\beta":       'β',
	"\\gamma":      'γ',
	"\\delta":      'δ',
	"\\epsilon":    'ϵ',
	"\\varepsilon": 'ε',
	"\\zeta":       'ζ',
	"\\eta":        'η',
	"\\theta":      'θ',
	"\\vartheta":   'ϑ',
	"\\iota":       'ι',
	"\\kappa":      'κ',
	"\\lambda":     'λ',
	"\\mu":         'μ',
	"\\nu":         'ν',
	"\\xi":         'ξ',
	"\\omicron":    'ο',
	"\\pi":         'π',
	"\\varpi":      'ϖ',
	"\\rho":        'ρ',
	"\\varrho":     'ϱ',
	"\\sigma":      'σ',
	"\\varsigma":   'ς',
	"\\tau":        'τ',
	"\\upsilon":    'υ',
	"\\phi":        'ϕ',
	"\\varphi":     'φ',
	"\\chi":        'χ',
	"\\psi":        'ψ',
	"\\omega":      'ω',

	// --- uppercase Greek -------------------------------------------------
	"\\Gamma":   'Γ',
	"\\Delta":   'Δ',
	"\\Theta":   'Θ',
	"\\Lambda": 'Λ',
	"\\Xi":      'Ξ',
	"\\Pi":      'Π',
	"\\Sigma":   'Σ',
	"\\Upsilon": 'Υ',
	"\\Phi":     'Φ',
	"\\Psi":     'Ψ',
	"\\Omega":   'Ω',

	// --- big operators ---------------------------------------------------
	"\\sum":      '∑',
	"\\prod":     '∏',
	"\\coprod":   '∐',
	"\\int":      '∫',
	"\\iint":     '∬',
	"\\iiint":    '∭',
	"\\oint":     '∮',
	"\\bigcup":   '⋃',
	"\\bigcap":   '⋂',
	"\\bigvee":   '⋁',
	"\\bigwedge": '⋀',
	"\\bigodot":  '⨀',
	"\\bigotimes": '⨂',
	"\\bigoplus":  '⨁',

	// --- relations -------------------------------------------------------
	"\\leq":     '≤',
	"\\le":      '≤',
	"\\geq":     '≥',
	"\\ge":      '≥',
	"\\neq":     '≠',
	"\\ne":      '≠',
	"\\equiv":   '≡',
	"\\approx":  '≈',
	"\\sim":     '∼',
	"\\simeq":   '≃',
	"\\propto":  '∝',
	"\\subset":  '⊂',
	"\\supset":  '⊃',
	"\\subseteq": '⊆',
	"\\supseteq": '⊇',
	"\\in":      '∈',
	"\\notin":   '∉',
	"\\ni":      '∋',

	// --- arrows ----------------------------------------------------------
	"\\to":           '→',
	"\\rightarrow":   '→',
	"\\leftarrow":    '←',
	"\\Rightarrow":   '⇒',
	"\\Leftarrow":    '⇐',
	"\\leftrightarrow": '↔',
	"\\Leftrightarrow": '⇔',
	"\\mapsto":       '↦',

	// --- binary operators ------------------------------------------------
	"\\pm":       '±',
	"\\mp":       '∓',
	"\\times":    '×',
	"\\div":      '÷',
	"\\cdot":     '⋅',
	"\\ast":      '∗',
	"\\star":     '⋆',
	"\\circ":     '∘',
	"\\bullet":   '∙',
	"\\oplus":    '⊕',
	"\\ominus":   '⊖',
	"\\otimes":   '⊗',
	"\\oslash":   '⊘',
	"\\odot":     '⊙',
	"\\cap":      '∩',
	"\\cup":      '∪',
	"\\setminus": '∖',

	// --- misc symbols ----------------------------------------------------
	"\\infty":    '∞',
	"\\partial":  '∂',
	"\\nabla":    '∇',
	"\\forall":   '∀',
	"\\exists":   '∃',
	"\\emptyset": '∅',
	"\\hbar":     'ℏ',
	"\\ell":      'ℓ',
	"\\Re":       'ℜ',
	"\\Im":       'ℑ',
	"\\aleph":    'ℵ',
	"\\sqrt":     '√',
	"\\neg":      '¬',
	"\\lnot":     '¬',
	"\\dots":     '…',
	"\\ldots":    '…',
	"\\cdots":    '⋯',
}

// lookupLatexSymbol returns the rune mapped to a backslash-prefixed LaTeX
// command. The boolean is false when the command is unknown.
//
// Callers SHOULD prefer this helper over poking at the map directly so the
// consolidation against `internal/math/symbols.go` stays a single edit.
func lookupLatexSymbol(cmd string) (rune, bool) {
	r, ok := latexMathSymbols[cmd]
	return r, ok
}
