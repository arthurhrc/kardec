package kardec

// Bibliography + citation primitives. Numeric (Vancouver-style)
// citations: every distinct key cited via Cite gets the next 1-based
// number, reused on subsequent Cite calls of the same key. Bibliography
// emits a "References" section with one paragraph per supplied entry,
// sorted in citation order with uncited entries appended at the end.
//
// Each emitted entry is preceded by an Anchor named
// "kardec-bib-<N>" so the [N] link Cite produces resolves to the
// matching paragraph in the rendered PDF.

// BibEntry is the user-supplied bibliography record. Fields are kept
// shallow and string-typed on purpose: composing the emitted line is
// the job of the renderer, not the BibEntry. Callers needing fine
// formatting control can override Bibliography's emit step by passing
// a pre-built block sequence instead — a v0.9 extension.
type BibEntry struct {
	Key     string // unique citation key (e.g., "Knuth1984")
	Author  string // "Knuth, D."
	Title   string // "Literate Programming"
	Year    int    // 1984
	Journal string // "The Computer Journal"
	Volume  string // "27"
	Pages   string // "97-111"
	URL     string // "https://..."
}

// BibAnchorPrefix is the leading string of the anchor names emitted
// before each Bibliography entry. Exposed so callers building manual
// cross-links know the convention.
const BibAnchorPrefix = "kardec-bib-"

// BibAnchorName returns the anchor name Bibliography emits for the
// given 1-based citation number.
func BibAnchorName(num int) string { return BibAnchorPrefix + itoaSmall(num) }

// Cite returns a Run carrying the canonical "[N]" reference for key,
// where N is assigned on the first call to Cite for that key and
// reused on subsequent calls. The Run carries a hyperlink to the
// matching bibliography entry; resolution waits for Bibliography to
// emit the anchor at build time.
//
// Citing an unknown key still allocates a number — the bibliography
// builder is allowed to come after the citations in source order.
// Callers can audit unresolved keys by comparing Document.CitedKeys
// (introduced as needed) against the entries they pass to
// Bibliography.
func (d *Document) Cite(key string) Run {
	if d.citations == nil {
		d.citations = make(map[string]int)
	}
	num, ok := d.citations[key]
	if !ok {
		num = len(d.citationOrder) + 1
		d.citations[key] = num
		d.citationOrder = append(d.citationOrder, key)
	}
	return Run{
		text: "[" + itoaSmall(num) + "]",
		link: "#" + BibAnchorName(num),
	}
}

// CitedKeys returns the citation keys in the order they were first
// referenced via Cite. Useful for cross-checking against the entries
// supplied to Bibliography.
func (d *Document) CitedKeys() []string {
	out := make([]string, len(d.citationOrder))
	copy(out, d.citationOrder)
	return out
}

// Bibliography appends a "References" heading followed by one
// paragraph per entry. Entries are emitted in citation order — the
// sequence in which Cite first referenced their keys — with any
// uncited entries appended at the end.
//
// Each entry's anchor matches BibAnchorName(N) so the [N] link Cite
// returned earlier resolves correctly. The emitter always allocates
// numbers up to len(entries) so even uncited entries get an anchor
// the user can link to manually.
//
// Format: "[N] Author. Title. Journal, Volume(Pages), Year. URL."
// — Vancouver-ish, easy to scan, no italic/bold for portability.
// Callers wanting custom formatting can pre-build blocks themselves
// and use Cite directly with manual anchors.
func (d *Document) Bibliography(entries ...BibEntry) *Document {
	if len(entries) == 0 {
		return d
	}
	d.Heading(2, Text("References"))

	byKey := make(map[string]BibEntry, len(entries))
	for _, e := range entries {
		byKey[e.Key] = e
	}
	emitted := make(map[string]bool, len(entries))

	// Citation-order pass: emit every cited entry that the caller
	// also supplied. Pre-allocate citation numbers for any keys
	// that appeared via Cite but had no matching BibEntry; the
	// anchor still emits with placeholder text so the link
	// resolves to "[?key]".
	for _, key := range d.citationOrder {
		entry, ok := byKey[key]
		if ok {
			d.appendBibEntry(d.citations[key], entry)
			emitted[key] = true
			continue
		}
		// Cited but missing entry — emit a placeholder line so
		// the link target still exists.
		d.appendBibEntry(d.citations[key], BibEntry{Key: key, Title: "[missing entry: " + key + "]"})
	}

	// Trailing pass: uncited entries get appended in order, each
	// allocated the next citation number so they remain
	// link-targetable from prose written after the Bibliography.
	for _, e := range entries {
		if emitted[e.Key] {
			continue
		}
		num := len(d.citationOrder) + 1
		if d.citations == nil {
			d.citations = make(map[string]int)
		}
		d.citations[e.Key] = num
		d.citationOrder = append(d.citationOrder, e.Key)
		d.appendBibEntry(num, e)
	}
	return d
}

// appendBibEntry emits the anchor + formatted paragraph for one
// bibliography entry. Pulled out so the in-order and trailing
// passes share a single emitter.
func (d *Document) appendBibEntry(num int, e BibEntry) {
	d.append(Anchor{name: BibAnchorName(num)})
	prefix := "[" + itoaSmall(num) + "] "
	body := formatBibEntryBody(e)
	d.append(Paragraph{
		runs:      []Run{Bold(prefix), Text(body)},
		styleName: StyleDefault,
	})
}

// formatBibEntryBody composes the visible text of one entry. Empty
// fields drop out so a sparsely populated entry still reads well.
func formatBibEntryBody(e BibEntry) string {
	parts := []string{}
	if e.Author != "" {
		parts = append(parts, e.Author + ".")
	}
	if e.Title != "" {
		parts = append(parts, e.Title + ".")
	}
	journal := e.Journal
	if e.Volume != "" {
		journal += " " + e.Volume
	}
	if e.Pages != "" {
		journal += "(" + e.Pages + ")"
	}
	if journal != "" {
		journal = trimSpaceLeft(journal)
		if e.Year > 0 {
			journal += ", " + itoaSmall(e.Year)
		}
		parts = append(parts, journal+".")
	} else if e.Year > 0 {
		parts = append(parts, itoaSmall(e.Year)+".")
	}
	if e.URL != "" {
		parts = append(parts, e.URL)
	}
	return joinWithSpace(parts)
}

// trimSpaceLeft removes leading ASCII spaces from s without pulling
// in strings.TrimLeft — keeps this file dependency-free.
func trimSpaceLeft(s string) string {
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	return s
}

// joinWithSpace concatenates parts separated by single spaces.
// Equivalent to strings.Join(parts, " ") but inlined to avoid the
// import for one call site.
func joinWithSpace(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += " " + parts[i]
	}
	return out
}
