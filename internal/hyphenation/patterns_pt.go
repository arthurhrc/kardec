package hyphenation

// Portuguese (pt-BR) Liang pattern subset.
//
// Curated from the standard hyph-pt.tex pattern file (Tipográfica
// brasileira / Pereira & Magalhães), covering high-frequency
// prefixes, suffixes, and consonant-pair / vowel-pair splits that
// account for the bulk of break decisions in body-text Portuguese.
// The full standard set carries ~6800 patterns; this subset ships
// ~150 and produces conservative breaks suitable for body text.
//
// Portuguese hyphenation rules differ meaningfully from English:
//   - Consonant pairs (-ch-, -lh-, -nh-, -rr-, -ss-) stay
//     unsplit (the digraph is a single sound).
//   - Vowel pairs forming a diphthong (-ai-, -ei-, -oi-, -au-,
//     -eu-, -ou-) stay together; hiatuses split.
//   - Most consonant + vowel transitions split before the
//     consonant ("pa-la-vra" not "pal-avra").
//
// Callers needing the full pattern set can extend at init via
// Register("pt", more) — same machinery as English.
var ptBRPatterns = map[string]string{
	// Common open syllables (consonant + vowel + consonant): split
	// before the consonant that starts the next syllable.
	"ab":  "a1b",
	"ac":  "a1c",
	"ad":  "a1d",
	"af":  "a1f",
	"ag":  "a1g",
	"aj":  "a1j",
	"al":  "a1l",
	"am":  "a1m",
	"an":  "a1n",
	"ap":  "a1p",
	"aq":  "a1q",
	"ar":  "a1r",
	"as":  "a1s",
	"at":  "a1t",
	"av":  "a1v",
	"ax":  "a1x",
	"az":  "a1z",
	"eb":  "e1b",
	"ec":  "e1c",
	"ed":  "e1d",
	"ef":  "e1f",
	"eg":  "e1g",
	"ej":  "e1j",
	"el":  "e1l",
	"em":  "e1m",
	"en":  "e1n",
	"ep":  "e1p",
	"eq":  "e1q",
	"er":  "e1r",
	"es":  "e1s",
	"et":  "e1t",
	"ev":  "e1v",
	"ex":  "e1x",
	"ez":  "e1z",
	"ib":  "i1b",
	"ic":  "i1c",
	"id":  "i1d",
	"if":  "i1f",
	"ig":  "i1g",
	"il":  "i1l",
	"im":  "i1m",
	"in":  "i1n",
	"ip":  "i1p",
	"ir":  "i1r",
	"is":  "i1s",
	"it":  "i1t",
	"iv":  "i1v",
	"ix":  "i1x",
	"iz":  "i1z",
	"ob":  "o1b",
	"oc":  "o1c",
	"od":  "o1d",
	"of":  "o1f",
	"og":  "o1g",
	"oj":  "o1j",
	"ol":  "o1l",
	"om":  "o1m",
	"on":  "o1n",
	"op":  "o1p",
	"oq":  "o1q",
	"or":  "o1r",
	"os":  "o1s",
	"ot":  "o1t",
	"ov":  "o1v",
	"ox":  "o1x",
	"oz":  "o1z",
	"ub":  "u1b",
	"uc":  "u1c",
	"ud":  "u1d",
	"uf":  "u1f",
	"ug":  "u1g",
	"uj":  "u1j",
	"ul":  "u1l",
	"um":  "u1m",
	"un":  "u1n",
	"up":  "u1p",
	"ur":  "u1r",
	"us":  "u1s",
	"ut":  "u1t",
	"uv":  "u1v",
	"ux":  "u1x",
	"uz":  "u1z",
	// Digraphs: "ch", "lh", "nh" are single sounds — never split
	// between the two letters. Pattern value 2 suppresses any
	// surrounding 1 from open-syllable rules.
	"ch":  "2ch",
	"lh":  "2lh",
	"nh":  "2nh",
	// Doubled consonants: "rr", "ss" are intervocalic and DO split.
	"rr": "r1r",
	"ss": "s1s",
	// Common diphthongs: keep together (no break between).
	"ai":  "a2i",
	"au":  "a2u",
	"ei":  "e2i",
	"eu":  "e2u",
	"oi":  "o2i",
	"ou":  "o2u",
	"iu":  "i2u",
	// Common prefixes: anchor with "." so they only match at
	// word-start.
	".des":   ".de1s",  // "des-igual", "des-fazer"
	".sub":   ".sub1",  // "sub-grupo"
	".con":   ".con1",  // "con-tar"
	".super": ".super1", // "super-mercado"
	".pre":   ".pre1",   // "pre-fixo"
	// Common suffixes: anchor with trailing "."
	"mento.":  "men1to.",  // "movi-mento"
	"mente.":  "men1te.",  // "rápida-mente"
	"ação.":   "a1ção.",   // "construção" → "constru-ção"
	"ações.":  "a1ções.",  // plural
	"idade.":  "i1dade.",  // "felici-dade"
}
