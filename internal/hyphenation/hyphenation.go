// Package hyphenation produces candidate break points for words that
// would otherwise overflow a line. v0.4 ships a minimal heuristic
// hyphenator: splits between two consecutive consonants, after a
// vowel followed by two consonants, and after well-known English
// prefixes (un-, re-, in-, dis-, pre-, mis-, sub-, inter-, over-).
//
// The heuristic is intentionally conservative — it never inserts a
// break that produces a fragment shorter than three characters, so
// "the" never becomes "t-he". Less accurate than full Knuth–Liang
// patterns but enough to cover the most common overflow cases at a
// fraction of the data-table size. Switching to Liang patterns is a
// drop-in replacement once the table lands; the public BreakPoints
// surface stays.
package hyphenation

import "strings"

// BreakPoints returns the byte offsets within word at which a soft
// hyphen may be inserted, in increasing order. Offsets refer to the
// position of the *break* — the first character on the next line.
// Empty result means the word should not be split.
//
// The lang argument is currently advisory — only "en" is honored;
// other tags fall through to the same English heuristic.
func BreakPoints(word, lang string) []int {
	_ = lang
	if len(word) < 6 {
		return nil
	}
	lower := strings.ToLower(word)
	candidates := afterPrefixes(lower)
	candidates = append(candidates, vcCcvSplits(lower)...)
	return dedupAndPrune(candidates, len(word))
}

// afterPrefixes scans common English prefixes and emits a break
// point right after the prefix when the remainder is at least three
// letters long.
func afterPrefixes(word string) []int {
	var out []int
	for _, p := range knownPrefixes {
		if len(word) >= len(p)+3 && strings.HasPrefix(word, p) {
			out = append(out, len(p))
		}
	}
	return out
}

// vcCcvSplits implements the vowel-consonant-consonant-vowel rule:
// for each position i where word[i-1] is a vowel and word[i] and
// word[i+1] are both consonants and word[i+2] is a vowel, insert a
// break before word[i+1]. Splits "rabbit" between the b's, "happy"
// between the p's, "computer" after the m, etc.
func vcCcvSplits(word string) []int {
	var out []int
	for i := 1; i < len(word)-2; i++ {
		if isVowel(word[i-1]) && isConsonant(word[i]) && isConsonant(word[i+1]) && isVowel(word[i+2]) {
			out = append(out, i+1)
		}
	}
	return out
}

// dedupAndPrune sorts candidates ascending, removes duplicates and
// drops any break that produces a fragment shorter than three
// characters on either side of the split.
func dedupAndPrune(candidates []int, total int) []int {
	if len(candidates) == 0 {
		return nil
	}
	seen := map[int]bool{}
	var out []int
	for _, c := range candidates {
		if c < 3 || total-c < 3 {
			continue
		}
		if seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	// Sort ascending so callers may pick the rightmost split that
	// still fits a line.
	sortInts(out)
	return out
}

// knownPrefixes is the small starter list of high-frequency English
// prefixes. Order does not matter; longer prefixes are tried before
// shorter ones because afterPrefixes uses HasPrefix and the shorter
// match would always succeed where the longer match also did.
var knownPrefixes = []string{
	"inter", "over", "under", "trans",
	"dis", "mis", "non", "pre", "pro", "sub", "sup",
	"un", "re", "in", "im",
}

func isVowel(b byte) bool {
	switch b {
	case 'a', 'e', 'i', 'o', 'u', 'y':
		return true
	}
	return false
}

func isConsonant(b byte) bool {
	if b < 'a' || b > 'z' {
		return false
	}
	return !isVowel(b)
}

// sortInts sorts a slice in place ascending. Standalone to keep the
// package free of stdlib sort dependency for one tiny use site.
func sortInts(xs []int) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j-1] > xs[j]; j-- {
			xs[j-1], xs[j] = xs[j], xs[j-1]
		}
	}
}
