package hyphenation

// Liang's hyphenation algorithm: at each position inside the word
// (with leading and trailing periods to anchor pattern matching),
// pick the maximum digit score from any matching pattern, then take
// odd scores as valid break points.
//
// The algorithm is implementation-independent; the quality of the
// output depends on the pattern set. Kardec ships a curated subset
// of the standard English (hyph-en-us) patterns covering common
// prefixes, suffixes, and consonant-pair splits. Callers needing
// full quality can extend the table at startup with their own
// pattern bytes:
//
//	hyphenation.Register("en", additionalPatterns)
//
// References:
//   - Knuth, D. E., "The TeXbook", Appendix H (1984)
//   - Liang, F. M., "Word Hy-phen-a-tion by Com-put-er" (1983 thesis)
//   - hyph-utf8 project (CTAN) — full multilingual pattern files

// liangBreakPoints applies the Liang algorithm to word using the
// patterns registered for lang. Returns the byte offsets at which a
// break is preferred (positions where the max pattern score is odd
// and produces fragments of ≥ 3 characters on each side).
//
// Returns nil when no patterns are registered for lang or the word
// is too short to split usefully.
func liangBreakPoints(word, lang string) []int {
	patterns := patternsFor(lang)
	if len(patterns) == 0 || len(word) < 6 {
		return nil
	}
	// Anchor with periods so prefix / suffix patterns ".con3" /
	// "tion4." can match at boundaries.
	dotted := "." + word + "."
	scores := make([]int, len(dotted)+1)
	for i := 0; i < len(dotted); i++ {
		for j := i + 1; j <= len(dotted); j++ {
			sub := dotted[i:j]
			pat, ok := patterns[sub]
			if !ok {
				continue
			}
			applyPattern(scores, pat, i)
		}
	}
	// Convert scores back to offsets in the original (un-dotted)
	// word. scores[k] corresponds to the gap before dotted[k]; the
	// gap before the leading "." is scores[0], between '.' and the
	// first letter is scores[1], etc. A break in word at position p
	// corresponds to scores[p+1].
	var out []int
	for p := 3; p <= len(word)-3; p++ {
		if scores[p+1]%2 == 1 {
			out = append(out, p)
		}
	}
	return out
}

// applyPattern walks pat (a sequence of letters interleaved with
// digit scores), takes the max score at each position relative to
// offset, and writes it back to scores. The pattern's letters are
// already known to match the slice they were keyed by — the helper
// only walks the digit positions.
func applyPattern(scores []int, pat string, offset int) {
	pos := offset
	for i := 0; i < len(pat); i++ {
		c := pat[i]
		if c >= '0' && c <= '9' {
			score := int(c - '0')
			if pos < len(scores) && score > scores[pos] {
				scores[pos] = score
			}
			continue
		}
		pos++
	}
}
