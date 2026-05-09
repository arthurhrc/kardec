package math

// SymbolCategory is the coarse typographic class assigned to a known LaTeX
// command. The math typography track uses it to pick spacing rules and to
// route the glyph through the correct font fallback chain.
type SymbolCategory uint8

// Recognized categories. The zero value is reserved for unknown symbols and
// is never returned together with ok==true from LookupSymbol.
const (
	// CategoryGreek covers lowercase and uppercase greek letters.
	CategoryGreek SymbolCategory = iota + 1
	// CategoryRelation covers relational operators rendered as a single glyph
	// (\leq, \geq, \neq, \approx, \to, ...).
	CategoryRelation
	// CategoryBinaryOp covers binary operators rendered as a single glyph
	// (\pm, \mp, \cdot, \times, ...).
	CategoryBinaryOp
	// CategoryBigOp covers large operators that participate in BigOp nodes
	// (\sum, \int, \prod, ...).
	CategoryBigOp
	// CategorySymbol covers everything else with a known Unicode rune that
	// does not fit one of the more specific classes (\infty, \partial, ...).
	CategorySymbol
)

// SymbolInfo is the value side of the symbol table. The Rune is the canonical
// Unicode codepoint the typography track should render; the Category drives
// spacing and font selection.
type SymbolInfo struct {
	// Rune is the canonical Unicode rune for the symbol.
	Rune rune
	// Category is the coarse typographic class.
	Category SymbolCategory
}

// symbolTable maps a LaTeX command (with leading backslash) to its canonical
// Unicode rune and category. The set covers the v1 parser surface; new
// entries can be added without breaking callers.
var symbolTable = map[string]SymbolInfo{
	// Lowercase Greek letters (U+03B1 .. U+03C9).
	"\\alpha":      {0x03B1, CategoryGreek},
	"\\beta":       {0x03B2, CategoryGreek},
	"\\gamma":      {0x03B3, CategoryGreek},
	"\\delta":      {0x03B4, CategoryGreek},
	"\\epsilon":    {0x03F5, CategoryGreek},
	"\\varepsilon": {0x03B5, CategoryGreek},
	"\\zeta":       {0x03B6, CategoryGreek},
	"\\eta":        {0x03B7, CategoryGreek},
	"\\theta":      {0x03B8, CategoryGreek},
	"\\vartheta":   {0x03D1, CategoryGreek},
	"\\iota":       {0x03B9, CategoryGreek},
	"\\kappa":      {0x03BA, CategoryGreek},
	"\\lambda":     {0x03BB, CategoryGreek},
	"\\mu":         {0x03BC, CategoryGreek},
	"\\nu":         {0x03BD, CategoryGreek},
	"\\xi":         {0x03BE, CategoryGreek},
	"\\omicron":    {0x03BF, CategoryGreek},
	"\\pi":         {0x03C0, CategoryGreek},
	"\\varpi":      {0x03D6, CategoryGreek},
	"\\rho":        {0x03C1, CategoryGreek},
	"\\varrho":     {0x03F1, CategoryGreek},
	"\\sigma":      {0x03C3, CategoryGreek},
	"\\varsigma":   {0x03C2, CategoryGreek},
	"\\tau":        {0x03C4, CategoryGreek},
	"\\upsilon":    {0x03C5, CategoryGreek},
	"\\phi":        {0x03D5, CategoryGreek},
	"\\varphi":     {0x03C6, CategoryGreek},
	"\\chi":        {0x03C7, CategoryGreek},
	"\\psi":        {0x03C8, CategoryGreek},
	"\\omega":      {0x03C9, CategoryGreek},

	// Uppercase Greek letters (only the ones LaTeX provides distinct
	// commands for; the rest borrow Latin shapes by convention).
	"\\Gamma":   {0x0393, CategoryGreek},
	"\\Delta":   {0x0394, CategoryGreek},
	"\\Theta":   {0x0398, CategoryGreek},
	"\\Lambda":  {0x039B, CategoryGreek},
	"\\Xi":      {0x039E, CategoryGreek},
	"\\Pi":      {0x03A0, CategoryGreek},
	"\\Sigma":   {0x03A3, CategoryGreek},
	"\\Upsilon": {0x03A5, CategoryGreek},
	"\\Phi":     {0x03A6, CategoryGreek},
	"\\Psi":     {0x03A8, CategoryGreek},
	"\\Omega":   {0x03A9, CategoryGreek},

	// Big operators (used as the head of a BigOp node).
	"\\sum":  {0x2211, CategoryBigOp},
	"\\int":  {0x222B, CategoryBigOp},
	"\\prod": {0x220F, CategoryBigOp},

	// Binary operators with a dedicated glyph.
	"\\pm":    {0x00B1, CategoryBinaryOp},
	"\\mp":    {0x2213, CategoryBinaryOp},
	"\\cdot":  {0x22C5, CategoryBinaryOp},
	"\\times": {0x00D7, CategoryBinaryOp},

	// Relational operators.
	"\\leq":        {0x2264, CategoryRelation},
	"\\geq":        {0x2265, CategoryRelation},
	"\\neq":        {0x2260, CategoryRelation},
	"\\approx":     {0x2248, CategoryRelation},
	"\\to":         {0x2192, CategoryRelation},
	"\\rightarrow": {0x2192, CategoryRelation},
	"\\leftarrow":  {0x2190, CategoryRelation},

	// Miscellaneous symbols.
	"\\infty":   {0x221E, CategorySymbol},
	"\\partial": {0x2202, CategorySymbol},
}

// LookupSymbol returns the canonical SymbolInfo for the given LaTeX command,
// including the leading backslash (e.g. "\\alpha"). The second return value
// is false when the command is not in the table; callers may still emit an
// Atom carrying the literal command string for graceful fallback rendering.
func LookupSymbol(name string) (SymbolInfo, bool) {
	info, ok := symbolTable[name]
	return info, ok
}

// IsBigOpCommand reports whether name is a known LaTeX big-operator command
// (\sum, \int, \prod, ...). The parser uses this to decide between an Atom
// node and a BigOp node when it sees a known command.
func IsBigOpCommand(name string) bool {
	info, ok := symbolTable[name]
	return ok && info.Category == CategoryBigOp
}
