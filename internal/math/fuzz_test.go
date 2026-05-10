package math

import "testing"

// FuzzParse drives random / corpus-derived input through the math
// parser. The test is structural: parsing must never panic, even on
// adversarial input. Pre-fuzz, an attacker who could supply a math
// expression in a Kardec template (rare but possible) could crash
// the renderer with deep nesting or malformed escapes; this fuzz
// keeps that surface honest.
func FuzzParse(f *testing.F) {
	// Seed corpus: a mix of valid expressions and shapes that
	// exercised parser bugs in development.
	for _, seed := range []string{
		"a^2 + b^2 = c^2",
		`\frac{1}{2}`,
		`\sqrt{\pi}`,
		`\int_0^\infty e^{-x^2} dx`,
		`\alpha + \beta`,
		"",
		`\`,             // lone backslash
		`\unknown`,      // unrecognised command
		`{{{`,           // unbalanced braces
		`a^^^^^^2`,      // chained ^
		`\frac{1}`,      // missing denominator
		`\sqrt`,         // missing body
		`x_1^2_2`,       // ambiguous sub/sup
		`\alpha\beta\gamma\delta`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, src string) {
		_, _ = Parse(src) // parse must never panic
	})
}
