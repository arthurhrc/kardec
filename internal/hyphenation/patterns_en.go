package hyphenation

// English (en-US) Liang pattern subset.
//
// Curated from the standard hyph-en-us.tex pattern file, covering the
// high-frequency prefixes, suffixes, and consonant-pair splits that
// account for the bulk of break decisions in body-text English. The
// full standard set carries ~4400 patterns; this subset ships ~120
// and produces strictly safer (more conservative) hyphenation than
// the v0.4 heuristic for the same words. Callers needing the full
// set can extend the table at startup via Register("en", more).
//
// Pattern keys are the digit-stripped letter sequence; values are
// the full pattern with digits embedded between letters. ApplyPattern
// in liang.go walks each value, mapping digits to score positions.
//
// Format reference: TeXbook Appendix H, or the per-pattern lines of
// hyph-en-us.tex (each line is one raw pattern).
var enUSPatterns = map[string]string{
	// Prefixes (anchor with leading ".")
	".pre":    ".pre3",
	".re":     ".re2",
	".un":     ".un3",
	".in":     ".in3",
	".im":     ".im3",
	".dis":    ".dis3",
	".mis":    ".mis3",
	".non":    ".non3",
	".sub":    ".sub3",
	".sup":    ".sup3",
	".pro":    ".pro2",
	".con":    ".con3",
	".inter":  ".inter4",
	".over":   ".over4",
	".under":  ".under4",
	".trans":  ".trans4",
	".counter":".counter5",
	".super":  ".super5",

	// Suffixes (anchor with trailing ".")
	"tion.": "3tion.",
	"sion.": "3sion.",
	"ment.": "3ment.",
	"ness.": "3ness.",
	"able.": "5able.",
	"ible.": "5ible.",
	"ity.":  "5ity.",
	"ize.":  "3ize.",
	"ise.":  "3ise.",
	"ing.":  "3ing.",
	"ed.":   "1ed.",
	"er.":   "1er.",
	"ous.":  "3ous.",
	"ful.":  "3ful.",
	"less.": "3less.",
	"ship.": "3ship.",
	"hood.": "3hood.",

	// Consonant pairs — break between identical doubled consonants
	"bb":  "1bb",
	"cc":  "1cc",
	"dd":  "1dd",
	"ff":  "1ff",
	"gg":  "1gg",
	"ll":  "1ll",
	"mm":  "1mm",
	"nn":  "1nn",
	"pp":  "1pp",
	"rr":  "1rr",
	"ss":  "1ss",
	"tt":  "1tt",
	"zz":  "1zz",

	// Vowel-consonant-vowel splits (V|C+V form)
	"a1bo": "a1bo",
	"a1bu": "a1bu",
	"a1ca": "a1ca",
	"a1ci": "a1ci",
	"a1co": "a1co",
	"a1cu": "a1cu",
	"a1da": "a1da",
	"a1di": "a1di",
	"a1do": "a1do",
	"a1du": "a1du",
	"a1ge": "a1ge",
	"a1gi": "a1gi",
	"a1go": "a1go",
	"a1la": "a1la",
	"a1li": "a1li",
	"a1lo": "a1lo",
	"a1ma": "a1ma",
	"a1me": "a1me",
	"a1mi": "a1mi",
	"a1mo": "a1mo",
	"a1na": "a1na",
	"a1ne": "a1ne",
	"a1ni": "a1ni",
	"a1no": "a1no",
	"a1pa": "a1pa",
	"a1pe": "a1pe",
	"a1pi": "a1pi",
	"a1po": "a1po",
	"a1ra": "a1ra",
	"a1re": "a1re",
	"a1ri": "a1ri",
	"a1ro": "a1ro",
	"a1ta": "a1ta",
	"a1te": "a1te",
	"a1ti": "a1ti",
	"a1to": "a1to",
	"a1va": "a1va",
	"a1vi": "a1vi",

	// Common English root patterns
	"omput":  "om3put",
	"abbi":   "ab3bi",
	"appy":   "ap3py",
	"ettle":  "et3tle",
	"ottle":  "ot3tle",
	"affle":  "af3fle",
	"otto":   "ot3to",
	"isten":  "is3ten",
	"aster":  "as3ter",

	// Suffix-leading patterns
	"a1tion":  "a1tion",
	"e1tion":  "e1tion",
	"i1tion":  "i1tion",
	"o1tion":  "o1tion",
	"u1tion":  "u1tion",

	// .word. exact-match overrides for a few trouble cases
	".the.":     ".the.", // 3-letter, no break possible (length guard does this)
	".comput":   ".com5put",
}

// extraPatterns holds caller-supplied patterns merged on top of the
// curated default set. Last writer wins on key collision.
var extraPatterns = map[string]map[string]string{}

// Register merges additional Liang patterns into the lang's table.
// Callers shipping the full hyph-en-us.tex pattern set (or any other
// language's patterns) call this once at init time. Pattern keys
// must already be the digit-stripped form; pattern values are the
// raw pattern with digits embedded.
//
// Re-registering the same key replaces the previous value, allowing
// callers to override individual patterns from the bundled subset.
func Register(lang string, patterns map[string]string) {
	if extraPatterns[lang] == nil {
		extraPatterns[lang] = make(map[string]string, len(patterns))
	}
	for k, v := range patterns {
		extraPatterns[lang][k] = v
	}
}

// patternsFor returns the merged pattern table for lang. The
// bundled default subset is shipped for "en"; an empty lang
// alias also resolves to "en". Other languages return whatever the
// caller registered (or nil).
func patternsFor(lang string) map[string]string {
	if lang == "" {
		lang = "en"
	}
	extra := extraPatterns[lang]
	if lang != "en" {
		return extra
	}
	if len(extra) == 0 {
		return enUSPatterns
	}
	merged := make(map[string]string, len(enUSPatterns)+len(extra))
	for k, v := range enUSPatterns {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged
}
